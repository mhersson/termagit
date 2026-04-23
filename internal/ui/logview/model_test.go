package logview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

func testMergeCommits() []git.LogEntry {
	return []git.LogEntry{
		{
			Hash:            "mmm",
			AbbreviatedHash: "mmm",
			Subject:         "Merge commit",
			AuthorName:      "Test",
			ParentHashes:    "aaa bbb",
		},
		{
			Hash:            "aaa",
			AbbreviatedHash: "aaa",
			Subject:         "Commit A",
			AuthorName:      "Test",
			ParentHashes:    "ccc",
		},
		{
			Hash:            "bbb",
			AbbreviatedHash: "bbb",
			Subject:         "Commit B",
			AuthorName:      "Test",
			ParentHashes:    "ccc",
		},
		{
			Hash:            "ccc",
			AbbreviatedHash: "ccc",
			Subject:         "Commit C",
			AuthorName:      "Test",
		},
	}
}

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
	assert.Equal(t, 0, m.cursor.Pos)
}

func TestLogModel_CursorDown_MovesToNextCommit(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")

	// Press j to move down
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	assert.Equal(t, 1, m.cursor.Pos)
}

func TestLogModel_CursorUp_MovesToPreviousCommit(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Pos = 2

	// Press k to move up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	assert.Equal(t, 1, m.cursor.Pos)
}

func TestLogModel_CursorDown_StopsAtBottom(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Pos = 2 // last commit

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	assert.Equal(t, 2, m.cursor.Pos) // stays at last
}

func TestLogModel_CursorUp_StopsAtTop(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Pos = 0

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	assert.Equal(t, 0, m.cursor.Pos) // stays at top
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
	m.cursor.Width = 80
	m.cursor.Height = 24

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
	m.cursor.Width = 80
	m.cursor.Height = 24

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
	m.cursor.Width = 80
	m.cursor.Height = 24

	view := m.View()

	assert.Contains(t, view, "main")
	assert.Contains(t, view, "feature")
}

func TestLogView_LastRow_ShowsLoadMoreHint(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, true, "main")
	m.hasMore = true
	m.cursor.Width = 80
	m.cursor.Height = 50 // tall enough to show all

	view := m.View()

	assert.Contains(t, view, "+ to show more")
}

func TestLogModel_YankHash_CopiesHash(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Pos = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	// Should return a command for yanking
	require.NotNil(t, cmd)
}

func TestLogModel_SelectOpensCommitViewOverlay(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Width = 80
	m.cursor.Height = 24

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
	m.cursor.Width = 80
	m.cursor.Height = 24

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
	m.cursor.Width = 80
	m.cursor.Height = 24

	// First open commit view
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, m.commitView)

	// j should be delegated to commit view, not move log cursor
	origLogCursor := m.cursor.Pos
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Log cursor should not have changed
	assert.Equal(t, origLogCursor, m.cursor.Pos)
	// Commit view should still be active
	assert.NotNil(t, m.commitView)
}

// Graph integration tests

func TestLogModel_GraphEnabled_HasMoreDisplayRowsThanCommits(t *testing.T) {
	commits := testMergeCommits()
	opts := &git.LogOpts{Graph: true}
	m := New(commits, nil, testTokens(), opts, false, "main")
	m.cursor.Width = 80
	m.cursor.Height = 24

	// With merge commits and graph enabled, displayRows should include connector rows
	assert.Greater(t, len(m.displayRows), len(m.commits),
		"graph should produce more display rows than commits due to connector rows")
}

func TestLogModel_GraphDisabled_DisplayRowsMatchCommits(t *testing.T) {
	commits := testMergeCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Width = 80
	m.cursor.Height = 24

	assert.Equal(t, len(m.commits), len(m.displayRows))
}

