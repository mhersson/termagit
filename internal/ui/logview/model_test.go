package logview

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

func testCommits() []git.LogEntry {
	return []git.LogEntry{
		{
			Hash:            "abc123def456abc123def456abc123def456abc1",
			AbbreviatedHash: "abc123d",
			Subject:         "First commit",
			AuthorName:      "Test User",
			AuthorEmail:     "test@example.com",
			AuthorDate:      "2024-01-15T10:30:00Z",
			Refs: []git.Ref{
				{Name: "HEAD", Kind: git.RefKindHead},
				{Name: "main", Kind: git.RefKindLocal},
			},
		},
		{
			Hash:            "def456abc123def456abc123def456abc123def4",
			AbbreviatedHash: "def456a",
			Subject:         "Second commit",
			AuthorName:      "Test User",
		},
		{
			Hash:            "ghi789abc123def456abc123def456abc123ghi7",
			AbbreviatedHash: "ghi789a",
			Subject:         "Third commit",
			AuthorName:      "Test User",
		},
	}
}

func TestLogModel_New_InitializesWithCommits(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")

	assert.Len(t, m.commits, 3)
	assert.Equal(t, 0, m.cursor)
}

func TestLogModel_CursorDown_MovesToNextCommit(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")

	// Press j to move down
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	assert.Equal(t, 1, m.cursor)
}

func TestLogModel_CursorUp_MovesToPreviousCommit(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor = 2

	// Press k to move up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	assert.Equal(t, 1, m.cursor)
}

func TestLogModel_CursorDown_StopsAtBottom(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor = 2 // last commit

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	assert.Equal(t, 2, m.cursor) // stays at last
}

func TestLogModel_CursorUp_StopsAtTop(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor = 0

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	assert.Equal(t, 0, m.cursor) // stays at top
}

func TestLogModel_Close_SendsCloseMsg(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseLogViewMsg)
	assert.True(t, ok)
}

func TestLogModel_Escape_SendsCloseMsg(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEscape})

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseLogViewMsg)
	assert.True(t, ok)
}

func TestLogModel_LoadMore_AppendsCommits(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, true, "main")
	m.hasMore = true

	// Press + to load more
	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})

	require.NotNil(t, cmd)
}

func TestLogModel_Filter_MatchesSubject(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.width = 80
	m.height = 24

	// Activate filter
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	assert.True(t, m.filterActive)

	// Type filter text
	m.filterInput.SetValue("First")
	m.applyFilter()

	// Should have filtered results
	assert.Len(t, m.filtered, 1)
	assert.Equal(t, 0, m.filtered[0]) // Index of "First commit"
}

func TestLogModel_Filter_MatchesHash(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.filterInput.SetValue("def456")
	m.applyFilter()

	assert.Len(t, m.filtered, 1)
	assert.Equal(t, 1, m.filtered[0])
}

func TestLogModel_Filter_MatchesAuthor(t *testing.T) {
	commits := []git.LogEntry{
		{AbbreviatedHash: "abc123d", Subject: "Commit A", AuthorName: "Alice"},
		{AbbreviatedHash: "def456a", Subject: "Commit B", AuthorName: "Bob"},
	}
	m := New(commits, nil, testTokens(), nil, false, "main")

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.filterInput.SetValue("Alice")
	m.applyFilter()

	assert.Len(t, m.filtered, 1)
	assert.Equal(t, 0, m.filtered[0])
}

func TestLogView_CommitRow_RendersHash(t *testing.T) {
	commits := []git.LogEntry{
		{AbbreviatedHash: "abc123d", Subject: "Test commit"},
	}
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.width = 80
	m.height = 24

	view := m.View()

	assert.Contains(t, view, "abc123d")
}

func TestLogView_CommitRow_RendersRefDecorations(t *testing.T) {
	commits := []git.LogEntry{
		{
			AbbreviatedHash: "abc123d",
			Subject:         "Test commit",
			Refs: []git.Ref{
				{Name: "main", Kind: git.RefKindLocal},
				{Name: "feature", Kind: git.RefKindLocal},
			},
		},
	}
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.width = 80
	m.height = 24

	view := m.View()

	assert.Contains(t, view, "main")
	assert.Contains(t, view, "feature")
}

func TestLogView_LastRow_ShowsLoadMoreHint(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, true, "main")
	m.hasMore = true
	m.width = 80
	m.height = 50 // tall enough to show all

	view := m.View()

	assert.Contains(t, view, "+ to show more")
}

func TestLogModel_YankHash_CopiesHash(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	// Should return a command for yanking
	require.NotNil(t, cmd)
}

func TestLogModel_SelectOpensCommitViewOverlay(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.width = 80
	m.height = 24

	// Press Enter to select
	newM, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	// Commit view overlay should be set
	require.NotNil(t, newM.commitView)
	assert.True(t, newM.commitView.CommitID() != "")
	require.NotNil(t, cmd) // Init cmd for loading commit data
}

func TestLogModel_CommitViewOverlay_QClosesOverlay(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.width = 80
	m.height = 24

	// First open commit view
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, m.commitView)

	// Press q to close commit view
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Commit view should be closed
	assert.Nil(t, m.commitView)
}

func TestLogModel_CommitViewOverlay_KeysDelegatedToCommitView(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.width = 80
	m.height = 24

	// First open commit view
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, m.commitView)

	// j should be delegated to commit view, not move log cursor
	origLogCursor := m.cursor
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Log cursor should not have changed
	assert.Equal(t, origLogCursor, m.cursor)
	// Commit view should still be active
	assert.NotNil(t, m.commitView)
}
