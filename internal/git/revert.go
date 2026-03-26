package git

import (
	"context"
	"fmt"
	"strconv"
)

// RevertOpts configures a revert operation.
type RevertOpts struct {
	Mainline int    // -m N (for merge commits)
	Strategy string // -s <strategy>
	GpgSign  string // -S <keyid>
	Edit     bool   // -e (open editor, default for revert)
	NoEdit   bool   // --no-edit (suppress editor)
	Signoff  bool   // --signoff
}

// revertArgs builds shared args for revert commands.
func revertArgs(opts RevertOpts) []string {
	var args []string

	if opts.Mainline > 0 {
		args = append(args, "-m", strconv.Itoa(opts.Mainline))
	}
	if opts.Strategy != "" {
		args = append(args, "--strategy", opts.Strategy)
	}
	if opts.GpgSign != "" {
		args = append(args, "-S"+opts.GpgSign)
	}
	if opts.Edit {
		args = append(args, "-e")
	}
	if opts.NoEdit {
		args = append(args, "--no-edit")
	}
	if opts.Signoff {
		args = append(args, "--signoff")
	}

	return args
}

// Revert reverts the given commits, creating new commits.
func (r *Repository) Revert(ctx context.Context, hashes []string, opts RevertOpts) error {
	args := append([]string{"revert"}, revertArgs(opts)...)
	args = append(args, hashes...)

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("revert: %w", err)
	}
	return nil
}

// RevertChanges applies the reverse of the given commits without committing.
// Equivalent to: git revert --no-commit <hashes>
func (r *Repository) RevertChanges(ctx context.Context, hashes []string, opts RevertOpts) error {
	args := append([]string{"revert", "--no-commit"}, revertArgs(opts)...)
	args = append(args, hashes...)

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("revert changes: %w", err)
	}
	return nil
}

// RevertContinue continues an in-progress revert after conflict resolution.
func (r *Repository) RevertContinue(ctx context.Context) error {
	_, err := r.runGit(ctx, "revert", "--continue")
	if err != nil {
		return fmt.Errorf("revert continue: %w", err)
	}
	return nil
}

// RevertSkip skips the current commit in an in-progress revert.
func (r *Repository) RevertSkip(ctx context.Context) error {
	_, err := r.runGit(ctx, "revert", "--skip")
	if err != nil {
		return fmt.Errorf("revert skip: %w", err)
	}
	return nil
}

// RevertAbort aborts an in-progress revert.
func (r *Repository) RevertAbort(ctx context.Context) error {
	_, err := r.runGit(ctx, "revert", "--abort")
	if err != nil {
		return fmt.Errorf("revert abort: %w", err)
	}
	return nil
}
