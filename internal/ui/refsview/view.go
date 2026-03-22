package refsview

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/ui/notification"
)

// View renders the refs view.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	vis := m.visibleLines()
	end := m.offset + vis
	if end > len(m.flatRows) {
		end = len(m.flatRows)
	}

	for i := m.offset; i < end; i++ {
		row := m.flatRows[i]
		isCursor := i == m.cursor

		var line string
		if row.isHeader {
			line = m.renderSectionHeader(row.sectionIdx)
		} else {
			sec := m.sections[row.sectionIdx]
			line = m.renderRefRow(sec.Items[row.itemIdx], sec.Kind)
		}

		if isCursor {
			line = m.tokens.Cursor.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	base := padToHeight(b.String(), m.height)

	// Overlay confirmation dialog if pending
	confirmMsg := m.ConfirmMessage()
	if confirmMsg != "" {
		d := notification.ConfirmDialog{Message: confirmMsg}
		confirmView := d.View(m.tokens, m.width-4)
		base = notification.CenterOverlay(base, confirmView, m.width, m.height)
	}

	return base
}

// renderSectionHeader renders a section title line.
// Matches Neogit refs_view/ui.lua section headings.
func (m Model) renderSectionHeader(sectionIdx int) string {
	sec := m.sections[sectionIdx]
	count := len(sec.Items)

	var parts []string

	switch sec.Kind {
	case RefsSectionLocal:
		// "Branches (N)"
		parts = append(parts, m.tokens.Branch.Render("Branches"))
	case RefsSectionRemote:
		// "Remote origin (url) (N)"
		parts = append(parts, m.tokens.Branch.Render("Remote "))
		parts = append(parts, m.tokens.Remote.Render(sec.RemoteName))
		if sec.RemoteURL != "" {
			parts = append(parts, m.tokens.Branch.Render(fmt.Sprintf(" (%s)", sec.RemoteURL)))
		}
	case RefsSectionTags:
		// "Tags (N)"
		parts = append(parts, m.tokens.Branch.Render("Tags"))
	}

	parts = append(parts, m.tokens.GraphWhite.Render(fmt.Sprintf(" (%d)", count)))

	return strings.Join(parts, "")
}

// renderRefRow renders a single ref entry.
// Matches Neogit refs_view/ui.lua Ref() function.
func (m Model) renderRefRow(ref git.RefEntry, kind RefsSectionKind) string {
	var parts []string

	// HEAD indicator: "@ " for current branch, "  " otherwise
	if ref.Head {
		parts = append(parts, m.tokens.GraphBoldPurple.Render("@ "))
	} else {
		parts = append(parts, "  ")
	}

	// Ref name: truncated to 34 chars, right-padded to 35
	nameStyle := m.nameStyle(kind)
	name := truncateString(ref.Name, 34)
	name = padRight(name, 35)
	parts = append(parts, nameStyle.Render(name))

	// Upstream info (for local branches with upstream)
	if ref.UpstreamName != "" {
		upstreamStyle := m.upstreamStatusStyle(ref.UpstreamStatus)
		parts = append(parts, upstreamStyle.Render(ref.UpstreamName))
		parts = append(parts, " ")
	}

	// Subject
	parts = append(parts, ref.Subject)

	return strings.Join(parts, "")
}

// nameStyle returns the style for a ref name based on its section kind.
func (m Model) nameStyle(kind RefsSectionKind) lipgloss.Style {
	switch kind {
	case RefsSectionLocal:
		return m.tokens.Branch
	case RefsSectionRemote:
		return m.tokens.Remote
	case RefsSectionTags:
		return m.tokens.Tag
	default:
		return m.tokens.Normal
	}
}

// upstreamStatusStyle returns the style for an upstream status indicator.
// Matches Neogit refs_view/ui.lua highlights table.
func (m Model) upstreamStatusStyle(status string) lipgloss.Style {
	switch status {
	case "+":
		return m.tokens.GraphCyan
	case "-":
		return m.tokens.GraphPurple
	case "<>":
		return m.tokens.GraphYellow
	case "=":
		return m.tokens.GraphGreen
	case "<":
		return m.tokens.GraphPurple
	case ">":
		return m.tokens.GraphCyan
	default:
		return m.tokens.GraphRed
	}
}

// truncateString truncates a string to maxRunes runes.
func truncateString(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}

// padRight pads a string with spaces to the given width in runes.
func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// padToHeight ensures the string has at least height lines for overlay placement.
func padToHeight(s string, height int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}
