package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRefs_LocalBranches(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create a second branch
	cmd := exec.Command("git", "branch", "feature/test")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	result, err := r.ListRefs(ctx)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(result.LocalBranches), 2)

	var found bool
	for _, ref := range result.LocalBranches {
		if ref.Name == "feature/test" {
			found = true
			assert.Equal(t, RefTypeLocalBranch, ref.Type)
			assert.Equal(t, "feature/test", ref.UnambiguousName)
			assert.NotEmpty(t, ref.Oid)
			assert.NotEmpty(t, ref.AbbrevOid)
		}
	}
	assert.True(t, found, "feature/test branch should be in local branches")
}

func TestListRefs_RemoteBranches(t *testing.T) {
	skipInShort(t)

	// Create a bare "remote" repo and clone it
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "bare.git")
	cloneDir := filepath.Join(dir, "clone")

	// Init bare repo with a commit
	cmd := exec.Command("git", "init", "--bare", bareDir)
	require.NoError(t, cmd.Run())

	// Clone the bare repo
	cmd = exec.Command("git", "clone", bareDir, cloneDir)
	require.NoError(t, cmd.Run())

	// Create an initial commit in clone
	readmePath := filepath.Join(cloneDir, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("# Test\n"), 0o644))

	for _, args := range [][]string{
		{"git", "-C", cloneDir, "add", "README.md"},
		{"git", "-C", cloneDir, "-c", "user.name=Test", "-c", "user.email=test@test.com", "commit", "-m", "init"},
		{"git", "-C", cloneDir, "push", "origin", "HEAD"},
	} {
		cmd = exec.Command(args[0], args[1:]...)
		require.NoError(t, cmd.Run(), "failed: %v", args)
	}

	r, err := Open(cloneDir, nil)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := r.ListRefs(ctx)
	require.NoError(t, err)

	assert.Contains(t, result.RemoteBranches, "origin", "should have origin remote group")
	if branches, ok := result.RemoteBranches["origin"]; ok {
		assert.GreaterOrEqual(t, len(branches), 1)
		for _, ref := range branches {
			assert.Equal(t, RefTypeRemoteBranch, ref.Type)
			assert.Equal(t, "origin", ref.Remote)
		}
	}
}

func TestListRefs_Tags(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create a tag
	cmd := exec.Command("git", "tag", "v1.0")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	result, err := r.ListRefs(ctx)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(result.Tags), 1)

	var found bool
	for _, ref := range result.Tags {
		if ref.Name == "v1.0" {
			found = true
			assert.Equal(t, RefTypeTag, ref.Type)
			assert.Equal(t, "tags/v1.0", ref.UnambiguousName)
		}
	}
	assert.True(t, found, "v1.0 tag should be in tags")
}

func TestListRefs_CurrentBranchMarkedAsHead(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	result, err := r.ListRefs(ctx)
	require.NoError(t, err)

	var headFound bool
	for _, ref := range result.LocalBranches {
		if ref.Head {
			headFound = true
		}
	}
	assert.True(t, headFound, "one local branch should be marked as HEAD")
}

func TestListRefs_EmptyRepo(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Repo has one branch and one commit, should not crash
	result, err := r.ListRefs(ctx)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, len(result.LocalBranches), 1)
}
