package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fast tests (in-memory) ---

func TestListBranches_ReturnsCurrent(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	branches, err := r.ListBranches(ctx)
	require.NoError(t, err)

	// Should have at least one branch (the default branch)
	require.NotEmpty(t, branches)

	// Find the current branch
	var current *Branch
	for i := range branches {
		if branches[i].IsCurrent {
			current = &branches[i]
			break
		}
	}
	require.NotNil(t, current, "should have a current branch")
	assert.NotEmpty(t, current.Name)
}

func TestListBranches_ReturnsAll(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Create additional branches
	head, err := r.raw.Head()
	require.NoError(t, err)

	err = r.raw.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewBranchReferenceName("feature-a"), head.Hash()),
	)
	require.NoError(t, err)

	err = r.raw.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewBranchReferenceName("feature-b"), head.Hash()),
	)
	require.NoError(t, err)

	branches, err := r.ListBranches(ctx)
	require.NoError(t, err)

	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Name
	}
	assert.Contains(t, names, "feature-a")
	assert.Contains(t, names, "feature-b")
	assert.GreaterOrEqual(t, len(branches), 3) // default + 2 new
}

func TestListBranches_TrackingInfo(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	head, err := r.raw.Head()
	require.NoError(t, err)

	// Create a remote tracking ref
	err = r.raw.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewRemoteReferenceName("origin", "main"), head.Hash()),
	)
	require.NoError(t, err)

	// Configure tracking
	cfg, err := r.raw.Config()
	require.NoError(t, err)

	branchName := head.Name().Short()
	cfg.Branches[branchName] = &config.Branch{
		Name:   branchName,
		Remote: "origin",
		Merge:  plumbing.NewBranchReferenceName("main"),
	}
	err = r.raw.SetConfig(cfg)
	require.NoError(t, err)

	branches, err := r.ListBranches(ctx)
	require.NoError(t, err)

	var current *Branch
	for i := range branches {
		if branches[i].IsCurrent {
			current = &branches[i]
			break
		}
	}
	require.NotNil(t, current)
	assert.Equal(t, "origin/main", current.Upstream)
}

func TestListRemoteBranches_ReturnsList(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	head, err := r.raw.Head()
	require.NoError(t, err)

	// Create remote tracking refs
	err = r.raw.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewRemoteReferenceName("origin", "main"), head.Hash()),
	)
	require.NoError(t, err)

	err = r.raw.Storer.SetReference(
		plumbing.NewHashReference(plumbing.NewRemoteReferenceName("origin", "develop"), head.Hash()),
	)
	require.NoError(t, err)

	branches, err := r.ListRemoteBranches(ctx)
	require.NoError(t, err)

	require.Len(t, branches, 2)
	for _, b := range branches {
		assert.True(t, b.IsRemote)
	}

	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Name
	}
	assert.Contains(t, names, "origin/main")
	assert.Contains(t, names, "origin/develop")
}

func TestCurrentBranch_ReturnsName(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	name, err := r.CurrentBranch(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, name)
}

func TestCurrentBranch_DetachedHead(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Detach HEAD by checking out a specific commit
	head, err := r.raw.Head()
	require.NoError(t, err)

	wt, err := r.raw.Worktree()
	require.NoError(t, err)
	err = wt.Checkout(&gogit.CheckoutOptions{Hash: head.Hash()})
	require.NoError(t, err)

	name, err := r.CurrentBranch(ctx)
	require.NoError(t, err)
	assert.Equal(t, "", name, "detached HEAD should return empty string")
}

// --- Slow tests (shell-out, real filesystem) ---

func TestCheckout_SwitchesBranch(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a new branch via git
	_, err := r.runGit(ctx, "branch", "feature")
	require.NoError(t, err)

	err = r.Checkout(ctx, "feature")
	require.NoError(t, err)

	name, err := r.CurrentBranch(ctx)
	require.NoError(t, err)
	assert.Equal(t, "feature", name)
}

func TestCheckoutNewBranch_CreatesAndSwitches(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.CheckoutNewBranch(ctx, "new-feature", "HEAD")
	require.NoError(t, err)

	name, err := r.CurrentBranch(ctx)
	require.NoError(t, err)
	assert.Equal(t, "new-feature", name)
}

