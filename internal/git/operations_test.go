package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageFile_StagesUntracked(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	addFile(t, r, "new.txt", "content")

	err := r.StageFile(ctx, "new.txt")
	require.NoError(t, err)

	status, err := r.Status(ctx)
	require.NoError(t, err)

	assert.Empty(t, status.Untracked)
	require.Len(t, status.Staged, 1)
	assert.Equal(t, "new.txt", status.Staged[0].Path())
}

func TestStageFile_StagesModified(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Modify existing file
	addFile(t, r, "README.md", "modified")

	err := r.StageFile(ctx, "README.md")
	require.NoError(t, err)

	status, err := r.Status(ctx)
	require.NoError(t, err)

	assert.Empty(t, status.Unstaged)
	require.Len(t, status.Staged, 1)
	assert.Equal(t, "README.md", status.Staged[0].Path())
}

func TestUnstageFile_UnstagesFile(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Add and stage a new file
	addFile(t, r, "staged.txt", "content")
	stageFile(t, r, "staged.txt")

	// Verify it's staged
	status, err := r.Status(ctx)
	require.NoError(t, err)
	require.Len(t, status.Staged, 1)

	// Unstage it
	err = r.UnstageFile(ctx, "staged.txt")
	require.NoError(t, err)

	// Verify it's no longer staged but is untracked
	status, err = r.Status(ctx)
	require.NoError(t, err)
	assert.Empty(t, status.Staged)
	require.Len(t, status.Untracked, 1)
	assert.Equal(t, "staged.txt", status.Untracked[0].Path())
}

func TestStageAll_StagesAllChanges(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create multiple changes
	addFile(t, r, "new1.txt", "content1")
	addFile(t, r, "new2.txt", "content2")
	addFile(t, r, "README.md", "modified")

	err := r.StageAll(ctx)
	require.NoError(t, err)

	status, err := r.Status(ctx)
	require.NoError(t, err)

	assert.Empty(t, status.Untracked)
	assert.Empty(t, status.Unstaged)
	assert.Len(t, status.Staged, 3)
}

func TestUnstageAll_UnstagesAllChanges(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Stage multiple files
	addFile(t, r, "new1.txt", "content1")
	addFile(t, r, "new2.txt", "content2")
	stageFile(t, r, "new1.txt")
	stageFile(t, r, "new2.txt")

	// Verify they're staged
	status, err := r.Status(ctx)
	require.NoError(t, err)
	require.Len(t, status.Staged, 2)

	// Unstage all
	err = r.UnstageAll(ctx)
	require.NoError(t, err)

	// Verify nothing is staged
	status, err = r.Status(ctx)
	require.NoError(t, err)
	assert.Empty(t, status.Staged)
	assert.Len(t, status.Untracked, 2)
}

func TestDiscardFile_DiscardsChanges(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Modify an existing file
	addFile(t, r, "README.md", "modified content")

	// Verify it shows as unstaged
	status, err := r.Status(ctx)
	require.NoError(t, err)
	require.Len(t, status.Unstaged, 1)

	// Discard changes
	err = r.DiscardFile(ctx, "README.md")
	require.NoError(t, err)

	// Verify no unstaged changes
	status, err = r.Status(ctx)
	require.NoError(t, err)
	assert.Empty(t, status.Unstaged)

	// Verify content is restored
	readmePath := filepath.Join(r.path, "README.md")
	content, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	assert.Equal(t, "# Test Repo\n", string(content))
}

func TestUntrackFile_RemovesFromIndex(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Add and commit a file, then untrack it
	addAndCommit(t, r, "tracked.txt", "content", "Add tracked file")

	err := r.UntrackFile(ctx, "tracked.txt")
	require.NoError(t, err)

	// File should still exist on disk
	filePath := filepath.Join(r.path, "tracked.txt")
	_, err = os.Stat(filePath)
	assert.NoError(t, err, "file should still exist on disk")

	// Should show as deleted in staging AND as untracked
	// (because file was removed from index but still exists on disk)
	status, err := r.Status(ctx)
	require.NoError(t, err)

	// Find the staged deletion entry
	var foundStagedDelete bool
	for _, entry := range status.Staged {
		if entry.Path() == "tracked.txt" && entry.Staged == FileStatusDeleted {
			foundStagedDelete = true
			break
		}
	}
	assert.True(t, foundStagedDelete, "file should be staged for deletion")

	// File should also appear as untracked (it still exists on disk)
	var foundUntracked bool
	for _, entry := range status.Untracked {
		if entry.Path() == "tracked.txt" {
			foundUntracked = true
			break
		}
	}
	assert.True(t, foundUntracked, "file should also appear as untracked")
}

func TestRenameFile_RenamesInIndex(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Commit a file first
	addAndCommit(t, r, "old.txt", "content", "Add file")

	err := r.RenameFile(ctx, "old.txt", "new.txt")
	require.NoError(t, err)

	// After git mv, the status should show either:
	// - A rename entry (kind 2 in porcelain v2), OR
	// - A delete of old.txt + add of new.txt (kind 1 entries)
	status, err := r.Status(ctx)
	require.NoError(t, err)

	// Verify the rename happened (check files exist correctly)
	oldPath := filepath.Join(r.path, "old.txt")
	newPath := filepath.Join(r.path, "new.txt")

	_, err = os.Stat(oldPath)
	assert.True(t, os.IsNotExist(err), "old file should not exist")

	_, err = os.Stat(newPath)
	assert.NoError(t, err, "new file should exist")

	// Verify something is staged (either rename or delete+add)
	assert.NotEmpty(t, status.Staged, "should have staged changes after rename")
}
