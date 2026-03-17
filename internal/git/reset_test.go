package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReset_Soft_KeepsChangesStaged(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create a second commit
	addAndCommit(t, r, "file.txt", "content", "Second commit")

	// Get current HEAD
	headBefore, err := r.HeadOID(ctx)
	require.NoError(t, err)

	// Reset soft to HEAD~1
	err = r.Reset(ctx, "HEAD~1", ResetSoft)
	require.NoError(t, err)

	// HEAD should have moved
	headAfter, err := r.HeadOID(ctx)
	require.NoError(t, err)
	require.NotEqual(t, headBefore, headAfter)

	// File should still exist and be staged
	status, err := r.runGit(ctx, "status", "--porcelain")
	require.NoError(t, err)
	require.Contains(t, status, "A  file.txt")
}

func TestReset_Hard_DiscardsAll(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create a second commit with a file
	addAndCommit(t, r, "file.txt", "content", "Second commit")

	// Reset hard to HEAD~1
	err := r.Reset(ctx, "HEAD~1", ResetHard)
	require.NoError(t, err)

	// File should be gone from working tree
	_, err = os.Stat(filepath.Join(r.path, "file.txt"))
	require.True(t, os.IsNotExist(err))
}

func TestResetFile_RestoresFileToTarget(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create file and commit
	addAndCommit(t, r, "file.txt", "original", "Add file")

	// Modify the file and stage it
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "file.txt"), []byte("modified"), 0o644))
	_, err := r.runGit(ctx, "add", "file.txt")
	require.NoError(t, err)

	// Reset the file in index
	err = r.ResetFile(ctx, "file.txt", "HEAD")
	require.NoError(t, err)

	// File should no longer be staged (but working tree still has modified)
	status, err := r.runGit(ctx, "status", "--porcelain")
	require.NoError(t, err)
	require.Contains(t, status, " M file.txt") // unstaged modification only
}
