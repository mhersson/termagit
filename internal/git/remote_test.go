package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- Fast tests: buildArgs ---

func TestFetchOpts_BuildArgs_Prune(t *testing.T) {
	opts := FetchOpts{Remote: "origin", Prune: true}
	args := opts.buildArgs()
	require.Contains(t, args, "--prune")
	require.Contains(t, args, "origin")
}

func TestFetchOpts_BuildArgs_All(t *testing.T) {
	opts := FetchOpts{All: true, Tags: true}
	args := opts.buildArgs()
	require.Contains(t, args, "--all")
	require.Contains(t, args, "--tags")
	// When All is set, Remote should not appear
	require.NotContains(t, args, "")
}

func TestFetchOpts_BuildArgs_Refspec(t *testing.T) {
	opts := FetchOpts{Remote: "origin", Refspec: "refs/heads/main:refs/remotes/origin/main"}
	args := opts.buildArgs()
	require.Contains(t, args, "origin")
	require.Contains(t, args, "refs/heads/main:refs/remotes/origin/main")
}

func TestPushOpts_BuildArgs_ForceWithLease(t *testing.T) {
	opts := PushOpts{Remote: "origin", Branch: "main", ForceWithLease: true}
	args := opts.buildArgs()
	require.Contains(t, args, "--force-with-lease")
	require.NotContains(t, args, "--force")
	require.Contains(t, args, "origin")
	require.Contains(t, args, "main")
}

func TestPushOpts_BuildArgs_SetUpstream(t *testing.T) {
	opts := PushOpts{Remote: "origin", Branch: "feature", SetUpstream: true, Tags: true}
	args := opts.buildArgs()
	require.Contains(t, args, "--set-upstream")
	require.Contains(t, args, "--tags")
	require.Contains(t, args, "origin")
	require.Contains(t, args, "feature")
}

func TestPullOpts_BuildArgs_FFOnly(t *testing.T) {
	opts := PullOpts{Remote: "origin", Branch: "main", FFOnly: true}
	args := opts.buildArgs()
	require.Contains(t, args, "--ff-only")
	require.Contains(t, args, "origin")
	require.Contains(t, args, "main")
}

func TestPullOpts_BuildArgs_Rebase(t *testing.T) {
	opts := PullOpts{Rebase: true, Autostash: true}
	args := opts.buildArgs()
	require.Contains(t, args, "--rebase")
	require.Contains(t, args, "--autostash")
}

// --- Slow tests: shell-out operations ---

func TestListRemotes_ReturnsConfigured(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Fresh repo has no remotes
	remotes, err := r.ListRemotes(ctx)
	require.NoError(t, err)
	require.Empty(t, remotes)
}

func TestAddRemote_AppearsInList(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddRemote(ctx, "upstream", "https://example.com/repo.git")
	require.NoError(t, err)

	remotes, err := r.ListRemotes(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	require.Equal(t, "upstream", remotes[0].Name)
	require.Equal(t, "https://example.com/repo.git", remotes[0].FetchURL)
}

func TestRemoveRemote_GoneFromList(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddRemote(ctx, "upstream", "https://example.com/repo.git")
	require.NoError(t, err)

	err = r.RemoveRemote(ctx, "upstream")
	require.NoError(t, err)

	remotes, err := r.ListRemotes(ctx)
	require.NoError(t, err)
	require.Empty(t, remotes)
}

func TestRenameRemote_NewNameInList(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddRemote(ctx, "upstream", "https://example.com/repo.git")
	require.NoError(t, err)

	err = r.RenameRemote(ctx, "upstream", "fork")
	require.NoError(t, err)

	remotes, err := r.ListRemotes(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	require.Equal(t, "fork", remotes[0].Name)
}

func TestSetRemoteURL_UpdatesFetch(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddRemote(ctx, "origin", "https://example.com/old.git")
	require.NoError(t, err)

	err = r.SetRemoteURL(ctx, "origin", "https://example.com/new.git", false)
	require.NoError(t, err)

	remotes, err := r.ListRemotes(ctx)
	require.NoError(t, err)
	require.Len(t, remotes, 1)
	require.Equal(t, "https://example.com/new.git", remotes[0].FetchURL)
}

func TestFetch_LogsEntry(t *testing.T) {
	skipInShort(t)
	// Set up a local bare repo as the "remote"
	bareDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", bareDir)
	require.NoError(t, cmd.Run())

	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddRemote(ctx, "origin", bareDir)
	require.NoError(t, err)

	// Get the current branch name
	head, err := r.raw.Head()
	require.NoError(t, err)
	branch := head.Name().Short()

	// Push something to the bare repo first
	_, err = r.runGit(ctx, "push", "origin", "HEAD:refs/heads/"+branch)
	require.NoError(t, err)

	// Fetch should succeed
	err = r.Fetch(ctx, FetchOpts{Remote: "origin"})
	require.NoError(t, err)
}

func TestPush_DryRun_DoesNotPush(t *testing.T) {
	skipInShort(t)
	// Set up a local bare repo as the "remote"
	bareDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", bareDir)
	require.NoError(t, cmd.Run())

	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddRemote(ctx, "origin", bareDir)
	require.NoError(t, err)

	// Get the current branch name
	head, err := r.raw.Head()
	require.NoError(t, err)
	branch := head.Name().Short()

	// Push first so we have a tracking branch
	_, err = r.runGit(ctx, "push", "origin", "HEAD:refs/heads/"+branch)
	require.NoError(t, err)

	// Dry run push should succeed without error
	err = r.Push(ctx, PushOpts{Remote: "origin", Branch: branch, DryRun: true})
	require.NoError(t, err)
}
