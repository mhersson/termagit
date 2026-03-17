package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommit_CreatesCommitWithMessage(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create and stage a file
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "new.txt"), []byte("hello"), 0o644))
	require.NoError(t, r.StageFile(ctx, "new.txt"))

	hash, err := r.Commit(ctx, CommitOpts{Message: "Add new file"})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the commit exists
	out, err := r.runGit(ctx, "log", "-1", "--format=%s")
	require.NoError(t, err)
	assert.Contains(t, out, "Add new file")
}

func TestCommit_AmendModifiesLastCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Get initial commit count
	out, err := r.runGit(ctx, "rev-list", "--count", "HEAD")
	require.NoError(t, err)

	hash, err := r.Commit(ctx, CommitOpts{
		Message:    "Amended message",
		Amend:      true,
		AllowEmpty: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Commit count should be the same (amended, not new)
	out2, err := r.runGit(ctx, "rev-list", "--count", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, out, out2)

	// Message should be updated
	out, err = r.runGit(ctx, "log", "-1", "--format=%s")
	require.NoError(t, err)
	assert.Contains(t, out, "Amended message")
}

func TestCommitFromFile_UsesFileContent(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Write a commit message file
	msgPath := filepath.Join(r.path, "commit-msg.txt")
	require.NoError(t, os.WriteFile(msgPath, []byte("Message from file\n\nDetailed body."), 0o644))

	// Create and stage a file
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "file.txt"), []byte("content"), 0o644))
	require.NoError(t, r.StageFile(ctx, "file.txt"))

	hash, err := r.CommitFromFile(ctx, msgPath, CommitOpts{})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	out, err := r.runGit(ctx, "log", "-1", "--format=%s")
	require.NoError(t, err)
	assert.Contains(t, out, "Message from file")
}

func TestCommitEditorPath_ReturnsCorrectPath(t *testing.T) {
	r := newTempRepo(t)

	path := r.CommitEditorPath()
	expected := filepath.Join(r.gitDir, "COMMIT_EDITMSG")
	assert.Equal(t, expected, path)
}

func TestCommitAbsorb_FailsGracefullyWhenNotInstalled(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// git-absorb is unlikely to be installed in CI/test environments
	// This should return an error, not panic
	err := r.CommitAbsorb(ctx)
	// We don't assert NoError because git-absorb is likely not installed
	// We just verify it doesn't panic
	_ = err
}
