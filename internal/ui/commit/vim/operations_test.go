package vim

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestVimEditor_dd_DeletesLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, 2, e.LineCount())
	assert.Equal(t, "line1\nline3", e.Content())
}

func TestVimEditor_dd_LastLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, 1, e.LineCount())
	assert.Equal(t, "line1", e.Content())
	assert.Equal(t, 0, e.Line(), "cursor should move to previous line")
}

func TestVimEditor_dd_OnlyLine_LeavesEmpty(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("only line")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, 1, e.LineCount())
	assert.Equal(t, "", e.Content())
}

func TestVimEditor_dw_DeletesWord(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, "world", e.Content())
}

func TestVimEditor_dw_MiddleOfWord(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, "heworld", e.Content())
}

func TestVimEditor_d_dollar_DeletestoLineEnd(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})

	assert.Equal(t, "hello ", e.Content())
}

func TestVimEditor_cw_ChangesWord(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, ModeInsert, e.Mode(), "should switch to insert mode")
	assert.Equal(t, "world", e.Content(), "word should be deleted")
}

func TestVimEditor_cc_ChangesLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 3)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, ModeInsert, e.Mode())
	assert.Equal(t, 2, e.LineCount(), "line should be deleted")
	assert.Equal(t, 0, e.Col(), "cursor should be at col 0")
}

func TestVimEditor_x_DeletesChar(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "ello", e.Content())
}

func TestVimEditor_x_MiddleOfLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "helo", e.Content())
}

func TestVimEditor_x_AtEndOfLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 4) // On 'o'
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "hell", e.Content())
	assert.Equal(t, 3, e.Col(), "cursor should clamp to new end")
}

func TestVimEditor_x_EmptyLine_NoOp(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "", e.Content())
}

func TestVimEditor_PendingOperator_ESC_Cancels(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	// Now in pending 'd' state
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})

	// Should be back to normal, pressing 'd' again shouldn't delete
	assert.Equal(t, "hello", e.Content())
	assert.Equal(t, ModeNormal, e.Mode())
}

func TestVimEditor_D_DeletestoLineEnd(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Equal(t, "hello ", e.Content())
	assert.Equal(t, ModeNormal, e.Mode(), "should stay in normal mode")
}

func TestVimEditor_D_AtLineStart_DeletesWholeLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Equal(t, "", e.Content())
}

func TestVimEditor_C_ChangesToLineEnd(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})

	assert.Equal(t, "hello ", e.Content())
	assert.Equal(t, ModeInsert, e.Mode(), "should switch to insert mode")
}

func TestVimEditor_C_AtLineStart_DeletesAndInserts(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})

	assert.Equal(t, "", e.Content())
	assert.Equal(t, ModeInsert, e.Mode())
}

// Register population tests

func TestVimEditor_Register_InitiallyEmpty(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	assert.Equal(t, "", e.Register())
	assert.False(t, e.RegisterIsLine())
}

func TestVimEditor_dd_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, "line2", e.Register())
	assert.True(t, e.RegisterIsLine())
}

func TestVimEditor_dw_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, "hello ", e.Register())
	assert.False(t, e.RegisterIsLine())
}

func TestVimEditor_d_dollar_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})

	assert.Equal(t, "world", e.Register())
	assert.False(t, e.RegisterIsLine())
}

func TestVimEditor_D_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Equal(t, "world", e.Register())
	assert.False(t, e.RegisterIsLine())
}

func TestVimEditor_x_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "h", e.Register())
	assert.False(t, e.RegisterIsLine())
}

func TestVimEditor_cc_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, "line1", e.Register())
	assert.True(t, e.RegisterIsLine())
}

func TestVimEditor_cw_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, "hello ", e.Register())
	assert.False(t, e.RegisterIsLine())
}

func TestVimEditor_C_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})

	assert.Equal(t, "world", e.Register())
	assert.False(t, e.RegisterIsLine())
}

// Replace (r) tests

func TestVimEditor_r_ReplacesCharUnderCursor(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	assert.Equal(t, "Xello", e.Content())
}

func TestVimEditor_r_MiddleOfLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	assert.Equal(t, "heXlo", e.Content())
}

func TestVimEditor_r_StaysInNormalMode(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	assert.Equal(t, ModeNormal, e.Mode())
}

func TestVimEditor_r_CursorStaysInPlace(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	assert.Equal(t, 2, e.Col(), "cursor should stay at same position")
}

func TestVimEditor_r_OnEmptyLine_NoOp(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})

	assert.Equal(t, "", e.Content())
}

func TestVimEditor_r_ESC_CancelsPending(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})

	assert.Equal(t, "hello", e.Content(), "escape should cancel replace")
	assert.Equal(t, ModeNormal, e.Mode())
}

// Undo (u) tests

func TestVimEditor_u_UndoesDD(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Equal(t, "line1\nline3", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	assert.Equal(t, "line1\nline2\nline3", e.Content())
	assert.Equal(t, 1, e.Line(), "cursor should restore to line 1")
	assert.Equal(t, 0, e.Col(), "cursor col should restore to 0")
}

func TestVimEditor_u_EmptyStack_NoOp(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	assert.Equal(t, "hello", e.Content())
	assert.Equal(t, 0, e.Line())
}

func TestVimEditor_u_UndoesX(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, "ello", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello", e.Content())
}

func TestVimEditor_u_UndoesDW(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	assert.Equal(t, "world", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello world", e.Content())
	assert.Equal(t, 0, e.Col())
}

func TestVimEditor_u_MultipleUndos(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	// dd (delete line1)
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	// dd (delete line2, now at top)
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Equal(t, "line3", e.Content())

	// First undo restores line2
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "line2\nline3", e.Content())

	// Second undo restores line1
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "line1\nline2\nline3", e.Content())
}

func TestVimEditor_u_UndoesInsertSession(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 4)
	e.SetMode(ModeNormal)

	// 'A' to append at end of line -> insert mode
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	assert.Equal(t, ModeInsert, e.Mode())

	// Type " world"
	for _, r := range " world" {
		if r == ' ' {
			e.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
		} else {
			e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}

	// Esc back to normal
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, "hello world", e.Content())

	// u should undo the entire insert session
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello", e.Content())
}

func TestVimEditor_u_UndoesReplace(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'X'}})
	assert.Equal(t, "Xello", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello", e.Content())
}

func TestVimEditor_u_UndoesJoin(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello\nworld")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	assert.Equal(t, "hello world", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello\nworld", e.Content())
}

func TestVimEditor_u_UndoesPaste(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	// yy then p to duplicate
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	assert.Equal(t, "line1\nline1\nline2", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "line1\nline2", e.Content())
}

func TestVimEditor_u_UndoesVisualLineDelete(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	// Vjd to delete lines 2-3
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Equal(t, "line1\nline4", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "line1\nline2\nline3\nline4", e.Content())
}

func TestVimEditor_u_UndoesD(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 5)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	assert.Equal(t, "hello", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello world", e.Content())
}

func TestVimEditor_u_UndoesC(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 5)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})
	assert.Equal(t, ModeInsert, e.Mode())

	// Esc back to normal
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "hello world", e.Content())
}

func TestVimEditor_u_UndoesOpenLineBelow(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	// o to open line below, type "new", Esc
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, "line1\nnew\nline2", e.Content())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "line1\nline2", e.Content())
}

func TestVimEditor_u_YankDoesNotCreateUndoEntry(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	// yy should NOT push an undo entry
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// u should be a no-op
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	assert.Equal(t, "line1\nline2", e.Content())
}
