package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/mhersson/conjit/internal/cmdlog"
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

func TestReadMergeState_NoMerge_ReturnsEmpty(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)

	head, subject, branch, err := r.ReadMergeState()
	require.NoError(t, err)
	require.Empty(t, head)
	require.Empty(t, subject)
	require.Empty(t, branch)
}

func TestReadMergeState_ActiveMerge_ReturnsState(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)

	// Create a branch with a commit
	cmd := exec.Command("git", "checkout", "-b", "feature")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	filePath := filepath.Join(r.path, "feature.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("feature content\n"), 0o644))

	cmd = exec.Command("git", "add", "feature.txt")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add feature")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Go back to master and make a conflicting change
	cmd = exec.Command("git", "checkout", "master")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	require.NoError(t, os.WriteFile(filePath, []byte("master content\n"), 0o644))

	cmd = exec.Command("git", "add", "feature.txt")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Add master version")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Start a merge that will conflict
	cmd = exec.Command("git", "merge", "feature", "--no-commit")
	cmd.Dir = r.path
	_ = cmd.Run() // This will return an error due to conflict, that's expected

	// Now check merge state
	head, subject, branch, err := r.ReadMergeState()
	require.NoError(t, err)
	require.NotEmpty(t, head)          // MERGE_HEAD should exist
	require.Contains(t, subject, "feature") // Subject from MERGE_MSG
	require.Equal(t, "feature", branch) // Branch being merged
}

func TestBisectState_NoBisect_ReturnsEmpty(t *testing.T) {
	r := newTempRepo(t)

	state, err := r.BisectState(context.Background())
	require.NoError(t, err)
	require.Empty(t, state.Items)
}

func TestBisectState_ActiveBisect_ReturnsItems(t *testing.T) {
	r := newTempRepo(t)

	// Create BISECT_LOG to simulate active bisect
	bisectLog := `# first bad commit: abc1234567890123456789012345678901234567890
git bisect start
# good: def1234567890123456789012345678901234567890 Initial commit
git bisect good def1234567890123456789012345678901234567890
# bad: abc1234567890123456789012345678901234567890 Bug introduced
git bisect bad abc1234567890123456789012345678901234567890
# skip: 111223344556677889900aabbccddeeff00112233 Unrelated change
git bisect skip 111223344556677889900aabbccddeeff00112233
`
	bisectLogPath := filepath.Join(r.gitDir, "BISECT_LOG")
	require.NoError(t, os.WriteFile(bisectLogPath, []byte(bisectLog), 0o644))

	state, err := r.BisectState(context.Background())
	require.NoError(t, err)

	// Should have 3 items: good, bad, skip
	require.Len(t, state.Items, 3)

	// First item should be good
	require.Equal(t, "good", state.Items[0].Action)
	require.Equal(t, "def1234567890123456789012345678901234567890", state.Items[0].Hash)
	require.Equal(t, "def1234", state.Items[0].AbbrevHash)
	require.Equal(t, "Initial commit", state.Items[0].Subject)

	// Second item should be bad
	require.Equal(t, "bad", state.Items[1].Action)
	require.Equal(t, "abc1234567890123456789012345678901234567890", state.Items[1].Hash)

	// Third item should be skip
	require.Equal(t, "skip", state.Items[2].Action)
}

func TestSequencerState_NoSequencer_ReturnsEmpty(t *testing.T) {
	r := newTempRepo(t)

	state, err := r.SequencerState(context.Background())
	require.NoError(t, err)
	require.Empty(t, state.Operation)
	require.Empty(t, state.Items)
}

func TestSequencerState_CherryPick_ReturnsState(t *testing.T) {
	r := newTempRepo(t)

	// Create CHERRY_PICK_HEAD to simulate cherry-pick in progress
	cpHead := filepath.Join(r.gitDir, "CHERRY_PICK_HEAD")
	require.NoError(t, os.WriteFile(cpHead, []byte("abc1234567890123456789012345678901234567890\n"), 0o644))

	state, err := r.SequencerState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "cherry-pick", state.Operation)
	require.Len(t, state.Items, 1)
	require.Equal(t, "abc1234", state.Items[0].AbbrevHash)
}

func TestSequencerState_Revert_ReturnsState(t *testing.T) {
	r := newTempRepo(t)

	// Create REVERT_HEAD to simulate revert in progress
	revertHead := filepath.Join(r.gitDir, "REVERT_HEAD")
	require.NoError(t, os.WriteFile(revertHead, []byte("def1234567890123456789012345678901234567890\n"), 0o644))

	state, err := r.SequencerState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "revert", state.Operation)
	require.Len(t, state.Items, 1)
	require.Equal(t, "def1234", state.Items[0].AbbrevHash)
}

