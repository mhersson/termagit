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

func TestReadRebaseTodo_NoRebase_ReturnsError(t *testing.T) {
	r := newTempRepo(t)

	_, err := r.ReadRebaseTodo()
	require.ErrorIs(t, err, ErrNoRebaseInProgress)
}

func TestReadRebaseTodo_ActiveRebase_ReturnsState(t *testing.T) {
	r := newTempRepo(t)

	// Simulate an interactive rebase by creating the rebase-merge directory
	rebaseDir := filepath.Join(r.gitDir, "rebase-merge")
	require.NoError(t, os.MkdirAll(rebaseDir, 0o755))

	// Create git-rebase-todo file (the remaining todo items)
	todoContent := `pick abc1234 First commit message
squash def5678 Second commit message
fixup ghi9012 Third commit message
`
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "git-rebase-todo"), []byte(todoContent), 0o644))

	// Create done file (already completed items)
	doneContent := `pick 111aaaa Initial setup
pick 222bbbb Add feature
`
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "done"), []byte(doneContent), 0o644))

	// Create stopped-sha file (current position)
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "stopped-sha"), []byte("abc1234\n"), 0o644))

	// Create onto file
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "onto"), []byte("main1234567890123456789012345678901234567890\n"), 0o644))

	// Create head-name file
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "head-name"), []byte("refs/heads/feature\n"), 0o644))

	state, err := r.ReadRebaseTodo()
	require.NoError(t, err)

	// Check state
	require.Equal(t, "feature", state.Branch)
	require.NotEmpty(t, state.Onto)

	// Should have 2 done items + 3 todo items = 5 total
	require.Len(t, state.Entries, 5)

	// First two should be done
	require.True(t, state.Entries[0].Done)
	require.True(t, state.Entries[1].Done)

	// Third should be stopped (current)
	require.True(t, state.Entries[2].Stopped)
	require.Equal(t, TodoPick, state.Entries[2].Action)
	require.Equal(t, "abc1234", state.Entries[2].AbbrevHash)
	require.Equal(t, "First commit message", state.Entries[2].Subject)

	// Fourth and fifth should be pending
	require.False(t, state.Entries[3].Done)
	require.False(t, state.Entries[3].Stopped)
	require.Equal(t, TodoSquash, state.Entries[3].Action)

	require.Equal(t, TodoFixup, state.Entries[4].Action)
}

func TestReadRebaseTodo_MarksCurrentStopped(t *testing.T) {
	r := newTempRepo(t)

	rebaseDir := filepath.Join(r.gitDir, "rebase-merge")
	require.NoError(t, os.MkdirAll(rebaseDir, 0o755))

	todoContent := `pick abc1234 First commit
pick def5678 Second commit
`
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "git-rebase-todo"), []byte(todoContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "stopped-sha"), []byte("abc1234\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "head-name"), []byte("refs/heads/test\n"), 0o644))

	state, err := r.ReadRebaseTodo()
	require.NoError(t, err)

	// Find the stopped entry
	var foundStopped bool
	for _, e := range state.Entries {
		if e.Stopped {
			foundStopped = true
			require.Equal(t, "abc1234", e.AbbrevHash)
		}
	}
	require.True(t, foundStopped, "should find stopped entry")
}

func TestReadRebaseTodo_HandlesMergeTodo(t *testing.T) {
	r := newTempRepo(t)

	rebaseDir := filepath.Join(r.gitDir, "rebase-merge")
	require.NoError(t, os.MkdirAll(rebaseDir, 0o755))

	// Test various rebase commands
	todoContent := `label onto
reset onto
pick abc1234 Some commit
exec npm test
break
label feature
merge -C def5678 feature
`
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "git-rebase-todo"), []byte(todoContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "head-name"), []byte("refs/heads/main\n"), 0o644))

	state, err := r.ReadRebaseTodo()
	require.NoError(t, err)

	// Should parse various action types
	require.Greater(t, len(state.Entries), 0)
}

