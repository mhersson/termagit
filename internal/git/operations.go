package git

import (
	"context"
	"fmt"
	"strings"
)

// sanitizePath rejects paths starting with a dash to prevent flag injection.
func sanitizePath(path string) (string, error) {
	if strings.HasPrefix(path, "-") {
		return "", fmt.Errorf("refusing path starting with dash: %q", path)
	}
	return path, nil
}

// StageFile stages a file in the index.
func (r *Repository) StageFile(ctx context.Context, path string) error {
	if _, err := sanitizePath(path); err != nil {
		return err
	}
	_, _, err := r.logOp(ctx, "git add "+path, func() (string, string, error) {
		wt, err := r.raw.Worktree()
		if err != nil {
			return "", "", fmt.Errorf("get worktree: %w", err)
		}
		_, err = wt.Add(path)
		if err != nil {
			return "", "", fmt.Errorf("stage file: %w", err)
		}
		return "", "", nil
	})
	return err
}

// UnstageFile removes a file from the staging area.
func (r *Repository) UnstageFile(ctx context.Context, path string) error {
	if _, err := sanitizePath(path); err != nil {
		return err
	}
	_, err := r.runGit(ctx, "reset", "--", path)
	if err != nil {
		return fmt.Errorf("unstage file %s: %w", path, err)
	}
	return nil
}

// StageAll stages all changes in the working tree.
func (r *Repository) StageAll(ctx context.Context) error {
	_, err := r.runGit(ctx, "add", "-A")
	if err != nil {
		return fmt.Errorf("stage all: %w", err)
	}
	return nil
}

// UnstageAll removes all files from the staging area.
func (r *Repository) UnstageAll(ctx context.Context) error {
	_, err := r.runGit(ctx, "reset")
	if err != nil {
		return fmt.Errorf("unstage all: %w", err)
	}
	return nil
}

// DiscardFile discards changes to a file in the working tree.
func (r *Repository) DiscardFile(ctx context.Context, path string) error {
	if _, err := sanitizePath(path); err != nil {
		return err
	}
	_, err := r.runGit(ctx, "checkout", "--", path)
	if err != nil {
		return fmt.Errorf("discard file %s: %w", path, err)
	}
	return nil
}

// UntrackFile removes a file from the index but keeps it in the working tree.
func (r *Repository) UntrackFile(ctx context.Context, path string) error {
	if _, err := sanitizePath(path); err != nil {
		return err
	}
	_, err := r.runGit(ctx, "rm", "--cached", "--", path)
	if err != nil {
		return fmt.Errorf("untrack file %s: %w", path, err)
	}
	return nil
}

// RenameFile renames a file in the index and working tree.
func (r *Repository) RenameFile(ctx context.Context, oldPath, newPath string) error {
	if _, err := sanitizePath(oldPath); err != nil {
		return err
	}
	if _, err := sanitizePath(newPath); err != nil {
		return err
	}
	_, err := r.runGit(ctx, "mv", "--", oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename file %s to %s: %w", oldPath, newPath, err)
	}
	return nil
}

// StageHunk stages a specific hunk of a file.
func (r *Repository) StageHunk(ctx context.Context, path string, hunk Hunk) error {
	patch := HunkToPatch(path, &hunk, false)
	if err := r.ApplyPatch(ctx, patch, "--cached"); err != nil {
		return fmt.Errorf("stage hunk in %s: %w", path, err)
	}
	return nil
}

// UnstageHunk unstages a specific hunk from the index.
func (r *Repository) UnstageHunk(ctx context.Context, path string, hunk Hunk) error {
	patch := HunkToPatch(path, &hunk, true)
	if err := r.ApplyPatch(ctx, patch, "--cached"); err != nil {
		return fmt.Errorf("unstage hunk in %s: %w", path, err)
	}
	return nil
}

// DiscardHunk discards a specific hunk from the working tree.
func (r *Repository) DiscardHunk(ctx context.Context, path string, hunk Hunk) error {
	patch := HunkToPatch(path, &hunk, true)
	if err := r.ApplyPatch(ctx, patch); err != nil {
		return fmt.Errorf("discard hunk in %s: %w", path, err)
	}
	return nil
}