func TestSequencerState_MultiCommitCherryPick_ReturnsItems(t *testing.T) {
	r := newTempRepo(t)

	// Create CHERRY_PICK_HEAD
	cpHead := filepath.Join(r.gitDir, "CHERRY_PICK_HEAD")
	require.NoError(t, os.WriteFile(cpHead, []byte("abc1234567890123456789012345678901234567890\n"), 0o644))

	// Create sequencer/todo for multi-commit cherry-pick
	seqDir := filepath.Join(r.gitDir, "sequencer")
	require.NoError(t, os.MkdirAll(seqDir, 0o755))

	todoContent := `pick abc1234567890123456789012345678901234567890 First commit
pick def1234567890123456789012345678901234567890 Second commit
pick ghi1234567890123456789012345678901234567890 Third commit
`
	require.NoError(t, os.WriteFile(filepath.Join(seqDir, "todo"), []byte(todoContent), 0o644))

	state, err := r.SequencerState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "cherry-pick", state.Operation)
	require.Len(t, state.Items, 3)
	require.Equal(t, "First commit", state.Items[0].Subject)
	require.Equal(t, "Second commit", state.Items[1].Subject)
	require.NotNil(t, state.Current)
}

func TestGetConfigValue_ReturnsValue_WhenConfigExists(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Set a config value
	cmd := exec.Command("git", "config", "core.commentChar", ";")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	// Read it back
	val, err := r.GetConfigValue(ctx, "core.commentChar")
	require.NoError(t, err)
	assert.Equal(t, ";", val)
}

func TestGetConfigValue_ReturnsEmpty_WhenConfigMissing(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Read a non-existent config key
	val, err := r.GetConfigValue(ctx, "nonexistent.key")
	require.NoError(t, err) // Should NOT return error for missing config
	assert.Empty(t, val)
}

func TestGetConfigValue_ReturnsDefault_ForUserEmail(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// user.email is typically set globally, but test the mechanism
	val, err := r.GetConfigValue(ctx, "user.email")
	require.NoError(t, err)
	// Value may or may not be set, but no error
	_ = val
}

func TestRunGit_CapturesStdout(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	out, err := r.runGit(ctx, "rev-parse", "--git-dir")
	require.NoError(t, err)
	assert.Contains(t, out, ".git")
}

func TestRunGit_WrapsStderrOnFailure(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	_, err := r.runGit(ctx, "log", "--invalid-option-xyz")
	require.Error(t, err)
	// Error message should contain stderr from git
	assert.Contains(t, err.Error(), "git log")
}

func TestRunGit_RespectsContextCancellation(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := r.runGit(ctx, "status")
	require.Error(t, err)
}

func TestRunGit_LogsEntry(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a logger
	logPath := filepath.Join(t.TempDir(), "test.log")
	logger, err := cmdlog.New(logPath, 1<<20, 2)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	r.logger = logger

	_, err = r.runGit(ctx, "status", "--porcelain")
	require.NoError(t, err)

	// Flush and read back
	require.NoError(t, logger.Close())

	entries, err := cmdlog.ReadRecent(logPath, 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Contains(t, entries[0].Command, "git status")
}

func TestRunGit_NilLogger_DoesNotPanic(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// logger is already nil in newTempRepo
	assert.Nil(t, r.logger)

	// Should not panic
	_, err := r.runGit(ctx, "status", "--porcelain")
	require.NoError(t, err)
}

func TestLogOp_LogsSuccess(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	logPath := filepath.Join(t.TempDir(), "test.log")
	logger, err := cmdlog.New(logPath, 1<<20, 2)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	r.logger = logger

	result1, result2, err := r.logOp(ctx, "test-op", func() (string, string, error) {
		return "out1", "out2", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "out1", result1)
	assert.Equal(t, "out2", result2)

	// Flush and verify log entry
	require.NoError(t, logger.Close())

	entries, err := cmdlog.ReadRecent(logPath, 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, "test-op", entries[0].Command)
	assert.Equal(t, 0, entries[0].ExitCode)
}

func TestLogOp_LogsFailure(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	logPath := filepath.Join(t.TempDir(), "test.log")
	logger, err := cmdlog.New(logPath, 1<<20, 2)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	r.logger = logger

	testErr := fmt.Errorf("something went wrong")
	_, _, err = r.logOp(ctx, "failing-op", func() (string, string, error) {
		return "", "", testErr
	})
	require.Error(t, err)
	assert.Equal(t, testErr, err)

	// Flush and verify log entry records the failure
	require.NoError(t, logger.Close())

	entries, err := cmdlog.ReadRecent(logPath, 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	assert.Equal(t, "failing-op", entries[0].Command)
	assert.Equal(t, 1, entries[0].ExitCode)
	assert.Contains(t, entries[0].Stderr, "something went wrong")
}

func TestLogOp_NilLogger_DoesNotPanic(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// logger is nil
	assert.Nil(t, r.logger)

	result1, result2, err := r.logOp(ctx, "test-op", func() (string, string, error) {
		return "a", "b", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "a", result1)
	assert.Equal(t, "b", result2)
}

func TestSequencerOperation_ReturnsEmpty_WhenNone(t *testing.T) {
	r := newTempRepo(t)

	// Clean repo with no sequencer directory at all
	assert.Equal(t, "", r.SequencerOperation())
}
