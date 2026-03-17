package git

import (
	"context"
	"fmt"
	"strings"
)

// GitWorktree represents a git worktree entry.
type GitWorktree struct {
	Path       string
	Head       string // full hash
	Branch     string
	IsBare     bool
	IsMain     bool
	IsLocked   bool
	LockReason string
}

// ListWorktrees returns all worktrees for the repository.
func (r *Repository) ListWorktrees(ctx context.Context) ([]GitWorktree, error) {
	out, err := r.runGit(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}

	wts, err := parseWorktreePorcelain(out)
	if err != nil {
		return nil, err
	}

	// Mark the first worktree as main
	if len(wts) > 0 {
		wts[0].IsMain = true
	}

	return wts, nil
}

// AddWorktree adds a worktree for an existing branch.
func (r *Repository) AddWorktree(ctx context.Context, path, branch string) error {
	_, err := r.runGit(ctx, "worktree", "add", path, branch)
	return err
}

// AddWorktreeNewBranch creates a new branch and adds a worktree for it.
func (r *Repository) AddWorktreeNewBranch(ctx context.Context, path, branch, base string) error {
	args := []string{"worktree", "add", "-b", branch, path}
	if base != "" {
		args = append(args, base)
	}
	_, err := r.runGit(ctx, args...)
	return err
}

// RemoveWorktree removes a linked worktree.
func (r *Repository) RemoveWorktree(ctx context.Context, path string, force bool) error {
	args := []string{"worktree", "remove", path}
	if force {
		args = []string{"worktree", "remove", "--force", path}
	}
	_, err := r.runGit(ctx, args...)
	return err
}

// MoveWorktree moves a linked worktree to a new path.
func (r *Repository) MoveWorktree(ctx context.Context, oldPath, newPath string) error {
	_, err := r.runGit(ctx, "worktree", "move", oldPath, newPath)
	return err
}

// LockWorktree prevents a worktree from being pruned.
func (r *Repository) LockWorktree(ctx context.Context, path, reason string) error {
	args := []string{"worktree", "lock", path}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	_, err := r.runGit(ctx, args...)
	return err
}

// UnlockWorktree removes the lock from a worktree.
func (r *Repository) UnlockWorktree(ctx context.Context, path string) error {
	_, err := r.runGit(ctx, "worktree", "unlock", path)
	return err
}

// GotoWorktree is a no-op placeholder; in the TUI layer, changing
// directory is handled by the app, not the git package.
func (r *Repository) GotoWorktree(_ context.Context, _ string) error {
	return nil
}

// parseWorktreePorcelain parses the output of `git worktree list --porcelain`.
// Each worktree block is separated by a blank line.
func parseWorktreePorcelain(output string) ([]GitWorktree, error) {
	var worktrees []GitWorktree
	var current *GitWorktree

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			current = &GitWorktree{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if current != nil {
			switch {
			case strings.HasPrefix(line, "HEAD "):
				current.Head = strings.TrimPrefix(line, "HEAD ")
			case strings.HasPrefix(line, "branch "):
				current.Branch = strings.TrimPrefix(line, "branch ")
			case line == "bare":
				current.IsBare = true
			case line == "locked":
				current.IsLocked = true
			case strings.HasPrefix(line, "locked "):
				current.IsLocked = true
				current.LockReason = strings.TrimPrefix(line, "locked ")
			}
		}
	}

	// Handle trailing entry without final blank line
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}
