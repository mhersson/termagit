package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Remote represents a configured git remote.
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// FetchOpts configures a git fetch operation.
type FetchOpts struct {
	Remote     string
	Prune      bool
	Tags       bool
	Force      bool
	All        bool
	Submodules bool
	Refspec    string
}

// buildArgs returns the git fetch arguments for these options.
func (o FetchOpts) buildArgs() []string {
	args := []string{"fetch"}
	if o.All {
		args = append(args, "--all")
	}
	if o.Prune {
		args = append(args, "--prune")
	}
	if o.Tags {
		args = append(args, "--tags")
	}
	if o.Force {
		args = append(args, "--force")
	}
	if o.Submodules {
		args = append(args, "--recurse-submodules")
	}
	if !o.All && o.Remote != "" {
		args = append(args, o.Remote)
	}
	if o.Refspec != "" {
		args = append(args, o.Refspec)
	}
	return args
}

// PullOpts configures a git pull operation.
type PullOpts struct {
	Remote    string
	Branch    string
	FFOnly    bool
	Rebase    bool
	Autostash bool
	Tags      bool
	Force     bool
}

// buildArgs returns the git pull arguments for these options.
func (o PullOpts) buildArgs() []string {
	args := []string{"pull"}
	if o.FFOnly {
		args = append(args, "--ff-only")
	}
	if o.Rebase {
		args = append(args, "--rebase")
	}
	if o.Autostash {
		args = append(args, "--autostash")
	}
	if o.Tags {
		args = append(args, "--tags")
	}
	if o.Force {
		args = append(args, "--force")
	}
	if o.Remote != "" {
		args = append(args, o.Remote)
	}
	if o.Branch != "" {
		args = append(args, o.Branch)
	}
	return args
}

// PushOpts configures a git push operation.
type PushOpts struct {
	Remote         string
	Branch         string
	Refspec        string
	GpgSign        string
	Force          bool
	ForceWithLease bool
	SetUpstream    bool
	DryRun         bool
	Tags           bool
	FollowTags     bool
	NoVerify       bool
}

// buildArgs returns the git push arguments for these options.
func (o PushOpts) buildArgs() []string {
	args := []string{"push"}
	if o.ForceWithLease {
		args = append(args, "--force-with-lease")
	} else if o.Force {
		args = append(args, "--force")
	}
	if o.SetUpstream {
		args = append(args, "--set-upstream")
	}
	if o.DryRun {
		args = append(args, "--dry-run")
	}
	if o.Tags {
		args = append(args, "--tags")
	}
	if o.FollowTags {
		args = append(args, "--follow-tags")
	}
	if o.NoVerify {
		args = append(args, "--no-verify")
	}
	if o.GpgSign != "" {
		args = append(args, fmt.Sprintf("--gpg-sign=%s", o.GpgSign))
	}
	if o.Remote != "" {
		args = append(args, o.Remote)
	}
	if o.Branch != "" {
		args = append(args, o.Branch)
	}
	if o.Refspec != "" {
		args = append(args, o.Refspec)
	}
	return args
}

// ListRemotes returns all configured remotes via go-git.
func (r *Repository) ListRemotes(ctx context.Context) ([]Remote, error) {
	rawRemotes, err := r.raw.Remotes()
	if err != nil {
		return nil, fmt.Errorf("list remotes: %w", err)
	}

	var remotes []Remote
	for _, rr := range rawRemotes {
		cfg := rr.Config()
		remote := Remote{
			Name: cfg.Name,
		}
		if len(cfg.URLs) > 0 {
			remote.FetchURL = cfg.URLs[0]
			remote.PushURL = cfg.URLs[0]
		}
		if len(cfg.URLs) > 1 {
			remote.PushURL = cfg.URLs[1]
		}
		remotes = append(remotes, remote)
	}

	return remotes, nil
}

// AddRemote adds a new remote to the repository.
func (r *Repository) AddRemote(ctx context.Context, name, url string) error {
	_, err := r.runGit(ctx, "remote", "add", name, url)
	if err != nil {
		return fmt.Errorf("add remote %s: %w", name, err)
	}
	// Reload go-git's view of the config
	r.reloadConfig()
	return nil
}

