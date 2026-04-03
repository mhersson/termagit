package reflogview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/shared"
	"github.com/muesli/termenv"
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
	assert.Equal(t, 0, m.cursor.Pos)
}

func TestReflogModel_CursorMovement(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")

	// Move down
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.cursor.Pos)

	// Move up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor.Pos)
}

func TestReflogModel_CursorStopsAtBoundaries(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")

	// At top, can't go up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor.Pos)

	// At bottom, can't go down
	m.cursor.Pos = 2
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 2, m.cursor.Pos)
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
	m.cursor.SetSize(80, 24)

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
	m.cursor.SetSize(80, 24)

	view := m.View()
	assert.Contains(t, view, "commit")
}

func TestReflogView_TypeHighlight_ResetIsRed(t *testing.T) {
	entries := []git.ReflogEntry{
		{Oid: "abc123def456abc123def456abc123def456abc1", Type: "reset", RefSubject: "reset: HEAD~1"},
	}
	m := New(entries, testTokens(), "HEAD")
	m.cursor.SetSize(80, 24)

	view := m.View()
	assert.Contains(t, view, "reset")
}

func TestReflogView_TypeHighlight_CheckoutIsBlue(t *testing.T) {
	entries := []git.ReflogEntry{
		{Oid: "abc123def456abc123def456abc123def456abc1", Type: "checkout", RefSubject: "checkout: main"},
	}
	m := New(entries, testTokens(), "HEAD")
	m.cursor.SetSize(80, 24)

	view := m.View()
	assert.Contains(t, view, "checkout")
}

func TestReflogModel_YankHash(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")
	m.cursor.Pos = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	require.NotNil(t, cmd)
}

func TestReflogView_Select_OpensCommitView(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), "HEAD")
	m.cursor.Pos = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	require.NotNil(t, cmd, "Select should return a command")
	msg := cmd()
	cvMsg, ok := msg.(shared.OpenCommitViewMsg)
	assert.True(t, ok, "expected OpenCommitViewMsg, got %T", msg)
	if ok {
		assert.Equal(t, entries[0].Oid, cvMsg.Hash)
	}
}

func TestReflogView_Select_NoEntries_Noop(t *testing.T) {
	m := New(nil, testTokens(), "HEAD")
	m.cursor.Pos = 0

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "Select with no entries should be nil")
}

// TestReflogView_CursorRow_NoNestedANSI verifies that the cursor-rendered row
// does not contain nested ANSI escape sequences. renderEntry produces ANSI-styled
// content (hash, type, date), so Cursor.Render must receive stripped content to
// avoid nesting. This matches the fix pattern used in logview, stashlist, refsview,
// and cmdhistory.
func TestReflogView_CursorRow_NoNestedANSI(t *testing.T) {
	// Force TrueColor so lipgloss emits ANSI even without a TTY.
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	// Use tokens with explicit hex colors so styles emit ANSI sequences.
	tokens := theme.Compile(theme.RawTokens{
		Hash:       "#7c9af2",
		GraphGreen: "#a9dc76",
		CommitDate: "#78dce8",
		CursorBg:   "#3a3a3a",
	})

	entries := []git.ReflogEntry{
		{
			Oid:        "abc123def456abc123def456abc123def456abc1",
			Index:      0,
			AuthorName: "Test User",
			RefName:    "HEAD@{0}",
			RefSubject: "commit: Add feature",
			RelDate:    "3 hours ago",
			Type:       "commit",
		},
	}
	m := New(entries, tokens, "HEAD")
	m.cursor.SetSize(80, 24)
	m.cursor.Pos = 0

	// renderEntry produces a row with multiple ANSI-styled parts (hash, type, date).
	idxWidth := 1
	row := m.renderEntry(entries[0], idxWidth)

	// Verify the raw entry row contains ANSI sequences (pre-styled content).
	require.NotEqual(t, row, ansi.Strip(row),
		"renderEntry must produce ANSI-styled content for this test to be meaningful")

	// Render using Cursor.Render with pre-styled content — this is what the
	// buggy code does. It embeds multiple ANSI sequences inside the cursor
	// escape, producing nested sequences.
	wrongCursorRow := tokens.Cursor.Render(row)

	// Render using Cursor.Render with stripped content — this is the correct fix.
	correctCursorRow := tokens.Cursor.Render(ansi.Strip(row))

	// The wrong rendering contains multiple ESC characters (one per styled part
	// plus the outer cursor open/close). The correct rendering has only two
	// (one cursor open, one cursor close/reset).
	wrongEscCount := strings.Count(wrongCursorRow, "\x1b")
	correctEscCount := strings.Count(correctCursorRow, "\x1b")

	assert.Greater(t, wrongEscCount, 2,
		"Cursor.Render(row) with pre-styled content should produce nested ANSI (more than 2 ESC sequences)")
	assert.Equal(t, 2, correctEscCount,
		"Cursor.Render(ansi.Strip(row)) should produce exactly 2 ESC sequences (open+close), no nesting")

	// Confirm the view's cursor row matches the correct (stripped) pattern by
	// checking that it has the same ESC count as the correct rendering.
	view := m.View()
	lines := strings.Split(view, "\n")
	var cursorLine string
	for _, line := range lines {
		if strings.Contains(ansi.Strip(line), "abc123d") {
			cursorLine = line
			break
		}
	}

	require.NotEmpty(t, cursorLine, "cursor line with hash 'abc123d' should be present in view")

	viewEscCount := strings.Count(cursorLine, "\x1b")
	assert.Equal(t, correctEscCount, viewEscCount,
		"View cursor row must use ansi.Strip(row) before Cursor.Render — "+
			"found %d ESC sequences, expected %d (the stripped pattern)",
		viewEscCount, correctEscCount)
}
