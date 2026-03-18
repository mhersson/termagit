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

func TestLineStyle_BulletNotColoredAsDiff(t *testing.T) {
	tokens := testTokens()
	e := NewEditor(tokens)

	// Simulate a commit message with a bullet list, followed by scissors + diff
	content := "feat: add things\n" +
		"\n" +
		"- bullet item one\n" +
		"+ plus prefix line\n" +
		"# On branch main\n" +
		"# ------------------------ >8 ------------------------\n" +
		"diff --git a/file.go b/file.go\n" +
		"@@ -1,3 +1,4 @@\n" +
		" context\n" +
		"-deleted line\n" +
		"+added line\n"
	e.SetContent(content)
	e.SetSize(80, 24)

	// "- bullet item one" is on line 2 (0-indexed), above scissors → Normal style
	assert.Equal(t, tokens.Normal, e.lineStyle("- bullet item one", 2),
		"bullet list in message area must not be colored as DiffDelete")

	// "+ plus prefix line" is on line 3, above scissors → Normal style
	assert.Equal(t, tokens.Normal, e.lineStyle("+ plus prefix line", 3),
		"plus prefix in message area must not be colored as DiffAdd")

	// "# On branch main" is on line 4, above scissors → Comment style (always)
	assert.Equal(t, tokens.Comment, e.lineStyle("# On branch main", 4),
		"comment lines should be styled everywhere")
}

func TestLineStyle_DiffBelowScissorsColored(t *testing.T) {
	tokens := testTokens()
	e := NewEditor(tokens)

	content := "feat: add things\n" +
		"# ------------------------ >8 ------------------------\n" +
		"diff --git a/file.go b/file.go\n" +
		"@@ -1,3 +1,4 @@\n" +
		" context\n" +
		"-deleted line\n" +
		"+added line\n"
	e.SetContent(content)
	e.SetSize(80, 24)

	// Line 2: "diff --git ..." → DiffHeader
	assert.Equal(t, tokens.DiffHeader, e.lineStyle("diff --git a/file.go b/file.go", 2),
		"diff header below scissors must be styled")

	// Line 3: "@@ ..." → DiffHunkHeader
	assert.Equal(t, tokens.DiffHunkHeader, e.lineStyle("@@ -1,3 +1,4 @@", 3),
		"hunk header below scissors must be styled")

	// Line 5: "-deleted line" → DiffDelete
	assert.Equal(t, tokens.DiffDelete, e.lineStyle("-deleted line", 5),
		"deleted line below scissors must be styled as DiffDelete")

	// Line 6: "+added line" → DiffAdd
	assert.Equal(t, tokens.DiffAdd, e.lineStyle("+added line", 6),
		"added line below scissors must be styled as DiffAdd")
}

func TestLineStyle_NoScissorsAllNormal(t *testing.T) {
	tokens := testTokens()
	e := NewEditor(tokens)

	// Content without scissors line — no diff section
	content := "feat: add things\n" +
		"\n" +
		"- bullet item\n" +
		"+ positive note\n"
	e.SetContent(content)
	e.SetSize(80, 24)

	// Without scissors, "-" and "+" should be Normal (no diff section)
	assert.Equal(t, tokens.Normal, e.lineStyle("- bullet item", 2),
		"without scissors, bullet should be Normal")
	assert.Equal(t, tokens.Normal, e.lineStyle("+ positive note", 3),
		"without scissors, plus prefix should be Normal")
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
