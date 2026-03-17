package git

import (
	"context"
	"fmt"
	"strings"
)

// BisectOpts configures a bisect start operation.
type BisectOpts struct {
	NoCheckout  bool // --no-checkout
	FirstParent bool // --first-parent
}

// BisectStart begins a bisect session.
func (r *Repository) BisectStart(ctx context.Context, bad string, goods []string, opts BisectOpts) error {
	args := []string{"bisect", "start"}

	if opts.NoCheckout {
		args = append(args, "--no-checkout")
	}
	if opts.FirstParent {
		args = append(args, "--first-parent")
	}

	args = append(args, bad)
	args = append(args, goods...)

	_, err := r.runGit(ctx, args...)
	return err
}

// BisectGood marks a commit as good.
func (r *Repository) BisectGood(ctx context.Context, hash string) error {
	args := []string{"bisect", "good"}
	if hash != "" {
		args = append(args, hash)
	}
	_, err := r.runGit(ctx, args...)
	return err
}

// BisectBad marks a commit as bad.
func (r *Repository) BisectBad(ctx context.Context, hash string) error {
	args := []string{"bisect", "bad"}
	if hash != "" {
		args = append(args, hash)
	}
	_, err := r.runGit(ctx, args...)
	return err
}

// BisectSkip marks a commit as untestable.
func (r *Repository) BisectSkip(ctx context.Context, hash string) error {
	args := []string{"bisect", "skip"}
	if hash != "" {
		args = append(args, hash)
	}
	_, err := r.runGit(ctx, args...)
	return err
}

// BisectReset ends the bisect session and returns to the original HEAD.
func (r *Repository) BisectReset(ctx context.Context) error {
	_, err := r.runGit(ctx, "bisect", "reset")
	return err
}

// BisectRun runs a script for automated bisecting.
func (r *Repository) BisectRun(ctx context.Context, cmd string, args []string) error {
	gitArgs := []string{"bisect", "run", cmd}
	gitArgs = append(gitArgs, args...)

	out, err := r.runGit(ctx, gitArgs...)
	if err != nil {
		return fmt.Errorf("bisect run: %s: %w", strings.TrimSpace(out), err)
	}
	return nil
}
