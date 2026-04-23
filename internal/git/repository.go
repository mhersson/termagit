package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/mhersson/termagit/internal/cmdlog"
)

// ErrNotARepo is returned when the path is not inside a git repository.
var ErrNotARepo = errors.New("not a git repository")

// BisectItem represents a single entry in the bisect log.
type BisectItem struct {
	Action     string // "good", "bad", "skip", "start"
	Hash       string // Full commit hash
	AbbrevHash string // Abbreviated hash (7 chars)
	Subject    string // Commit message subject
	Finished   bool   // True for the final identified commit
}

// BisectState represents the current state of a git bisect operation.
type BisectState struct {
	Items    []BisectItem // Bisect log entries
	Current  *LogEntry    // Current commit being tested
	Finished bool         // True if bisect has finished
}

// SequencerItem represents a single entry in the sequencer todo.
type SequencerItem struct {
	Action     string // "pick" or "revert"
	Hash       string // Full commit hash
	AbbrevHash string // Abbreviated hash
	Subject    string // Commit message subject
}

// SequencerState represents the current state of a cherry-pick or revert operation.
type SequencerState struct {
	Operation string          // "cherry-pick" or "revert"
	Items     []SequencerItem // Todo entries
	Current   *SequencerItem  // Currently stopped on
}

// Repository wraps a go-git repository with logging and convenience methods.
type Repository struct {
	raw    *gogit.Repository
	path   string         // absolute working tree root
	gitDir string         // absolute path to .git/
	logger *cmdlog.Logger // may be nil
}

// Open opens a git repository by walking up from path to find .git.
// Returns ErrNotARepo if not inside a git repository.
func Open(path string, logger *cmdlog.Logger) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("get absolute path: %w", err)
	}

	// Walk up the directory tree to find .git
	current := absPath
	for {
		gitDir := filepath.Join(current, ".git")
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			// Found it
			raw, err := gogit.PlainOpen(current)
			if err != nil {
				return nil, fmt.Errorf("open repository: %w", err)
			}
			return &Repository{
				raw:    raw,
				path:   current,
				gitDir: gitDir,
				logger: logger,
			}, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return nil, ErrNotARepo
		}
		current = parent
	}
}

// Wrap creates a Repository from an existing go-git repository.
// This is primarily used for tests with in-memory repositories.
func Wrap(raw *gogit.Repository, path string, logger *cmdlog.Logger) *Repository {
	return &Repository{
		raw:    raw,
		path:   path,
		gitDir: filepath.Join(path, ".git"),
		logger: logger,
	}
}

// Path returns the absolute path to the working tree root.
func (r *Repository) Path() string {
	return r.path
}

// GitDir returns the absolute path to the .git directory.
func (r *Repository) GitDir() string {
	return r.gitDir
}

// HeadInfo returns the current branch name and the subject of HEAD commit.
// For detached HEAD, branch is "HEAD".
// For unborn HEAD (no commits yet), returns branch name and empty subject.
func (r *Repository) HeadInfo(ctx context.Context) (branch, subject string, err error) {
	return r.logOp(ctx, "git rev-parse --abbrev-ref HEAD && git log -1 --format=%s", func() (string, string, error) {
		// Check for unborn HEAD first
		if unborn, branchName := r.isUnbornHEAD(ctx); unborn {
			return branchName, "", nil
		}

		head, err := r.raw.Head()
		if err != nil {
			return "", "", fmt.Errorf("get HEAD: %w", err)
		}

		// Check if detached
		if !head.Name().IsBranch() {
			branch = "HEAD"
		} else {
			branch = head.Name().Short()
		}

		// Get commit message
		commit, err := r.raw.CommitObject(head.Hash())
		if err != nil {
			return "", "", fmt.Errorf("get commit: %w", err)
		}

		// Subject is the first line of the message
		subject = strings.Split(commit.Message, "\n")[0]

		return branch, subject, nil
	})
}

// HeadOID returns the full 40-character hash of HEAD.
// Returns empty string for unborn HEAD (no commits yet).
func (r *Repository) HeadOID(ctx context.Context) (string, error) {
	oid, err := r.logOpSingle(ctx, "git rev-parse HEAD", func() (string, error) {
		// Check for unborn HEAD first
		if unborn, _ := r.isUnbornHEAD(ctx); unborn {
			return "", nil
		}

		head, err := r.raw.Head()
		if err != nil {
			return "", fmt.Errorf("get HEAD: %w", err)
		}
		return head.Hash().String(), nil
	})
	return oid, err
}

