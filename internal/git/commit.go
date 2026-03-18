package git

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// CommitOpts configures a git commit operation.
type CommitOpts struct {
	Message      string
	All          bool     // -a / --all
	AllowEmpty   bool     // --allow-empty
	Verbose      bool     // -v
	NoVerify     bool     // --no-verify (hooks)
	ResetAuthor  bool     // --reset-author
	Signoff      bool     // -s
	GpgSign      string   // -S <keyid>
	Author       string   // --author
	ReuseMessage string   // -C <commit>
	Amend        bool     // --amend
	Fixup        string   // --fixup=<commit> (also supports "amend:<commit>" and "reword:<commit>")
	Squash       string   // --squash=<commit>
	NoEdit       bool     // --no-edit
	OnlyFiles    []string // paths to stage before committing
}

// Commit creates a new commit from the staged changes.
// Returns the abbreviated hash of the new commit.
func (r *Repository) Commit(ctx context.Context, opts CommitOpts) (string, error) {
	args := []string{"commit"}
	args = appendCommitArgs(args, opts)

	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	args = append(args, opts.OnlyFiles...)

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return "", err
	}

	return parseCommitHash(ctx, r, out)
}

// CommitFromFile creates a commit using message from a file.
func (r *Repository) CommitFromFile(ctx context.Context, path string, opts CommitOpts) (string, error) {
	args := []string{"commit", "-F", path}
	args = appendCommitArgs(args, opts)
	args = append(args, opts.OnlyFiles...)

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return "", err
	}

	return parseCommitHash(ctx, r, out)
}

// CommitEditorPath returns the path of the COMMIT_EDITMSG file.
func (r *Repository) CommitEditorPath() string {
	return filepath.Join(r.gitDir, "COMMIT_EDITMSG")
}

// CommitAbsorb runs git absorb if available.
// Returns an error if git-absorb is not installed.
func (r *Repository) CommitAbsorb(ctx context.Context) error {
	_, err := r.runGit(ctx, "absorb")
	return err
}

// appendCommitArgs adds common commit flags to the args slice.
func appendCommitArgs(args []string, opts CommitOpts) []string {
	if opts.All {
		args = append(args, "--all")
	}
	if opts.AllowEmpty {
		args = append(args, "--allow-empty")
	}
	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.NoVerify {
		args = append(args, "--no-verify")
	}
	if opts.ResetAuthor {
		args = append(args, "--reset-author")
	}
	if opts.Signoff {
		args = append(args, "-s")
	}
	if opts.GpgSign != "" {
		args = append(args, "-S"+opts.GpgSign)
	}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	if opts.ReuseMessage != "" {
		args = append(args, "-C", opts.ReuseMessage)
	}
	if opts.Amend {
		args = append(args, "--amend")
	}
	if opts.Fixup != "" {
		args = append(args, "--fixup="+opts.Fixup)
	}
	if opts.Squash != "" {
		args = append(args, "--squash="+opts.Squash)
	}
	if opts.NoEdit {
		args = append(args, "--no-edit")
	}
	return args
}

// parseCommitHash extracts the commit hash from git commit output or falls back to rev-parse.
func parseCommitHash(ctx context.Context, r *Repository, output string) (string, error) {
	// Try to parse from output: "[main abc1234] message"
	if idx := strings.Index(output, " "); idx > 0 {
		bracket := output[:idx]
		if strings.HasPrefix(bracket, "[") {
			parts := strings.SplitN(bracket, " ", 2)
			if len(parts) == 2 {
				hash := strings.TrimSuffix(parts[1], "]")
				if len(hash) >= 7 {
					return hash, nil
				}
			}
		}
	}

	// Fallback: get hash via rev-parse
	out, err := r.runGit(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("get commit hash: %w", err)
	}
	return strings.TrimSpace(out), nil
}
