package stashlist

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.RawTokens{})
}

func testStashes() []git.StashEntry {
	return []git.StashEntry{
		{Index: 0, Name: "stash@{0}", Message: "WIP on main: abc1234 work in progress", Branch: "main", Hash: "aaa111"},
		{Index: 1, Name: "stash@{1}", Message: "On feature: fix tests", Branch: "feature", Hash: "bbb222"},
		{Index: 2, Name: "stash@{2}", Message: "WIP on main: def5678 another save", Branch: "main", Hash: "ccc333"},
	}
}

func TestStashListModel_Init_ShowsStashes(t *testing.T) {
	m := New(testStashes(), nil, testTokens())

	assert.Len(t, m.stashes, 3)
	assert.Equal(t, 0, m.cursor)
}

func TestStashListModel_Navigation(t *testing.T) {
	m := New(testStashes(), nil, testTokens())

	// Move down
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.cursor)

	// Move up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor)

	// Can't go below 0
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor)

	// Go to bottom
	m.cursor = 2
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, m.cursor) // stays at bottom
}

func TestStashListModel_Close_SendsCloseMsg(t *testing.T) {
	m := New(testStashes(), nil, testTokens())

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseStashListMsg)
	assert.True(t, ok)
}

func TestStashListModel_Select_OpensCommitView(t *testing.T) {
	m := New(testStashes(), nil, testTokens())
	m.cursor = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	require.NotNil(t, cmd)
	msg := cmd()
	cvMsg, ok := msg.(OpenCommitViewMsg)
	assert.True(t, ok)
	assert.Equal(t, "stash@{0}", cvMsg.Hash)
}

func TestStashListModel_Discard_RequiresConfirm(t *testing.T) {
	m := New(testStashes(), nil, testTokens())
	m.cursor = 1

	// Press x
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, confirmDropStash, m.confirmMode)
	assert.Equal(t, 1, m.confirmIdx)

	confirmMsg := m.ConfirmMessage()
	assert.Contains(t, confirmMsg, "stash@{1}")
}

func TestStashListModel_Discard_CancelOnN(t *testing.T) {
	m := New(testStashes(), nil, testTokens())
	m.cursor = 1

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, confirmDropStash, m.confirmMode)

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, confirmNone, m.confirmMode)
}

func TestStashListModel_Yank_EmitsYankMsg(t *testing.T) {
	m := New(testStashes(), nil, testTokens())
	m.cursor = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	require.NotNil(t, cmd)
	msg := cmd()
	ym, ok := msg.(YankMsg)
	assert.True(t, ok)
	assert.Equal(t, "stash@{0}", ym.Text)
}

func TestStashListView_StashRow_RendersNameAndMessage(t *testing.T) {
	m := New(testStashes(), nil, testTokens())
	m.width = 80
	m.height = 24

	view := m.View()

	assert.Contains(t, view, "stash@{0}")
	assert.Contains(t, view, "WIP on main: abc1234 work in progress")
	assert.Contains(t, view, "stash@{1}")
}

func TestStashListView_HeaderShowsCount(t *testing.T) {
	m := New(testStashes(), nil, testTokens())
	m.width = 80
	m.height = 24

	view := m.View()

	assert.Contains(t, view, "Stashes (3)")
}

func TestStashListModel_EmptyList_NoErrors(t *testing.T) {
	m := New(nil, nil, testTokens())
	m.width = 80
	m.height = 24

	// Should not panic on empty list
	view := m.View()
	assert.Contains(t, view, "Stashes (0)")

	// Navigation should not crash
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
}
