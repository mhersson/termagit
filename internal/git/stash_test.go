package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListStashes_EmptyRepo_ReturnsEmpty(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)

	stashes, err := r.ListStashes(context.Background())
	require.NoError(t, err)
	require.Empty(t, stashes)
}

func TestListStashes_WithStashes_ReturnsEntries(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)

	// Create a modified file
	filePath := filepath.Join(r.path, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("original content\n"), 0o644))

	// Stage and commit
	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add test file")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Modify the file
	require.NoError(t, os.WriteFile(filePath, []byte("modified content\n"), 0o644))

	// Create first stash
	cmd = exec.Command("git", "stash", "push", "-m", "First stash")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Modify again
	require.NoError(t, os.WriteFile(filePath, []byte("another modification\n"), 0o644))

	// Create second stash
	cmd = exec.Command("git", "stash", "push", "-m", "Second stash")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// List stashes
	stashes, err := r.ListStashes(context.Background())
	require.NoError(t, err)
	require.Len(t, stashes, 2)

	// Stashes are returned newest first
	require.Equal(t, 0, stashes[0].Index)
	require.Equal(t, "stash@{0}", stashes[0].Name)
	require.Contains(t, stashes[0].Message, "Second stash")

	require.Equal(t, 1, stashes[1].Index)
	require.Equal(t, "stash@{1}", stashes[1].Name)
	require.Contains(t, stashes[1].Message, "First stash")
}

func TestListStashes_ParsesWIPMessage(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)

	// Create a modified file
	filePath := filepath.Join(r.path, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("original content\n"), 0o644))

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add test file")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Modify and stash without message (WIP stash)
	require.NoError(t, os.WriteFile(filePath, []byte("modified\n"), 0o644))

	cmd = exec.Command("git", "stash", "push")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	stashes, err := r.ListStashes(context.Background())
	require.NoError(t, err)
	require.Len(t, stashes, 1)

	// WIP stash should have "WIP on" in the message
	require.Contains(t, stashes[0].Message, "WIP on")
}

// --- Tests for stash write operations ---

// stashSetup creates a temp repo with a committed file and a dirty working tree,
// ready for stash operations.
func stashSetup(t *testing.T) (*Repository, string) {
	t.Helper()
	r := newTempRepo(t)

	filePath := filepath.Join(r.path, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("original content\n"), 0o644))

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add test file")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Modify to create dirty state
	require.NoError(t, os.WriteFile(filePath, []byte("modified content\n"), 0o644))

	return r, filePath
}

func TestStash_CreatesEntry(t *testing.T) {
	skipInShort(t)
	r, _ := stashSetup(t)
	ctx := context.Background()

	err := r.Stash(ctx, StashOpts{Message: "test stash"})
	require.NoError(t, err)

	stashes, err := r.ListStashes(ctx)
	require.NoError(t, err)
	require.Len(t, stashes, 1)
	require.Contains(t, stashes[0].Message, "test stash")
}

func TestStash_IncludeUntracked(t *testing.T) {
	skipInShort(t)
	r, _ := stashSetup(t)
	ctx := context.Background()

	// Create an untracked file
	untrackedPath := filepath.Join(r.path, "untracked.txt")
	require.NoError(t, os.WriteFile(untrackedPath, []byte("untracked\n"), 0o644))

	err := r.Stash(ctx, StashOpts{
		Message:          "with untracked",
		IncludeUntracked: true,
	})
	require.NoError(t, err)

	// Untracked file should be gone after stash
	_, err = os.Stat(untrackedPath)
	require.True(t, os.IsNotExist(err), "untracked file should be removed after stash -u")
}

func TestStashPop_AppliesAndRemoves(t *testing.T) {
	skipInShort(t)
	r, filePath := stashSetup(t)
	ctx := context.Background()

	err := r.Stash(ctx, StashOpts{Message: "pop test"})
	require.NoError(t, err)

	// File should be back to original
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, "original content\n", string(data))

	// Pop the stash
	err = r.StashPop(ctx, 0)
	require.NoError(t, err)

	// File should be modified again
	data, err = os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, "modified content\n", string(data))

	// Stash list should be empty
	stashes, err := r.ListStashes(ctx)
	require.NoError(t, err)
	require.Empty(t, stashes)
}

func TestStashApply_AppliesAndKeeps(t *testing.T) {
	skipInShort(t)
	r, filePath := stashSetup(t)
	ctx := context.Background()

	err := r.Stash(ctx, StashOpts{Message: "apply test"})
	require.NoError(t, err)

	// Apply the stash
	err = r.StashApply(ctx, 0)
	require.NoError(t, err)

	// File should be modified
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, "modified content\n", string(data))

	// Stash should still be in the list
	stashes, err := r.ListStashes(ctx)
	require.NoError(t, err)
	require.Len(t, stashes, 1)
}

func TestStashDrop_RemovesEntry(t *testing.T) {
	skipInShort(t)
	r, _ := stashSetup(t)
	ctx := context.Background()

	err := r.Stash(ctx, StashOpts{Message: "drop test"})
	require.NoError(t, err)

	stashes, err := r.ListStashes(ctx)
	require.NoError(t, err)
	require.Len(t, stashes, 1)

	err = r.StashDrop(ctx, 0)
	require.NoError(t, err)

	stashes, err = r.ListStashes(ctx)
	require.NoError(t, err)
	require.Empty(t, stashes)
}

func TestStashBranch_ChecksOutNewBranch(t *testing.T) {
	skipInShort(t)
	r, filePath := stashSetup(t)
	ctx := context.Background()

	err := r.Stash(ctx, StashOpts{Message: "branch test"})
	require.NoError(t, err)

	err = r.StashBranch(ctx, "stash-branch", 0)
	require.NoError(t, err)

	// Should be on the new branch
	out, err := r.runGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	require.NoError(t, err)
	require.Equal(t, "stash-branch", strings.TrimSpace(out))

	// File should have the stashed modification
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, "modified content\n", string(data))

	// Stash should be dropped
	stashes, err := r.ListStashes(ctx)
	require.NoError(t, err)
	require.Empty(t, stashes)
}
