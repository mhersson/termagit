package cmdhistory

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/cmdlog"
	"github.com/mhersson/termagit/internal/theme"
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
	assert.Equal(t, 0, m.cursor.Pos)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor.Pos)
}

func TestCmdHistory_View_ExpandedShowsError(t *testing.T) {
	entries := []cmdlog.Entry{
		{
			Command:    "git push origin main",
			ExitCode:   128,
			Stderr:     "",
			Error:      "exit status 128",
			DurationMs: 500,
		},
	}
	m := New(entries, testTokens(), 80, 24)
	// Expand
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)
	v := m.View()
	assert.Contains(t, v, "exit status 128")
}

func TestCmdHistory_View_ExpandedShowsErrorAndStderr(t *testing.T) {
	entries := []cmdlog.Entry{
		{
			Command:    "git push origin main",
			ExitCode:   128,
			Stderr:     "fatal: remote rejected\n",
			Error:      "exit status 128",
			DurationMs: 500,
		},
	}
	m := New(entries, testTokens(), 80, 24)
	// Expand
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = newModel.(Model)
	v := m.View()
	assert.Contains(t, v, "fatal: remote rejected")
	assert.Contains(t, v, "exit status 128")
}

func TestCmdHistory_Update_K_MovesUp(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	// Move to entry 1 first
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor.Pos)

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor.Pos)
}

func TestCmdHistory_EntryHeight_Folded(t *testing.T) {
	m := New(testEntries(), testTokens(), 80, 24)
	// All entries start folded, height should be 1
	assert.Equal(t, 1, entryHeight(&m, 0))
	assert.Equal(t, 1, entryHeight(&m, 1))
	assert.Equal(t, 1, entryHeight(&m, 2))
}

func TestCmdHistory_EntryHeight_ExpandedWithStdout(t *testing.T) {
	entries := []cmdlog.Entry{
		{Command: "git status", ExitCode: 0, Stdout: "line1\nline2\nline3\n", DurationMs: 42},
	}
	m := New(entries, testTokens(), 80, 24)
	m.folded[0] = false
	// 1 (header) + 3 (stdout lines) + 1 (trailing blank) = 5
	assert.Equal(t, 5, entryHeight(&m, 0))
}

func TestCmdHistory_EntryHeight_ExpandedWithStderrAndError(t *testing.T) {
	entries := []cmdlog.Entry{
		{
			Command:    "git push",
			ExitCode:   128,
			Stderr:     "fatal: remote rejected\nerror: failed\n",
			Error:      "exit status 128",
			DurationMs: 500,
		},
	}
	m := New(entries, testTokens(), 80, 24)
	m.folded[0] = false
	// 1 (header) + 2 (stderr lines) + 1 (error line) + 1 (trailing blank) = 5
	assert.Equal(t, 5, entryHeight(&m, 0))
}

func TestCmdHistory_EntryHeight_ExpandedEmptyOutput(t *testing.T) {
	entries := []cmdlog.Entry{
		{Command: "git add .", ExitCode: 0, DurationMs: 5},
	}
	m := New(entries, testTokens(), 80, 24)
	m.folded[0] = false
	// 1 (header) + 0 (no output) + 1 (trailing blank) = 2
	assert.Equal(t, 2, entryHeight(&m, 0))
}

func TestCmdHistory_EnsureCursorVisible_ExpandedAbovePushesCursorDown(t *testing.T) {
	// 5 entries, height=10, headerRows=2 → VisibleLines=8
	// Entry 0 expanded with 5-line output → entryHeight=7
	// Entry 1 folded → 1 line. Total 8 = fits exactly.
	// Entry 2 would push to 9 lines → need to scroll.
	entries := make([]cmdlog.Entry, 5)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	entries[0].Stdout = "a\nb\nc\nd\ne\n" // 5 lines

	m := New(entries, testTokens(), 80, 10)
	m.folded[0] = false // expanded: 1+5+1=7 lines
	m.cursor.Pos = 3    // cursor at entry 3
	m.cursor.Offset = 0

	ensureCursorVisible(&m)

	// Entries 0 (7 lines) + 1 (1) + 2 (1) + 3 (1) = 10 lines > 8 visible
	// Need to bump offset until cursor fits
	assert.True(t, m.cursor.Offset > 0, "offset should have increased")
	// Verify cursor entry fits: sum heights from Offset to Pos must be <= VisibleLines
	linesUsed := 0
	for i := m.cursor.Offset; i <= m.cursor.Pos; i++ {
		linesUsed += entryHeight(&m, i)
	}
	assert.LessOrEqual(t, linesUsed, m.cursor.VisibleLines())
}

func TestCmdHistory_EnsureCursorVisible_CursorAboveOffset(t *testing.T) {
	entries := make([]cmdlog.Entry, 10)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}

	m := New(entries, testTokens(), 80, 24)
	m.cursor.Offset = 5
	m.cursor.Pos = 3

	ensureCursorVisible(&m)

	assert.Equal(t, 3, m.cursor.Offset, "offset should snap to cursor when cursor is above")
}

