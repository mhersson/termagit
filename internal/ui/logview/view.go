package logview

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mhersson/conjit/internal/git"
)

// View renders the log view.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

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
	commitCount := len(m.commits)
	if len(m.filtered) > 0 {
		commitCount = len(m.filtered)
	}

	// Render commits from offset
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

		row := m.renderCommitRow(c, isCursor)
		b.WriteString(row)
		b.WriteString("\n")
		linesUsed++

		// Show expanded details if there's room
		if idx < len(m.expanded) && m.expanded[idx] {
			details := m.renderCommitDetails(c)
			detailLines := strings.Count(details, "\n")

			// Only show details if they fit
			if linesUsed+detailLines <= maxLines {
				b.WriteString(details)
				linesUsed += detailLines
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

func (m Model) renderCommitRow(c git.LogEntry, isCursor bool) string {
	// Format: hash refs subject          author    time
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

	// Calculate space available for refs + subject
	rightSideWidth := authorColWidth + timeColWidth
	middleWidth := m.width - hashWidth - rightSideWidth

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
		refsStr := ""
		if refs != "" {
			refsStr = refs + " "
		}
		row = fmt.Sprintf("%s %s%s %s %s",
			m.tokens.Hash.Render(hash),
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