func TestLogModel_GraphEnabled_CursorSkipsConnectorRows(t *testing.T) {
	commits := testMergeCommits()
	opts := &git.LogOpts{Graph: true}
	m := New(commits, nil, testTokens(), opts, false, "main")
	m.cursor.Width = 80
	m.cursor.Height = 24

	// Cursor starts at 0, which should be a commit row
	assert.GreaterOrEqual(t, m.displayRows[m.cursor.Pos].commitIdx, 0)

	// Press j — cursor should move to the next commit row, skipping connector rows
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.GreaterOrEqual(t, m.displayRows[m.cursor.Pos].commitIdx, 0,
		"cursor should always land on a commit row, not a connector row")
}

func TestLogModel_GraphEnabled_MaxCursorOnCommitRow(t *testing.T) {
	commits := testMergeCommits()
	opts := &git.LogOpts{Graph: true}
	m := New(commits, nil, testTokens(), opts, false, "main")
	m.cursor.Width = 80
	m.cursor.Height = 50

	// Go to bottom
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	assert.GreaterOrEqual(t, m.displayRows[m.cursor.Pos].commitIdx, 0,
		"max cursor should be on a commit row")
}

func TestLogModel_GraphEnabled_LoadMoreRecomputesGraph(t *testing.T) {
	// Start with 2 linear commits
	commits := []git.LogEntry{
		{Hash: "aaa", AbbreviatedHash: "aaa", Subject: "Commit A", AuthorName: "Test", ParentHashes: "bbb"},
		{Hash: "bbb", AbbreviatedHash: "bbb", Subject: "Commit B", AuthorName: "Test"},
	}
	opts := &git.LogOpts{Graph: true}
	m := New(commits, nil, testTokens(), opts, true, "main")
	m.cursor.Width = 120
	m.cursor.Height = 40

	assert.Equal(t, 2, len(m.commits))

	// Simulate loading more commits
	newModel, _ := m.Update(CommitsLoadedMsg{
		Commits: []logCommit{
			{hash: "ccc", abbrevHash: "ccc", subject: "Commit C", authorName: "Test", parentHashes: "ddd"},
			{hash: "ddd", abbrevHash: "ddd", subject: "Commit D", authorName: "Test"},
		},
		HasMore: false,
	})
	m = newModel.(Model)

	// All 4 commits should be present and displayRows recomputed
	assert.Equal(t, 4, len(m.commits))
	// Count commit rows in displayRows
	commitRowCount := 0
	for _, dr := range m.displayRows {
		if dr.commitIdx >= 0 {
			commitRowCount++
		}
	}
	assert.Equal(t, 4, commitRowCount, "all 4 commits should have display rows")
	assert.False(t, m.hasMore)
}

func TestLogView_GraphEnabled_RendersGraphChars(t *testing.T) {
	commits := testMergeCommits()
	opts := &git.LogOpts{Graph: true}
	m := New(commits, nil, testTokens(), opts, false, "main")
	m.cursor.Width = 120
	m.cursor.Height = 40

	view := m.View()
	assert.Contains(t, view, "•", "graph-enabled view should contain commit marker")
}

func TestLogView_GraphDisabled_NoGraphChars(t *testing.T) {
	commits := testCommits()
	m := New(commits, nil, testTokens(), nil, false, "main")
	m.cursor.Width = 120
	m.cursor.Height = 40

	view := m.View()
	assert.NotContains(t, view, "•", "graph-disabled view should not contain commit marker")
	assert.NotContains(t, view, "│", "graph-disabled view should not contain branch lines")
}

func TestLogView_GraphOnlyRow_NotHighlightedByCursor(t *testing.T) {
	commits := testMergeCommits()
	opts := &git.LogOpts{Graph: true}
	m := New(commits, nil, testTokens(), opts, false, "main")
	m.cursor.Width = 120
	m.cursor.Height = 40

	view := m.View()
	// The view should render without crashing and contain commit data
	assert.Contains(t, view, "Merge commit")
	assert.Contains(t, view, "Commit A")
}
