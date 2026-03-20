package commitview

import (
	"fmt"
	"strings"

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

	return m.viewport.View()
}

// renderContent builds the full commit view content for the viewport.
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
