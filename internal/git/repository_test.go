package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_FindsGitRepository(t *testing.T) {
	r := newTempRepo(t)

	// Open from the root should work
	repo, err := Open(r.path, nil)
	require.NoError(t, err)
	assert.Equal(t, r.path, repo.Path())
}

func TestOpen_FindsGitFromSubdirectory(t *testing.T) {
	r := newTempRepo(t)

	// Create a subdirectory
	subdir := filepath.Join(r.path, "subdir", "nested")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	// Open from subdirectory should find the repo
	repo, err := Open(subdir, nil)
	require.NoError(t, err)
	assert.Equal(t, r.path, repo.Path())
}

func TestOpen_ReturnsErrNotARepo_WhenNotInGitRepo(t *testing.T) {
	dir := t.TempDir() // Empty directory, not a git repo

	_, err := Open(dir, nil)
	assert.ErrorIs(t, err, ErrNotARepo)
}

func TestWrap_CreatesRepositoryFromRaw(t *testing.T) {
	r := newMemRepo(t)

	wrapped := Wrap(r.raw, "/test/path", nil)
	assert.Equal(t, "/test/path", wrapped.Path())
	assert.Equal(t, "/test/path/.git", wrapped.GitDir())
}

func TestPath_ReturnsWorkingTreeRoot(t *testing.T) {
	r := newTempRepo(t)
	assert.Equal(t, r.path, r.Path())
}

func TestGitDir_ReturnsGitDirectory(t *testing.T) {
	r := newTempRepo(t)
	expected := filepath.Join(r.path, ".git")
	assert.Equal(t, expected, r.GitDir())
}

func TestHeadInfo_ReturnsBranchAndSubject(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	branch, subject, err := r.HeadInfo(ctx)
	require.NoError(t, err)

	// In-memory repo starts on master (go-git default)
	assert.Contains(t, []string{"master", "main"}, branch)
	assert.Equal(t, "Initial commit", subject)
}

func TestHeadInfo_ReturnsHEAD_WhenDetached(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Get current HEAD and checkout directly
	head, err := r.raw.Head()
	require.NoError(t, err)

	wt, err := r.raw.Worktree()
	require.NoError(t, err)

	// Checkout the commit directly to create detached HEAD
	err = wt.Checkout(&git.CheckoutOptions{
		Hash: head.Hash(),
	})
	require.NoError(t, err)

	branch, _, err := r.HeadInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "HEAD", branch)
}

func TestHeadOID_ReturnsFullHash(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	oid, err := r.HeadOID(ctx)
	require.NoError(t, err)

	// Should be 40 hex characters
	assert.Len(t, oid, 40)
	assert.Regexp(t, "^[0-9a-f]{40}$", oid)
}

func TestAheadBehind_ReturnsZero_WhenNoUpstream(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	ahead, behind, err := r.AheadBehind(ctx)
	require.NoError(t, err)

	assert.Equal(t, 0, ahead)
	assert.Equal(t, 0, behind)
}

func TestRebaseInProgress_ReturnsFalse_WhenNoRebase(t *testing.T) {
	r := newTempRepo(t)

	assert.False(t, r.RebaseInProgress())
}

func TestRebaseInProgress_ReturnsTrue_WhenRebaseInteractiveExists(t *testing.T) {
	r := newTempRepo(t)

	// Create .git/rebase-merge directory to simulate rebase
	rebaseDir := filepath.Join(r.gitDir, "rebase-merge")
	require.NoError(t, os.MkdirAll(rebaseDir, 0o755))

	assert.True(t, r.RebaseInProgress())
}

func TestMergeInProgress_ReturnsFalse_WhenNoMerge(t *testing.T) {
	r := newTempRepo(t)

	assert.False(t, r.MergeInProgress())
}

func TestMergeInProgress_ReturnsTrue_WhenMergeHeadExists(t *testing.T) {
	r := newTempRepo(t)

	// Create MERGE_HEAD file to simulate merge
	mergeHead := filepath.Join(r.gitDir, "MERGE_HEAD")
	require.NoError(t, os.WriteFile(mergeHead, []byte("abc123\n"), 0o644))

	assert.True(t, r.MergeInProgress())
}

func TestCherryPickInProgress_ReturnsFalse_WhenNoCherryPick(t *testing.T) {
	r := newTempRepo(t)

	assert.False(t, r.CherryPickInProgress())
}

func TestCherryPickInProgress_ReturnsTrue_WhenCherryPickHeadExists(t *testing.T) {
	r := newTempRepo(t)

	// Create CHERRY_PICK_HEAD file
	cpHead := filepath.Join(r.gitDir, "CHERRY_PICK_HEAD")
	require.NoError(t, os.WriteFile(cpHead, []byte("abc123\n"), 0o644))

	assert.True(t, r.CherryPickInProgress())
}

func TestRevertInProgress_ReturnsFalse_WhenNoRevert(t *testing.T) {
	r := newTempRepo(t)

	assert.False(t, r.RevertInProgress())
}

func TestRevertInProgress_ReturnsTrue_WhenRevertHeadExists(t *testing.T) {
	r := newTempRepo(t)

	// Create REVERT_HEAD file
	revertHead := filepath.Join(r.gitDir, "REVERT_HEAD")
	require.NoError(t, os.WriteFile(revertHead, []byte("abc123\n"), 0o644))

	assert.True(t, r.RevertInProgress())
}

func TestBisectInProgress_ReturnsFalse_WhenNoBisect(t *testing.T) {
	r := newTempRepo(t)

	assert.False(t, r.BisectInProgress())
}

func TestBisectInProgress_ReturnsTrue_WhenBisectLogExists(t *testing.T) {
	r := newTempRepo(t)

	// Create BISECT_LOG file
	bisectLog := filepath.Join(r.gitDir, "BISECT_LOG")
	require.NoError(t, os.WriteFile(bisectLog, []byte("# bisect log\n"), 0o644))

	assert.True(t, r.BisectInProgress())
}

func TestSequencerOperation_ReturnsEmpty_WhenNoSequencer(t *testing.T) {
	r := newTempRepo(t)

	assert.Equal(t, "", r.SequencerOperation())
}

func TestSequencerOperation_ReturnsOperation_WhenTodoExists(t *testing.T) {
	r := newTempRepo(t)

	// Create sequencer/todo file
	seqDir := filepath.Join(r.gitDir, "sequencer")
	require.NoError(t, os.MkdirAll(seqDir, 0o755))

	todoPath := filepath.Join(seqDir, "todo")
	require.NoError(t, os.WriteFile(todoPath, []byte("pick abc123 Some commit\npick def456 Another commit\n"), 0o644))

	assert.Equal(t, "pick", r.SequencerOperation())
}

func TestRunGit_ExecutesCommand(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	out, err := r.runGit(ctx, "status", "--porcelain")
	require.NoError(t, err)
	assert.NotNil(t, out) // May be empty but not nil
}

func TestRunGit_ReturnsError_OnFailure(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	_, err := r.runGit(ctx, "invalid-command")
	assert.Error(t, err)
}
