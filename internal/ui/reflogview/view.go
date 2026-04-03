package reflogview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mhersson/termagit/internal/git"
)

// View renders the reflog view.
func (m Model) View() string {
	if m.cursor.Width == 0 || m.cursor.Height == 0 {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(m.tokens.SectionHeader.Render(m.header))
	b.WriteString("\n\n")

	// Entries list
	vis := m.cursor.VisibleLines()
	end := m.cursor.Offset + vis
	if end > len(m.entries) {
		end = len(m.entries)
	}

	// Calculate max index width for alignment
	maxIdx := len(m.entries) - 1
	idxWidth := len(fmt.Sprintf("%d", maxIdx)) + 1

	for i := m.cursor.Offset; i < end; i++ {
		e := m.entries[i]
		isCursor := i == m.cursor.Pos

		row := m.renderEntry(e, idxWidth)

		if isCursor {
			row = m.tokens.Cursor.Render(ansi.Strip(row))
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderEntry(e git.ReflogEntry, idxWidth int) string {
	var parts []string

	// Hash (7 chars)
	hash := e.Oid
	if len(hash) > 7 {
		hash = hash[:7]
	}
	parts = append(parts, m.tokens.Hash.Render(hash))

	// Index (right-aligned)
	idxStr := fmt.Sprintf("%*d", idxWidth, e.Index)
	parts = append(parts, " ")
	parts = append(parts, idxStr)

	// Type (right-aligned to 16 chars, colored by type)
	typeStyle := m.styleForType(e.Type)
	typeStr := fmt.Sprintf("%16s", e.Type)
	parts = append(parts, " ")
	parts = append(parts, typeStyle.Render(typeStr))

	// Subject (remove the type prefix since we show it separately)
	subject := extractSubject(e.RefSubject)
	parts = append(parts, " ")
	parts = append(parts, subject)

	// Date (right-aligned) - shown at end
	date := formatRelDate(e.RelDate)
	parts = append(parts, "  ")
	parts = append(parts, m.tokens.CommitDate.Render(date))

	return strings.Join(parts, "")
}

// styleForType returns the style for a reflog entry type.
// Matches Neogit's reflog_view/ui.lua lines 12-26.
func (m Model) styleForType(typ string) lipgloss.Style {
	switch typ {
	case "commit", "merge":
		return m.tokens.GraphGreen
	case "reset":
		return m.tokens.GraphRed
	case "checkout", "branch":
		return m.tokens.GraphBlue
	case "cherry-pick", "revert":
		return m.tokens.GraphYellow
	case "amend":
		return m.tokens.GraphPurple
	default:
		// Includes: rebase, pull, clone, other
		if strings.HasPrefix(typ, "rebase") {
			return m.tokens.GraphPurple
		}
		return m.tokens.GraphCyan
	}
}

// extractSubject removes the type prefix from a reflog subject.
// e.g., "commit: add feature" -> "add feature"
func extractSubject(subject string) string {
	colonIdx := strings.Index(subject, ": ")
	if colonIdx >= 0 {
		return strings.TrimSpace(subject[colonIdx+2:])
	}
	return subject
}

// formatRelDate formats a relative date for display.
// e.g., "3 hours ago" -> "3h"
func formatRelDate(relDate string) string {
	// Keep short for now
	parts := strings.Split(relDate, " ")
	if len(parts) >= 2 {
		num := parts[0]
		unit := parts[1]
		switch {
		case strings.HasPrefix(unit, "second"):
			return num + "s"
		case strings.HasPrefix(unit, "minute"):
			return num + "m"
		case strings.HasPrefix(unit, "hour"):
			return num + "h"
		case strings.HasPrefix(unit, "day"):
			return num + "d"
		case strings.HasPrefix(unit, "week"):
			return num + "w"
		case strings.HasPrefix(unit, "month"):
			return num + "mo"
		case strings.HasPrefix(unit, "year"):
			return num + "y"
		}
	}
	return relDate
}
