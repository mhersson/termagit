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
