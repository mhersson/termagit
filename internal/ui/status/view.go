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

	// Format header text
	var header string
	if len(s.Items) > 0 {
		header = fmt.Sprintf("%s %s (%d)", sign, s.Title, len(s.Items))
	} else {
		header = fmt.Sprintf("%s %s", sign, s.Title)
	}

	// Apply section-specific styling
	if onHeader {
		b.WriteString(m.tokens.Cursor.Render(header))
	} else {
		style := getSectionHeaderStyle(m.tokens, s.Kind)
		b.WriteString(style.Render(header))
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

// getSectionHeaderStyle returns the appropriate style for a section header.
func getSectionHeaderStyle(tokens Tokens, kind SectionKind) lipgloss.Style {
	switch kind {
	case SectionSequencer:
		return tokens.Picking // Will be overridden by actual operation type
	case SectionRebase:
		return tokens.Rebasing
	case SectionBisect:
		return tokens.Bisecting
	case SectionStashes:
		return tokens.Stashes
	default:
		return tokens.SectionHeader
	}
}

// renderItem renders a single item based on its type.
func renderItem(m Model, sectionIdx, itemIdx int, item *Item, sectionKind SectionKind) string {
	var b strings.Builder

	onItem := m.cursor.Section == sectionIdx && m.cursor.Item == itemIdx && m.cursor.Hunk == -1

	// Sequencer/Rebase/Bisect items (have Action)
	if item.Action != "" {
		switch sectionKind {
		case SectionRebase:
			b.WriteString(renderRebaseItem(m, item, onItem))
		case SectionBisect:
			b.WriteString(renderBisectItem(m, item, onItem))
		default:
			b.WriteString(renderSequencerItem(m, item, onItem))
		}
		return b.String()
	}

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
		return b.String()
	}

	// Stash entry
	if item.Stash != nil {
		b.WriteString(renderStashItem(m, item, onItem))
		return b.String()
	}

	// Commit entry
	if item.Commit != nil {
		b.WriteString(renderCommitItem(m, item, onItem))
		return b.String()
	}

	return b.String()
}

// renderSequencerItem renders a cherry-pick or revert item.
// Format: action (6 chars) + hash (7 chars) + subject
func renderSequencerItem(m Model, item *Item, onItem bool) string {
	action := padRight(item.Action, 6)
	hash := item.ActionHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	subject := item.ActionSubject

	if onItem {
		line := fmt.Sprintf("  %s %s %s", action, hash, subject)
		return m.tokens.Cursor.Render(line) + "\n"
	}

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(m.tokens.GraphOrange.Render(action))
	b.WriteString(" ")
	b.WriteString(m.tokens.Hash.Render(hash))
	b.WriteString(" ")
	b.WriteString(subject)
	b.WriteString("\n")
	return b.String()
}

// renderRebaseItem renders a rebase todo item.
// Format: [>] action (6 chars) + hash (7 chars) + subject
// > prefix for stopped item, done items in RebaseDone style
func renderRebaseItem(m Model, item *Item, onItem bool) string {
	prefix := "  "
	if item.ActionStopped {
		prefix = "> "
	}

	action := padRight(item.Action, 6)
	hash := item.ActionHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	subject := item.ActionSubject

	if onItem {
		line := fmt.Sprintf("%s%s %s %s", prefix, action, hash, subject)
		return m.tokens.Cursor.Render(line) + "\n"
	}

	var b strings.Builder
	b.WriteString(prefix)
	if item.ActionDone {
		b.WriteString(m.tokens.RebaseDone.Render(action))
		b.WriteString(" ")
		b.WriteString(m.tokens.RebaseDone.Render(hash))
		b.WriteString(" ")
		b.WriteString(m.tokens.RebaseDone.Render(subject))
	} else {
		b.WriteString(m.tokens.GraphOrange.Render(action))
		b.WriteString(" ")
		b.WriteString(m.tokens.Hash.Render(hash))
		b.WriteString(" ")
		b.WriteString(subject)
	}
	b.WriteString("\n")
	return b.String()
}

// renderBisectItem renders a bisect log item.
// Format: [>] action (5 chars) + hash (7 chars) + subject
// > prefix for current, good=green, bad=red
func renderBisectItem(m Model, item *Item, onItem bool) string {
	prefix := "  "
	// Could mark current but bisect log doesn't typically have a current marker

	action := padRight(item.Action, 5)
	hash := item.ActionHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	subject := item.ActionSubject

	if onItem {
		line := fmt.Sprintf("%s%s %s %s", prefix, action, hash, subject)
		return m.tokens.Cursor.Render(line) + "\n"
	}

	var b strings.Builder
	b.WriteString(prefix)

	// Color action based on type
	switch item.Action {
	case "good":
		b.WriteString(m.tokens.GraphGreen.Render(action))
	case "bad":
		b.WriteString(m.tokens.GraphRed.Render(action))
	case "skip":
		b.WriteString(m.tokens.GraphBlue.Render(action))
	default:
		b.WriteString(m.tokens.GraphOrange.Render(action))
	}

	b.WriteString(" ")
	b.WriteString(m.tokens.Hash.Render(hash))
	b.WriteString(" ")
	b.WriteString(subject)
	b.WriteString("\n")
	return b.String()
}

// renderStashItem renders a stash entry.
// Format: stash@{N}: message
func renderStashItem(m Model, item *Item, onItem bool) string {
	line := fmt.Sprintf("  %s: %s", item.Stash.Name, item.Stash.Message)
	if onItem {
		return m.tokens.Cursor.Render(line) + "\n"
	}

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(m.tokens.SubtleText.Render(item.Stash.Name))
	b.WriteString(": ")
	b.WriteString(item.Stash.Message)
	b.WriteString("\n")
	return b.String()
}

// renderCommitItem renders a commit entry.
// Format: hash (7 chars) + [refs] + subject
func renderCommitItem(m Model, item *Item, onItem bool) string {
	hash := item.Commit.AbbreviatedHash
	subject := item.Commit.Subject

	if onItem {
		line := fmt.Sprintf("  %s %s", hash, subject)
		return m.tokens.Cursor.Render(line) + "\n"
	}

	var b strings.Builder
	b.WriteString("  ")
	b.WriteString(m.tokens.Hash.Render(hash))

	// Render refs if present
	if len(item.Commit.Refs) > 0 {
		b.WriteString(" ")
		b.WriteString(renderRefs(m, item.Commit.Refs))
	}

	b.WriteString(" ")
	b.WriteString(subject)
	b.WriteString("\n")
	return b.String()
}

// renderRefs renders commit ref decorations.
func renderRefs(m Model, refs []git.Ref) string {
	if len(refs) == 0 {
		return ""
	}

	var parts []string
	for _, ref := range refs {
		switch ref.Kind {
		case git.RefKindLocal:
			parts = append(parts, m.tokens.Branch.Render(ref.Name))
		case git.RefKindRemote:
			remoteName := ref.Remote + "/" + ref.Name
			parts = append(parts, m.tokens.Remote.Render(remoteName))
		case git.RefKindTag:
			parts = append(parts, m.tokens.Tag.Render(ref.Name))
		case git.RefKindHead:
			parts = append(parts, m.tokens.Bold.Render("HEAD"))
		}
	}

	return "(" + strings.Join(parts, ", ") + ")"
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
