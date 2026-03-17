package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevert_RevertsCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit to revert
	addAndCommitDisk(t, r, "to-revert.txt", "content to revert", "Add file to revert")

	hash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	hash = strings.TrimSpace(hash)

	// Revert it
	err = r.Revert(ctx, []string{hash}, RevertOpts{NoEdit: true})
	require.NoError(t, err)

	// File should no longer exist
	_, err = os.Stat(filepath.Join(r.path, "to-revert.txt"))
	assert.True(t, os.IsNotExist(err), "file should be gone after revert")

	// A new commit should have been created
	newHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	assert.NotEqual(t, strings.TrimSpace(hash), strings.TrimSpace(newHash),
		"HEAD should have advanced")
}

func TestRevertChanges_DoesNotCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit to revert
	addAndCommitDisk(t, r, "to-revert.txt", "content", "Add file")
	hash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	hash = strings.TrimSpace(hash)

	headBefore := hash

	// Revert changes only (no commit)
	err = r.RevertChanges(ctx, []string{hash}, RevertOpts{})
	require.NoError(t, err)

	// HEAD should not have changed
	headAfter, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, headBefore, strings.TrimSpace(headAfter),
		"HEAD should not change with RevertChanges")
}

func TestRevertAbort_ClearsState(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a file and commit
	addAndCommitDisk(t, r, "base.txt", "base", "Add base")

	// Create a conflicting scenario: modify the file, commit, then try to revert
	// a commit that also modifies the same file
	addAndCommitDisk(t, r, "base.txt", "version 2", "Change base")
	hash2, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	hash2 = strings.TrimSpace(hash2)

	addAndCommitDisk(t, r, "base.txt", "version 3", "Change base again")

	// Revert the middle commit - should conflict because version 3 depends on it
	err = r.Revert(ctx, []string{hash2}, RevertOpts{NoEdit: true})
	require.Error(t, err, "revert should fail due to conflict")
	require.True(t, r.RevertInProgress(), "revert should be in progress")

	// Abort
	err = r.RevertAbort(ctx)
	require.NoError(t, err)
	require.False(t, r.RevertInProgress(), "revert should not be in progress after abort")
}

func TestRevertContinue_AfterConflictResolution(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Same conflict setup as above
	addAndCommitDisk(t, r, "base.txt", "base", "Add base")
	addAndCommitDisk(t, r, "base.txt", "version 2", "Change base")
	hash2, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	hash2 = strings.TrimSpace(hash2)

	addAndCommitDisk(t, r, "base.txt", "version 3", "Change base again")

	// Revert should fail
	err = r.Revert(ctx, []string{hash2}, RevertOpts{NoEdit: true})
	require.Error(t, err)
	require.True(t, r.RevertInProgress())

	// Resolve conflict
	require.NoError(t, os.WriteFile(
		filepath.Join(r.path, "base.txt"),
		[]byte("resolved"), 0o644))
	_, err = r.runGit(ctx, "add", "base.txt")
	require.NoError(t, err)

	// Continue
	err = r.RevertContinue(ctx)
	require.NoError(t, err)
	require.False(t, r.RevertInProgress())
}
