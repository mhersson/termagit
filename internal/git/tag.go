package git

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

// TagOpts configures a tag creation operation.
type TagOpts struct {
	Annotate  bool
	Sign      bool
	Force     bool
	Message   string
	LocalUser string // -u <keyid>
}

// ListTags returns all tag names in the repository, sorted alphabetically.
func (r *Repository) ListTags(ctx context.Context) ([]string, error) {
	var tags []string

	_, _, err := r.logOp(ctx, "git tag -l", func() (string, string, error) {
		iter, err := r.raw.Tags()
		if err != nil {
			return "", "", fmt.Errorf("list tags: %w", err)
		}

		err = iter.ForEach(func(ref *plumbing.Reference) error {
			tags = append(tags, ref.Name().Short())
			return nil
		})
		if err != nil {
			return "", "", fmt.Errorf("iterate tags: %w", err)
		}

		sort.Strings(tags)
		return strings.Join(tags, "\n"), "", nil
	})

	return tags, err
}

// CreateTag creates a new tag at the given hash.
func (r *Repository) CreateTag(ctx context.Context, name, hash string, opts TagOpts) error {
	args := []string{"tag"}

	if opts.Annotate || opts.Message != "" {
		args = append(args, "-a")
	}
	if opts.Sign {
		args = append(args, "-s")
	}
	if opts.Force {
		args = append(args, "-f")
	}
	if opts.LocalUser != "" {
		args = append(args, "-u", opts.LocalUser)
	}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}

	args = append(args, name, hash)

	_, err := r.runGit(ctx, args...)
	return err
}

// DeleteTag deletes a tag by name.
func (r *Repository) DeleteTag(ctx context.Context, name string) error {
	_, err := r.runGit(ctx, "tag", "-d", name)
	return err
}

// PushTag pushes a single tag to a remote.
func (r *Repository) PushTag(ctx context.Context, remote, name string) error {
	_, err := r.runGit(ctx, "push", remote, "refs/tags/"+name)
	return err
}

// PruneRemoteTags removes remote tags that no longer exist on the remote.
func (r *Repository) PruneRemoteTags(ctx context.Context, remote string) error {
	_, err := r.runGit(ctx, "fetch", "--prune", remote, "+refs/tags/*:refs/tags/*")
	return err
}