func TestCreateBranch_ExistsAfterCreate(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.CreateBranch(ctx, "new-branch", "HEAD")
	require.NoError(t, err)

	// Should appear in branch list
	out, err := r.runGit(ctx, "branch", "--list", "new-branch")
	require.NoError(t, err)
	assert.Contains(t, out, "new-branch")
}

func TestRenameBranch_OldNameGone(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a branch, then rename it
	_, err := r.runGit(ctx, "branch", "old-name")
	require.NoError(t, err)

	err = r.RenameBranch(ctx, "old-name", "new-name")
	require.NoError(t, err)

	// Old name should be gone
	out, err := r.runGit(ctx, "branch", "--list")
	require.NoError(t, err)
	assert.NotContains(t, out, "old-name")
	assert.Contains(t, out, "new-name")
}

func TestDeleteBranch_BranchGone(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	_, err := r.runGit(ctx, "branch", "to-delete")
	require.NoError(t, err)

	err = r.DeleteBranch(ctx, "to-delete", false)
	require.NoError(t, err)

	out, err := r.runGit(ctx, "branch", "--list")
	require.NoError(t, err)
	assert.NotContains(t, out, "to-delete")
}

func TestDeleteBranch_Force_DeletesUnmerged(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create branch with unique commit
	_, err := r.runGit(ctx, "checkout", "-b", "unmerged-branch")
	require.NoError(t, err)

	filePath := filepath.Join(r.path, "unmerged.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("unmerged content"), 0o644))
	_, err = r.runGit(ctx, "add", "unmerged.txt")
	require.NoError(t, err)
	_, err = r.runGit(ctx, "commit", "-m", "unmerged commit")
	require.NoError(t, err)

	// Switch back to original branch
	_, err = r.runGit(ctx, "checkout", "-")
	require.NoError(t, err)

	// Normal delete should fail
	err = r.DeleteBranch(ctx, "unmerged-branch", false)
	assert.Error(t, err)

	// Force delete should succeed
	err = r.DeleteBranch(ctx, "unmerged-branch", true)
	require.NoError(t, err)
}

func TestSetBranchConfig_SetsValue(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Get current branch name
	name, err := r.CurrentBranch(ctx)
	require.NoError(t, err)

	err = r.SetBranchConfig(ctx, name, "description", "test description")
	require.NoError(t, err)

	// Verify it was set
	out, err := r.runGit(ctx, "config", "--get", "branch."+name+".description")
	require.NoError(t, err)
	assert.Contains(t, out, "test description")
}

func TestCurrentPushRemote_ReturnsEmptyWhenOnlyUpstreamConfigured(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Configure upstream tracking (remote + merge) but NOT pushRemote
	cfg, err := r.raw.Config()
	require.NoError(t, err)

	head, err := r.raw.Head()
	require.NoError(t, err)
	branchName := head.Name().Short()

	cfg.Remotes["origin"] = &config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.com/repo.git"},
	}
	cfg.Branches[branchName] = &config.Branch{
		Name:   branchName,
		Remote: "origin",
		Merge:  plumbing.NewBranchReferenceName(branchName),
	}
	require.NoError(t, r.raw.SetConfig(cfg))

	remote, branch, err := r.CurrentPushRemote(ctx)
	require.NoError(t, err)
	assert.Empty(t, remote, "push remote should be empty when only upstream is configured")
	assert.Empty(t, branch)
}

func TestCurrentPushRemote_ReturnsValueWhenExplicitlyConfigured(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	branchName, err := r.CurrentBranch(ctx)
	require.NoError(t, err)

	// Configure an explicit pushRemote
	_, err = r.runGit(ctx, "config", "branch."+branchName+".pushRemote", "myfork")
	require.NoError(t, err)

	// Re-open to pick up config
	r2, err := Open(r.path, nil)
	require.NoError(t, err)

	remote, branch, err := r2.CurrentPushRemote(ctx)
	require.NoError(t, err)
	assert.Equal(t, "myfork", remote)
	assert.Equal(t, branchName, branch)
}
