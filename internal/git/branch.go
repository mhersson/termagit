package git

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

var safeConfigKeyRe = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)

// Branch represents a git branch with tracking information.
type Branch struct {
	Name       string
	IsRemote   bool
	IsCurrent  bool
	Upstream   string // e.g. "origin/main"
	Ahead      int
	Behind     int
	LastCommit LogEntry
}

// ListBranches returns all local branches.
func (r *Repository) ListBranches(ctx context.Context) ([]Branch, error) {
	// Get current branch name (works even for unborn HEAD)
	currentName := ""
	if unborn, branchName := r.isUnbornHEAD(ctx); unborn {
		currentName = branchName
	} else {
		head, err := r.raw.Head()
		if err != nil {
			return nil, fmt.Errorf("get HEAD: %w", err)
		}
		if head.Name().IsBranch() {
			currentName = head.Name().Short()
		}
	}

	refs, err := r.raw.References()
	if err != nil {
		return nil, fmt.Errorf("list references: %w", err)
	}

	cfg, _ := r.raw.Config()

	var branches []Branch
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsBranch() {
			return nil
		}

		name := ref.Name().Short()
		b := Branch{
			Name:      name,
			IsCurrent: name == currentName,
		}

		// Check for tracking info
		if cfg != nil {
			if branchCfg, ok := cfg.Branches[name]; ok && branchCfg.Remote != "" {
				mergeBranch := branchCfg.Merge.Short()
				b.Upstream = branchCfg.Remote + "/" + mergeBranch
			}
		}

		branches = append(branches, b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterate branches: %w", err)
	}

	// For unborn HEAD, we may have no branches yet but we know the current branch name
	// Add a placeholder entry for it if not found
	if currentName != "" {
		found := false
		for _, b := range branches {
			if b.Name == currentName {
				found = true
				break
			}
		}
		if !found {
			branches = append([]Branch{{Name: currentName, IsCurrent: true}}, branches...)
		}
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	return branches, nil
}

// ListRemoteBranches returns all remote tracking branches.
func (r *Repository) ListRemoteBranches(ctx context.Context) ([]Branch, error) {
	refs, err := r.raw.References()
	if err != nil {
		return nil, fmt.Errorf("list references: %w", err)
	}

	var branches []Branch
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsRemote() {
			return nil
		}

		name := ref.Name().Short() // e.g. "origin/main"
		b := Branch{
			Name:     name,
			IsRemote: true,
		}
		branches = append(branches, b)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterate remote branches: %w", err)
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	return branches, nil
}

// CurrentBranch returns the name of the current branch.
// Returns empty string if HEAD is detached.
// Returns branch name even for unborn HEAD (no commits yet).
func (r *Repository) CurrentBranch(ctx context.Context) (string, error) {
	name, _, err := r.logOp(ctx, "git rev-parse --abbrev-ref HEAD", func() (string, string, error) {
		// Check for unborn HEAD first
		if unborn, branchName := r.isUnbornHEAD(ctx); unborn {
			return branchName, "", nil
		}

		head, err := r.raw.Head()
		if err != nil {
			return "", "", fmt.Errorf("get HEAD: %w", err)
		}
		if !head.Name().IsBranch() {
			return "", "", nil // detached
		}
		return head.Name().Short(), "", nil
	})
	return name, err
}

// CurrentUpstream returns the remote and branch name of the current branch's upstream.
func (r *Repository) CurrentUpstream(ctx context.Context) (remote, branch string, err error) {
	head, err := r.raw.Head()
	if err != nil {
		return "", "", fmt.Errorf("get HEAD: %w", err)
	}
	if !head.Name().IsBranch() {
		return "", "", nil
	}

	cfg, err := r.raw.Config()
	if err != nil {
		return "", "", nil //nolint:nilerr // config unreadable → treat as no upstream configured
	}

	branchName := head.Name().Short()
	branchCfg, ok := cfg.Branches[branchName]
	if !ok || branchCfg.Remote == "" {
		return "", "", nil
	}

	return branchCfg.Remote, branchCfg.Merge.Short(), nil
}

// CurrentPushRemote returns the push remote and branch for the current branch.
// Only returns a value when branch.<name>.pushRemote is explicitly configured.
func (r *Repository) CurrentPushRemote(ctx context.Context) (remote, branch string, err error) {
	head, err := r.raw.Head()
	if err != nil {
		return "", "", fmt.Errorf("get HEAD: %w", err)
	}
	if !head.Name().IsBranch() {
		return "", "", nil
	}

	branchName := head.Name().Short()

	cfg, err := r.raw.Config()
	if err != nil {
		return "", "", nil //nolint:nilerr // config unreadable → treat as no push remote configured
	}

	// go-git doesn't directly expose pushRemote, so check raw config sections
	for _, raw := range cfg.Raw.Sections {
		if raw.Name != "branch" {
			continue
		}
		for _, sub := range raw.Subsections {
			if sub.Name != branchName {
				continue
			}
			pushRemote := sub.Option("pushRemote")
			if pushRemote != "" {
				return pushRemote, branchName, nil
			}
		}
	}

	return "", "", nil
}

// Checkout switches to the given branch.
func (r *Repository) Checkout(ctx context.Context, name string) error {
	_, err := r.runGit(ctx, "checkout", name)
	if err != nil {
		return fmt.Errorf("checkout %s: %w", name, err)
	}
	return nil
}

// CheckoutNewBranch creates a new branch at base and switches to it.
func (r *Repository) CheckoutNewBranch(ctx context.Context, name, base string) error {
	_, err := r.runGit(ctx, "checkout", "-b", name, base)
	if err != nil {
		return fmt.Errorf("checkout new branch %s: %w", name, err)
	}
	return nil
}

// CreateBranch creates a new branch at base without switching to it.
func (r *Repository) CreateBranch(ctx context.Context, name, base string) error {
	_, err := r.runGit(ctx, "branch", name, base)
	if err != nil {
		return fmt.Errorf("create branch %s: %w", name, err)
	}
	return nil
}

// MoveBranch forcibly moves a branch to point at a different commit.
// Equivalent to `git branch -f <name> <target>`.
func (r *Repository) MoveBranch(ctx context.Context, name, target string) error {
	_, err := r.runGit(ctx, "branch", "-f", name, target)
	if err != nil {
		return fmt.Errorf("move branch %s to %s: %w", name, target, err)
	}
	return nil
}

// RenameBranch renames a branch.
func (r *Repository) RenameBranch(ctx context.Context, oldName, newName string) error {
	_, err := r.runGit(ctx, "branch", "-m", oldName, newName)
	if err != nil {
		return fmt.Errorf("rename branch %s to %s: %w", oldName, newName, err)
	}
	return nil
}

// DeleteBranch deletes a branch. If force is true, uses -D instead of -d.
func (r *Repository) DeleteBranch(ctx context.Context, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.runGit(ctx, "branch", flag, name)
	if err != nil {
		return fmt.Errorf("delete branch %s: %w", name, err)
	}
	return nil
}

// SetUpstream sets the upstream for a local branch.
func (r *Repository) SetUpstream(ctx context.Context, local, remote string) error {
	_, err := r.runGit(ctx, "branch", "--set-upstream-to="+remote, local)
	if err != nil {
		return fmt.Errorf("set upstream for %s: %w", local, err)
	}
	return nil
}

// SetBranchConfig sets a branch configuration value.
func (r *Repository) SetBranchConfig(ctx context.Context, branch, key, value string) error {
	if !safeConfigKeyRe.MatchString(branch) {
		return fmt.Errorf("invalid branch name for config key: %q", branch)
	}
	configKey := "branch." + branch + "." + key
	_, err := r.runGit(ctx, "config", configKey, value)
	if err != nil {
		return fmt.Errorf("set branch config %s.%s: %w", branch, key, err)
	}
	return nil
}

// SpinOffBranch creates a new branch at HEAD, then resets the current branch to upstream.
func (r *Repository) SpinOffBranch(ctx context.Context, name string) error {
	// Create the new branch at current HEAD
	_, err := r.runGit(ctx, "branch", name)
	if err != nil {
		return fmt.Errorf("spin-off create branch: %w", err)
	}

	// Find upstream to reset to
	remote, branch, err := r.CurrentUpstream(ctx)
	if err != nil || remote == "" {
		// No upstream, just checkout the new branch
		return r.Checkout(ctx, name)
	}

	// Reset current branch to upstream
	_, err = r.runGit(ctx, "reset", "--hard", remote+"/"+branch)
	if err != nil {
		return fmt.Errorf("spin-off reset: %w", err)
	}

	// Checkout the new branch
	return r.Checkout(ctx, name)
}

// SpinOutBranch creates a new branch at HEAD without checking it out.
func (r *Repository) SpinOutBranch(ctx context.Context, name string) error {
	_, err := r.runGit(ctx, "branch", name)
	if err != nil {
		return fmt.Errorf("spin-out branch: %w", err)
	}
	return nil
}

// RecentBranches returns branches sorted by most recently checked out.
func (r *Repository) RecentBranches(ctx context.Context) ([]Branch, error) {
	out, err := r.runGit(ctx, "branch", "--sort=-committerdate", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("recent branches: %w", err)
	}

	var branches []Branch
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branches = append(branches, Branch{Name: line})
	}
	return branches, nil
}

// PullRequestURL generates a URL for creating a pull request for the given branch.
func (r *Repository) PullRequestURL(ctx context.Context, branch string) (string, error) {
	out, err := r.runGit(ctx, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(out)
	webURL := remoteURLToWeb(remoteURL)
	if webURL == "" {
		return "", fmt.Errorf("cannot determine web URL from remote: %s", remoteURL)
	}

	return webURL + "/compare/" + url.PathEscape(branch) + "?expand=1", nil
}