// isUnbornHEAD checks if HEAD points to an unborn branch (no commits yet).
// If so, returns (true, branchName). Otherwise returns (false, "").
func (r *Repository) isUnbornHEAD(ctx context.Context) (bool, string) {
	_, err := r.raw.Head()
	if err == nil {
		return false, ""
	}

	// Check if this is plumbing.ErrReferenceNotFound
	if !errors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, ""
	}

	// HEAD doesn't resolve to a commit, but we can still get the symbolic ref
	out, err := r.runGit(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return false, ""
	}

	return true, strings.TrimSpace(out)
}

// AheadBehind returns the number of commits ahead and behind the upstream.
// Returns (0, 0) if there is no upstream configured.
func (r *Repository) AheadBehind(ctx context.Context) (ahead, behind int, err error) {
	// Get current branch
	head, err := r.raw.Head()
	if err != nil {
		return 0, 0, nil //nolint:nilerr // no HEAD (empty repo) → no tracking info
	}

	if !head.Name().IsBranch() {
		return 0, 0, nil // Detached HEAD has no upstream
	}

	// Try to get the upstream reference
	cfg, err := r.raw.Config()
	if err != nil {
		return 0, 0, nil //nolint:nilerr // config unreadable → treat as no upstream
	}

	branchName := head.Name().Short()
	branchCfg, ok := cfg.Branches[branchName]
	if !ok || branchCfg.Remote == "" {
		return 0, 0, nil // No upstream configured
	}

	// Get upstream reference
	remoteBranch := branchCfg.Merge.Short()
	upstreamRef := plumbing.NewRemoteReferenceName(branchCfg.Remote, remoteBranch)

	upstream, err := r.raw.Reference(upstreamRef, true)
	if err != nil {
		return 0, 0, nil //nolint:nilerr // upstream ref not fetched locally → 0,0 is correct
	}

	// Use git rev-list for accurate ahead/behind counting
	aheadCount, behindCount, err := r.countAheadBehind(ctx, head.Hash().String(), upstream.Hash().String())
	if err != nil {
		return 0, 0, nil //nolint:nilerr // rev-list failure → fall back to 0,0
	}

	return aheadCount, behindCount, nil
}

func (r *Repository) countAheadBehind(ctx context.Context, local, upstream string) (ahead, behind int, err error) {
	out, err := r.runGit(ctx, "rev-list", "--count", "--left-right", local+"..."+upstream)
	if err != nil {
		return 0, 0, err
	}

	parts := strings.Fields(strings.TrimSpace(out))
	if len(parts) != 2 {
		return 0, 0, nil
	}

	ahead, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse ahead count %q: %w", parts[0], err)
	}
	behind, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse behind count %q: %w", parts[1], err)
	}
	return ahead, behind, nil
}

// RebaseInProgress returns true if a rebase is in progress.
func (r *Repository) RebaseInProgress() bool {
	// Check for both rebase-merge (interactive) and rebase-apply (am-style)
	if _, err := os.Stat(filepath.Join(r.gitDir, "rebase-merge")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(r.gitDir, "rebase-apply")); err == nil {
		return true
	}
	return false
}

// MergeInProgress returns true if a merge is in progress.
func (r *Repository) MergeInProgress() bool {
	_, err := os.Stat(filepath.Join(r.gitDir, "MERGE_HEAD"))
	return err == nil
}

// CherryPickInProgress returns true if a cherry-pick is in progress.
func (r *Repository) CherryPickInProgress() bool {
	_, err := os.Stat(filepath.Join(r.gitDir, "CHERRY_PICK_HEAD"))
	return err == nil
}

// RevertInProgress returns true if a revert is in progress.
func (r *Repository) RevertInProgress() bool {
	_, err := os.Stat(filepath.Join(r.gitDir, "REVERT_HEAD"))
	return err == nil
}

// BisectInProgress returns true if a bisect is in progress.
func (r *Repository) BisectInProgress() bool {
	_, err := os.Stat(filepath.Join(r.gitDir, "BISECT_LOG"))
	return err == nil
}

