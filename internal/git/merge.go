package git

import (
	"context"
	"fmt"
)

// MergeOpts configures a merge operation.
type MergeOpts struct {
	Branch         string
	Message        string
	Strategy       string
	StrategyOption string
	DiffAlgorithm  string
	GpgSign        string
	NoFF           bool
	FFOnly         bool
	Squash         bool
	NoCommit       bool
}

// Merge merges the given branch into the current branch.
func (r *Repository) Merge(ctx context.Context, opts MergeOpts) error {
	args := []string{"merge"}

	if opts.NoFF {
		args = append(args, "--no-ff")
	}
	if opts.FFOnly {
		args = append(args, "--ff-only")
	}
	if opts.Squash {
		args = append(args, "--squash")
	}
	if opts.NoCommit {
		args = append(args, "--no-commit")
	}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}
	if opts.Strategy != "" {
		args = append(args, "-s", opts.Strategy)
	}
	if opts.StrategyOption != "" {
		args = append(args, "-X", opts.StrategyOption)
	}
	if opts.DiffAlgorithm != "" {
		args = append(args, "-X", "diff-algorithm="+opts.DiffAlgorithm)
	}
	if opts.GpgSign != "" {
		args = append(args, "-S"+opts.GpgSign)
	}

	args = append(args, opts.Branch)

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("merge %s: %w", opts.Branch, err)
	}
	return nil
}

// MergeAbort aborts an in-progress merge.
func (r *Repository) MergeAbort(ctx context.Context) error {
	_, err := r.runGit(ctx, "merge", "--abort")
	if err != nil {
		return fmt.Errorf("merge abort: %w", err)
	}
	return nil
}

// MergeCommit commits an in-progress merge (after conflict resolution).
func (r *Repository) MergeCommit(ctx context.Context) error {
	_, err := r.runGit(ctx, "commit", "--no-edit")
	if err != nil {
		return fmt.Errorf("merge commit: %w", err)
	}
	return nil
}
