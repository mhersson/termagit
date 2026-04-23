package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addAndCommitDisk creates a file on-disk, stages, and commits.
func addAndCommitDisk(t *testing.T, r *Repository, path, content, msg string) {
	t.Helper()
	ctx := context.Background()

	fullPath := filepath.Join(r.path, path)
	require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
	require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))

	_, err := r.runGit(ctx, "add", path)
	require.NoError(t, err)

	out, err := r.runGit(ctx, "commit", "-m", msg,
		"--author", "Test User <test@example.com>",
		"--date", "2024-01-15T10:30:00+00:00")
	require.NoError(t, err, "commit failed: %s", out)
}

// trimOutput trims whitespace from git command output.
func trimOutput(s string) string {
	return string([]byte(s[:len(s)-1])) // strip trailing newline
}

// validateHex tests

func TestValidateHex_ValidShortHash(t *testing.T) {
	err := validateHex("abc1234")
	assert.NoError(t, err)
}

func TestValidateHex_ValidFullHash(t *testing.T) {
	err := validateHex("abc1234def5678901234567890abcdef12345678")
	assert.NoError(t, err)
}

func TestValidateHex_RejectsQuote(t *testing.T) {
	err := validateHex("abc'def")
	assert.Error(t, err)
}

func TestValidateHex_RejectsSpace(t *testing.T) {
	err := validateHex("abc def")
	assert.Error(t, err)
}

func TestValidateHex_RejectsEmpty(t *testing.T) {
	err := validateHex("")
	assert.Error(t, err)
}

func TestRewordCommit_RejectsInvalidHash(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.RewordCommit(ctx, "abc'def0", "msg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hex")
}

func TestModifyCommit_RejectsInvalidHash(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.ModifyCommit(ctx, "abc'def0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid hex")
}

func TestRebase_NonInteractive_RebasesOnto(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit on main
	addAndCommitDisk(t, r, "main.txt", "main content", "Main commit")

	// Create a branch and add a commit
	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "feature.txt", "feature content", "Feature commit")

	// Go back to main and add another commit to diverge
	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "main2.txt", "more main", "Another main commit")

	// Now rebase feature onto main
	_, err = r.runGit(ctx, "checkout", "feature")
	require.NoError(t, err)

	err = r.Rebase(ctx, RebaseOpts{Onto: "master"})
	require.NoError(t, err)

	// feature.txt should still exist
	_, err = os.Stat(filepath.Join(r.path, "feature.txt"))
	assert.NoError(t, err, "feature.txt should exist after rebase")

	// main2.txt should also exist (rebased onto latest main)
	_, err = os.Stat(filepath.Join(r.path, "main2.txt"))
	assert.NoError(t, err, "main2.txt should exist after rebase")
}

func TestRebaseAbort_RestoresState(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create conflicting changes
	addAndCommitDisk(t, r, "conflict.txt", "original", "Add conflict file")

	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "feature version", "Feature change")

	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "main version", "Main change")

	_, err = r.runGit(ctx, "checkout", "feature")
	require.NoError(t, err)

	// Rebase should fail with conflict
	err = r.Rebase(ctx, RebaseOpts{Onto: "master"})
	require.Error(t, err, "rebase should fail due to conflict")

	// Should be in rebase state
	require.True(t, r.RebaseInProgress(), "rebase should be in progress")

	// Abort should succeed
	err = r.RebaseAbort(ctx)
	require.NoError(t, err)

	// Should no longer be in rebase state
	require.False(t, r.RebaseInProgress(), "rebase should not be in progress after abort")
}

func TestRebaseContinue_AfterConflictResolution(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create conflicting changes
	addAndCommitDisk(t, r, "conflict.txt", "original", "Add conflict file")

	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "feature version", "Feature change")

	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "main version", "Main change")

	_, err = r.runGit(ctx, "checkout", "feature")
	require.NoError(t, err)

	// Rebase onto master - will conflict
	err = r.Rebase(ctx, RebaseOpts{Onto: "master"})
	require.Error(t, err)
	require.True(t, r.RebaseInProgress())

	// Resolve the conflict
	require.NoError(t, os.WriteFile(
		filepath.Join(r.path, "conflict.txt"),
		[]byte("resolved"), 0o644))
	_, err = r.runGit(ctx, "add", "conflict.txt")
	require.NoError(t, err)

	// Continue the rebase
	err = r.RebaseContinue(ctx)
	require.NoError(t, err)

	// Should no longer be in rebase state
	require.False(t, r.RebaseInProgress())
}

func TestRebaseSkip_SkipsConflictingCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create conflicting changes
	addAndCommitDisk(t, r, "conflict.txt", "original", "Add conflict file")

	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "feature version", "Feature change")

	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "main version", "Main change")

	_, err = r.runGit(ctx, "checkout", "feature")
	require.NoError(t, err)

	// Rebase onto master - will conflict
	err = r.Rebase(ctx, RebaseOpts{Onto: "master"})
	require.Error(t, err)
	require.True(t, r.RebaseInProgress())

	// Skip the conflicting commit
	err = r.RebaseSkip(ctx)
	require.NoError(t, err)

	// Should no longer be in rebase state
	require.False(t, r.RebaseInProgress())
}