// SequencerOperation returns the current sequencer operation if any.
// Returns empty string if no sequencer operation is in progress.
func (r *Repository) SequencerOperation() string {
	todoPath := filepath.Join(r.gitDir, "sequencer", "todo")
	data, err := readGitFile(todoPath)
	if err != nil {
		return ""
	}

	// First line contains the operation
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: "pick abc123 message" or "revert abc123 message"
		parts := strings.Fields(line)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

// ReadMergeState returns merge information when a merge is in progress.
// Returns (head, subject, branch, error):
// - head: the commit hash being merged (from MERGE_HEAD)
// - subject: the first line of the merge message (from MERGE_MSG)
// - branch: the branch name being merged (extracted from MERGE_MSG)
// Returns empty strings when no merge is in progress.
func (r *Repository) ReadMergeState() (head, subject, branch string, err error) {
	// Read MERGE_HEAD
	mergeHeadPath := filepath.Join(r.gitDir, "MERGE_HEAD")
	data, err := readGitFile(mergeHeadPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", "", "", nil // No merge in progress
		}
		return "", "", "", fmt.Errorf("read MERGE_HEAD: %w", err)
	}
	head = strings.TrimSpace(string(data))

	// Read MERGE_MSG
	mergeMsgPath := filepath.Join(r.gitDir, "MERGE_MSG")
	msgData, err := readGitFile(mergeMsgPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return head, "", "", fmt.Errorf("read MERGE_MSG: %w", err)
		}
		// MERGE_MSG doesn't exist, just return head
		return head, "", "", nil
	}

	msg := string(msgData)
	lines := strings.Split(msg, "\n")
	if len(lines) > 0 {
		subject = strings.TrimSpace(lines[0])
	}

	// Extract branch name from subject
	// Format: "Merge branch 'feature'" or "Merge branch 'feature' into main"
	if strings.HasPrefix(subject, "Merge branch '") {
		rest := strings.TrimPrefix(subject, "Merge branch '")
		if idx := strings.Index(rest, "'"); idx > 0 {
			branch = rest[:idx]
		}
	}

	return head, subject, branch, nil
}

// BisectState returns the current state of a git bisect operation.
// Returns empty state if no bisect is in progress.
func (r *Repository) BisectState(ctx context.Context) (BisectState, error) {
	bisectLogPath := filepath.Join(r.gitDir, "BISECT_LOG")
	data, err := readGitFile(bisectLogPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return BisectState{}, nil // No bisect in progress
		}
		return BisectState{}, fmt.Errorf("read BISECT_LOG: %w", err)
	}

	state := BisectState{}
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse comment lines: # good: HASH subject
		if strings.HasPrefix(line, "# ") {
			item := parseBisectComment(line)
			if item != nil {
				state.Items = append(state.Items, *item)
			}
			continue
		}

		// Check for "first bad commit" marker
		if strings.Contains(line, "first bad commit") {
			state.Finished = true
		}
	}

	// Read the current commit being tested from BISECT_EXPECTED_REV
	expectedPath := filepath.Join(r.gitDir, "BISECT_EXPECTED_REV")
	if expectedData, err := readGitFile(expectedPath); err == nil {
		rev := strings.TrimSpace(string(expectedData))
		if rev != "" {
			entry, err := r.commitDetailForBisect(ctx, rev)
			if err == nil {
				state.Current = entry
			}
		}
	}

	return state, nil
}

// commitDetailForBisect returns a LogEntry with fuller details for bisect display.
func (r *Repository) commitDetailForBisect(ctx context.Context, rev string) (*LogEntry, error) {
	out, err := r.runGit(ctx, "show", "--no-patch",
		"--format=%H%n%an%n%ae%n%aI%n%cn%n%ce%n%cI%n%s%n%b", rev)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) < 8 {
		return nil, fmt.Errorf("unexpected show output for %s", rev)
	}

	entry := &LogEntry{
		Hash:           lines[0],
		AuthorName:     lines[1],
		AuthorEmail:    lines[2],
		AuthorDate:     lines[3],
		CommitterName:  lines[4],
		CommitterEmail: lines[5],
		CommitterDate:  lines[6],
		Subject:        lines[7],
	}

	if len(entry.Hash) >= 7 {
		entry.AbbreviatedHash = entry.Hash[:7]
	}

	if len(lines) > 8 {
		entry.Body = strings.Join(lines[8:], "\n")
	}

	return entry, nil
}

