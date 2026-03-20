package commitview

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/mhersson/conjit/internal/git"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.loading {
		return "Loading commit " + m.commitID + "..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	if !m.ready || m.info == nil {
		return "No commit data"
	}

	// Render content with cursor highlighting
	content := m.renderContentWithCursor()

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

	return strings.Join(lines[startLine:endLine], "\n")
}

// renderContentWithCursor builds the commit view content with cursor highlighting.
func (m Model) renderContentWithCursor() string {
	var b strings.Builder
	lineNum := 0

	// Header line: "Commit <full hash>" with highlight
	headerLine := "Commit " + m.info.Hash
	// Pad to full width for background highlight
	if len(headerLine) < m.width {
		headerLine += strings.Repeat(" ", m.width-len(headerLine))
	}
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(headerLine))
	} else {
		b.WriteString(m.tokens.CommitViewHeader.Render(headerLine))
		b.WriteString("\n")
	}
	lineNum++

	// Author info
	authorLine := m.tokens.SubtleText.Render("Author:     ") + m.info.AuthorName + " <" + m.info.AuthorEmail + ">"
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(authorLine))
	} else {
		b.WriteString(authorLine)
		b.WriteString("\n")
	}
	lineNum++

	// Author date
	dateLine := m.tokens.SubtleText.Render("AuthorDate: ") + m.info.AuthorDate
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(dateLine))
	} else {
		b.WriteString(dateLine)
		b.WriteString("\n")
	}
	lineNum++

	// Committer info (if different from author)
	if m.info.CommitterName != "" && m.info.CommitterName != m.info.AuthorName {
		committerLine := m.tokens.SubtleText.Render("Committer:  ") + m.info.CommitterName
		if m.info.CommitterEmail != "" {
			committerLine += " <" + m.info.CommitterEmail + ">"
		}
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(committerLine))
		} else {
			b.WriteString(committerLine)
			b.WriteString("\n")
		}
		lineNum++

		commitDateLine := m.tokens.SubtleText.Render("CommitDate: ") + m.info.CommitterDate
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(commitDateLine))
		} else {
			b.WriteString(commitDateLine)
			b.WriteString("\n")
		}
		lineNum++
	}

	// Blank line before subject/body
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(""))
	} else {
		b.WriteString("\n")
	}
	lineNum++

	// Commit subject (first line of message)
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(m.info.Subject))
	} else {
		b.WriteString(m.info.Subject)
		b.WriteString("\n")
	}
	lineNum++

	// Commit body (if any)
	if m.info.Body != "" {
		// Blank line
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(""))
		} else {
			b.WriteString("\n")
		}
		lineNum++

		// Body lines
		bodyLines := strings.Split(m.info.Body, "\n")
		for _, bodyLine := range bodyLines {
			if lineNum == m.cursorLine {
				b.WriteString(m.renderCursorLine(bodyLine))
			} else {
				b.WriteString(bodyLine)
				b.WriteString("\n")
			}
			lineNum++
		}
	}

	// File overview
	if m.overview != nil && len(m.overview.Files) > 0 {
		// Blank line
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(""))
		} else {
			b.WriteString("\n")
		}
		lineNum++

		// Summary line
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(m.overview.Summary))
		} else {
			b.WriteString(m.overview.Summary)
			b.WriteString("\n")
		}
		lineNum++

		for _, f := range m.overview.Files {
			var fileLine strings.Builder
			fileLine.WriteString(m.tokens.FilePath.Render(f.Path))
			fileLine.WriteString(" | ")
			fileLine.WriteString(m.tokens.Number.Render(f.Changes))
			if f.IsBinary {
				fileLine.WriteString(" (binary)")
			} else {
				fileLine.WriteString(" ")
				fileLine.WriteString(m.tokens.DiffAdd.Render(f.Insertions))
				fileLine.WriteString(m.tokens.DiffDelete.Render(f.Deletions))
			}

			if lineNum == m.cursorLine {
				b.WriteString(m.renderCursorLine(fileLine.String()))
			} else {
				b.WriteString(fileLine.String())
				b.WriteString("\n")
			}
			lineNum++
		}
	}

	// Diffs
	if len(m.diffs) > 0 {
		// Blank line
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(""))
		} else {
			b.WriteString("\n")
		}
		lineNum++

		for _, diff := range m.diffs {
			lineNum = m.renderFileDiffWithCursor(&b, &diff, lineNum)
		}
	}

	return b.String()
}

// renderCursorLine renders a line with cursor styling (block cursor on first char).
func (m Model) renderCursorLine(line string) string {
	if len(line) == 0 {
		return m.tokens.CursorBlock.Render(" ") + "\n"
	}

	// Get first rune (handles multi-byte UTF-8)
	firstRune, size := utf8.DecodeRuneInString(line)
	rest := line[size:]

	// First character: reverse video, rest: cursor line background
	return m.tokens.CursorBlock.Render(string(firstRune)) + m.tokens.Cursor.Render(rest) + "\n"
}

