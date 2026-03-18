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

func TestAppendCommitArgs_Fixup(t *testing.T) {
	args := appendCommitArgs([]string{"commit"}, CommitOpts{Fixup: "abc1234"})
	assert.Contains(t, args, "--fixup=abc1234")
}

func TestAppendCommitArgs_FixupAmend(t *testing.T) {
	args := appendCommitArgs([]string{"commit"}, CommitOpts{Fixup: "amend:abc1234"})
	assert.Contains(t, args, "--fixup=amend:abc1234")
}

func TestAppendCommitArgs_FixupReword(t *testing.T) {
	args := appendCommitArgs([]string{"commit"}, CommitOpts{Fixup: "reword:abc1234"})
	assert.Contains(t, args, "--fixup=reword:abc1234")
}

func TestAppendCommitArgs_Squash(t *testing.T) {
	args := appendCommitArgs([]string{"commit"}, CommitOpts{Squash: "def5678"})
	assert.Contains(t, args, "--squash=def5678")
}

func TestAppendCommitArgs_NoEdit(t *testing.T) {
	args := appendCommitArgs([]string{"commit"}, CommitOpts{NoEdit: true})
	assert.Contains(t, args, "--no-edit")
}

func TestAppendCommitArgs_FixupWithNoEdit(t *testing.T) {
	args := appendCommitArgs([]string{"commit"}, CommitOpts{Fixup: "abc1234", NoEdit: true})
	assert.Contains(t, args, "--fixup=abc1234")
	assert.Contains(t, args, "--no-edit")
}

func TestCommit_FixupCreatesFixupCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit to fixup
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "base.txt"), []byte("base"), 0o644))
	require.NoError(t, r.StageFile(ctx, "base.txt"))
	_, err := r.Commit(ctx, CommitOpts{Message: "Base commit"})
	require.NoError(t, err)

	// Get the base commit hash
	baseHash, err := r.runGit(ctx, "rev-parse", "--short", "HEAD")
	require.NoError(t, err)
	baseHash = strings.TrimSpace(baseHash)

	// Create another file and stage it
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "fix.txt"), []byte("fix"), 0o644))
	require.NoError(t, r.StageFile(ctx, "fix.txt"))

	// Create fixup commit
	hash, err := r.Commit(ctx, CommitOpts{Fixup: baseHash, NoEdit: true})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the commit message starts with "fixup!"
	out, err := r.runGit(ctx, "log", "-1", "--format=%s")
	require.NoError(t, err)
	assert.Contains(t, out, "fixup!")
}

func TestCommit_SquashCreatesSquashCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit to squash into
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "base.txt"), []byte("base"), 0o644))
	require.NoError(t, r.StageFile(ctx, "base.txt"))
	_, err := r.Commit(ctx, CommitOpts{Message: "Base commit"})
	require.NoError(t, err)

	baseHash, err := r.runGit(ctx, "rev-parse", "--short", "HEAD")
	require.NoError(t, err)
	baseHash = strings.TrimSpace(baseHash)

	// Create another file and stage it
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "squash.txt"), []byte("squash"), 0o644))
	require.NoError(t, r.StageFile(ctx, "squash.txt"))

	// Create squash commit
	hash, err := r.Commit(ctx, CommitOpts{Squash: baseHash, NoEdit: true})
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify the commit message starts with "squash!"
	out, err := r.runGit(ctx, "log", "-1", "--format=%s")
	require.NoError(t, err)
	assert.Contains(t, out, "squash!")
}
