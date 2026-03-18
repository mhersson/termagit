package git

import (
	"context"
	"fmt"
)

// StageFile stages a file in the index.
func (r *Repository) StageFile(ctx context.Context, path string) error {
	return r.logOpVoid(ctx, "git add "+path, func() error {
		wt, err := r.raw.Worktree()
		if err != nil {
			return fmt.Errorf("get worktree: %w", err)
		}
		_, err = wt.Add(path)
		if err != nil {
			return fmt.Errorf("stage file: %w", err)
		}
		return nil
	})
}

// UnstageFile removes a file from the staging area.
func (r *Repository) UnstageFile(ctx context.Context, path string) error {
	_, err := r.runGit(ctx, "reset", "--", path)
	return err
}

// StageAll stages all changes in the working tree.
func (r *Repository) StageAll(ctx context.Context) error {
	_, err := r.runGit(ctx, "add", "-A")
	return err
}

// UnstageAll removes all files from the staging area.
func (r *Repository) UnstageAll(ctx context.Context) error {
	_, err := r.runGit(ctx, "reset")
	return err
}

// DiscardFile discards changes to a file in the working tree.
func (r *Repository) DiscardFile(ctx context.Context, path string) error {
	_, err := r.runGit(ctx, "checkout", "--", path)
	return err
}

// UntrackFile removes a file from the index but keeps it in the working tree.
func (r *Repository) UntrackFile(ctx context.Context, path string) error {
	_, err := r.runGit(ctx, "rm", "--cached", path)
	return err
}

// RenameFile renames a file in the index and working tree.
func (r *Repository) RenameFile(ctx context.Context, oldPath, newPath string) error {
	_, err := r.runGit(ctx, "mv", oldPath, newPath)
	return err
}

// StageHunk stages a specific hunk of a file.
func (r *Repository) StageHunk(ctx context.Context, path string, hunk Hunk) error {
	patch := HunkToPatch(path, &hunk, false)
	return r.ApplyPatch(ctx, patch, "--cached")
}

// UnstageHunk unstages a specific hunk from the index.
func (r *Repository) UnstageHunk(ctx context.Context, path string, hunk Hunk) error {
	patch := HunkToPatch(path, &hunk, true)
	return r.ApplyPatch(ctx, patch, "--cached")
}

// DiscardHunk discards a specific hunk from the working tree.
func (r *Repository) DiscardHunk(ctx context.Context, path string, hunk Hunk) error {
	patch := HunkToPatch(path, &hunk, true)
	return r.ApplyPatch(ctx, patch)
}

// UnstageStaged unstages all staged files.
func (r *Repository) UnstageStaged(ctx context.Context) error {
	return r.UnstageAll(ctx)
}

// logOpVoid wraps a void operation with logging.
func (r *Repository) logOpVoid(ctx context.Context, equiv string, fn func() error) error {
	_, _, err := r.logOp(ctx, equiv, func() (string, string, error) {
		return "", "", fn()
	})
	return err
}
