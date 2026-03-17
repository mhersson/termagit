package git

import (
	"context"
	"fmt"
)

// ResetMode represents the mode for a git reset operation.
type ResetMode string

const (
	ResetSoft     ResetMode = "soft"
	ResetMixed    ResetMode = "mixed"
	ResetHard     ResetMode = "hard"
	ResetKeep     ResetMode = "keep"
	ResetIndex    ResetMode = "index"    // git reset (index only)
	ResetWorktree ResetMode = "worktree" // git checkout --
)

// Reset resets the current HEAD to the specified target with the given mode.
func (r *Repository) Reset(ctx context.Context, target string, mode ResetMode) error {
	switch mode {
	case ResetIndex:
		_, err := r.runGit(ctx, "reset", "--", target)
		return err
	case ResetWorktree:
		_, err := r.runGit(ctx, "checkout", "--", target)
		return err
	case ResetSoft, ResetMixed, ResetHard, ResetKeep:
		_, err := r.runGit(ctx, "reset", "--"+string(mode), target)
		return err
	default:
		return fmt.Errorf("reset: unknown mode %q", mode)
	}
}

// ResetFile resets a single file in the index to the specified target revision.
func (r *Repository) ResetFile(ctx context.Context, path, target string) error {
	_, err := r.runGit(ctx, "reset", target, "--", path)
	return err
}