func TestCmdHistory_EnsureCursorVisible_AllFolded(t *testing.T) {
	entries := make([]cmdlog.Entry, 20)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}

	m := New(entries, testTokens(), 80, 12) // VisibleLines=10
	m.cursor.Pos = 15
	m.cursor.Offset = 0

	ensureCursorVisible(&m)

	// Cursor at 15, visible=10, so offset should be at least 6
	assert.Equal(t, 6, m.cursor.Offset)
}

func TestCmdHistory_G_GoesToBottom(t *testing.T) {
	entries := make([]cmdlog.Entry, 10)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	m := New(entries, testTokens(), 80, 24)
	assert.Equal(t, 0, m.cursor.Pos)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(Model)
	assert.Equal(t, 9, m.cursor.Pos)
}

func TestCmdHistory_GG_GoesToTop(t *testing.T) {
	entries := make([]cmdlog.Entry, 10)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	m := New(entries, testTokens(), 80, 24)

	// Move to bottom first
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(Model)
	assert.Equal(t, 9, m.cursor.Pos)

	// Press g then g
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)
	assert.Equal(t, 0, m.cursor.Pos)
	assert.Equal(t, 0, m.cursor.Offset)
}

func TestCmdHistory_CtrlD_HalfPageDown(t *testing.T) {
	entries := make([]cmdlog.Entry, 20)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	m := New(entries, testTokens(), 80, 12) // VisibleLines=10, half=5
	assert.Equal(t, 0, m.cursor.Pos)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = newModel.(Model)
	assert.Equal(t, 5, m.cursor.Pos)
}

func TestCmdHistory_CtrlU_HalfPageUp(t *testing.T) {
	entries := make([]cmdlog.Entry, 20)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	m := New(entries, testTokens(), 80, 12) // VisibleLines=10, half=5

	// Move down first
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(Model)
	assert.Equal(t, 19, m.cursor.Pos)

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = newModel.(Model)
	assert.Equal(t, 14, m.cursor.Pos)
}

func TestCmdHistory_CtrlF_PageDown(t *testing.T) {
	entries := make([]cmdlog.Entry, 30)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	m := New(entries, testTokens(), 80, 12) // VisibleLines=10

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	m = newModel.(Model)
	assert.Equal(t, 10, m.cursor.Pos)
}

func TestCmdHistory_CtrlB_PageUp(t *testing.T) {
	entries := make([]cmdlog.Entry, 30)
	for i := range entries {
		entries[i] = cmdlog.Entry{Command: "git status", ExitCode: 0, DurationMs: 1}
	}
	m := New(entries, testTokens(), 80, 12) // VisibleLines=10

	// Move to bottom
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = newModel.(Model)
	assert.Equal(t, 29, m.cursor.Pos)

	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
	m = newModel.(Model)
	assert.Equal(t, 19, m.cursor.Pos)
}

func TestCmdHistory_View_OnlyRendersVisibleEntries(t *testing.T) {
	entries := make([]cmdlog.Entry, 20)
	for i := range entries {
		entries[i] = cmdlog.Entry{
			Command:    fmt.Sprintf("git cmd-%d", i),
			ExitCode:   0,
			DurationMs: 1,
		}
	}
	m := New(entries, testTokens(), 80, 7) // VisibleLines=5
	v := m.View()

	// First 5 entries should be visible (offset=0)
	assert.Contains(t, v, "git cmd-0")
	assert.Contains(t, v, "git cmd-4")
	// Entry beyond viewport should NOT be visible
	assert.NotContains(t, v, "git cmd-5")
	assert.NotContains(t, v, "git cmd-19")
}

func TestCmdHistory_View_RendersFromOffset(t *testing.T) {
	entries := make([]cmdlog.Entry, 20)
	for i := range entries {
		entries[i] = cmdlog.Entry{
			Command:    fmt.Sprintf("git cmd-%d", i),
			ExitCode:   0,
			DurationMs: 1,
		}
	}
	m := New(entries, testTokens(), 80, 7) // VisibleLines=5
	m.cursor.Offset = 5
	m.cursor.Pos = 5
	v := m.View()

	// Entries before offset should NOT be visible
	assert.NotContains(t, v, "git cmd-0")
	assert.NotContains(t, v, "git cmd-4")
	// Entries from offset should be visible
	assert.Contains(t, v, "git cmd-5")
	assert.Contains(t, v, "git cmd-9")
	// Entry beyond viewport should NOT be visible
	assert.NotContains(t, v, "git cmd-10")
}

func TestCmdHistory_View_ZeroDimensions_ReturnsEmpty(t *testing.T) {
	m := New(testEntries(), testTokens(), 0, 0)
	assert.Equal(t, "", m.View())
}
