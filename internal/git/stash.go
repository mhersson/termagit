package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// StashEntry represents a stash in the stash list.
type StashEntry struct {
	Index   int    // Stash index (0, 1, 2, ...)
	Name    string // Full stash name (e.g., "stash@{0}")
	Message string // Stash message (e.g., "WIP on main: abc123 commit message")
	Branch  string // Branch the stash was created on
	Hash    string // Commit hash of the stash
}

// ListStashes returns all stash entries in the repository.
// Returns entries newest-first (stash@{0} is first).
func (r *Repository) ListStashes(ctx context.Context) ([]StashEntry, error) {
	// Use git stash list with a custom format
	// %gd = short reflog selector (stash@{N})
	// %H = commit hash
	// %s = subject (the stash message)
	out, err := r.runGit(ctx, "stash", "list", "--format=%gd:%H:%s")
	if err != nil {
		// Empty stash list returns exit code 0 but empty output
		// Some git versions might return error on empty, handle gracefully
		if strings.Contains(err.Error(), "exit status") {
			return nil, nil
		}
		return nil, fmt.Errorf("list stashes: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var entries []StashEntry
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry, err := parseStashLine(line)
		if err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// StashOpts configures a git stash push operation.
type StashOpts struct {
	Message          string
	IncludeUntracked bool // -u / --include-untracked
	All              bool // -a / --all (includes ignored files, incompatible with -u)
	KeepIndex        bool // --keep-index
}

// Stash creates a new stash entry from the current working tree state.
func (r *Repository) Stash(ctx context.Context, opts StashOpts) error {
	args := []string{"stash", "push"}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}
	if opts.IncludeUntracked {
		args = append(args, "--include-untracked")
	}
	if opts.All {
		args = append(args, "--all")
	}
	if opts.KeepIndex {
		args = append(args, "--keep-index")
	}
	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("stash push: %w", err)
	}
	return nil
}

// StashPop applies a stash entry and removes it from the stash list.
func (r *Repository) StashPop(ctx context.Context, index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	_, err := r.runGit(ctx, "stash", "pop", ref)
	if err != nil {
		return fmt.Errorf("stash pop %s: %w", ref, err)
	}
	return nil
}

// StashApply applies a stash entry without removing it from the stash list.
func (r *Repository) StashApply(ctx context.Context, index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	_, err := r.runGit(ctx, "stash", "apply", ref)
	if err != nil {
		return fmt.Errorf("stash apply %s: %w", ref, err)
	}
	return nil
}

// StashDrop removes a stash entry from the stash list without applying it.
func (r *Repository) StashDrop(ctx context.Context, index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	_, err := r.runGit(ctx, "stash", "drop", ref)
	if err != nil {
		return fmt.Errorf("stash drop %s: %w", ref, err)
	}
	return nil
}

// StashBranch creates a new branch from a stash entry and drops the stash.
func (r *Repository) StashBranch(ctx context.Context, name string, index int) error {
	ref := fmt.Sprintf("stash@{%d}", index)
	_, err := r.runGit(ctx, "stash", "branch", name, ref)
	if err != nil {
		return fmt.Errorf("stash branch %s %s: %w", name, ref, err)
	}
	return nil
}

// StashRename renames a stash entry by dropping it and re-storing with a new message.
// WARNING: This is not atomic. If the store fails after the drop, the stash is lost.
func (r *Repository) StashRename(ctx context.Context, index int, newName string) error {
	ref := fmt.Sprintf("stash@{%d}", index)

	// Get the stash commit hash before dropping
	out, err := r.runGit(ctx, "rev-parse", ref)
	if err != nil {
		return fmt.Errorf("stash rename: resolve %s: %w", ref, err)
	}
	hash := strings.TrimSpace(out)

	// Drop the old entry
	_, err = r.runGit(ctx, "stash", "drop", ref)
	if err != nil {
		return fmt.Errorf("stash rename: drop %s: %w", ref, err)
	}

	// Store the commit back with the new message
	_, err = r.runGit(ctx, "stash", "store", "-m", newName, hash)
	if err != nil {
		return fmt.Errorf("stash rename: store as %q: %w", newName, err)
	}

	return nil
}

// StashCreateWipRef creates a stash commit and stores it under refs/wip/<branch>.
// Unlike regular stash, this doesn't modify the working tree or index.
func (r *Repository) StashCreateWipRef(ctx context.Context, opts StashOpts) error {
	// Create a stash commit without modifying working tree
	args := []string{"stash", "create"}
	if opts.IncludeUntracked {
		args = append(args, "--include-untracked")
	}
	if opts.All {
		args = append(args, "--all")
	}
	out, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("stash create: %w", err)
	}
	hash := strings.TrimSpace(out)
	if hash == "" {
		return fmt.Errorf("stash create: nothing to stash")
	}

	// Determine ref name
	branchOut, err := r.runGit(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("stash wip ref: resolve HEAD: %w", err)
	}
	branch := strings.TrimSpace(branchOut)
	ref := "refs/wip/" + branch

	// Store under the wip ref
	msg := "wip on " + branch
	if opts.Message != "" {
		msg = opts.Message
	}
	_, err = r.runGit(ctx, "update-ref", "-m", msg, ref, hash)
	if err != nil {
		return fmt.Errorf("stash wip ref: update-ref %s: %w", ref, err)
	}
	return nil
}

// StashShowPatch returns the diff of a stash entry as a patch string.
func (r *Repository) StashShowPatch(ctx context.Context, index int) (string, error) {
	ref := fmt.Sprintf("stash@{%d}", index)
	out, err := r.runGit(ctx, "stash", "show", "-p", ref)
	if err != nil {
		return "", fmt.Errorf("stash show -p %s: %w", ref, err)
	}
	return out, nil
}

// parseStashLine parses a line from git stash list output.
// Format: stash@{N}:HASH:message
func parseStashLine(line string) (StashEntry, error) {
	// Split on first two colons only (message may contain colons)
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 3 {
		return StashEntry{}, fmt.Errorf("invalid stash line: %s", line)
	}

	name := parts[0]   // stash@{N}
	hash := parts[1]   // commit hash
	message := parts[2] // stash message

	// Parse index from name (e.g., "stash@{0}" -> 0)
	index := 0
	if strings.HasPrefix(name, "stash@{") && strings.HasSuffix(name, "}") {
		idxStr := name[7 : len(name)-1]
		if n, err := strconv.Atoi(idxStr); err == nil {
			index = n
		}
	}

	// Extract branch from message if present
	// Format: "WIP on branch: hash message" or "On branch: message"
	branch := ""
	if strings.HasPrefix(message, "WIP on ") {
		rest := strings.TrimPrefix(message, "WIP on ")
		if colonIdx := strings.Index(rest, ":"); colonIdx > 0 {
			branch = rest[:colonIdx]
		}
	} else if strings.HasPrefix(message, "On ") {
		rest := strings.TrimPrefix(message, "On ")
		if colonIdx := strings.Index(rest, ":"); colonIdx > 0 {
			branch = rest[:colonIdx]
		}
	}

	return StashEntry{
		Index:   index,
		Name:    name,
		Hash:    hash,
		Message: message,
		Branch:  branch,
	}, nil
}
