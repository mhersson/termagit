package git

import (
	"os"
	"path/filepath"
	"testing"

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