// RemoveRemote removes a remote from the repository.
func (r *Repository) RemoveRemote(ctx context.Context, name string) error {
	_, err := r.runGit(ctx, "remote", "remove", name)
	if err != nil {
		return fmt.Errorf("remove remote %s: %w", name, err)
	}
	r.reloadConfig()
	return nil
}

// RenameRemote renames a remote.
func (r *Repository) RenameRemote(ctx context.Context, oldName, newName string) error {
	_, err := r.runGit(ctx, "remote", "rename", oldName, newName)
	if err != nil {
		return fmt.Errorf("rename remote %s -> %s: %w", oldName, newName, err)
	}
	r.reloadConfig()
	return nil
}

// SetRemoteURL updates a remote's URL. If push is true, sets the push URL;
// otherwise sets the fetch URL.
func (r *Repository) SetRemoteURL(ctx context.Context, name, url string, push bool) error {
	args := []string{"remote", "set-url"}
	if push {
		args = append(args, "--push")
	}
	args = append(args, name, url)

	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("set remote URL %s: %w", name, err)
	}
	r.reloadConfig()
	return nil
}

// PruneRemote removes stale remote-tracking references.
func (r *Repository) PruneRemote(ctx context.Context, name string) error {
	_, err := r.runGit(ctx, "remote", "prune", name)
	if err != nil {
		return fmt.Errorf("prune remote %s: %w", name, err)
	}
	return nil
}

// Fetch downloads objects and refs from a remote.
func (r *Repository) Fetch(ctx context.Context, opts FetchOpts) error {
	args := opts.buildArgs()
	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	return nil
}

// Pull fetches and integrates changes from a remote.
func (r *Repository) Pull(ctx context.Context, opts PullOpts) error {
	args := opts.buildArgs()
	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	return nil
}

// Push sends local commits to a remote.
func (r *Repository) Push(ctx context.Context, opts PushOpts) error {
	args := opts.buildArgs()
	_, err := r.runGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	return nil
}

// PushTags pushes all tags to a remote.
func (r *Repository) PushTags(ctx context.Context, remote string) error {
	_, err := r.runGit(ctx, "push", remote, "--tags")
	if err != nil {
		return fmt.Errorf("push tags to %s: %w", remote, err)
	}
	return nil
}

// CommitURL returns the web URL for a commit on a known hosting service.
// Returns empty string if the remote URL is not from a known host.
func (r *Repository) CommitURL(ctx context.Context, hash string) (string, error) {
	remotes, err := r.ListRemotes(ctx)
	if err != nil {
		return "", err
	}
	if len(remotes) == 0 {
		return "", nil
	}

	// Use origin if available, otherwise first remote
	url := ""
	for _, rm := range remotes {
		if rm.Name == "origin" {
			url = rm.FetchURL
			break
		}
	}
	if url == "" {
		url = remotes[0].FetchURL
	}

	webURL := remoteURLToWeb(url)
	if webURL == "" {
		return "", nil
	}

	return webURL + "/commit/" + hash, nil
}

// remoteURLToWeb converts a git remote URL to a web URL.
func remoteURLToWeb(rawURL string) string {
	url := rawURL

	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.HasPrefix(url, "git@") {
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
		url = "https://" + url
	}

	// Handle ssh:// URLs
	if strings.HasPrefix(url, "ssh://") {
		url = strings.TrimPrefix(url, "ssh://")
		// Remove user@ if present
		if at := strings.Index(url, "@"); at >= 0 {
			url = url[at+1:]
		}
		url = "https://" + url
	}

	// Strip .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Only return for known hosts
	if strings.Contains(url, "github.com") ||
		strings.Contains(url, "gitlab.com") ||
		strings.Contains(url, "bitbucket.org") {
		return url
	}

	return ""
}

// reloadConfig re-opens the repository to pick up config changes
// made via shell-out commands.
func (r *Repository) reloadConfig() {
	// go-git caches config; re-opening from path forces a refresh
	if raw, err := git.PlainOpen(r.path); err == nil {
		r.raw = raw
	}
}
