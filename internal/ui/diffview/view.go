package diffview

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/mhersson/conjit/internal/git"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content string

	if m.loading {
		content = "Loading diff..."
	} else if m.err != nil {
		content = fmt.Sprintf("Error: %v", m.err)
	} else if len(m.files) == 0 {
		content = "No changes"
	} else {
		content = m.renderContentWithCursor()
	}

	// Apply viewport scrolling
	lines := strings.Split(content, "\n")
	startLine := m.viewport.YOffset
	endLine := startLine + m.viewport.Height

	if startLine > len(lines) {
		startLine = len(lines)
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	visibleLines := lines[startLine:endLine]

	// Pad with empty lines to fill full height
	for len(visibleLines) < m.height {
		visibleLines = append(visibleLines, "")
	}

	return strings.Join(visibleLines, "\n")
}

// renderContentWithCursor builds the diff view content with cursor highlighting.
func (m Model) renderContentWithCursor() string {
	var b strings.Builder
	lineNum := 0

	// Header line
	headerLine := m.header
	if pad := m.width - len(m.header); pad > 0 {
		headerLine += strings.Repeat(" ", pad)
	}
	m.writeLine(&b, headerLine, lineNum, m.tokens.FloatHeaderHighlight.Render)
	lineNum++

	// Stat block
	if m.stats != nil && len(m.stats.Files) > 0 {
		lineNum = m.renderStatBlock(&b, lineNum)
	}

	// File diffs
	for fi, diff := range m.files {
		lineNum = m.renderFileDiff(&b, &diff, fi, lineNum)
	}

	return b.String()
}

