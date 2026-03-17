package vim

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// View renders the editor content.
func (e *Editor) View() string {
	if e.width == 0 || e.height == 0 {
		return ""
	}

	var b strings.Builder
	lineCount := e.buffer.LineCount()

	for i := 0; i < lineCount; i++ {
		line := e.buffer.Line(i)

		if e.mode == ModeVisualLine && e.isLineSelected(i) {
			// Render selected line with selection highlight
			b.WriteString(e.renderSelectedLine(line, i))
		} else if i == e.cursor.Line {
			// Render line with cursor
			b.WriteString(e.renderLineWithCursor(line))
		} else {
			// Normal line rendering
			b.WriteString(e.tokens.Normal.Render(line))
		}

		if i < lineCount-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// isLineSelected returns true if the line is part of the visual selection.
func (e *Editor) isLineSelected(line int) bool {
	start := e.selStart
	end := e.selEnd
	if start > end {
		start, end = end, start
	}
	return line >= start && line <= end
}

// renderLineWithCursor renders a line with the cursor at the current position.
func (e *Editor) renderLineWithCursor(line string) string {
	col := e.cursor.Col
	lineRunes := []rune(line)

	// Handle empty line
	if len(lineRunes) == 0 {
		// Show a space with cursor style for visibility
		return e.tokens.CursorBlock.Render(" ")
	}

	// Clamp col to valid range
	if col < 0 {
		col = 0
	}
	if e.mode == ModeInsert {
		// Insert mode allows cursor past last char
		if col > len(lineRunes) {
			col = len(lineRunes)
		}
	} else {
		// Normal mode cursor is on a char
		if col >= len(lineRunes) {
			col = len(lineRunes) - 1
		}
	}

	var result strings.Builder

	// Text before cursor
	if col > 0 {
		result.WriteString(e.tokens.Normal.Render(string(lineRunes[:col])))
	}

	if e.mode == ModeInsert {
		// Insert mode - cursor is between characters, but we show it on the next char
		// or a space if at end of line
		if col < len(lineRunes) {
			result.WriteString(e.tokens.CursorBlock.Render(string(lineRunes[col])))
			if col+1 < len(lineRunes) {
				result.WriteString(e.tokens.Normal.Render(string(lineRunes[col+1:])))
			}
		} else {
			// At end of line - show cursor as a space
			result.WriteString(e.tokens.CursorBlock.Render(" "))
		}
	} else {
		// Block cursor in normal/visual mode
		if col < len(lineRunes) {
			result.WriteString(e.tokens.CursorBlock.Render(string(lineRunes[col])))
			// Text after cursor
			if col+1 < len(lineRunes) {
				result.WriteString(e.tokens.Normal.Render(string(lineRunes[col+1:])))
			}
		}
	}

	return result.String()
}

// renderSelectedLine renders a line with selection highlight.
func (e *Editor) renderSelectedLine(line string, lineNum int) string {
	var result strings.Builder

	if line == "" {
		line = " " // Show selection on empty lines
	}

	if lineNum == e.cursor.Line {
		// Selected line with cursor
		col := e.cursor.Col
		lineRunes := []rune(line)

		if col >= len(lineRunes) {
			col = len(lineRunes) - 1
		}
		if col < 0 {
			col = 0
		}

		// Render with both selection and cursor
		if col > 0 {
			result.WriteString(e.tokens.Selection.Render(string(lineRunes[:col])))
		}
		// Cursor position - show with cursor block over selection
		result.WriteString(e.tokens.CursorBlock.Render(string(lineRunes[col])))
		if col+1 < len(lineRunes) {
			result.WriteString(e.tokens.Selection.Render(string(lineRunes[col+1:])))
		}
	} else {
		// Selected line without cursor
		result.WriteString(e.tokens.Selection.Render(line))
	}

	return result.String()
}

// runeKey creates a tea.KeyMsg for a single rune.
func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}