// parseBisectComment parses a bisect log comment line.
// Format: # action: HASH subject
func parseBisectComment(line string) *BisectItem {
	line = strings.TrimPrefix(line, "# ")

	// Skip "first bad commit" lines
	if strings.HasPrefix(line, "first bad commit") {
		return nil
	}

	// Find the colon that separates action from the rest
	colonIdx := strings.Index(line, ":")
	if colonIdx < 0 {
		return nil
	}

	action := strings.TrimSpace(line[:colonIdx])
	rest := strings.TrimSpace(line[colonIdx+1:])

	// Valid actions
	switch action {
	case "good", "bad", "skip", "start":
		// OK
	default:
		return nil
	}

	if rest == "" {
		return &BisectItem{Action: action}
	}

	// Split into hash and subject
	parts := strings.SplitN(rest, " ", 2)
	hash := parts[0]
	subject := ""
	if len(parts) > 1 {
		subject = parts[1]
	}

	abbrev := hash
	if len(hash) >= 7 {
		abbrev = hash[:7]
	}

	return &BisectItem{
		Action:     action,
		Hash:       hash,
		AbbrevHash: abbrev,
		Subject:    subject,
	}
}

// SequencerState returns the current state of a cherry-pick or revert operation.
// Returns empty state if no sequencer operation is in progress.
func (r *Repository) SequencerState(ctx context.Context) (SequencerState, error) {
	state := SequencerState{}

	// Determine operation type
	if r.CherryPickInProgress() {
		state.Operation = "cherry-pick"
	} else if r.RevertInProgress() {
		state.Operation = "revert"
	} else {
		return state, nil // No sequencer operation
	}

	// Read sequencer/todo if it exists
	todoPath := filepath.Join(r.gitDir, "sequencer", "todo")
	data, err := readGitFile(todoPath)
	if err != nil {
		// Single cherry-pick/revert without sequencer
		// Read the HEAD file for current operation
		var headPath string
		if state.Operation == "cherry-pick" {
			headPath = filepath.Join(r.gitDir, "CHERRY_PICK_HEAD")
		} else {
			headPath = filepath.Join(r.gitDir, "REVERT_HEAD")
		}

		headData, err := readGitFile(headPath)
		if err != nil {
			return state, nil //nolint:nilerr // HEAD file missing mid-operation → return partial state without commit hash
		}

		hash := strings.TrimSpace(string(headData))
		abbrev := hash
		if len(hash) >= 7 {
			abbrev = hash[:7]
		}

		item := SequencerItem{
			Action:     state.Operation[:1], // "p" or "r"
			Hash:       hash,
			AbbrevHash: abbrev,
		}
		state.Items = []SequencerItem{item}
		state.Current = &state.Items[0]

		return state, nil
	}

	// Parse sequencer/todo
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Format: pick HASH subject
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		action := parts[0]
		hash := parts[1]
		subject := ""
		if len(parts) > 2 {
			subject = strings.Join(parts[2:], " ")
		}

		abbrev := hash
		if len(hash) >= 7 {
			abbrev = hash[:7]
		}

		state.Items = append(state.Items, SequencerItem{
			Action:     action,
			Hash:       hash,
			AbbrevHash: abbrev,
			Subject:    subject,
		})
	}

	if len(state.Items) > 0 {
		state.Current = &state.Items[0]
	}

	return state, nil
}

// runGit executes a git command and returns stdout.
// Logs the command via cmdlog if logger is set.
func (r *Repository) runGit(ctx context.Context, args ...string) (string, error) {
	stdout, _, err := r.runGitFull(ctx, args...)
	return stdout, err
}

// runGitWithEnv executes a git command with extra environment variables.
func (r *Repository) runGitWithEnv(ctx context.Context, env []string, args ...string) error {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path
	cmd.Env = append(os.Environ(), env...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	cmdErr := cmd.Run()

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	r.logGitCmd(start, args, stdout, stderr, cmdErr)

	if cmdErr != nil {
		return fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr), cmdErr)
	}

	return nil
}

// runGitFull executes a git command and returns stdout, stderr, and error.
// The output is returned even when there's an error (for commands like diff --no-index).
func (r *Repository) runGitFull(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	return r.runGitFullWithStdin(ctx, nil, args...)
}

// runGitWithStdin executes a git command with data piped to stdin.
func (r *Repository) runGitWithStdin(ctx context.Context, stdin string, args ...string) (string, error) {
	stdout, _, err := r.runGitFullWithStdin(ctx, strings.NewReader(stdin), args...)
	return stdout, err
}

const maxGitOutput = 64 * 1024 * 1024 // 64 MB

