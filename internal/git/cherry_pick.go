package git

import (
	"context"
	"fmt"
	"strconv"
)

// CherryPickOpts configures a cherry-pick operation.
type CherryPickOpts struct {
	Mainline           int    // -m N (for merge commits)
	Strategy           string // -s <strategy>
	GpgSign            string // -S <keyid>
	FF                 bool   // --ff
	ReferenceInMessage bool   // -x
	Edit               bool   // -e
	Signoff            bool   // -s (signoff)
}

// cherryPickArgs builds shared args for cherry-pick commands.
func cherryPickArgs(opts CherryPickOpts) []string {
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
	if opts.FF {
		args = append(args, "--ff")
	}
	if opts.ReferenceInMessage {
		args = append(args, "-x")
	}
	if opts.Edit {
		args = append(args, "-e")
	}
	if opts.Signoff {
		args = append(args, "--signoff")
	}

	return args
}

// CherryPick cherry-picks the given commits onto the current branch.
func (r *Repository) CherryPick(ctx context.Context, hashes []string, opts CherryPickOpts) error {
	args := append([]string{"cherry-pick"}, cherryPickArgs(opts)...)
	args = append(args, hashes...)

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("cherry-pick: %w", err)
	}
	return nil
}

// CherryPickContinue continues an in-progress cherry-pick after conflict resolution.
func (r *Repository) CherryPickContinue(ctx context.Context) error {
	_, err := r.runGit(ctx, "cherry-pick", "--continue")
	if err != nil {
		return fmt.Errorf("cherry-pick continue: %w", err)
	}
	return nil
}

// CherryPickSkip skips the current commit in an in-progress cherry-pick.
func (r *Repository) CherryPickSkip(ctx context.Context) error {
	_, err := r.runGit(ctx, "cherry-pick", "--skip")
	if err != nil {
		return fmt.Errorf("cherry-pick skip: %w", err)
	}
	return nil
}

// CherryPickAbort aborts an in-progress cherry-pick.
func (r *Repository) CherryPickAbort(ctx context.Context) error {
	_, err := r.runGit(ctx, "cherry-pick", "--abort")
	if err != nil {
		return fmt.Errorf("cherry-pick abort: %w", err)
	}
	return nil
}

// CherryPickApply applies the changes from the given commits without committing.
// Equivalent to: git cherry-pick --no-commit <hashes>
func (r *Repository) CherryPickApply(ctx context.Context, hashes []string, opts CherryPickOpts) error {
	args := append([]string{"cherry-pick", "--no-commit"}, cherryPickArgs(opts)...)
	args = append(args, hashes...)

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("cherry-pick apply: %w", err)
	}
	return nil
}

// CherryPickDonate cherry-picks commits onto the destination branch, then removes them
// from the source branch. This is the "donate" operation from Neogit.
// Flow: checkout dst → cherry-pick → checkout src → rebase to drop donated commits.
func (r *Repository) CherryPickDonate(ctx context.Context, hashes []string, src, dst string, opts CherryPickOpts) error {
	// Step 1: Checkout destination branch
	if _, err := r.runGit(ctx, "checkout", dst); err != nil {
		return fmt.Errorf("checkout %s: %w", dst, err)
	}

	// Step 2: Cherry-pick the commits onto destination
	if err := r.CherryPick(ctx, hashes, opts); err != nil {
		// If cherry-pick fails, try to get back to src
		_, _ = r.runGit(ctx, "cherry-pick", "--abort")
		_, _ = r.runGit(ctx, "checkout", src)
		return fmt.Errorf("cherry-pick onto %s: %w", dst, err)
	}

	// Step 3: Checkout back to source branch
	if _, err := r.runGit(ctx, "checkout", src); err != nil {
		return fmt.Errorf("checkout %s: %w", src, err)
	}

	// Step 4: Rebase source to drop the donated commits
	// For each commit, use rebase --onto to skip it
	// We use the first commit's parent as the new base
	parentRef := hashes[0] + "^"
	lastHash := hashes[len(hashes)-1]
	if _, err := r.runGit(ctx, "rebase", "--onto", parentRef, lastHash); err != nil {
		return fmt.Errorf("rebase to drop commits: %w", err)
	}

	return nil
}

