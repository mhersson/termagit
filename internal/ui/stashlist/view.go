package stashlist

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/ui/notification"
	"github.com/mhersson/termagit/internal/ui/shared"
)

// View renders the stash list view.
func (m Model) View() string {
	if m.cursor.Width == 0 || m.cursor.Height == 0 {
		return ""
	}

	var b strings.Builder

	// Header: "Stashes (N)"
	b.WriteString(m.tokens.SectionHeader.Render(fmt.Sprintf("Stashes (%d)", len(m.stashes))))
	b.WriteString("\n\n")

	// Stash entries
	vis := m.cursor.VisibleLines()
	end := m.cursor.Offset + vis
	if end > len(m.stashes) {
		end = len(m.stashes)
	}

	for i := m.cursor.Offset; i < end; i++ {
		isCursor := i == m.cursor.Pos

		row := m.renderStashRow(m.stashes[i])

		if isCursor {
			row = m.tokens.Cursor.Render(ansi.Strip(row))
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	base := shared.PadToHeight(b.String(), m.cursor.Height)

	// Overlay confirmation dialog if pending
	confirmMsg := m.ConfirmMessage()
	if confirmMsg != "" {
		d := notification.ConfirmDialog{Message: confirmMsg}
		confirmView := d.View(m.tokens, m.cursor.Width-4)
		base = notification.CenterOverlay(base, confirmView, m.cursor.Width, m.cursor.Height)
	}

	return base
}

// renderStashRow renders a single stash entry.
// Matches Neogit stash_list_view/ui.lua: "stash@{N}" with Comment highlight, then message.
func (m Model) renderStashRow(entry git.StashEntry) string {
	label := fmt.Sprintf("stash@{%d}", entry.Index)
	return m.tokens.Comment.Render(label) + " " + entry.Message
}