func TestFormatTodoEntries_PickLine(t *testing.T) {
	entries := []TodoEntry{
		{Action: TodoPick, AbbrevHash: "abc1234", Subject: "add feature X"},
	}
	got := FormatTodoEntries(entries)
	assert.Equal(t, "pick abc1234 add feature X\n", got)
}

func TestFormatTodoEntries_MultipleActions(t *testing.T) {
	entries := []TodoEntry{
		{Action: TodoPick, AbbrevHash: "abc1234", Subject: "first commit"},
		{Action: TodoSquash, AbbrevHash: "def5678", Subject: "fix typo"},
		{Action: TodoReword, AbbrevHash: "ghi9012", Subject: "update README"},
	}
	got := FormatTodoEntries(entries)
	expected := "pick abc1234 first commit\nsquash def5678 fix typo\nreword ghi9012 update README\n"
	assert.Equal(t, expected, got)
}

func TestFormatTodoEntries_DropLine_CommentPrefix(t *testing.T) {
	entries := []TodoEntry{
		{Action: TodoDrop, AbbrevHash: "jkl3456", Subject: "wip: remove me"},
	}
	got := FormatTodoEntries(entries)
	assert.Equal(t, "# drop jkl3456 wip: remove me\n", got)
}

func TestFormatTodoEntries_ExecLine_NoHash(t *testing.T) {
	entries := []TodoEntry{
		{Action: TodoExec, Subject: "make test"},
	}
	got := FormatTodoEntries(entries)
	assert.Equal(t, "exec make test\n", got)
}

func TestFormatTodoEntries_BreakLine(t *testing.T) {
	entries := []TodoEntry{
		{Action: TodoBreak},
	}
	got := FormatTodoEntries(entries)
	assert.Equal(t, "break\n", got)
}

func TestFormatTodoEntries_MixedEntries(t *testing.T) {
	entries := []TodoEntry{
		{Action: TodoPick, AbbrevHash: "abc1234", Subject: "add feature X"},
		{Action: TodoExec, Subject: "make test"},
		{Action: TodoBreak},
		{Action: TodoDrop, AbbrevHash: "jkl3456", Subject: "wip: remove me"},
		{Action: TodoFixup, AbbrevHash: "mno7890", Subject: "fix whitespace"},
	}
	got := FormatTodoEntries(entries)
	expected := strings.Join([]string{
		"pick abc1234 add feature X",
		"exec make test",
		"break",
		"# drop jkl3456 wip: remove me",
		"fixup mno7890 fix whitespace",
		"",
	}, "\n")
	assert.Equal(t, expected, got)
}

func TestWriteRebaseTodo_WritesFile(t *testing.T) {
	r := newTempRepo(t)

	// Create rebase-merge directory (simulate in-progress rebase)
	rebaseDir := filepath.Join(r.gitDir, "rebase-merge")
	require.NoError(t, os.MkdirAll(rebaseDir, 0o755))

	// Write an initial todo file
	require.NoError(t, os.WriteFile(filepath.Join(rebaseDir, "git-rebase-todo"), []byte(""), 0o644))

	entries := []TodoEntry{
		{Action: TodoPick, AbbrevHash: "abc1234", Subject: "first commit"},
		{Action: TodoSquash, AbbrevHash: "def5678", Subject: "second commit"},
	}

	err := r.WriteRebaseTodo(entries)
	require.NoError(t, err)

	// Verify file contents
	data, err := os.ReadFile(filepath.Join(rebaseDir, "git-rebase-todo"))
	require.NoError(t, err)
	expected := "pick abc1234 first commit\nsquash def5678 second commit\n"
	assert.Equal(t, expected, string(data))
}

func TestWriteRebaseTodo_NoRebase_ReturnsError(t *testing.T) {
	r := newTempRepo(t)

	entries := []TodoEntry{
		{Action: TodoPick, AbbrevHash: "abc1234", Subject: "first commit"},
	}

	err := r.WriteRebaseTodo(entries)
	require.Error(t, err)
}

