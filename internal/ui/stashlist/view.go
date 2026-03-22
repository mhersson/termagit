package stashlist

import (
	"fmt"
	"strings"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/ui/notification"
)

// View renders the stash list view.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	// Header: "Stashes (N)"
	b.WriteString(m.tokens.SectionHeader.Render(fmt.Sprintf("Stashes (%d)", len(m.stashes))))
	b.WriteString("\n\n")

	// Stash entries
	vis := m.visibleLines()
	end := m.offset + vis
	if end > len(m.stashes) {
		end = len(m.stashes)
	}

	for i := m.offset; i < end; i++ {
		isCursor := i == m.cursor

		row := m.renderStashRow(m.stashes[i])

		if isCursor {
			row = m.tokens.Cursor.Render(row)
		}

		b.WriteString(row)
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

// renderStashRow renders a single stash entry.
// Matches Neogit stash_list_view/ui.lua: "stash@{N}" with Comment highlight, then message.
func (m Model) renderStashRow(entry git.StashEntry) string {
	label := fmt.Sprintf("stash@{%d}", entry.Index)
	return m.tokens.Comment.Render(label) + " " + entry.Message
}

// padToHeight ensures the string has at least height lines for overlay placement.
func padToHeight(s string, height int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}
