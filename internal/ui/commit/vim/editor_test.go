package vim

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestVimEditor_New_RespectsInitialMode(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	assert.Equal(t, ModeNormal, e.Mode(), "should respect ModeNormal initial mode")

	e2 := NewEditor(testTokens(), ModeInsert)
	assert.Equal(t, ModeInsert, e2.Mode(), "should respect ModeInsert initial mode")
}

func TestVimEditor_ESC_SwitchesToNormal(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	assert.Equal(t, ModeInsert, e.Mode())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, ModeNormal, e.Mode())
}

func TestVimEditor_i_SwitchesToInsert(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	assert.Equal(t, ModeInsert, e.Mode())
}

func TestVimEditor_a_SwitchesToInsert_MovesRight(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, ModeInsert, e.Mode())
	assert.Equal(t, 1, e.Col(), "cursor should move right for 'a'")
}

func TestVimEditor_A_SwitchesToInsert_GoesToLineEnd(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	assert.Equal(t, ModeInsert, e.Mode())
	assert.Equal(t, 5, e.Col(), "cursor should be at end of line for 'A'")
}

func TestVimEditor_o_NewLineBelow_InsertMode(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	assert.Equal(t, ModeInsert, e.Mode())
	assert.Equal(t, 1, e.Line(), "cursor should be on new line below")
	assert.Equal(t, 3, e.LineCount(), "should have 3 lines now")
	assert.Equal(t, "", e.LineContent(1), "new line should be empty")
}

func TestVimEditor_O_NewLineAbove_InsertMode(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}})
	assert.Equal(t, ModeInsert, e.Mode())
	assert.Equal(t, 1, e.Line(), "cursor should be on new line above (now line 1)")
	assert.Equal(t, 3, e.LineCount(), "should have 3 lines now")
	assert.Equal(t, "", e.LineContent(1), "new line should be empty")
}

func TestVimEditor_hjkl_MoveCursor(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello\nworld")
	e.SetCursor(0, 2)
	e.SetMode(ModeNormal)

	// h = left
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, 1, e.Col())

	// l = right
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.Equal(t, 2, e.Col())

	// j = down
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, e.Line())

	// k = up
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, e.Line())
}

func TestVimEditor_0Dollar_LineMotions(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 5)
	e.SetMode(ModeNormal)

	// 0 = line start
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	assert.Equal(t, 0, e.Col())

	// $ = line end
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	assert.Equal(t, 10, e.Col(), "should be at last char 'd'")
}

func TestVimEditor_ggG_BufferMotions(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	// gg = buffer start
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	assert.Equal(t, 0, e.Line())

	// G = buffer end
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	assert.Equal(t, 2, e.Line())
}

func TestVimEditor_wb_WordMotions(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("hello world")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	// w = word forward
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	assert.Equal(t, 6, e.Col(), "should be at 'world'")

	// b = word backward
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	assert.Equal(t, 0, e.Col(), "should be back at 'hello'")
}

func TestVimEditor_InsertMode_TypingAddsText(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("")

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	assert.Equal(t, "Hi", e.Content())
}

func TestVimEditor_InsertMode_Backspace(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("Hello")
	e.SetCursor(0, 5)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "Hell", e.Content())
}

func TestVimEditor_InsertMode_Enter_CreatesNewLine(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("HelloWorld")
	e.SetCursor(0, 5)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, 2, e.LineCount())
	assert.Equal(t, "Hello", e.LineContent(0))
	assert.Equal(t, "World", e.LineContent(1))
}

func TestVimEditor_InsertMode_Space_InsertsSpace(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("HelloWorld")
	e.SetCursor(0, 5)

	e.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	assert.Equal(t, "Hello World", e.Content())
	assert.Equal(t, 6, e.Col(), "cursor should advance after space")
}

func TestVimEditor_InsertMode_Tab_InsertsTab(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("Hello")
	e.SetCursor(0, 5)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, "Hello\t", e.Content())
}

func TestVimEditor_NormalMode_TypingDoesNotAddText(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("Hello")
	e.SetMode(ModeNormal)

	// In normal mode, random typing should not insert text
	// (only recognized commands do something)
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	assert.Equal(t, "Hello", e.Content(), "z should not modify text in normal mode")
}

