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

func TestCherryPick_AppliesCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit on a feature branch
	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "feature.txt", "feature content", "Feature commit")

	// Get the hash of the feature commit
	featureHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	featureHash = strings.TrimSpace(featureHash)

	// Go back to master
	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)

	// Cherry-pick the feature commit
	err = r.CherryPick(ctx, []string{featureHash}, CherryPickOpts{})
	require.NoError(t, err)

	// Verify feature.txt exists on master
	_, err = os.Stat(filepath.Join(r.path, "feature.txt"))
	assert.NoError(t, err, "feature.txt should exist after cherry-pick")
}

func TestCherryPickAbort_ClearsState(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create conflicting changes
	addAndCommitDisk(t, r, "conflict.txt", "original", "Add file")

	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "feature version", "Feature change")

	featureHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	featureHash = strings.TrimSpace(featureHash)

	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "main version", "Main change")

	// Cherry-pick should fail
	err = r.CherryPick(ctx, []string{featureHash}, CherryPickOpts{})
	require.Error(t, err)
	require.True(t, r.CherryPickInProgress())

	// Abort
	err = r.CherryPickAbort(ctx)
	require.NoError(t, err)
	require.False(t, r.CherryPickInProgress())
}

func TestCherryPickContinue_AfterConflictResolution(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create conflicting changes
	addAndCommitDisk(t, r, "conflict.txt", "original", "Add file")

	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "feature version", "Feature change")

	featureHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	featureHash = strings.TrimSpace(featureHash)

	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "main version", "Main change")

	// Cherry-pick should fail
	err = r.CherryPick(ctx, []string{featureHash}, CherryPickOpts{})
	require.Error(t, err)
	require.True(t, r.CherryPickInProgress())

	// Resolve conflict
	require.NoError(t, os.WriteFile(
		filepath.Join(r.path, "conflict.txt"),
		[]byte("resolved"), 0o644))
	_, err = r.runGit(ctx, "add", "conflict.txt")
	require.NoError(t, err)

	// Continue
	err = r.CherryPickContinue(ctx)
	require.NoError(t, err)
	require.False(t, r.CherryPickInProgress())
}

func TestCherryPickApply_DoesNotCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit on a feature branch
	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "feature.txt", "feature content", "Feature commit")

	featureHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	featureHash = strings.TrimSpace(featureHash)

	// Go back to master
	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)

	// Get master HEAD before apply
	masterBefore, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)

	// Apply (no commit)
	err = r.CherryPickApply(ctx, []string{featureHash}, CherryPickOpts{})
	require.NoError(t, err)

	// HEAD should not have changed (no commit made)
	masterAfter, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(masterBefore), strings.TrimSpace(masterAfter),
		"HEAD should not change with apply (no commit)")

	// But the file should be in the working tree
	_, err = os.Stat(filepath.Join(r.path, "feature.txt"))
	assert.NoError(t, err, "feature.txt should exist after cherry-pick apply")
}

func TestCherryPickDonate_MovesCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create initial commit on master
	addAndCommitDisk(t, r, "base.txt", "base", "Base commit")

	// Create a target branch
	_, err := r.runGit(ctx, "branch", "target")
	require.NoError(t, err)

	// Add a commit on master that we'll donate
	addAndCommitDisk(t, r, "donate.txt", "donated content", "Commit to donate")
	donateHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	donateHash = strings.TrimSpace(donateHash)

	// Record master HEAD before donate
	masterBefore, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)

	// Donate the commit from master to target
	err = r.CherryPickDonate(ctx, []string{donateHash}, "master", "target", CherryPickOpts{})
	require.NoError(t, err)

	// We should be back on master
	currentBranch, err := r.runGit(ctx, "symbolic-ref", "--short", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "master", strings.TrimSpace(currentBranch))

	// Master should no longer have donate.txt
	_, err = os.Stat(filepath.Join(r.path, "donate.txt"))
	assert.True(t, os.IsNotExist(err), "donate.txt should not exist on master after donate")

	// Master HEAD should have changed (commit was dropped)
	masterAfter, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	assert.NotEqual(t, strings.TrimSpace(masterBefore), strings.TrimSpace(masterAfter),
		"master HEAD should change after rebase")

	// Target branch should have donate.txt
	_, err = r.runGit(ctx, "checkout", "target")
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(r.path, "donate.txt"))
	assert.NoError(t, err, "donate.txt should exist on target after donate")
}
