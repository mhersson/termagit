package logview

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/graph"
)

// View renders the log view.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Check for commit view overlay first
	if m.commitView != nil {
		return m.renderCommitViewOverlay()
	}

	return m.renderLogContent()
}

// renderLogContent renders the log view content.
func (m Model) renderLogContent() string {
	var b strings.Builder
	linesUsed := 0

	// Header
	b.WriteString(m.tokens.SectionHeader.Render(m.header))
	b.WriteString("\n\n")
	linesUsed += 2

	// Filter input (if active)
	if m.filterActive {
		b.WriteString("Filter: ")
		b.WriteString(m.filterInput.View())
		b.WriteString("\n\n")
		linesUsed += 2
	} else if m.filterText != "" {
		b.WriteString(m.tokens.SubtleText.Render(fmt.Sprintf("Filtered: %q (%d matches)", m.filterText, len(m.filtered))))
		b.WriteString("\n\n")
		linesUsed += 2
	}

	// Reserve space for load more hint
	reservedLines := 0
	if m.hasMore {
		reservedLines = 3 // blank + hint + blank
	}

	maxLines := m.height - reservedLines

	if m.graphEnabled && len(m.displayRows) > 0 {
		// Graph mode: iterate displayRows
		rowCount := len(m.displayRows)
		for i := m.offset; i < rowCount && linesUsed < maxLines; i++ {
			dr := m.displayRows[i]
			if dr.commitIdx >= 0 && dr.commitIdx < len(m.commits) {
				// Commit row
				c := m.commits[dr.commitIdx]
				isCursor := i == m.cursor
				row := m.renderCommitRow(c, isCursor, dr.graphCells)
				b.WriteString(row)
				b.WriteString("\n")
				linesUsed++

				// Show expanded details if there's room
				if dr.commitIdx < len(m.expanded) && m.expanded[dr.commitIdx] {
					details := m.renderCommitDetails(c)
					detailLines := strings.Count(details, "\n")
					if linesUsed+detailLines <= maxLines {
						b.WriteString(details)
						linesUsed += detailLines
					}
				}
			} else {
				// Graph-only connector row
				row := m.renderGraphOnlyRow(dr.graphCells)
				b.WriteString(row)
				b.WriteString("\n")
				linesUsed++
			}
		}
	} else {
		// Non-graph mode: iterate commits directly
		commitCount := len(m.commits)
		if len(m.filtered) > 0 {
			commitCount = len(m.filtered)
		}

		for i := m.offset; i < commitCount && linesUsed < maxLines; i++ {
			idx := i
			if len(m.filtered) > 0 && i < len(m.filtered) {
				idx = m.filtered[i]
			}

			if idx >= len(m.commits) {
				continue
			}

			c := m.commits[idx]
			isCursor := i == m.cursor

			row := m.renderCommitRow(c, isCursor, nil)
			b.WriteString(row)
			b.WriteString("\n")
			linesUsed++

			// Show expanded details if there's room
			if idx < len(m.expanded) && m.expanded[idx] {
				details := m.renderCommitDetails(c)
				detailLines := strings.Count(details, "\n")
				if linesUsed+detailLines <= maxLines {
					b.WriteString(details)
					linesUsed += detailLines
				}
			}
		}
	}

	// Load more hint
	if m.hasMore {
		b.WriteString("\n")
		hint := m.tokens.GraphBlue.Bold(true).Render("Type") +
			m.tokens.GraphCyan.Bold(true).Render(" + ") +
			m.tokens.GraphBlue.Bold(true).Render("to show more history")
		b.WriteString(hint)
		b.WriteString("\n")
	}

	return b.String()
}