func TestVimEditor_V_EntersVisualLineMode(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(1, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.Equal(t, ModeVisualLine, e.Mode())
	assert.Equal(t, 1, e.SelectionStart(), "selection should start at current line")
	assert.Equal(t, 1, e.SelectionEnd(), "selection should end at current line")
}

func TestVimEditor_VisualLine_ESC_ReturnsToNormal(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	e.SetContent("line1\nline2\nline3")
	e.SetMode(ModeNormal)
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	assert.Equal(t, ModeVisualLine, e.Mode())

	e.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, ModeNormal, e.Mode())
}

func TestVimEditor_CtrlF_PageDown(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	// Create 20 lines
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line")
	}
	e.SetContent(strings.Join(lines, "\n"))
	e.SetSize(80, 10) // 10 line viewport
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlF})

	// Should move down by height-1 (9 lines)
	assert.Equal(t, 9, e.Line())
}

func TestVimEditor_CtrlB_PageUp(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line")
	}
	e.SetContent(strings.Join(lines, "\n"))
	e.SetSize(80, 10)
	e.SetCursor(15, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlB})

	// Should move up by height-1 (9 lines)
	assert.Equal(t, 6, e.Line())
}

func TestVimEditor_CtrlD_HalfPageDown(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line")
	}
	e.SetContent(strings.Join(lines, "\n"))
	e.SetSize(80, 10)
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlD})

	// Should move down by height/2 (5 lines)
	assert.Equal(t, 5, e.Line())
}

func TestVimEditor_CtrlU_HalfPageUp(t *testing.T) {
	e := NewEditor(testTokens(), ModeNormal)
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line")
	}
	e.SetContent(strings.Join(lines, "\n"))
	e.SetSize(80, 10)
	e.SetCursor(15, 0)
	e.SetMode(ModeNormal)

	e.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})

	// Should move up by height/2 (5 lines)
	assert.Equal(t, 10, e.Line())
}

func TestVimEditor_Paste_MultilineCreatesLines(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("")

	// Simulate pasting "line1\nline2\nline3" via KeyRunes
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("line1\nline2\nline3")})

	assert.Equal(t, 3, e.LineCount(), "paste with 2 newlines should create 3 lines")
	assert.Equal(t, "line1", e.LineContent(0))
	assert.Equal(t, "line2", e.LineContent(1))
	assert.Equal(t, "line3", e.LineContent(2))
}

func TestVimEditor_Paste_CursorPositionAfterMultiline(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("")

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hello\nworld")})

	assert.Equal(t, 1, e.Line(), "cursor should be on second line after paste")
	assert.Equal(t, 5, e.Col(), "cursor should be at end of 'world'")
}

func TestVimEditor_Paste_CRLFHandled(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("")

	// Simulate Windows-style \r\n paste
	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("aaa\r\nbbb\r\nccc")})

	assert.Equal(t, 3, e.LineCount(), "CRLF paste should create 3 lines")
	assert.Equal(t, "aaa", e.LineContent(0), "no \\r should remain in line content")
	assert.Equal(t, "bbb", e.LineContent(1))
	assert.Equal(t, "ccc", e.LineContent(2))
}

func TestVimEditor_Paste_ContentRoundTrip(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("")

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("first\nsecond\nthird")})

	assert.Equal(t, "first\nsecond\nthird", e.Content())
}

func TestVimEditor_Paste_IntoExistingContent(t *testing.T) {
	e := NewEditor(testTokens(), ModeInsert)
	e.SetContent("before after")
	e.SetCursor(0, 7) // between "before " and "after"

	e.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("mid1\nmid2\n")})

	assert.Equal(t, "before mid1\nmid2\nafter", e.Content())
}

// testTokens creates minimal tokens for testing
func testTokens() Tokens {
	return Tokens{
		Normal:      lipgloss.NewStyle(),
		CursorBlock: lipgloss.NewStyle().Reverse(true),
		Selection:   lipgloss.NewStyle().Background(lipgloss.Color("#444444")),
		Comment:     lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")),

		// Diff styles
		DiffAdd:        lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		DiffDelete:     lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		DiffContext:    lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		DiffHunkHeader: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")),
		DiffHeader:     lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
	}
}
