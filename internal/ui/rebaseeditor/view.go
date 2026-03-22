package rebaseeditor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

// View renders the rebase editor.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if m.loading {
		return "Loading rebase todo..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	var b strings.Builder

	// Top bar
	b.WriteString(m.renderTopBar())
	b.WriteString("\n")

	// Entry list
	for i, entry := range m.entries {
		b.WriteString(m.renderEntry(i, entry))
		b.WriteString("\n")
	}

	// Exec command input prompt
	if m.execActive {
		b.WriteString("\n")
		b.WriteString("  exec command: ")
		b.WriteString(m.execInput.View())
		b.WriteString("\n")
	}

	// Help block
	b.WriteString("\n")
	b.WriteString(m.renderHelpBlock())

	return b.String()
}

// renderTopBar renders the header bar with centered title.
func (m Model) renderTopBar() string {
	title := "Rebase"
	badge := m.tokens.EditorModeNormal.Render(" [NORMAL] ")
	badgeWidth := lipgloss.Width(badge)

	titleWidth := len(title)
	centerPos := (m.width - titleWidth) / 2
	gapAfterBadge := centerPos - badgeWidth
	if gapAfterBadge < 1 {
		gapAfterBadge = 1
	}

	titleStyle := m.tokens.Bold.Background(m.tokens.EditorBar.GetBackground())
	styledTitle := titleStyle.Render(title)

	gap := m.tokens.EditorBar.Render(strings.Repeat(" ", gapAfterBadge))
	rightFill := m.width - badgeWidth - gapAfterBadge - titleWidth
	if rightFill < 0 {
		rightFill = 0
	}
	fill := m.tokens.EditorBar.Render(strings.Repeat(" ", rightFill))

	return badge + gap + styledTitle + fill
}

// renderEntry renders a single rebase todo entry.
func (m Model) renderEntry(idx int, entry git.TodoEntry) string {
	// Cursor prefix
	prefix := "  "
	if idx == m.cursor {
		prefix = "> "
	}

	line := renderEntryLine(entry, m.tokens)
	return prefix + line
}

// renderEntryLine renders the entry content (action + hash + subject) with styling.
func renderEntryLine(entry git.TodoEntry, tokens theme.Tokens) string {
	action := string(entry.Action)

	switch entry.Action {
	case git.TodoBreak:
		return tokens.GraphOrange.Render("break")

	case git.TodoExec:
		padded := fmt.Sprintf("%-7s", action)
		styledAction := tokens.GraphOrange.Render(padded)
		return styledAction + tokens.Normal.Render(entry.Subject)

	case git.TodoDrop:
		// Dropped lines rendered with SubtleText (commented out)
		padded := fmt.Sprintf("%-7s", action)
		line := "# " + padded + entry.AbbrevHash + " " + entry.Subject
		return tokens.SubtleText.Render(line)
	}

	// Standard commit entries: action hash subject
	padded := fmt.Sprintf("%-7s", action)

	var styledAction string
	if entry.Done {
		styledAction = tokens.RebaseDone.Render(padded)
	} else {
		styledAction = tokens.GraphOrange.Render(padded)
	}

	styledHash := tokens.Hash.Render(entry.AbbrevHash)
	styledSubject := tokens.Normal.Render(" " + entry.Subject)

	return styledAction + styledHash + styledSubject
}

// renderHelpBlock renders the help text at the bottom of the editor.
// Key labels are dynamically generated from the configured key bindings.
func (m Model) renderHelpBlock() string {
	keys := m.keys
	comment := "#"

	// Collect all key labels for padding
	labels := []string{
		keys.Pick.Help().Key,
		keys.Reword.Help().Key,
		keys.Edit.Help().Key,
		keys.Squash.Help().Key,
		keys.Fixup.Help().Key,
		keys.Execute.Help().Key,
		keys.Drop.Help().Key,
		keys.Submit.Help().Key,
		keys.Abort.Help().Key,
		keys.MoveUp.Help().Key,
		keys.MoveDown.Help().Key,
		keys.OpenCommit.Help().Key,
	}
	padding := maxLen(labels)

	pad := func(s string) string {
		return fmt.Sprintf("%-*s", padding, s)
	}

	lines := []string{
		fmt.Sprintf("%s Commands:", comment),
		fmt.Sprintf("%s   %s pick   = use commit", comment, pad(keys.Pick.Help().Key)),
		fmt.Sprintf("%s   %s reword = use commit, but edit the commit message", comment, pad(keys.Reword.Help().Key)),
		fmt.Sprintf("%s   %s edit   = use commit, but stop for amending", comment, pad(keys.Edit.Help().Key)),
		fmt.Sprintf("%s   %s squash = use commit, but meld into previous commit", comment, pad(keys.Squash.Help().Key)),
		fmt.Sprintf(`%s   %s fixup  = like "squash", but discard this commit's log message`, comment, pad(keys.Fixup.Help().Key)),
		fmt.Sprintf("%s   %s exec   = run command (the rest of the line) using shell", comment, pad(keys.Execute.Help().Key)),
		fmt.Sprintf("%s   %s drop   = remove commit", comment, pad(keys.Drop.Help().Key)),
		fmt.Sprintf("%s   %s undo last change", comment, pad("u")),
		fmt.Sprintf("%s   %s tell Git to make it happen", comment, pad(keys.Submit.Help().Key)),
		fmt.Sprintf("%s   %s tell Git that you changed your mind, i.e. abort", comment, pad(keys.Abort.Help().Key)),
		fmt.Sprintf("%s   %s move the commit up", comment, pad(keys.MoveUp.Help().Key)),
		fmt.Sprintf("%s   %s move the commit down", comment, pad(keys.MoveDown.Help().Key)),
		fmt.Sprintf("%s   %s show the commit another buffer", comment, pad(keys.OpenCommit.Help().Key)),
		comment,
		fmt.Sprintf("%s These lines can be re-ordered; they are executed from top to bottom.", comment),
		comment,
		fmt.Sprintf("%s If you remove a line here THAT COMMIT WILL BE LOST.", comment),
		comment,
		fmt.Sprintf("%s However, if you remove everything, the rebase will be aborted.", comment),
		comment,
	}

	var b strings.Builder
	for _, line := range lines {
		b.WriteString(m.tokens.SubtleText.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}

// maxLen returns the maximum length of strings in the slice.
func maxLen(ss []string) int {
	max := 0
	for _, s := range ss {
		if len(s) > max {
			max = len(s)
		}
	}
	return max
}
