package vim

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestVimEditor_VisualLine_jk_ExtendsSelection(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0) // Start on line2
	e.SetMode(ModeNormal)

	// Enter visual line mode
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.Equal(t, ModeVisualLine, e.Mode())
	assert.Equal(t, 1, e.SelectionStart())
	assert.Equal(t, 1, e.SelectionEnd())

	// Move down - extends selection
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, e.SelectionStart())
	assert.Equal(t, 2, e.SelectionEnd())

	// Move down again
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, e.SelectionStart())
	assert.Equal(t, 3, e.SelectionEnd())
}

func TestVimEditor_VisualLine_k_CanExtendUp(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(2, 0) // Start on line3
	e.SetMode(ModeNormal)

	// Enter visual line mode
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})

	// Move up - extends selection upward
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 1, e.SelectionStart())
	assert.Equal(t, 2, e.SelectionEnd())
}

func TestVimEditor_VisualLine_d_DeletesSelection(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	// Select line2 and line3
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, e.SelectionStart())
	assert.Equal(t, 2, e.SelectionEnd())

	// Delete selection
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, ModeNormal, e.Mode())
	assert.Equal(t, 2, e.LineCount())
	assert.Equal(t, "line1\nline4", e.Content())
}

func TestVimEditor_VisualLine_c_ChangesSelection(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	// Select line2 and line3
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Change selection (delete + insert mode)
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, ModeInsert, e.Mode())
	assert.Equal(t, 2, e.LineCount())
	assert.Equal(t, "line1\nline4", e.Content())
}

func TestVimEditor_VisualLine_AllLines_LeavesEmpty(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	// Select all lines
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Delete
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, 1, e.LineCount())
	assert.Equal(t, "", e.Content())
}

func TestVimEditor_VisualLine_d_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	assert.Equal(t, "line2\nline3", e.Register())
	assert.True(t, e.RegisterIsLine())
}

func TestVimEditor_VisualLine_c_PopulatesRegister(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, "line2\nline3", e.Register())
	assert.True(t, e.RegisterIsLine())
}

func TestVimEditor_VisualLine_y_YanksSelection(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3\nline4")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.Equal(t, "line2\nline3", e.Register())
	assert.True(t, e.RegisterIsLine())
	assert.Equal(t, ModeNormal, e.Mode())
	assert.Equal(t, "line1\nline2\nline3\nline4", e.Content(), "content should be unchanged")
}

func TestVimEditor_VisualLine_ESC_ClearsSelection(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})

	assert.Equal(t, ModeNormal, e.Mode())
	// Content should be unchanged
	assert.Equal(t, "line1\nline2\nline3", e.Content())
}
