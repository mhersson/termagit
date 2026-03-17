package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
