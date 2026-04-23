package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var hexRe = regexp.MustCompile(`^[0-9a-fA-F]+$`)

func validateHex(s string) error {
	if !hexRe.MatchString(s) {
		return fmt.Errorf("invalid hex string: %q", s)
	}
	return nil
}

// RebaseOpts configures a rebase operation.
type RebaseOpts struct {
	Onto                      string // --onto <target>
	Interactive               bool   // -i
	Autosquash                bool   // --autosquash
	Autostash                 bool   // --autostash
	KeepEmpty                 bool   // --keep-empty
	UpdateRefs                bool   // --update-refs
	RebaseMerges              string // "" | "no-rebase-cousins" | "rebase-cousins"
	NoVerify                  bool   // --no-verify
	GpgSign                   string // -S <keyid>
	CommitterDateIsAuthorDate bool   // --committer-date-is-author-date
	IgnoreDate                bool   // --ignore-date
}

// rebaseArgs builds the argument list for a rebase command.
func rebaseArgs(opts RebaseOpts) []string {
	var args []string

	if opts.Interactive {
		args = append(args, "-i")
	}
	if opts.Autosquash {
		args = append(args, "--autosquash")
	}
	if opts.Autostash {
		args = append(args, "--autostash")
	}
	if opts.KeepEmpty {
		args = append(args, "--keep-empty")
	}
	if opts.UpdateRefs {
		args = append(args, "--update-refs")
	}
	if opts.RebaseMerges != "" {
		args = append(args, "--rebase-merges="+opts.RebaseMerges)
	}
	if opts.NoVerify {
		args = append(args, "--no-verify")
	}
	if opts.GpgSign != "" {
		args = append(args, "-S"+opts.GpgSign)
	}
	if opts.CommitterDateIsAuthorDate {
		args = append(args, "--committer-date-is-author-date")
	}
	if opts.IgnoreDate {
		args = append(args, "--ignore-date")
	}

	return args
}

// Rebase starts a rebase onto the given target.
// For interactive rebases, sets GIT_SEQUENCE_EDITOR=true so git writes the
// todo file and pauses. The caller can then read the todo via ReadRebaseTodo,
// modify it, and call RebaseContinue.
func (r *Repository) Rebase(ctx context.Context, opts RebaseOpts) error {
	args := append([]string{"rebase"}, rebaseArgs(opts)...)

	if opts.Onto != "" {
		args = append(args, opts.Onto)
	}

	if opts.Interactive {
		// Use GIT_SEQUENCE_EDITOR=true to accept the default todo and pause
		if err := r.runGitWithEnv(ctx, []string{"GIT_SEQUENCE_EDITOR=true"}, args...); err != nil {
			return fmt.Errorf("rebase interactive: %w", err)
		}
		return nil
	}

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("rebase: %w", err)
	}
	return nil
}

// RebaseContinue continues an in-progress rebase.
func (r *Repository) RebaseContinue(ctx context.Context) error {
	_, err := r.runGit(ctx, "rebase", "--continue")
	if err != nil {
		return fmt.Errorf("rebase continue: %w", err)
	}
	return nil
}

// RebaseSkip skips the current commit in an in-progress rebase.
func (r *Repository) RebaseSkip(ctx context.Context) error {
	_, err := r.runGit(ctx, "rebase", "--skip")
	if err != nil {
		return fmt.Errorf("rebase skip: %w", err)
	}
	return nil
}

// RebaseAbort aborts an in-progress rebase and restores the original state.
func (r *Repository) RebaseAbort(ctx context.Context) error {
	_, err := r.runGit(ctx, "rebase", "--abort")
	if err != nil {
		return fmt.Errorf("rebase abort: %w", err)
	}
	return nil
}

// RebaseOnto rebases a range of commits onto a specific target.
// Equivalent to: git rebase --onto <onto> <from>
func (r *Repository) RebaseOnto(ctx context.Context, onto, from string, opts RebaseOpts) error {
	args := append([]string{"rebase"}, rebaseArgs(opts)...)
	args = append(args, "--onto", onto, from)

	if opts.Interactive {
		if err := r.runGitWithEnv(ctx, []string{"GIT_SEQUENCE_EDITOR=true"}, args...); err != nil {
			return fmt.Errorf("rebase onto %s: %w", onto, err)
		}
		return nil
	}

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("rebase onto %s: %w", onto, err)
	}
	return nil
}

// DropCommit removes a commit from history via rebase --onto.
// Equivalent to: git rebase --onto <hash>^ <hash>
func (r *Repository) DropCommit(ctx context.Context, hash string) error {
	_, err := r.runGit(ctx, "rebase", "--onto", hash+"^", hash)
	if err != nil {
		return fmt.Errorf("drop commit %s: %w", hash, err)
	}
	return nil
}

// RewordCommit rewrites the commit message of the given commit using
// interactive rebase with a custom sequence editor that changes "pick" to "reword".
func (r *Repository) RewordCommit(ctx context.Context, hash, message string) error {
	if err := validateHex(hash); err != nil {
		return err
	}
	// Use GIT_SEQUENCE_EDITOR to change "pick <hash>" to "reword <hash>"
	sedCmd := fmt.Sprintf("sed -i.bak '0,/^pick %s/s//reword %s/' \"$1\"", hash[:7], hash[:7])
	env := []string{
		"GIT_SEQUENCE_EDITOR=" + sedCmd,
		"GIT_EDITOR=true",
	}

	// Start interactive rebase from the commit's parent
	args := []string{"rebase", "-i", hash + "^"}
	if err := r.runGitWithEnv(ctx, env, args...); err != nil {
		return fmt.Errorf("reword commit %s: %w", hash, err)
	}

	return nil
}

// ModifyCommit stops the rebase at the given commit for amending.
// Uses interactive rebase with a sequence editor that changes "pick" to "edit".
func (r *Repository) ModifyCommit(ctx context.Context, hash string) error {
	if err := validateHex(hash); err != nil {
		return err
	}
	// Use GIT_SEQUENCE_EDITOR to change "pick <hash>" to "edit <hash>"
	sedCmd := fmt.Sprintf("sed -i.bak '0,/^pick %s/s//edit %s/' \"$1\"", hash[:7], hash[:7])
	env := []string{
		"GIT_SEQUENCE_EDITOR=" + sedCmd,
	}

	args := []string{"rebase", "-i", hash + "^"}
	if err := r.runGitWithEnv(ctx, env, args...); err != nil && !r.RebaseInProgress() {
		return fmt.Errorf("modify commit %s: %w", hash, err)
	}

	return nil
}

// Autosquash runs rebase --interactive --autosquash on HEAD.
// The sequence editor is set to "true" to accept the reordered todo as-is.
func (r *Repository) Autosquash(ctx context.Context, opts RebaseOpts) error {
	// Find the merge base with the upstream or use the root
	base, err := r.runGit(ctx, "merge-base", "HEAD", opts.Onto)
	if err != nil {
		return fmt.Errorf("find merge base: %w", err)
	}

	target := strings.TrimSpace(base)
	args := append([]string{"rebase", "-i", "--autosquash"}, rebaseArgs(opts)...)
	args = append(args, target)

	if err = r.runGitWithEnv(ctx, []string{"GIT_SEQUENCE_EDITOR=true"}, args...); err != nil {
		return fmt.Errorf("autosquash: %w", err)
	}
	return nil
}
