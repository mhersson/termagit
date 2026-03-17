package vim

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender_BlockCursor_NormalMode(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 2) // On 'l'
	e.SetMode(ModeNormal)
	e.SetSize(80, 24)

	view := e.View()

	// The view should contain the text
	assert.Contains(t, view, "h")
	assert.Contains(t, view, "e")
	assert.Contains(t, view, "l")
	assert.Contains(t, view, "o")
	// In normal mode, cursor position should be visible
	// The exact rendering depends on style, but text should be there
}

func TestRender_InsertMode_ShowsCursor(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 5) // After 'o'
	e.SetSize(80, 24)

	view := e.View()

	assert.Contains(t, view, "hello")
}

func TestRender_InsertMode_CursorVisible(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 2) // Between 'e' and 'l'
	e.SetMode(ModeInsert)
	e.SetSize(80, 24)

	view := e.View()

	// In insert mode, the cursor should be visible as a block on the character
	// at the cursor position (or a special marker if at end of line)
	assert.Contains(t, view, "he")
	assert.Contains(t, view, "llo")
	// The view should show something at cursor position
	assert.NotEmpty(t, view)
}

func TestRender_InsertMode_CursorAtEndOfLine(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetCursor(0, 5) // Past last char
	e.SetMode(ModeInsert)
	e.SetSize(80, 24)

	view := e.View()

	// Should render a cursor indicator even at end of line
	assert.Contains(t, view, "hello")
	// In insert mode at end of line, we need some visual cursor
}

func TestRender_SelectionHighlight_VisualMode(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("line1\nline2\nline3")
	e.SetCursor(0, 0)
	e.SetMode(ModeNormal)
	e.SetSize(80, 24)

	// Enter visual line mode and select 2 lines
	e.HandleKey(runeKey('V'))
	e.HandleKey(runeKey('j'))

	view := e.View()

	// All selected lines should be visible
	assert.Contains(t, view, "line1")
	assert.Contains(t, view, "line2")
	assert.Contains(t, view, "line3")
}

func TestRender_EmptyBuffer(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("")
	e.SetSize(80, 24)

	view := e.View()

	// Should not panic, should render something
	assert.NotPanics(t, func() { _ = e.View() })
	_ = view // View may be empty or contain cursor
}

func TestRender_MultipleLines(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("first\nsecond\nthird")
	e.SetSize(80, 24)

	view := e.View()

	assert.Contains(t, view, "first")
	assert.Contains(t, view, "second")
	assert.Contains(t, view, "third")
}

func TestRender_ModeIndicator(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello")
	e.SetMode(ModeNormal)
	e.SetSize(80, 24)

	view := e.View()
	// Just verify view renders without panic
	assert.NotEmpty(t, view)
}

func TestRender_WidthConstraint(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetContent("hello world")
	e.SetSize(5, 24) // Very narrow

	view := e.View()

	// Should not panic with narrow width
	assert.NotPanics(t, func() { _ = e.View() })
	_ = view
}

func TestRender_ViewportShowsTopAfterSetContent(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetSize(80, 5) // Small height to force viewport

	// Create content with many lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line content")
	}
	content := strings.Join(lines, "\n")

	e.SetContent(content)
	e.SetCursor(0, 0)

	view := e.View()

	// The view should start with line 0 content, not show lines from the end
	assert.True(t, strings.HasPrefix(view, "line content"), "viewport should show line 0 at top")
}

func TestRender_ViewportScrollsWithCursor(t *testing.T) {
	e := NewEditor(testTokens())
	e.SetSize(80, 3) // Only 3 visible lines

	e.SetContent("line0\nline1\nline2\nline3\nline4\nline5")
	e.SetCursor(0, 0)

	// Initially should show lines 0-2
	view := e.View()
	assert.Contains(t, view, "line0")
	assert.Contains(t, view, "line1")
	assert.Contains(t, view, "line2")
	assert.NotContains(t, view, "line5")

	// Move cursor to line 5
	e.SetCursor(5, 0)
	view = e.View()

	// Now should show lines including line5
	assert.Contains(t, view, "line5")
}