// renderCommitViewOverlay renders the commit view in the lower portion of the terminal.
func (m Model) renderCommitViewOverlay() string {
	cvContent := m.commitView.View()
	logContent := m.renderLogContent()

	// Split content into lines
	logLines := strings.Split(logContent, "\n")
	cvLines := strings.Split(cvContent, "\n")

	// Commit view gets 70% of screen height
	cvHeight := m.height * 70 / 100
	maxLogLines := m.height - cvHeight
	if maxLogLines < 0 {
		maxLogLines = 0
	}

	var b strings.Builder

	// Render log lines that appear above the commit view
	for i := 0; i < maxLogLines && i < len(logLines); i++ {
		b.WriteString(logLines[i])
		b.WriteString("\n")
	}

	// Render commit view content
	for i := 0; i < cvHeight; i++ {
		if i < len(cvLines) {
			b.WriteString(cvLines[i])
		}
		if i < cvHeight-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderCommitRow(c git.LogEntry, isCursor bool, graphCells graph.Row) string {
	// Format: hash [graph] refs subject          author    time
	// Author/time are always at fixed positions from right edge

	// Fixed widths
	hashWidth := 8        // 7 chars + space
	authorColWidth := 16  // author column
	timeColWidth := 12    // time column (e.g., "10 hours", "2 days")
	minSubjectWidth := 15

	// Hash (7 chars)
	hash := c.AbbreviatedHash
	if len(hash) > 7 {
		hash = hash[:7]
	}

	// Author (truncate to fit column using rune count for proper UTF-8 handling)
	author := c.AuthorName
	if utf8.RuneCountInString(author) > authorColWidth {
		author = string([]rune(author)[:authorColWidth])
	}

	// Relative time (fuller format like Neogit)
	relTime := formatRelativeTimeFull(c.When)

	// Graph column
	graphStr := ""
	graphVisualWidth := 0
	if len(graphCells) > 0 {
		graphStr = m.renderGraphCells(graphCells)
		graphVisualWidth = graphCellWidth(graphCells) + 1 // +1 for space after graph
	}

	// Calculate space available for refs + subject
	rightSideWidth := authorColWidth + timeColWidth
	middleWidth := m.width - hashWidth - graphVisualWidth - rightSideWidth

	if middleWidth < minSubjectWidth {
		middleWidth = minSubjectWidth
	}

	// Refs (branches, tags) - no parentheses, just space-separated
	refs := ""
	if len(c.Refs) > 0 {
		refs = m.renderRefsFlat(c.Refs)
	}

	// Calculate subject width
	refsWidth := 0
	if refs != "" {
		refsWidth = len(stripAnsi(refs)) + 1 // +1 for space after refs
	}
	subjectWidth := middleWidth - refsWidth
	if subjectWidth < minSubjectWidth {
		subjectWidth = minSubjectWidth
	}

	// Truncate subject if needed
	subject := c.Subject
	if len(subject) > subjectWidth {
		subject = subject[:subjectWidth-1] // leave room, no ellipsis like neogit
	}

	// Build the row with proper alignment
	var row string
	if m.width > 60 {
		// Wide terminal: show full format
		graphPart := ""
		if graphStr != "" {
			graphPart = graphStr + " "
		}
		refsStr := ""
		if refs != "" {
			refsStr = refs + " "
		}
		row = fmt.Sprintf("%s %s%s%s %s %s",
			m.tokens.Hash.Render(hash),
			graphPart,
			refsStr,
			padRight(subject, subjectWidth),
			m.tokens.CommitAuthor.Render(padRight(author, authorColWidth)),
			m.tokens.CommitDate.Render(padRight(relTime, timeColWidth)),
		)
	} else {
		// Narrow terminal: simplified format
		row = fmt.Sprintf("%s %s",
			m.tokens.Hash.Render(hash),
			subject,
		)
	}

	if isCursor {
		row = m.tokens.Cursor.Render(row)
	}

	return row
}

// renderGraphCells renders graph cells with appropriate styling.
func (m Model) renderGraphCells(cells graph.Row) string {
	var b strings.Builder
	for _, cell := range cells {
		b.WriteString(m.graphColorStyle(cell.Color).Render(cell.Text))
	}
	return b.String()
}

// renderGraphOnlyRow renders a graph-only connector row (no commit data).
func (m Model) renderGraphOnlyRow(cells graph.Row) string {
	// Left-pad with hashWidth spaces to align with commit rows
	padding := strings.Repeat(" ", 8) // hashWidth = 8
	return padding + m.renderGraphCells(cells)
}

// graphCellWidth returns the visual width of graph cells.
func graphCellWidth(cells graph.Row) int {
	w := 0
	for _, c := range cells {
		w += utf8.RuneCountInString(c.Text)
	}
	return w
}

// graphColorStyle returns the lipgloss style for a graph color name.
func (m Model) graphColorStyle(color string) lipgloss.Style {
	switch color {
	case "Red":
		return m.tokens.GraphRed
	case "Green":
		return m.tokens.GraphGreen
	case "Blue":
		return m.tokens.GraphBlue
	case "Yellow":
		return m.tokens.GraphYellow
	case "Cyan":
		return m.tokens.GraphCyan
	case "Purple":
		return m.tokens.GraphPurple
	case "Gray":
		return m.tokens.GraphGray
	case "White":
		return m.tokens.GraphWhite
	case "Orange":
		return m.tokens.GraphOrange
	default:
		return m.tokens.GraphPurple
	}
}

// renderRefsFlat renders refs without parentheses, space-separated.
func (m Model) renderRefsFlat(refs []git.Ref) string {
	var parts []string

	for _, ref := range refs {
		switch ref.Kind {
		case git.RefKindHead:
			continue // Skip HEAD
		case git.RefKindLocal:
			parts = append(parts, m.tokens.Branch.Render(ref.Name))
		case git.RefKindRemote:
			remoteName := ref.Remote + "/" + ref.Name
			parts = append(parts, m.tokens.Remote.Render(remoteName))
		case git.RefKindTag:
			parts = append(parts, m.tokens.Tag.Render(ref.Name))
		}
	}

	return strings.Join(parts, " ")
}

// stripAnsi removes ANSI escape codes to get actual string length.
func stripAnsi(s string) string {
	// Simple approach: count visible characters
	// ANSI codes are \x1b[...m
	result := make([]byte, 0, len(s))
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}

// padRight pads a string with spaces on the right to reach the target width.
// Uses rune count for proper UTF-8 handling (e.g., æ, ø, å are 1 visual char).
func padRight(s string, width int) string {
	runeCount := utf8.RuneCountInString(s)
	if runeCount >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runeCount)
}

// formatRelativeTimeFull formats a time as a full relative duration (e.g., "10 hours", "2 days").
func formatRelativeTimeFull(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	now := time.Now()
	diff := now.Sub(t)

	pluralize := func(n int, unit string) string {
		if n == 1 {
			return fmt.Sprintf("%d %s", n, unit)
		}
		return fmt.Sprintf("%d %ss", n, unit)
	}

	switch {
	case diff < time.Minute:
		secs := int(diff.Seconds())
		return pluralize(secs, "second")
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return pluralize(mins, "minute")
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return pluralize(hours, "hour")
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return pluralize(days, "day")
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		return pluralize(weeks, "week")
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		return pluralize(months, "month")
	default:
		years := int(diff.Hours() / 24 / 365)
		return pluralize(years, "year")
	}
}

func (m Model) renderCommitDetails(c git.LogEntry) string {
	var b strings.Builder
	indent := "        " // 8 spaces (aligned with hash width + space)

	// Author
	b.WriteString(indent)
	b.WriteString(m.tokens.SubtleText.Render("Author:     "))
	b.WriteString(c.AuthorName)
	if c.AuthorEmail != "" {
		b.WriteString(" <")
		b.WriteString(c.AuthorEmail)
		b.WriteString(">")
	}
	b.WriteString("\n")

	// AuthorDate
	if c.AuthorDate != "" {
		b.WriteString(indent)
		b.WriteString(m.tokens.SubtleText.Render("AuthorDate: "))
		b.WriteString(c.AuthorDate)
		b.WriteString("\n")
	}

	// Committer
	if c.CommitterName != "" {
		b.WriteString(indent)
		b.WriteString(m.tokens.SubtleText.Render("Commit:     "))
		b.WriteString(c.CommitterName)
		if c.CommitterEmail != "" {
			b.WriteString(" <")
			b.WriteString(c.CommitterEmail)
			b.WriteString(">")
		}
		b.WriteString("\n")
	}

	// CommitterDate
	if c.CommitterDate != "" {
		b.WriteString(indent)
		b.WriteString(m.tokens.SubtleText.Render("CommitDate: "))
		b.WriteString(c.CommitterDate)
		b.WriteString("\n")
	}

	// Body
	if c.Body != "" {
		b.WriteString("\n")
		for _, line := range strings.Split(c.Body, "\n") {
			b.WriteString(indent)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}