// renderFileDiffWithCursor renders a single file diff with cursor tracking.
func (m Model) renderFileDiffWithCursor(b *strings.Builder, diff *git.FileDiff, lineNum int) int {
	// File header
	modeText := "modified"
	if diff.IsNew {
		modeText = "new file"
	} else if diff.IsDelete {
		modeText = "deleted"
	} else if diff.OldPath != "" {
		modeText = "renamed"
	}

	fileHeader := modeText + " " + diff.Path
	if lineNum == m.cursorLine {
		b.WriteString(m.renderCursorLine(fileHeader))
	} else {
		b.WriteString(m.tokens.DiffHunkHeader.Render(fileHeader))
		b.WriteString("\n")
	}
	lineNum++

	// Hunks
	for _, hunk := range diff.Hunks {
		// Hunk header
		if lineNum == m.cursorLine {
			b.WriteString(m.renderCursorLine(hunk.Header))
		} else {
			b.WriteString(m.tokens.DiffHunkHeader.Render(hunk.Header))
			b.WriteString("\n")
		}
		lineNum++

		// Hunk lines
		for _, line := range hunk.Lines {
			var styledLine string
			switch line.Op {
			case git.DiffOpAdd:
				styledLine = m.tokens.DiffAdd.Render("+" + line.Content)
			case git.DiffOpDelete:
				styledLine = m.tokens.DiffDelete.Render("-" + line.Content)
			default:
				styledLine = m.tokens.DiffContext.Render(" " + line.Content)
			}

			if lineNum == m.cursorLine {
				// For cursor line, render the raw text with cursor styling
				var rawLine string
				switch line.Op {
				case git.DiffOpAdd:
					rawLine = "+" + line.Content
				case git.DiffOpDelete:
					rawLine = "-" + line.Content
				default:
					rawLine = " " + line.Content
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

// renderContent builds the full commit view content for the viewport (without cursor).
func (m Model) renderContent() string {
	var b strings.Builder

	// Header line: "Commit <full hash>" with highlight
	headerLine := "Commit " + m.info.Hash
	// Pad to full width for background highlight
	if len(headerLine) < m.width {
		headerLine += strings.Repeat(" ", m.width-len(headerLine))
	}
	b.WriteString(m.tokens.CommitViewHeader.Render(headerLine))
	b.WriteString("\n")

	// Author info
	b.WriteString(m.tokens.SubtleText.Render("Author:     "))
	b.WriteString(m.info.AuthorName)
	b.WriteString(" <")
	b.WriteString(m.info.AuthorEmail)
	b.WriteString(">\n")

	// Author date
	b.WriteString(m.tokens.SubtleText.Render("AuthorDate: "))
	b.WriteString(m.info.AuthorDate)
	b.WriteString("\n")

	// Committer info (if different from author)
	if m.info.CommitterName != "" && m.info.CommitterName != m.info.AuthorName {
		b.WriteString(m.tokens.SubtleText.Render("Committer:  "))
		b.WriteString(m.info.CommitterName)
		if m.info.CommitterEmail != "" {
			b.WriteString(" <")
			b.WriteString(m.info.CommitterEmail)
			b.WriteString(">")
		}
		b.WriteString("\n")

		b.WriteString(m.tokens.SubtleText.Render("CommitDate: "))
		b.WriteString(m.info.CommitterDate)
		b.WriteString("\n")
	}

	// Blank line before subject/body
	b.WriteString("\n")

	// Commit subject (first line of message)
	b.WriteString(m.info.Subject)
	b.WriteString("\n")

	// Commit body (if any)
	if m.info.Body != "" {
		b.WriteString("\n")
		b.WriteString(m.info.Body)
		b.WriteString("\n")
	}

	// File overview
	if m.overview != nil && len(m.overview.Files) > 0 {
		b.WriteString("\n")
		b.WriteString(m.overview.Summary)
		b.WriteString("\n")

		for _, f := range m.overview.Files {
			b.WriteString(m.tokens.FilePath.Render(f.Path))
			b.WriteString(" | ")
			b.WriteString(m.tokens.Number.Render(f.Changes))
			if f.IsBinary {
				b.WriteString(" (binary)")
			} else {
				b.WriteString(" ")
				b.WriteString(m.tokens.DiffAdd.Render(f.Insertions))
				b.WriteString(m.tokens.DiffDelete.Render(f.Deletions))
			}
			b.WriteString("\n")
		}
	}

	// Diffs
	if len(m.diffs) > 0 {
		b.WriteString("\n")
		for _, diff := range m.diffs {
			m.renderFileDiff(&b, &diff)
		}
	}

	return b.String()
}

// renderFileDiff renders a single file diff.
func (m Model) renderFileDiff(b *strings.Builder, diff *git.FileDiff) {
	// File header
	modeText := "modified"
	if diff.IsNew {
		modeText = "new file"
	} else if diff.IsDelete {
		modeText = "deleted"
	} else if diff.OldPath != "" {
		modeText = "renamed"
	}

	b.WriteString(m.tokens.DiffHunkHeader.Render(modeText + " " + diff.Path))
	b.WriteString("\n")

	// Hunks
	for _, hunk := range diff.Hunks {
		// Hunk header
		b.WriteString(m.tokens.DiffHunkHeader.Render(hunk.Header))
		b.WriteString("\n")

		// Hunk lines
		for _, line := range hunk.Lines {
			switch line.Op {
			case git.DiffOpAdd:
				b.WriteString(m.tokens.DiffAdd.Render("+" + line.Content))
			case git.DiffOpDelete:
				b.WriteString(m.tokens.DiffDelete.Render("-" + line.Content))
			default:
				b.WriteString(m.tokens.DiffContext.Render(" " + line.Content))
			}
			b.WriteString("\n")
		}
	}
}