// renderContent builds the full content without cursor (for line counting).
func (m Model) renderContent() string {
	var b strings.Builder

	// Header
	b.WriteString(m.header)
	b.WriteString("\n")

	// Stat block
	if m.stats != nil && len(m.stats.Files) > 0 {
		b.WriteString(m.stats.Summary)
		b.WriteString("\n")
		for _, f := range m.stats.Files {
			b.WriteString(f.Path)
			b.WriteString(" | ")
			b.WriteString(f.Changes)
			if f.IsBinary {
				b.WriteString(" (binary)")
			} else {
				b.WriteString(" ")
				b.WriteString(f.Insertions)
				b.WriteString(f.Deletions)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n") // empty line after stats
	}

	// File diffs
	for fi, diff := range m.files {
		// File header
		b.WriteString(m.formatFileHeader(&diff, fi))
		b.WriteString("\n")
		// Separator
		b.WriteString(strings.Repeat("─", m.width))
		b.WriteString("\n")
		// Hunks
		for _, hunk := range diff.Hunks {
			b.WriteString(hunk.Header)
			b.WriteString("\n")
			for _, line := range hunk.Lines {
				switch line.Op {
				case git.DiffOpAdd:
					b.WriteString("+" + line.Content)
				case git.DiffOpDelete:
					b.WriteString("-" + line.Content)
				default:
					b.WriteString(" " + line.Content)
				}
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// renderStatBlock renders the stat overview block.
func (m Model) renderStatBlock(b *strings.Builder, lineNum int) int {
	// Summary line
	m.writeLine(b, m.stats.Summary, lineNum, nil)
	lineNum++

	// Compute file path padding for alignment
	maxPathLen := 0
	for _, f := range m.stats.Files {
		if len(f.Path) > maxPathLen {
			maxPathLen = len(f.Path)
		}
	}

	// File stat lines
	for _, f := range m.stats.Files {
		var fileLine strings.Builder
		fileLine.WriteString(m.tokens.FilePath.Render(padRight(f.Path, maxPathLen)))
		fileLine.WriteString("  | ")
		fileLine.WriteString(m.tokens.Number.Render(padLeft(f.Changes, 5)))
		if f.IsBinary {
			fileLine.WriteString("  (binary)")
		} else {
			fileLine.WriteString("  ")
			fileLine.WriteString(m.tokens.DiffAdd.Render(f.Insertions))
			fileLine.WriteString(m.tokens.DiffDelete.Render(f.Deletions))
		}

		m.writeLine(b, fileLine.String(), lineNum, nil)
		lineNum++
	}

	// Empty line after stats
	m.writeLine(b, "", lineNum, nil)
	lineNum++

	return lineNum
}

// renderFileDiff renders a single file's diff with cursor tracking.
func (m Model) renderFileDiff(b *strings.Builder, diff *git.FileDiff, fileIdx int, lineNum int) int {
	// File header with counters
	fileHeader := m.formatFileHeader(diff, fileIdx)
	m.writeLine(b, fileHeader, lineNum, nil)
	lineNum++

	// Separator line
	sep := strings.Repeat("─", m.width)
	m.writeLine(b, sep, lineNum, m.tokens.DiffHeader.Render)
	lineNum++

	// Line number tracking
	showLineNos := m.cfg != nil && !m.cfg.UI.DisableLineNumbers

	// Hunks
	for _, hunk := range diff.Hunks {
		// Hunk header
		m.writeLine(b, hunk.Header, lineNum, m.tokens.DiffHunkHeader.Render)
		lineNum++

		// Hunk lines
		oldLine := hunk.OldStart
		newLine := hunk.NewStart
		for _, line := range hunk.Lines {
			var prefix string
			if showLineNos {
				prefix = m.renderLineNumbers(line.Op, oldLine, newLine)
			}

			var styledLine string
			switch line.Op {
			case git.DiffOpAdd:
				styledLine = prefix + m.tokens.DiffAdd.Render("+"+line.Content)
				newLine++
			case git.DiffOpDelete:
				styledLine = prefix + m.tokens.DiffDelete.Render("-"+line.Content)
				oldLine++
			default:
				styledLine = prefix + m.tokens.DiffContext.Render(" "+line.Content)
				oldLine++
				newLine++
			}

			if lineNum == m.cursorLine {
				var rawLine string
				switch line.Op {
				case git.DiffOpAdd:
					rawLine = prefix + "+" + line.Content
				case git.DiffOpDelete:
					rawLine = prefix + "-" + line.Content
				default:
					rawLine = prefix + " " + line.Content
				}
				b.WriteString(m.renderCursorLine(rawLine))
			} else {
				b.WriteString(styledLine)
				b.WriteString("\n")
			}
			lineNum++
		}
	}

	return lineNum
}

// formatFileHeader formats the file header line with file and hunk counters.
func (m Model) formatFileHeader(diff *git.FileDiff, fileIdx int) string {
	filePath := m.tokens.FilePath.Render(diff.Path)

	totalFiles := len(m.files)
	totalHunks := len(diff.Hunks)

	counter := m.tokens.SubtleText.Render(
		fmt.Sprintf("file %d/%d  hunk %d", fileIdx+1, totalFiles, totalHunks),
	)

	// Calculate spacing between path and counter
	pathLen := len(diff.Path)
	counterLen := len(fmt.Sprintf("file %d/%d  hunk %d", fileIdx+1, totalFiles, totalHunks))
	spacing := m.width - pathLen - counterLen
	if spacing < 2 {
		spacing = 2
	}

	return filePath + strings.Repeat(" ", spacing) + counter
}

// renderLineNumbers renders the old/new line number gutter.
func (m Model) renderLineNumbers(op git.DiffOp, oldLine, newLine int) string {
	var oldStr, newStr string

	switch op {
	case git.DiffOpAdd:
		oldStr = "   "
		newStr = fmt.Sprintf("%3d", newLine)
	case git.DiffOpDelete:
		oldStr = fmt.Sprintf("%3d", oldLine)
		newStr = "   "
	default:
		oldStr = fmt.Sprintf("%3d", oldLine)
		newStr = fmt.Sprintf("%3d", newLine)
	}

	return m.tokens.SubtleText.Render(oldStr+" "+newStr) + "  "
}

// writeLine writes a line with optional styling, handling cursor.
func (m Model) writeLine(b *strings.Builder, line string, lineNum int, styleFn func(...string) string) {
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(line))
		return
	}

	if styleFn != nil {
		b.WriteString(styleFn(line))
	} else {
		b.WriteString(line)
	}
	b.WriteString("\n")
}

// renderCursorLine renders a line with cursor styling.
func (m Model) renderCursorLine(line string) string {
	stripped := ansi.Strip(line)
	if len(stripped) == 0 {
		return m.tokens.CursorBlock.Render(" ") + "\n"
	}

	firstRune, size := utf8.DecodeRuneInString(stripped)
	rest := stripped[size:]

	return m.tokens.CursorBlock.Render(string(firstRune)) + m.tokens.Cursor.Render(rest) + "\n"
}

// padRight pads a string to the given width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// padLeft pads a string to the given width with leading spaces.
func padLeft(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(s)) + s
}
