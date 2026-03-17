package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/mhersson/conjit/internal/cmdlog"
)

// ErrNotARepo is returned when the path is not inside a git repository.
var ErrNotARepo = errors.New("not a git repository")

// Repository wraps a go-git repository with logging and convenience methods.
type Repository struct {
	raw    *git.Repository
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
			raw, err := git.PlainOpen(current)
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
func Wrap(raw *git.Repository, path string, logger *cmdlog.Logger) *Repository {
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
func (r *Repository) HeadInfo(ctx context.Context) (branch, subject string, err error) {
	return r.logOp(ctx, "git rev-parse --abbrev-ref HEAD && git log -1 --format=%s", func() (string, string, error) {
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
func (r *Repository) HeadOID(ctx context.Context) (string, error) {
	oid, _, err := r.logOpSingle(ctx, "git rev-parse HEAD", func() (string, error) {
		head, err := r.raw.Head()
		if err != nil {
			return "", fmt.Errorf("get HEAD: %w", err)
		}
		return head.Hash().String(), nil
	})
	return oid, err
}

// AheadBehind returns the number of commits ahead and behind the upstream.
// Returns (0, 0) if there is no upstream configured.
func (r *Repository) AheadBehind(ctx context.Context) (ahead, behind int, err error) {
	// Get current branch
	head, err := r.raw.Head()
	if err != nil {
		return 0, 0, nil // No HEAD means no tracking
	}

	if !head.Name().IsBranch() {
		return 0, 0, nil // Detached HEAD has no upstream
	}

	// Try to get the upstream reference
	cfg, err := r.raw.Config()
	if err != nil {
		return 0, 0, nil
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
		return 0, 0, nil // Upstream doesn't exist locally
	}

	// Count commits
	headCommit, err := r.raw.CommitObject(head.Hash())
	if err != nil {
		return 0, 0, fmt.Errorf("get head commit: %w", err)
	}

	upstreamCommit, err := r.raw.CommitObject(upstream.Hash())
	if err != nil {
		return 0, 0, fmt.Errorf("get upstream commit: %w", err)
	}

	// Find merge base and count commits
	// For simplicity, we'll use git rev-list for accurate counting
	// This is a common case where shelling out is more reliable
	aheadCount, behindCount, err := r.countAheadBehind(ctx, head.Hash().String(), upstream.Hash().String())
	if err != nil {
		// Fallback: just return 0,0 if we can't count
		_ = headCommit
		_ = upstreamCommit
		return 0, 0, nil
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

	_, _ = fmt.Sscanf(parts[0], "%d", &ahead)
	_, _ = fmt.Sscanf(parts[1], "%d", &behind)
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
	data, err := os.ReadFile(todoPath)
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

// runGit executes a git command and returns stdout.
// Logs the command via cmdlog if logger is set.
func (r *Repository) runGit(ctx context.Context, args ...string) (string, error) {
	stdout, _, err := r.runGitFull(ctx, args...)
	return stdout, err
}

// runGitFull executes a git command and returns stdout, stderr, and error.
// The output is returned even when there's an error (for commands like diff --no-index).
func (r *Repository) runGitFull(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.path

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	cmdErr := cmd.Run()
	duration := time.Since(start)

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	// Log the command
	if r.logger != nil {
		exitCode := 0
		if cmdErr != nil {
			if exitErr, ok := cmdErr.(*exec.ExitError); ok {
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
			DurationMs: duration.Milliseconds(),
		})
	}

	if cmdErr != nil {
		err = fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), strings.TrimSpace(stderr), cmdErr)
	}

	return stdout, stderr, err
}

// logOp wraps a go-git operation with logging.
// equiv is the equivalent git command for logging purposes.
func (r *Repository) logOp(ctx context.Context, equiv string, fn func() (string, string, error)) (string, string, error) {
	start := time.Now()
	result1, result2, err := fn()
	duration := time.Since(start)

	if r.logger != nil {
		exitCode := 0
		stderr := ""
		if err != nil {
			exitCode = 1
			stderr = err.Error()
		}
		_ = r.logger.Append(cmdlog.Entry{
			Timestamp:  start,
			Command:    equiv,
			Dir:        r.path,
			ExitCode:   exitCode,
			Stdout:     result1,
			Stderr:     stderr,
			DurationMs: duration.Milliseconds(),
		})
	}

	return result1, result2, err
}

// logOpSingle wraps a go-git operation that returns a single value with logging.
func (r *Repository) logOpSingle(ctx context.Context, equiv string, fn func() (string, error)) (string, string, error) {
	result, _, err := r.logOp(ctx, equiv, func() (string, string, error) {
		r, e := fn()
		return r, "", e
	})
	return result, "", err
}
