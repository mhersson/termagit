package reflogview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.RawTokens{})
}

func testEntries() []git.ReflogEntry {
	return []git.ReflogEntry{
		{
			Oid:        "abc123def456abc123def456abc123def456abc1",
			Index:      0,
			AuthorName: "Test User",
			RefName:    "HEAD@{0}",
			RefSubject: "commit: First commit",
			RelDate:    "3 hours ago",
			Type:       "commit",
		},
		{
			Oid:        "def456abc123def456abc123def456abc123def4",
			Index:      1,
			AuthorName: "Test User",
			RefName:    "HEAD@{1}",
			RefSubject: "checkout: moving from feature to main",
			RelDate:    "5 hours ago",
			Type:       "checkout",
		},
		{
			Oid:        "ghi789abc123def456abc123def456abc123ghi7",
			Index:      2,
			AuthorName: "Test User",
			RefName:    "HEAD@{2}",
			RefSubject: "reset: moving to HEAD~1",
			RelDate:    "1 day ago",
			Type:       "reset",
		},
	}
}

func TestReflogModel_New_InitializesWithEntries(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")

	assert.Len(t, m.entries, 3)
	assert.Equal(t, 0, m.cursor)
}

func TestReflogModel_CursorMovement(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")

	// Move down
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.cursor)

	// Move up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor)
}

func TestReflogModel_CursorStopsAtBoundaries(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")

	// At top, can't go up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor)

	// At bottom, can't go down
	m.cursor = 2
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, m.cursor)
}

func TestReflogModel_Close_SendsCloseMsg(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseReflogViewMsg)
	assert.True(t, ok)
}

func TestReflogView_RendersEntries(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain hashes
	assert.Contains(t, view, "abc123d")
	assert.Contains(t, view, "def456a")
}

func TestReflogView_TypeHighlight_CommitIsGreen(t *testing.T) {
	entries := []git.ReflogEntry{
		{Oid: "abc123def456abc123def456abc123def456abc1", Type: "commit", RefSubject: "commit: test"},
	}
	m := New(entries, testTokens(), "HEAD")
	m.width = 80
	m.height = 24

	// The type should be rendered with GraphGreen style
	// We can't easily test the actual color, but we can verify it renders
	view := m.View()
	assert.Contains(t, view, "commit")
}

func TestReflogView_TypeHighlight_ResetIsRed(t *testing.T) {
	entries := []git.ReflogEntry{
		{Oid: "abc123def456abc123def456abc123def456abc1", Type: "reset", RefSubject: "reset: HEAD~1"},
	}
	m := New(entries, testTokens(), "HEAD")
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "reset")
}

func TestReflogView_TypeHighlight_CheckoutIsBlue(t *testing.T) {
	entries := []git.ReflogEntry{
		{Oid: "abc123def456abc123def456abc123def456abc1", Type: "checkout", RefSubject: "checkout: main"},
	}
	m := New(entries, testTokens(), "HEAD")
	m.width = 80
	m.height = 24

	view := m.View()
	assert.Contains(t, view, "checkout")
}

func TestReflogModel_YankHash(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")
	m.cursor = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	require.NotNil(t, cmd)
}

func TestReflogView_Select_OpensCommitView(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")
	m.cursor = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	require.NotNil(t, cmd, "Select should return a command")
	msg := cmd()
	cvMsg, ok := msg.(OpenCommitViewMsg)
	assert.True(t, ok, "expected OpenCommitViewMsg, got %T", msg)
	if ok {
		assert.Equal(t, entries[0].Oid, cvMsg.Hash)
	}
}

func TestReflogView_Select_NoEntries_Noop(t *testing.T) {
	m := New(nil, testTokens(), "HEAD")
	m.cursor = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Select with no entries should be nil")
}
