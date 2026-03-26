package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoRebaseInProgress is returned when no rebase is in progress.
var ErrNoRebaseInProgress = errors.New("no rebase in progress")

// TodoAction represents a rebase todo action.
type TodoAction string

const (
	TodoPick   TodoAction = "pick"
	TodoReword TodoAction = "reword"
	TodoEdit   TodoAction = "edit"
	TodoSquash TodoAction = "squash"
	TodoFixup  TodoAction = "fixup"
	TodoExec   TodoAction = "exec"
	TodoDrop   TodoAction = "drop"
	TodoBreak  TodoAction = "break"
	TodoLabel  TodoAction = "label"
	TodoReset  TodoAction = "reset"
	TodoMerge  TodoAction = "merge"
)

// TodoEntry represents a single entry in the rebase todo list.
type TodoEntry struct {
	Action     TodoAction // The rebase action (pick, squash, etc.)
	Hash       string     // Full commit hash (if available)
	AbbrevHash string     // Abbreviated commit hash
	Subject    string     // Commit message subject
	Done       bool       // Already applied
	Stopped    bool       // Current position (stopped here)
}

// RebaseState represents the current state of an interactive rebase.
type RebaseState struct {
	Branch   string      // Branch being rebased
	Onto     string      // Base commit (what we're rebasing onto)
	OntoRef  string      // Reference name of onto (if available)
	Entries  []TodoEntry // All entries (done + pending)
	Current  int         // Index of current entry in Entries
	Total    int         // Total number of entries
}

// ReadRebaseTodo reads the current rebase state from .git/rebase-merge or .git/rebase-apply.
// Returns ErrNoRebaseInProgress if no rebase is in progress.
func (r *Repository) ReadRebaseTodo() (RebaseState, error) {
	// Check for interactive rebase (rebase-merge)
	rebaseMergeDir := filepath.Join(r.gitDir, "rebase-merge")
	if _, err := os.Stat(rebaseMergeDir); err == nil {
		return r.readRebaseMergeState(rebaseMergeDir)
	}

	// Check for am-style rebase (rebase-apply)
	rebaseApplyDir := filepath.Join(r.gitDir, "rebase-apply")
	if _, err := os.Stat(rebaseApplyDir); err == nil {
		return r.readRebaseApplyState(rebaseApplyDir)
	}

	return RebaseState{}, ErrNoRebaseInProgress
}

// readRebaseMergeState reads state from an interactive rebase (rebase-merge directory).
func (r *Repository) readRebaseMergeState(dir string) (RebaseState, error) {
	state := RebaseState{}

	// Read head-name (branch being rebased)
	if data, err := os.ReadFile(filepath.Join(dir, "head-name")); err == nil {
		branchRef := strings.TrimSpace(string(data))
		state.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
	}

	// Read onto (base commit)
	if data, err := os.ReadFile(filepath.Join(dir, "onto")); err == nil {
		state.Onto = strings.TrimSpace(string(data))
	}

	// Read stopped-sha (current position)
	stoppedSha := ""
	if data, err := os.ReadFile(filepath.Join(dir, "stopped-sha")); err == nil {
		stoppedSha = strings.TrimSpace(string(data))
	}

	// Read done file (completed entries)
	if data, err := os.ReadFile(filepath.Join(dir, "done")); err == nil {
		entries := parseTodoEntries(string(data), true, stoppedSha)
		state.Entries = append(state.Entries, entries...)
	}

	// Read git-rebase-todo (pending entries)
	if data, err := os.ReadFile(filepath.Join(dir, "git-rebase-todo")); err == nil {
		entries := parseTodoEntries(string(data), false, stoppedSha)
		state.Entries = append(state.Entries, entries...)
	}

	// If we have entries but none stopped, mark the first pending as stopped
	if stoppedSha != "" {
		markStopped(&state, stoppedSha)
	}

	state.Total = len(state.Entries)

	// Count current position
	for i, e := range state.Entries {
		if e.Stopped {
			state.Current = i + 1 // 1-indexed for display
			break
		}
	}

	return state, nil
}

// readRebaseApplyState reads state from an am-style rebase (rebase-apply directory).
func (r *Repository) readRebaseApplyState(dir string) (RebaseState, error) {
	state := RebaseState{}

	// Read head-name
	if data, err := os.ReadFile(filepath.Join(dir, "head-name")); err == nil {
		branchRef := strings.TrimSpace(string(data))
		state.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
	}

	// Read onto
	if data, err := os.ReadFile(filepath.Join(dir, "onto")); err == nil {
		state.Onto = strings.TrimSpace(string(data))
	}

	// For rebase-apply, we don't have the same todo format
	// Just return basic state
	return state, nil
}

// parseTodoEntries parses lines from a git-rebase-todo or done file.
func parseTodoEntries(content string, done bool, stoppedSha string) []TodoEntry {
	var entries []TodoEntry

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		entry := parseTodoLine(line)
		if entry != nil {
			entry.Done = done
			// Check if this is the stopped entry
			if stoppedSha != "" && strings.HasPrefix(stoppedSha, entry.AbbrevHash) {
				entry.Stopped = true
			}
			entries = append(entries, *entry)
		}
	}

	return entries
}

