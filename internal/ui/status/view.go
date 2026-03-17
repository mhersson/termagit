package status

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/conjit/internal/git"
)

// view renders the status buffer.
func view(m Model) string {
	if m.loading {
		return "Loading..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	var b strings.Builder

	// Render HEAD bar
	b.WriteString(renderHeadBar(m))
	b.WriteString("\n\n")

	// Render sections
	for i, s := range m.sections {
		if s.Hidden {
			continue
		}
		b.WriteString(renderSection(m, i, &s))
	}

	// Render notification if present
	if m.notification != "" {
		b.WriteString("\n")
		b.WriteString(m.notification)
	}

	return b.String()
}

// renderHeadBar renders the HEAD information bar.
func renderHeadBar(m Model) string {
	var b strings.Builder

	// Head line
	headLabel := padRight("Head:", 10)
	b.WriteString(m.tokens.Bold.Render(headLabel))

	if m.head.AbbrevOid != "" {
		b.WriteString(m.tokens.Hash.Render(m.head.AbbrevOid))
		b.WriteString(" ")
	}

	if m.head.Detached {
		b.WriteString(m.tokens.Branch.Render("(detached)"))
	} else {
		b.WriteString(m.tokens.Branch.Render(m.head.Branch))
	}

	if m.head.Subject != "" {
		b.WriteString("  ")
		b.WriteString(m.head.Subject)
	}

	// Merge line (if applicable)
	if m.head.UpstreamBranch != "" {
		b.WriteString("\n")
		mergeLabel := padRight("Merge:", 10)
		b.WriteString(m.tokens.Bold.Render(mergeLabel))

		if m.head.UpstreamOid != "" {
			b.WriteString(m.tokens.Hash.Render(abbreviateOID(m.head.UpstreamOid)))
			b.WriteString(" ")
		}

		remoteBranch := m.head.UpstreamRemote + "/" + m.head.UpstreamBranch
		b.WriteString(m.tokens.Remote.Render(remoteBranch))

		if m.head.UpstreamSubject != "" {
			b.WriteString("  ")
			b.WriteString(m.head.UpstreamSubject)
		}
	}

	// Push line (if applicable)
	if m.head.PushBranch != "" {
		b.WriteString("\n")
		pushLabel := padRight("Push:", 10)
		b.WriteString(m.tokens.Bold.Render(pushLabel))

		if m.head.PushOid != "" {
			b.WriteString(m.tokens.Hash.Render(abbreviateOID(m.head.PushOid)))
			b.WriteString(" ")
		}

		remoteBranch := m.head.PushRemote + "/" + m.head.PushBranch
		b.WriteString(m.tokens.Remote.Render(remoteBranch))

		if m.head.PushSubject != "" {
			b.WriteString("  ")
			b.WriteString(m.head.PushSubject)
		}
	}

	// Tag line (if applicable)
	if m.head.Tag != "" {
		b.WriteString("\n")
		tagLabel := padRight("Tag:", 10)
		b.WriteString(m.tokens.Bold.Render(tagLabel))
		b.WriteString(m.tokens.Tag.Render(m.head.Tag))

		if m.head.TagDistance > 0 {
			fmt.Fprintf(&b, " (%d)", m.head.TagDistance)
		}
	}

	return b.String()
}

// renderSection renders a single section.
func renderSection(m Model, sectionIdx int, s *Section) string {
	var b strings.Builder

	// Section header
	onHeader := m.cursor.Section == sectionIdx && m.cursor.Item == -1
	sign := ">"
	if !s.Folded {
		sign = "v"
	}

	header := fmt.Sprintf("%s %s (%d)", sign, s.Title, len(s.Items))
	if onHeader {
		b.WriteString(m.tokens.Cursor.Render(header))
	} else {
		b.WriteString(m.tokens.SectionHeader.Render(header))
	}
	b.WriteString("\n")

	// Items (only if not folded)
	if !s.Folded {
		for i, item := range s.Items {
			b.WriteString(renderItem(m, sectionIdx, i, &item, s.Kind))
		}
	}

	return b.String() + "\n"
}

// renderItem renders a single item (file entry).
func renderItem(m Model, sectionIdx, itemIdx int, item *Item, sectionKind SectionKind) string {
	var b strings.Builder

	onItem := m.cursor.Section == sectionIdx && m.cursor.Item == itemIdx && m.cursor.Hunk == -1

	// File entry
	if item.Entry != nil {
		modeText := getModeText(item.Entry, sectionKind)
		path := item.Entry.Path()

		// Item sign
		sign := ">"
		if item.Expanded {
			sign = "v"
		}

		line := fmt.Sprintf("  %s %s %s", sign, padRight(modeText, 12), path)

		if onItem {
			b.WriteString(m.tokens.Cursor.Render(line))
		} else {
			// Apply change style based on mode
			b.WriteString(styleForMode(m.tokens, item.Entry, sectionKind).Render(line))
		}
		b.WriteString("\n")

		// Inline diff (if expanded)
		if item.Expanded && len(item.Hunks) > 0 {
			for hunkIdx, hunk := range item.Hunks {
				isFolded := len(item.HunksFolded) > hunkIdx && item.HunksFolded[hunkIdx]
				b.WriteString(renderHunk(m, sectionIdx, itemIdx, hunkIdx, &hunk, isFolded))
			}
		} else if item.Expanded && item.HunksLoading {
			b.WriteString("      Loading diff...\n")
		}
	}

	// Stash entry
	if item.Stash != nil {
		line := fmt.Sprintf("  %s: %s", item.Stash.Name, item.Stash.Message)
		if onItem {
			b.WriteString(m.tokens.Cursor.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Commit entry
	if item.Commit != nil {
		line := fmt.Sprintf("  %s %s", item.Commit.AbbreviatedHash, item.Commit.Subject)
		if onItem {
			b.WriteString(m.tokens.Cursor.Render(line))
		} else {
			b.WriteString(m.tokens.Hash.Render(item.Commit.AbbreviatedHash))
			b.WriteString(" ")
			b.WriteString(item.Commit.Subject)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderHunk renders a diff hunk.
func renderHunk(m Model, sectionIdx, itemIdx, hunkIdx int, hunk *git.Hunk, folded bool) string {
	var b strings.Builder

	onHunk := m.cursor.Section == sectionIdx &&
		m.cursor.Item == itemIdx &&
		m.cursor.Hunk == hunkIdx &&
		m.cursor.Line == -1 // Only on hunk header if Line == -1

	// Hunk header with fold indicator
	sign := "v"
	if folded {
		sign = ">"
	}
	header := "    " + sign + " " + hunk.Header
	if onHunk {
		b.WriteString(m.tokens.Cursor.Render(header))
	} else {
		b.WriteString(m.tokens.DiffHunkHeader.Render(header))
	}
	b.WriteString("\n")

	// Diff lines (only if not folded)
	if !folded {
		for lineIdx, line := range hunk.Lines {
			onLine := m.cursor.Section == sectionIdx &&
				m.cursor.Item == itemIdx &&
				m.cursor.Hunk == hunkIdx &&
				m.cursor.Line == lineIdx

			var lineStr string
			switch line.Op {
			case git.DiffOpAdd:
				lineStr = "      +" + line.Content
				if onLine {
					b.WriteString(m.tokens.Cursor.Render(lineStr))
				} else {
					b.WriteString(m.tokens.DiffAdd.Render(lineStr))
				}
			case git.DiffOpDelete:
				lineStr = "      -" + line.Content
				if onLine {
					b.WriteString(m.tokens.Cursor.Render(lineStr))
				} else {
					b.WriteString(m.tokens.DiffDelete.Render(lineStr))
				}
			case git.DiffOpContext:
				lineStr = "       " + line.Content
				if onLine {
					b.WriteString(m.tokens.Cursor.Render(lineStr))
				} else {
					b.WriteString(m.tokens.DiffContext.Render(lineStr))
				}
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// getModeText returns the mode text for a file entry.
func getModeText(entry *git.StatusEntry, sectionKind SectionKind) string {
	var status git.FileStatus
	switch sectionKind {
	case SectionStaged:
		status = entry.Staged
	case SectionUntracked:
		return "" // Untracked files don't show mode
	default:
		status = entry.Unstaged
	}

	// Check for unmerged
	if entry.UnmergedMode != "" {
		return git.ModeText[entry.UnmergedMode]
	}

	key := string(status)
	if text, ok := git.ModeText[key]; ok {
		return text
	}
	return ""
}

// styleForMode returns the appropriate style for a file's change type.
func styleForMode(tokens Tokens, entry *git.StatusEntry, sectionKind SectionKind) lipgloss.Style {
	var status git.FileStatus
	switch sectionKind {
	case SectionStaged:
		status = entry.Staged
	default:
		status = entry.Unstaged
	}

	switch status {
	case git.FileStatusModified:
		return tokens.ChangeModified
	case git.FileStatusNew, git.FileStatusAdded:
		return tokens.ChangeAdded
	case git.FileStatusDeleted:
		return tokens.ChangeDeleted
	case git.FileStatusRenamed:
		return tokens.ChangeRenamed
	case git.FileStatusCopied:
		return tokens.ChangeCopied
	case git.FileStatusUntracked:
		return tokens.ChangeUntracked
	default:
		return tokens.Normal
	}
}

// padRight pads a string to the right with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
