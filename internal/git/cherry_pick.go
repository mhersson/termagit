package git

import (
	"context"
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
		args = append(args, "-m", itoa(opts.Mainline))
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
	return err
}

// CherryPickContinue continues an in-progress cherry-pick after conflict resolution.
func (r *Repository) CherryPickContinue(ctx context.Context) error {
	_, err := r.runGit(ctx, "cherry-pick", "--continue")
	return err
}

// CherryPickSkip skips the current commit in an in-progress cherry-pick.
func (r *Repository) CherryPickSkip(ctx context.Context) error {
	_, err := r.runGit(ctx, "cherry-pick", "--skip")
	return err
}

// CherryPickAbort aborts an in-progress cherry-pick.
func (r *Repository) CherryPickAbort(ctx context.Context) error {
	_, err := r.runGit(ctx, "cherry-pick", "--abort")
	return err
}

// CherryPickApply applies the changes from the given commits without committing.
// Equivalent to: git cherry-pick --no-commit <hashes>
func (r *Repository) CherryPickApply(ctx context.Context, hashes []string, opts CherryPickOpts) error {
	args := append([]string{"cherry-pick", "--no-commit"}, cherryPickArgs(opts)...)
	args = append(args, hashes...)

	_, err := r.runGit(ctx, args...)
	return err
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
