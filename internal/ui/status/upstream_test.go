package status

import (
	"os"
	"os/exec"
	"testing"

	"github.com/mhersson/conjit/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initTestRepo creates a real git repo in a temp dir with an initial commit.
// Returns the opened *git.Repository and cleanup is automatic via t.TempDir().
func initTestRepo(t *testing.T) *git.Repository {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	run("init", "-b", "main")
	run("commit", "--allow-empty", "-m", "initial")

	repo, err := git.Open(dir, nil)
	require.NoError(t, err)
	return repo
}

// configureUpstream sets branch.main.remote and branch.main.merge to simulate
// an upstream tracking branch.
func configureUpstream(t *testing.T, repo *git.Repository, remote, branch string) {
	t.Helper()
	dir := repoPath(t, repo)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}

	run("config", "branch.main.remote", remote)
	run("config", "branch.main.merge", "refs/heads/"+branch)
}

// repoPath extracts the working directory path from a *git.Repository.
func repoPath(t *testing.T, repo *git.Repository) string {
	t.Helper()
	// Use CurrentBranch to verify the repo works, then get path from the repo.
	// The git.Repository stores path internally; we get it via the repo's path field.
	// Since path is unexported, we'll use the repo's WorkDir method if available.
	// Fallback: store the dir from initTestRepo.
	// Actually, let's just use repo.Path() if exported.
	return repo.Path()
}

func TestGetUpstreamRef_ReturnsRefWhenConfigured(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	repo := initTestRepo(t)
	configureUpstream(t, repo, "origin", "main")

	// Re-open to pick up config changes
	repo, err := git.Open(repo.Path(), nil)
	require.NoError(t, err)

	ref := getUpstreamRef(repo)
	assert.Equal(t, "origin/main", ref)
}

func TestGetUpstreamRef_ReturnsEmptyWhenNoUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	repo := initTestRepo(t)

	ref := getUpstreamRef(repo)
	assert.Empty(t, ref)
}

func TestGetPushRemoteRef_ReturnsRefWhenConfigured(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	repo := initTestRepo(t)
	configureUpstream(t, repo, "origin", "main")

	// Re-open to pick up config changes
	repo, err := git.Open(repo.Path(), nil)
	require.NoError(t, err)

	ref := getPushRemoteRef(repo)
	assert.Equal(t, "origin/main", ref)
}

func TestGetPushRemoteRef_ReturnsEmptyWhenNoUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	repo := initTestRepo(t)

	ref := getPushRemoteRef(repo)
	assert.Empty(t, ref)
}

func TestGetUpstreamRef_DifferentRemoteAndBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	repo := initTestRepo(t)
	configureUpstream(t, repo, "upstream", "develop")

	repo, err := git.Open(repo.Path(), nil)
	require.NoError(t, err)

	ref := getUpstreamRef(repo)
	assert.Equal(t, "upstream/develop", ref)
}

func TestLoadStatusCmd_PopulatesHeadStateUpstream(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	repo := initTestRepo(t)
	configureUpstream(t, repo, "origin", "main")

	repo, err := git.Open(repo.Path(), nil)
	require.NoError(t, err)

	// Create a fake remote ref so LogAhead/LogBehind don't fail
	dir := repo.Path()
	cmd := exec.Command("git", "update-ref", "refs/remotes/origin/main", "HEAD")
	cmd.Dir = dir
	out, runErr := cmd.CombinedOutput()
	require.NoError(t, runErr, "update-ref: %s", out)

	// Re-open to pick up ref changes
	repo, err = git.Open(dir, nil)
	require.NoError(t, err)

	// Execute loadStatusCmd and capture the message
	fn := loadStatusCmd(repo, nil)
	msg := fn()

	loaded, ok := msg.(statusLoadedMsg)
	require.True(t, ok, "expected statusLoadedMsg, got %T", msg)
	require.NoError(t, loaded.err)

	assert.Equal(t, "origin", loaded.head.UpstreamRemote)
	assert.Equal(t, "main", loaded.head.UpstreamBranch)
}