// parseTodoLine parses a single line from the rebase todo.
// Format: action [hash] [subject] or action [label/command]
func parseTodoLine(line string) *TodoEntry {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	action := TodoAction(strings.ToLower(parts[0]))

	// Handle abbreviated actions
	switch parts[0] {
	case "p":
		action = TodoPick
	case "r":
		action = TodoReword
	case "e":
		action = TodoEdit
	case "s":
		action = TodoSquash
	case "f":
		action = TodoFixup
	case "x":
		action = TodoExec
	case "d":
		action = TodoDrop
	case "b":
		action = TodoBreak
	case "l":
		action = TodoLabel
	case "t":
		action = TodoReset
	case "m":
		action = TodoMerge
	}

	entry := &TodoEntry{Action: action}

	// Commands without hash
	switch action {
	case TodoBreak:
		return entry
	case TodoLabel, TodoReset, TodoExec:
		if len(parts) > 1 {
			entry.Subject = strings.Join(parts[1:], " ")
		}
		return entry
	case TodoMerge:
		// merge -C <hash> <label>
		if len(parts) >= 3 {
			entry.AbbrevHash = strings.TrimPrefix(parts[2], "-C")
			if len(parts) > 3 {
				entry.Subject = strings.Join(parts[3:], " ")
			}
		}
		return entry
	}

	// Commands with hash
	if len(parts) >= 2 {
		entry.AbbrevHash = parts[1]
	}
	if len(parts) >= 3 {
		entry.Subject = strings.Join(parts[2:], " ")
	}

	return entry
}

// markStopped marks the correct entry as stopped.
func markStopped(state *RebaseState, stoppedSha string) {
	// Look through entries to find and mark the stopped one
	for i := range state.Entries {
		if strings.HasPrefix(stoppedSha, state.Entries[i].AbbrevHash) ||
			strings.HasPrefix(state.Entries[i].AbbrevHash, stoppedSha) {
			state.Entries[i].Stopped = true
			return
		}
	}
}

// FormatTodoEntries serializes rebase todo entries back to git-rebase-todo format.
// Dropped entries are prefixed with "# ", exec/break lines omit the hash.
func FormatTodoEntries(entries []TodoEntry) string {
	var b strings.Builder
	for _, e := range entries {
		switch e.Action {
		case TodoBreak:
			b.WriteString("break\n")
		case TodoExec:
			b.WriteString("exec " + e.Subject + "\n")
		case TodoLabel, TodoReset:
			b.WriteString(string(e.Action) + " " + e.Subject + "\n")
		case TodoDrop:
			b.WriteString("# drop " + e.AbbrevHash + " " + e.Subject + "\n")
		default:
			b.WriteString(string(e.Action) + " " + e.AbbrevHash + " " + e.Subject + "\n")
		}
	}
	return b.String()
}

// WriteRebaseTodo writes the given entries to .git/rebase-merge/git-rebase-todo.
// Returns an error if no interactive rebase is in progress.
func (r *Repository) WriteRebaseTodo(entries []TodoEntry) error {
	rebaseMergeDir := filepath.Join(r.gitDir, "rebase-merge")
	if _, err := os.Stat(rebaseMergeDir); err != nil {
		return fmt.Errorf("write rebase todo: %w", ErrNoRebaseInProgress)
	}

	todoPath := filepath.Join(rebaseMergeDir, "git-rebase-todo")
	content := FormatTodoEntries(entries)
	if err := os.WriteFile(todoPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write rebase todo: %w", err)
	}
	return nil
}

// GenerateRebaseTodo generates rebase todo entries for commits in the range (base..HEAD].
// The entries are returned in execution order (oldest first), matching git rebase -i.
func (r *Repository) GenerateRebaseTodo(ctx context.Context, base string) ([]TodoEntry, error) {
	// Use git log to get commits from base (exclusive) to HEAD, oldest first
	out, err := r.runGit(ctx, "log", "--reverse", "--format=%h %s", base+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("generate rebase todo: %w", err)
	}

	var entries []TodoEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		entry := TodoEntry{
			Action:     TodoPick,
			AbbrevHash: parts[0],
		}
		if len(parts) > 1 {
			entry.Subject = parts[1]
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("generate rebase todo: no commits in range %s..HEAD", base)
	}

	return entries, nil
}

// RebaseWithTodo runs an interactive rebase using a pre-prepared todo file.
// It writes the entries to a temp file, then runs git rebase -i with a
// GIT_SEQUENCE_EDITOR that copies the prepared todo over git's generated one.
func (r *Repository) RebaseWithTodo(ctx context.Context, base string, entries []TodoEntry, opts RebaseOpts) error {
	// Write entries to a temp file
	content := FormatTodoEntries(entries)
	tmpFile, err := os.CreateTemp("", "termagit-rebase-todo-*")
	if err != nil {
		return fmt.Errorf("rebase with todo: create temp: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("rebase with todo: write temp: %w", err)
	}
	_ = tmpFile.Close()

	// Build the sequence editor command that copies our prepared todo over git's
	seqEditor := fmt.Sprintf("cp '%s'", strings.ReplaceAll(tmpPath, "'", "'\\''"))

	args := append([]string{"rebase"}, rebaseArgs(opts)...)
	args = append(args, "-i", base)

	env := []string{"GIT_SEQUENCE_EDITOR=" + seqEditor}
	_, err = r.runGitWithEnv(ctx, env, args...)
	return err
}

// RebaseCurrentStep returns the current step number in the rebase (1-indexed).
func (r *Repository) RebaseCurrentStep() (int, error) {
	state, err := r.ReadRebaseTodo()
	if err != nil {
		return 0, err
	}
	return state.Current, nil
}

// RebaseAutosquash runs an interactive rebase with --autosquash to fold
// fixup!/squash! commits into their targets. The target parameter is the
// commit SHA that marks the start of the range (the rebase will be done
// onto target~1, i.e. the parent of target). GIT_SEQUENCE_EDITOR is set
// to "true" so the todo list is auto-accepted without user interaction.
func (r *Repository) RebaseAutosquash(ctx context.Context, target string) error {
	args := []string{
		"-c", "sequence.editor=true",
		"rebase", "-i", "--autosquash", "--autostash", "--keep-empty",
		target + "~1",
	}
	_, err := r.runGit(ctx, args...)
	return err
}