func TestRebaseAutosquash_SquashesFixupIntoTarget(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create base commit
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "base.txt"), []byte("base"), 0o644))
	require.NoError(t, r.StageFile(ctx, "base.txt"))
	_, err := r.Commit(ctx, CommitOpts{Message: "Base commit"})
	require.NoError(t, err)

	// Get the base commit hash (full)
	baseHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	baseHash = strings.TrimSpace(baseHash)

	// Create a normal commit after base
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "normal.txt"), []byte("normal"), 0o644))
	require.NoError(t, r.StageFile(ctx, "normal.txt"))
	_, err = r.Commit(ctx, CommitOpts{Message: "Normal commit"})
	require.NoError(t, err)

	// Create fixup commit targeting base
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "fix.txt"), []byte("fix"), 0o644))
	require.NoError(t, r.StageFile(ctx, "fix.txt"))
	_, err = r.Commit(ctx, CommitOpts{Fixup: baseHash, NoEdit: true})
	require.NoError(t, err)

	// Before autosquash: 4 commits (Initial + Base + Normal + fixup!)
	countBefore, err := r.runGit(ctx, "rev-list", "--count", "HEAD")
	require.NoError(t, err)
	require.Equal(t, "4", strings.TrimSpace(countBefore))

	// Run autosquash targeting the base commit's parent
	err = r.RebaseAutosquash(ctx, baseHash)
	require.NoError(t, err)

	// After autosquash: 3 commits (Initial + Base(with fix) + Normal)
	countAfter, err := r.runGit(ctx, "rev-list", "--count", "HEAD")
	require.NoError(t, err)
	require.Equal(t, "3", strings.TrimSpace(countAfter))

	// Verify the fixup commit message is gone (base commit absorbed it)
	logOut, err := r.runGit(ctx, "log", "--oneline")
	require.NoError(t, err)
	assert.NotContains(t, logOut, "fixup!")
}

func TestRebaseAutosquash_WithUnstagedChanges_Succeeds(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create base commit
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "base.txt"), []byte("base"), 0o644))
	require.NoError(t, r.StageFile(ctx, "base.txt"))
	_, err := r.Commit(ctx, CommitOpts{Message: "Base commit"})
	require.NoError(t, err)

	baseHash, err := r.runGit(ctx, "rev-parse", "HEAD")
	require.NoError(t, err)
	baseHash = strings.TrimSpace(baseHash)

	// Create a normal commit after base
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "normal.txt"), []byte("normal"), 0o644))
	require.NoError(t, r.StageFile(ctx, "normal.txt"))
	_, err = r.Commit(ctx, CommitOpts{Message: "Normal commit"})
	require.NoError(t, err)

	// Create fixup commit targeting base
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "fix.txt"), []byte("fix"), 0o644))
	require.NoError(t, r.StageFile(ctx, "fix.txt"))
	_, err = r.Commit(ctx, CommitOpts{Fixup: baseHash, NoEdit: true})
	require.NoError(t, err)

	// Introduce an unstaged change (dirty worktree)
	require.NoError(t, os.WriteFile(filepath.Join(r.path, "base.txt"), []byte("dirty"), 0o644))

	// Autosquash must succeed despite dirty worktree (--autostash)
	err = r.RebaseAutosquash(ctx, baseHash)
	require.NoError(t, err, "RebaseAutosquash should succeed with unstaged changes")

	// Fixup was squashed: 3 commits instead of 4
	countAfter, err := r.runGit(ctx, "rev-list", "--count", "HEAD")
	require.NoError(t, err)
	require.Equal(t, "3", strings.TrimSpace(countAfter))

	// Unstaged change must have been restored
	content, err := os.ReadFile(filepath.Join(r.path, "base.txt"))
	require.NoError(t, err)
	assert.Equal(t, "dirty", string(content), "unstaged change should survive autosquash")
}
