package cmdhistory

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/cmdlog"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTokens() theme.Tokens {
	t := theme.Fallback()
	return theme.Compile(t.Raw())
}

func testEntries() []cmdlog.Entry {
	return []cmdlog.Entry{
		{
			Timestamp:  time.Now(),
			Command:    "git status",
			ExitCode:   0,
			Stdout:     "M internal/config/config.go\n",
			Stderr:     "",
			DurationMs: 42,
		},
		{
			Timestamp:  time.Now(),
			Command:    "git push origin main",
			ExitCode:   128,
			Stdout:     "",
			Stderr:     "! [rejected] main -> main (non-fast-forward)\nerror: failed to push\n",
			DurationMs: 1203,
		},
		{
			Timestamp:  time.Now(),
			Command:    "git add .",
			ExitCode:   0,
			Stdout:     "",
			Stderr:     "",
			DurationMs: 5,
		},
	}
}

func TestCmdHistory_New_LoadsEntries(t *testing.T) {
	entries := testEntries()
	m := New(entries, testTokens(), 80, 24)
	assert.Equal(t, len(entries), len(m.entries))
}

func TestCmdHistory_View_RendersEntryRow(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	v := m.View()
	assert.Contains(t, v, "git status")
	assert.Contains(t, v, "git push origin main")
}

func TestCmdHistory_View_ExitCodeZero_ShowsZero(t *testing.T) {
	entries := []cmdlog.Entry{
		{Command: "git status", ExitCode: 0, DurationMs: 42},
	}
	m := New(entries, testTokens(), 80, 24)
	v := m.View()
	assert.Contains(t, v, "  0")
}

func TestCmdHistory_View_ExitCodeNonZero_Shows128(t *testing.T) {
	entries := []cmdlog.Entry{
		{Command: "git push origin main", ExitCode: 128, DurationMs: 1203},
	}
	m := New(entries, testTokens(), 80, 24)
	v := m.View()
	assert.Contains(t, v, "128")
}

func TestCmdHistory_View_FoldedByDefault(t *testing.T) {
	entries := []cmdlog.Entry{
		{Command: "git status", ExitCode: 0, Stdout: "secret output\n", DurationMs: 42},
	}
	m := New(entries, testTokens(), 80, 24)
	v := m.View()
	// Output should not be visible when folded
	assert.NotContains(t, v, "secret output")
}

func TestCmdHistory_Update_Tab_TogglesFold(t *testing.T) {
	entries := []cmdlog.Entry{
		{Command: "git status", ExitCode: 0, Stdout: "visible output\n", DurationMs: 42},
	}
	m := New(entries, testTokens(), 80, 24)

	// Toggle open
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)
	v := m.View()
	assert.Contains(t, v, "visible output")

	// Toggle closed
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)
	v = m.View()
	assert.NotContains(t, v, "visible output")
}

func TestCmdHistory_Update_Escape_SendsClose(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseMsg)
	assert.True(t, ok, "expected CloseMsg")
}

func TestCmdHistory_Update_Q_SendsClose(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseMsg)
	assert.True(t, ok, "expected CloseMsg")
}

func TestCmdHistory_View_ExpandedShowsOutput(t *testing.T) {
	entries := []cmdlog.Entry{
		{
			Command:    "git push origin main",
			ExitCode:   128,
			Stdout:     "",
			Stderr:     "! [rejected] main -> main (non-fast-forward)\n",
			DurationMs: 1203,
		},
	}
	m := New(entries, testTokens(), 80, 24)
	// Expand
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)
	v := m.View()
	assert.Contains(t, v, "[rejected]")
}

func TestCmdHistory_Update_J_MovesDown(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	assert.Equal(t, 0, m.cursor)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)
}

func TestCmdHistory_Update_K_MovesUp(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	// Move to entry 1 first
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor)
}
