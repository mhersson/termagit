package commitselect

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCommits() []git.LogEntry {
	return []git.LogEntry{
		{Hash: "abc1234567890abc1234567890abc1234567890ab", AbbreviatedHash: "abc1234", Subject: "Add feature X"},
		{Hash: "def5678901234def5678901234def5678901234de", AbbreviatedHash: "def5678", Subject: "Fix bug Y"},
		{Hash: "ghi9012345678ghi9012345678ghi9012345678gh", AbbreviatedHash: "ghi9012", Subject: "Refactor Z"},
	}
}

func TestNew_InitializesWithCursorAtZero(t *testing.T) {
	m := New(testCommits(), 80, 24)
	assert.Equal(t, 0, m.cursor)
	assert.False(t, m.done)
	assert.False(t, m.aborted)
}

func TestNew_EmptyCommits(t *testing.T) {
	m := New(nil, 80, 24)
	assert.Equal(t, 0, m.cursor)
	assert.Empty(t, m.commits)
}

func TestUpdate_MoveDown(t *testing.T) {
	m := New(testCommits(), 80, 24)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := updated.(Model)
	assert.Equal(t, 1, result.cursor)
}

func TestUpdate_MoveUp(t *testing.T) {
	m := New(testCommits(), 80, 24)
	m.cursor = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result := updated.(Model)
	assert.Equal(t, 1, result.cursor)
}

func TestUpdate_MoveDownWrapsAtBottom(t *testing.T) {
	m := New(testCommits(), 80, 24)
	m.cursor = 2 // last item

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	result := updated.(Model)
	assert.Equal(t, 2, result.cursor, "cursor should not go past last item")
}

func TestUpdate_MoveUpClampsAtTop(t *testing.T) {
	m := New(testCommits(), 80, 24)
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	result := updated.(Model)
	assert.Equal(t, 0, result.cursor, "cursor should not go below zero")
}

func TestUpdate_ArrowDownMovesCursor(t *testing.T) {
	m := New(testCommits(), 80, 24)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	result := updated.(Model)
	assert.Equal(t, 1, result.cursor)
}

func TestUpdate_ArrowUpMovesCursor(t *testing.T) {
	m := New(testCommits(), 80, 24)
	m.cursor = 1

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	result := updated.(Model)
	assert.Equal(t, 0, result.cursor)
}

func TestUpdate_EnterSelectsCommit(t *testing.T) {
	m := New(testCommits(), 80, 24)
	m.cursor = 1

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(Model)
	assert.True(t, result.done)
	assert.False(t, result.aborted)

	// The command should produce a SelectedMsg
	require.NotNil(t, cmd)
	msg := cmd()
	sel, ok := msg.(SelectedMsg)
	require.True(t, ok)
	assert.Equal(t, "def5678", sel.Hash)
	assert.Equal(t, "Fix bug Y", sel.Subject)
}

func TestUpdate_EscapeAborts(t *testing.T) {
	m := New(testCommits(), 80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	result := updated.(Model)
	assert.True(t, result.done)
	assert.True(t, result.aborted)

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(AbortedMsg)
	assert.True(t, ok)
}

func TestUpdate_QAborts(t *testing.T) {
	m := New(testCommits(), 80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	result := updated.(Model)
	assert.True(t, result.done)
	assert.True(t, result.aborted)

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(AbortedMsg)
	assert.True(t, ok)
}

func TestUpdate_EnterOnEmptyCommitsAborts(t *testing.T) {
	m := New(nil, 80, 24)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := updated.(Model)
	assert.True(t, result.done)
	assert.True(t, result.aborted)

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(AbortedMsg)
	assert.True(t, ok)
}

func TestView_ContainsHeader(t *testing.T) {
	m := New(testCommits(), 80, 24)
	view := m.View()
	assert.Contains(t, view, "Select a commit with <cr>, or <esc> to abort")
}

func TestView_ContainsCommitHashes(t *testing.T) {
	m := New(testCommits(), 80, 24)
	view := m.View()
	assert.Contains(t, view, "abc1234")
	assert.Contains(t, view, "def5678")
	assert.Contains(t, view, "ghi9012")
}

func TestView_ContainsCommitSubjects(t *testing.T) {
	m := New(testCommits(), 80, 24)
	view := m.View()
	assert.Contains(t, view, "Add feature X")
	assert.Contains(t, view, "Fix bug Y")
	assert.Contains(t, view, "Refactor Z")
}

func TestDone_ReturnsState(t *testing.T) {
	m := New(testCommits(), 80, 24)
	assert.False(t, m.Done())

	m.done = true
	assert.True(t, m.Done())
}

func TestAborted_ReturnsState(t *testing.T) {
	m := New(testCommits(), 80, 24)
	assert.False(t, m.Aborted())

	m.aborted = true
	assert.True(t, m.Aborted())
}

func TestSetSize_UpdatesDimensions(t *testing.T) {
	m := New(testCommits(), 80, 24)
	m.SetSize(120, 40)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestUpdate_CtrlDScrollsDown(t *testing.T) {
	// Create many commits to need scrolling
	commits := make([]git.LogEntry, 50)
	for i := range commits {
		commits[i] = git.LogEntry{AbbreviatedHash: "abc1234", Subject: "Commit"}
	}
	m := New(commits, 80, 10)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	result := updated.(Model)
	assert.Greater(t, result.cursor, 0, "ctrl-d should move cursor down")
}

func TestUpdate_CtrlUScrollsUp(t *testing.T) {
	commits := make([]git.LogEntry, 50)
	for i := range commits {
		commits[i] = git.LogEntry{AbbreviatedHash: "abc1234", Subject: "Commit"}
	}
	m := New(commits, 80, 10)
	m.cursor = 25

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	result := updated.(Model)
	assert.Less(t, result.cursor, 25, "ctrl-u should move cursor up")
}