const maxGitFileSize = 10 * 1024 * 1024 // 10 MB

// readGitFile reads a file from the .git directory with a size limit.
func readGitFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxGitFileSize {
		return nil, fmt.Errorf("git file too large (%d bytes): %s", info.Size(), path)
	}
	return os.ReadFile(path)
}

// limitedWriter wraps a bytes.Buffer with a maximum size.
// Writes that would exceed the limit are silently discarded.
type limitedWriter struct {
	buf     bytes.Buffer
	max     int
	overrun bool
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.buf.Len()+len(p) > w.max {
		w.overrun = true
		return len(p), nil
	}
	return w.buf.Write(p)
}

// runGitFullWithStdin executes a git command with optional stdin and returns stdout, stderr, and error.
func (r *Repository) runGitFullWithStdin(ctx context.Context, stdin io.Reader, args ...string) (stdout, stderr string, err error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path

	stdoutBuf := &limitedWriter{max: maxGitOutput}
	stderrBuf := &limitedWriter{max: maxGitOutput}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	if stdin != nil {
		cmd.Stdin = stdin
	}

	cmdErr := cmd.Run()

	stdout = stdoutBuf.buf.String()
	stderr = stderrBuf.buf.String()

	r.logGitCmd(start, args, stdout, stderr, cmdErr)

	if cmdErr != nil {
		err = fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr), cmdErr)
	}

	return stdout, stderr, err
}

// logGitCmd logs a git command execution if a logger is configured.
func (r *Repository) logGitCmd(start time.Time, args []string, stdout, stderr string, cmdErr error) {
	if r.logger == nil {
		return
	}
	exitCode := 0
	var errMsg string
	if cmdErr != nil {
		errMsg = cmdErr.Error()
		var exitErr *exec.ExitError
		if errors.As(cmdErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	_ = r.logger.Append(cmdlog.Entry{
		Timestamp:  start,
		Command:    "git " + strings.Join(args, " "),
		Dir:        r.path,
		ExitCode:   exitCode,
		Stdout:     stdout,
		Stderr:     stderr,
		Error:      errMsg,
		DurationMs: time.Since(start).Milliseconds(),
	})
}

// logOp wraps a go-git operation with logging.
// equiv is the equivalent git command for logging purposes.
func (r *Repository) logOp(_ context.Context, equiv string, fn func() (string, string, error)) (string, string, error) {
	start := time.Now()
	result1, result2, err := fn()
	duration := time.Since(start)

	if r.logger != nil {
		exitCode := 0
		var errMsg string
		if err != nil {
			exitCode = 1
			errMsg = err.Error()
		}
		_ = r.logger.Append(cmdlog.Entry{
			Timestamp:  start,
			Command:    equiv,
			Dir:        r.path,
			ExitCode:   exitCode,
			Stdout:     result1,
			Error:      errMsg,
			DurationMs: duration.Milliseconds(),
		})
	}

	return result1, result2, err
}

// logOpSingle wraps a go-git operation that returns a single value with logging.
func (r *Repository) logOpSingle(ctx context.Context, equiv string, fn func() (string, error)) (string, error) {
	result, _, err := r.logOp(ctx, equiv, func() (string, string, error) {
		s, e := fn()
		return s, "", e
	})
	return result, err
}

// GetConfigValue reads a git config value by key.
// Returns empty string (not error) if the key doesn't exist.
func (r *Repository) GetConfigValue(ctx context.Context, key string) (string, error) {
	out, err := r.runGit(ctx, "config", "--get", key)
	if err != nil {
		return "", nil //nolint:nilerr // git config --get exits 1 for absent keys; treat as empty, not error
	}
	return strings.TrimSpace(out), nil
}

// GetGlobalConfigValue reads a global git config value by key.
// Returns empty string (not error) if the key doesn't exist.
func (r *Repository) GetGlobalConfigValue(ctx context.Context, key string) (string, error) {
	out, err := r.runGit(ctx, "config", "--global", "--get", key)
	if err != nil {
		return "", nil //nolint:nilerr // git config --get exits 1 for absent keys; treat as empty, not error
	}
	return strings.TrimSpace(out), nil
}

// SetConfigValue sets a git config key to the given value.
func (r *Repository) SetConfigValue(ctx context.Context, key, value string) error {
	_, err := r.runGit(ctx, "config", key, value)
	if err != nil {
		return fmt.Errorf("set config %s: %w", key, err)
	}
	return nil
}
