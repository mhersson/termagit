package vim

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestVimEditor_dd_DeletesLine(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, 2, e.LineCount())
	assert.Equal(t, "line1\nline3", e.Content())
}

func TestVimEditor_dd_LastLine(t *testing.T) {
	e := NewEditor(testTokens())
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
	e := NewEditor(testTokens())
	e.SetContent("only line")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, 1, e.LineCount())
	assert.Equal(t, "", e.Content())
}

func TestVimEditor_dw_DeletesWord(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, "world", e.Content())
}

func TestVimEditor_dw_MiddleOfWord(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, "heworld", e.Content())
}

func TestVimEditor_d_dollar_DeletestoLineEnd(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})

	assert.Equal(t, "hello ", e.Content())
}

func TestVimEditor_cw_ChangesWord(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	assert.Equal(t, ModeInsert, e.Mode(), "should switch to insert mode")
	assert.Equal(t, "world", e.Content(), "word should be deleted")
}

func TestVimEditor_cc_ChangesLine(t *testing.T) {
	e := NewEditor(testTokens())
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
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "ello", e.Content())
}

func TestVimEditor_x_MiddleOfLine(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "helo", e.Content())
}

func TestVimEditor_x_AtEndOfLine(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 4) // On 'o'
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "hell", e.Content())
	assert.Equal(t, 3, e.Col(), "cursor should clamp to new end")
}

func TestVimEditor_x_EmptyLine_NoOp(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, "", e.Content())
}

func TestVimEditor_PendingOperator_ESC_Cancels(t *testing.T) {
	e := NewEditor(testTokens())
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
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Equal(t, "hello ", e.Content())
	assert.Equal(t, ModeNormal, e.Mode(), "should stay in normal mode")
}

func TestVimEditor_D_AtLineStart_DeletesWholeLine(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})

	assert.Equal(t, "", e.Content())
}

func TestVimEditor_C_ChangesToLineEnd(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetCursor(0, 6)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})

	assert.Equal(t, "hello ", e.Content())
	assert.Equal(t, ModeInsert, e.Mode(), "should switch to insert mode")
}

func TestVimEditor_C_AtLineStart_DeletesAndInserts(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'C'}})

	assert.Equal(t, "", e.Content())
	assert.Equal(t, ModeInsert, e.Mode())
}
