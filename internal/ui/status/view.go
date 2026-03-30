package status

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/shared"
)

// view renders the status buffer.
func view(m Model) string {
	if m.loading {
		return "Loading..."
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	// Overlay commit view or popup if active — use cached lines when available
	if m.commitView != nil || m.popup != nil {
		// Use cached content if warm, otherwise fall back to renderContent
		var lines []string
		var cursorLine int
		if !m.contentDirty && len(m.cachedBaseLines) > 0 {
			lines = m.cachedBaseLines
			cursorLine = computeCursorLine(m)
		} else {
			content, cl := renderContent(m)
			lines = strings.Split(content, "\n")
			cursorLine = cl
		}

		// Apply cursor styling to the cursor line (popup = no block cursor)
		if cursorLine >= 0 && cursorLine < len(lines) {
			stripped := ansi.Strip(lines[cursorLine])
			lines[cursorLine] = m.tokens.Cursor.Render(stripped)
		}

		// Apply viewport-like scrolling
		startLine := m.viewport.YOffset
		endLine := startLine + m.viewport.Height
		if endLine > len(lines) {
			endLine = len(lines)
		}
		if startLine > len(lines) {
			startLine = len(lines)
		}

		// Ensure cursor is visible
		if cursorLine < startLine {
			startLine = cursorLine
			endLine = startLine + m.viewport.Height
		} else if cursorLine >= endLine {
			endLine = cursorLine + 1
			startLine = endLine - m.viewport.Height
			if startLine < 0 {
				startLine = 0
			}
		}

		// Build visible content with strings.Builder instead of Join
		var b strings.Builder
		for i := startLine; i < endLine; i++ {
			if i > startLine {
				b.WriteByte('\n')
			}
			b.WriteString(lines[i])
		}
		visibleContent := b.String()

		if m.commitView != nil {
			return renderCommitViewOverlay(m, visibleContent)
		}
		return renderPopupOverlay(m, visibleContent)
	}

	// If viewport has content, use it; otherwise render directly
	var content string
	if m.viewport.Width > 0 && m.viewport.Height > 0 {
		content = m.viewport.View()
	} else {
		// Fallback to direct rendering (for tests or before WindowSizeMsg)
		content, _ = renderContent(m)
	}

	return content
}

// renderPopupOverlay renders the popup on top of the status buffer.
func renderPopupOverlay(m Model, statusContent string) string {
	popupView := m.popup.View()

	// Split status content into lines
	statusLines := strings.Split(statusContent, "\n")

	// Split popup content into lines
	popupLines := strings.Split(popupView, "\n")

	// Calculate popup position (bottom-anchored)
	popupHeight := len(popupLines)
	statusHeight := m.height

	// Position popup at bottom, render status above it
	startLine := statusHeight - popupHeight
	if startLine < 0 {
		startLine = 0
	}

	var b strings.Builder

	// Render status lines that appear above the popup
	for i := 0; i < startLine && i < len(statusLines); i++ {
		b.WriteString(statusLines[i])
		b.WriteString("\n")
	}

	// Render popup
	b.WriteString(popupView)

	return b.String()
}

// renderCommitViewOverlay renders the commit view in the lower portion of the terminal.
func renderCommitViewOverlay(m Model, statusContent string) string {
	cvContent := m.commitView.View()

	// Split content into lines
	statusLines := strings.Split(statusContent, "\n")
	cvLines := strings.Split(cvContent, "\n")

	// Commit view gets 60% of screen height (matches SetSize in update.go)
	cvHeight := m.height * 60 / 100
	maxStatusLines := m.height - cvHeight
	if maxStatusLines < 0 {
		maxStatusLines = 0
	}

	var b strings.Builder

	// Render status lines, padding to fill maxStatusLines
	for i := 0; i < maxStatusLines; i++ {
		if i < len(statusLines) {
			b.WriteString(statusLines[i])
		}
		b.WriteString("\n")
	}

	// Render commit view content, padding to fill cvHeight
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

// statusHints is the static set of hint bar key-action pairs.
var statusHints = []struct {
	key    string
	action string
}{
	{"<tab>", "toggle"},
	{"s", "stage"},
	{"u", "unstage"},
	{"x", "discard"},
	{"c", "commit"},
	{"?", "help"},
}

// renderHintBar renders the hint bar at the top of the status buffer.
// Format: "Hint: <tab> toggle | s stage | u unstage | x discard | c commit | ? help"
func renderHintBar(m Model) string {
	var b strings.Builder

	// "Hint:" in subtle style
	b.WriteString(m.tokens.SubtleText.Render("Hint: "))

	for i, h := range statusHints {
		if i > 0 {
			b.WriteString(m.tokens.SubtleText.Render(" | "))
		}
		b.WriteString(m.tokens.PopupSection.Render(h.key))
		b.WriteString(m.tokens.SubtleText.Render(" " + h.action))
	}

	return b.String()
}

// renderWithBlockCursor renders a line with a block cursor at position 0.
// The first character is shown with reverse video, rest has cursor line background.
func renderWithBlockCursor(tokens theme.Tokens, line string) string {
	stripped := ansi.Strip(line)
	if len(stripped) == 0 {
		return tokens.CursorBlock.Render(" ") + "\n"
	}

	// Get first visible rune (handles multi-byte UTF-8)
	firstRune, size := utf8.DecodeRuneInString(stripped)
	rest := stripped[size:]

	// First character: reverse video, rest: cursor line background
	return tokens.CursorBlock.Render(string(firstRune)) + tokens.Cursor.Render(rest) + "\n"
}

// renderCursorLine renders a line with cursor styling.
// When popup is active, shows cursor background only (no block cursor).
// When popup is not active, shows full block cursor on first character.
func renderCursorLine(m Model, line string) string {
	if m.popup != nil {
		// Popup has focus - show cursor line background but no block cursor
		return m.tokens.Cursor.Render(line) + "\n"
	}
	return renderWithBlockCursor(m.tokens, line)
}

// renderHeadBar renders the HEAD information bar.
func renderHeadBar(m Model) string {
	var b strings.Builder

	padding := 10
	if m.cfg != nil && m.cfg.UI.HEADPadding > 0 {
		padding = m.cfg.UI.HEADPadding
	}

	// Head line
	headLabel := shared.PadRight("Head:", padding)
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
		b.WriteString(" ")
		b.WriteString(m.head.Subject)
	} else if m.head.AbbrevOid == "" {
		// No OID and no subject means unborn HEAD (no commits yet)
		b.WriteString(" ")
		b.WriteString(m.tokens.SubtleText.Render("(No commits yet)"))
	}

	// Merge line (if applicable)
	if m.head.UpstreamBranch != "" {
		b.WriteString("\n")
		mergeLabel := shared.PadRight("Merge:", padding)
		b.WriteString(m.tokens.Bold.Render(mergeLabel))

		if m.head.UpstreamOid != "" {
			b.WriteString(m.tokens.Hash.Render(abbreviateOID(m.head.UpstreamOid)))
			b.WriteString(" ")
		}

		remoteBranch := m.head.UpstreamRemote + "/" + m.head.UpstreamBranch
		b.WriteString(m.tokens.Remote.Render(remoteBranch))

		if m.head.UpstreamSubject != "" {
			b.WriteString(" ")
			b.WriteString(m.head.UpstreamSubject)
		}
	}

	// Push line (if applicable)
	if m.head.PushBranch != "" {
		b.WriteString("\n")
		pushLabel := shared.PadRight("Push:", padding)
		b.WriteString(m.tokens.Bold.Render(pushLabel))

		if m.head.PushOid != "" {
			b.WriteString(m.tokens.Hash.Render(abbreviateOID(m.head.PushOid)))
			b.WriteString(" ")
		}

		remoteBranch := m.head.PushRemote + "/" + m.head.PushBranch
		b.WriteString(m.tokens.Remote.Render(remoteBranch))

		if m.head.PushSubject != "" {
			b.WriteString(" ")
			b.WriteString(m.head.PushSubject)
		}
	}

	// Tag line (if applicable)
	if m.head.Tag != "" {
		b.WriteString("\n")
		tagLabel := shared.PadRight("Tag:", padding)
		b.WriteString(m.tokens.Bold.Render(tagLabel))
		b.WriteString(m.tokens.Tag.Render(m.head.Tag))

		if m.head.TagDistance > 0 {
			fmt.Fprintf(&b, " (%d)", m.head.TagDistance)
		}
	}

	return b.String()
}

// getSectionHeaderStyle returns the appropriate style for a section header.
func getSectionHeaderStyle(tokens theme.Tokens, kind SectionKind) lipgloss.Style {
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

// renderSequencerItem renders a cherry-pick or revert item.
// Format: action (6 chars) + hash (7 chars) + subject
func renderSequencerItem(m Model, item *Item, onItem bool) string {
	action := shared.PadRight(item.Action, 6)
	hash := item.ActionHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	subject := item.ActionSubject

	if onItem {
		line := fmt.Sprintf("  %s %s %s", action, hash, subject)
		return renderCursorLine(m, line)
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

	action := shared.PadRight(item.Action, 6)
	hash := item.ActionHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	subject := item.ActionSubject

	if onItem {
		line := fmt.Sprintf("%s%s %s %s", prefix, action, hash, subject)
		return renderCursorLine(m, line)
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

	action := shared.PadRight(item.Action, 5)
	hash := item.ActionHash
	if len(hash) > 7 {
		hash = hash[:7]
	}
	subject := item.ActionSubject

	if onItem {
		line := fmt.Sprintf("%s%s %s %s", prefix, action, hash, subject)
		return renderCursorLine(m, line)
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

// renderBisectDetailItem renders the "Bisecting at" commit details.
// Matches Neogit's BisectDetailsSection layout.
func renderBisectDetailItem(m Model, entry *git.LogEntry, onItem bool) string {
	var b strings.Builder

	// Author line
	author := fmt.Sprintf("  %s%s <%s>", shared.PadRight("Author:", 12), entry.AuthorName, entry.AuthorEmail)
	if onItem {
		b.WriteString(renderCursorLine(m, author))
	} else {
		b.WriteString("  ")
		b.WriteString(m.tokens.SubtleText.Render(shared.PadRight("Author:", 12)))
		fmt.Fprintf(&b, "%s <%s>", entry.AuthorName, entry.AuthorEmail)
		b.WriteString("\n")
	}

	// AuthorDate line
	b.WriteString("  ")
	b.WriteString(m.tokens.SubtleText.Render(shared.PadRight("AuthorDate:", 12)))
	b.WriteString(entry.AuthorDate)
	b.WriteString("\n")

	// Committer line
	b.WriteString("  ")
	b.WriteString(m.tokens.SubtleText.Render(shared.PadRight("Committer:", 12)))
	fmt.Fprintf(&b, "%s <%s>", entry.CommitterName, entry.CommitterEmail)
	b.WriteString("\n")

	// CommitDate line
	b.WriteString("  ")
	b.WriteString(m.tokens.SubtleText.Render(shared.PadRight("CommitDate:", 12)))
	b.WriteString(entry.CommitterDate)
	b.WriteString("\n")

	// Blank line + description
	if entry.Body != "" {
		b.WriteString("\n")
		for _, line := range strings.Split(entry.Body, "\n") {
			if line != "" {
				b.WriteString("  ")
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderStashItem renders a stash entry.
// Format: stash@{N}: message
func renderStashItem(m Model, item *Item, onItem bool) string {
	line := fmt.Sprintf("  %s: %s", item.Stash.Name, item.Stash.Message)
	if onItem {
		return renderCursorLine(m, line)
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
		return renderCursorLine(m, line)
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

	parts := make([]string, 0, len(refs))
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
func styleForMode(tokens theme.Tokens, entry *git.StatusEntry, sectionKind SectionKind) lipgloss.Style {
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

// renderContent renders the status buffer and returns the content along with
// the visual line number where the cursor is positioned (0-indexed).
func renderContent(m Model) (content string, cursorLine int) {
	if m.loading {
		return "Loading...", 0
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err), 0
	}

	var b strings.Builder
	lineNum := 0
	cursorLine = 0

	// Render hint bar (unless disabled)
	if m.cfg == nil || !m.cfg.UI.DisableHint {
		b.WriteString(renderHintBar(m))
		b.WriteString("\n")
		lineNum++
		b.WriteString("\n")
		lineNum++
	}

	// Render HEAD bar - count lines
	headBar := renderHeadBar(m)
	b.WriteString(headBar)
	lineNum += strings.Count(headBar, "\n")
	b.WriteString("\n\n")
	lineNum += 2

	// Render sections
	for i, s := range m.sections {
		if s.Hidden {
			continue
		}

		// Track if cursor is on this section header
		if m.cursor.Section == i && m.cursor.Item == -1 {
			cursorLine = lineNum
		}

		sectionContent, sectionLines := renderSectionWithLineTracking(m, i, &s, lineNum, &cursorLine)
		b.WriteString(sectionContent)
		lineNum += sectionLines
	}

	return b.String(), cursorLine
}

// renderContentBase renders content without cursor styling (cacheable).
// Uses a sentinel cursor position so no cursor checks match.
func renderContentBase(m Model) string {
	m.cursor = Cursor{Section: -1, Item: -1, Hunk: -1, Line: -1}
	content, _ := renderContent(m)
	return content
}

// invalidateContent marks the render cache as stale.
func (m *Model) invalidateContent() {
	m.contentDirty = true
}

// ensureContent populates the render cache if needed.
func (m *Model) ensureContent() {
	if !m.contentDirty && m.cachedBaseContent != "" {
		return
	}
	m.cachedBaseContent = renderContentBase(*m)
	m.cachedBaseLines = strings.Split(m.cachedBaseContent, "\n")
	m.contentDirty = false
}

// computeCursorLine maps the cursor position to a visual line number
// without rendering. Must match renderContent's line accounting exactly.
func computeCursorLine(m Model) int {
	lineNum := 0

	// Hint bar (2 lines: hint text + blank line)
	if m.cfg == nil || !m.cfg.UI.DisableHint {
		lineNum += 2
	}

	// Head bar: Head is always 1 line. Optional Merge/Push/Tag add 1 each.
	// renderHeadBar uses \n between lines (no trailing \n).
	// renderContent counts strings.Count(headBar, "\n") = N-1 for N lines,
	// then adds 2 for "\n\n" after head bar. Total advance = N+1.
	headLines := 1
	if m.head.UpstreamBranch != "" {
		headLines++
	}
	if m.head.PushBranch != "" {
		headLines++
	}
	if m.head.Tag != "" {
		headLines++
	}
	lineNum += headLines + 1 // (headLines-1) newlines within + 2 after = headLines+1

	// Sections
	for i, s := range m.sections {
		if s.Hidden {
			continue
		}

		// Section header
		if m.cursor.Section == i && m.cursor.Item == -1 {
			return lineNum
		}
		lineNum++ // section header line

		// Items (only if not folded)
		if !s.Folded {
			for j, item := range s.Items {
				// Item line
				if m.cursor.Section == i && m.cursor.Item == j && m.cursor.Hunk == -1 {
					return lineNum
				}

				// Count item lines based on type
				if item.BisectDetail != nil {
					// Bisect detail: Author + AuthorDate + Committer + CommitDate = 4 lines
					count := 4
					if item.BisectDetail.Body != "" {
						bodyLines := strings.Split(item.BisectDetail.Body, "\n")
						count += 1 + len(bodyLines) // blank line + body lines
					}
					lineNum += count
					continue
				}

				lineNum++ // single-line item (file, stash, commit, sequencer, etc.)

				// Expanded hunks
				if item.Expanded && len(item.Hunks) > 0 {
					for h, hunk := range item.Hunks {
						if m.cursor.Section == i && m.cursor.Item == j && m.cursor.Hunk == h && m.cursor.Line == -1 {
							return lineNum
						}
						lineNum++ // hunk header

						isFolded := len(item.HunksFolded) > h && item.HunksFolded[h]
						if !isFolded {
							for l := range hunk.Lines {
								if m.cursor.Section == i && m.cursor.Item == j && m.cursor.Hunk == h && m.cursor.Line == l {
									return lineNum
								}
								lineNum++
							}
						}
					}
				} else if item.Expanded && item.HunksLoading {
					lineNum++ // "Loading diff..." line
				}
			}
		}

		lineNum++ // trailing blank line after section
	}

	return lineNum
}

// renderWithBlockCursorNoNewline renders block cursor without trailing newline.
func renderWithBlockCursorNoNewline(tokens theme.Tokens, line string) string {
	if len(line) == 0 {
		return tokens.CursorBlock.Render(" ")
	}
	firstRune, size := utf8.DecodeRuneInString(line)
	rest := line[size:]
	return tokens.CursorBlock.Render(string(firstRune)) + tokens.Cursor.Render(rest)
}

// applyViewportWithCursor builds viewport content from cached base lines,
// applying cursor styling to the cursor line, and updates the viewport.
func (m *Model) applyViewportWithCursor() {
	m.ensureContent()
	cursorLine := computeCursorLine(*m)

	// Build content with cursor styling applied
	var b strings.Builder
	for i, line := range m.cachedBaseLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == cursorLine {
			stripped := ansi.Strip(line)
			if m.popup != nil {
				b.WriteString(m.tokens.Cursor.Render(stripped))
			} else {
				b.WriteString(renderWithBlockCursorNoNewline(m.tokens, stripped))
			}
		} else {
			b.WriteString(line)
		}
	}
	m.viewport.SetContent(b.String())
	ensureCursorVisible(m, cursorLine)
}

// renderSectionWithLineTracking renders a section and updates cursorLine if cursor is within.
// Returns the rendered content and the number of newlines written.
func renderSectionWithLineTracking(m Model, sectionIdx int, s *Section, startLine int, cursorLine *int) (string, int) {
	var b strings.Builder
	lineNum := startLine

	// Section header
	onHeader := m.cursor.Section == sectionIdx && m.cursor.Item == -1
	sign := ">"
	if !s.Folded {
		sign = "v"
	}

	if onHeader {
		*cursorLine = lineNum
	}

	// Build header with styled title and normal count
	style := getSectionHeaderStyle(m.tokens, s.Kind)

	// "Bisecting at" shows OID instead of count
	isBisectDetails := s.Title == "Bisecting at" && len(s.Items) > 0 && s.Items[0].BisectDetail != nil

	if onHeader {
		var header string
		if isBisectDetails {
			header = fmt.Sprintf("%s %s %s", sign, s.Title, s.Items[0].BisectDetail.AbbreviatedHash)
		} else if len(s.Items) > 0 {
			header = fmt.Sprintf("%s %s (%d)", sign, s.Title, len(s.Items))
		} else {
			header = fmt.Sprintf("%s %s", sign, s.Title)
		}
		b.WriteString(renderCursorLine(m, header))
	} else {
		b.WriteString(style.Render(fmt.Sprintf("%s %s", sign, s.Title)))
		if isBisectDetails {
			b.WriteString(" ")
			b.WriteString(m.tokens.Hash.Render(s.Items[0].BisectDetail.AbbreviatedHash))
		} else if len(s.Items) > 0 {
			fmt.Fprintf(&b, " (%d)", len(s.Items))
		}
		b.WriteString("\n")
	}
	lineNum++

	// Items (only if not folded)
	if !s.Folded {
		for i, item := range s.Items {
			itemContent, itemLines := renderItemWithLineTracking(m, sectionIdx, i, &item, s.Kind, lineNum, cursorLine)
			b.WriteString(itemContent)
			lineNum += itemLines
		}
	}

	b.WriteString("\n")
	return b.String(), lineNum - startLine + 1 // +1 for trailing blank line
}

// renderItemWithLineTracking renders an item and updates cursorLine if cursor is on this item.
// Returns the rendered content and the number of newlines written.
func renderItemWithLineTracking(m Model, sectionIdx, itemIdx int, item *Item, sectionKind SectionKind, startLine int, cursorLine *int) (string, int) {
	var b strings.Builder
	lineNum := startLine

	onItem := m.cursor.Section == sectionIdx && m.cursor.Item == itemIdx && m.cursor.Hunk == -1

	if onItem {
		*cursorLine = lineNum
	}

	// Bisect details item ("Bisecting at" section)
	if item.BisectDetail != nil {
		content := renderBisectDetailItem(m, item.BisectDetail, onItem)
		b.WriteString(content)
		return b.String(), strings.Count(content, "\n")
	}

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
		return b.String(), 1
	}

	// File entry
	if item.Entry != nil {
		modeText := getModeText(item.Entry, sectionKind)
		path := item.Entry.Path

		sign := ">"
		if item.Expanded {
			sign = "v"
		}

		if onItem {
			var line string
			if modeText == "" {
				line = fmt.Sprintf("  %s %s", sign, path)
			} else {
				line = fmt.Sprintf("  %s %s %s", sign, shared.PadRight(modeText, 12), path)
			}
			b.WriteString(renderCursorLine(m, line))
		} else {
			b.WriteString("  ")
			b.WriteString(sign)
			b.WriteString(" ")
			if modeText != "" {
				b.WriteString(styleForMode(m.tokens, item.Entry, sectionKind).Render(shared.PadRight(modeText, 12)))
			}
			b.WriteString(path)
			b.WriteString("\n")
		}
		lineNum++

		// Inline diff (if expanded)
		if item.Expanded && len(item.Hunks) > 0 {
			for hunkIdx, hunk := range item.Hunks {
				isFolded := len(item.HunksFolded) > hunkIdx && item.HunksFolded[hunkIdx]
				hunkContent, hunkLines := renderHunkWithLineTracking(m, sectionIdx, itemIdx, hunkIdx, &hunk, isFolded, lineNum, cursorLine)
				b.WriteString(hunkContent)
				lineNum += hunkLines
			}
		} else if item.Expanded && item.HunksLoading {
			b.WriteString("      Loading diff...\n")
			lineNum++
		}
		return b.String(), lineNum - startLine
	}

	// Stash entry
	if item.Stash != nil {
		b.WriteString(renderStashItem(m, item, onItem))
		return b.String(), 1
	}

	// Commit entry
	if item.Commit != nil {
		b.WriteString(renderCommitItem(m, item, onItem))
		return b.String(), 1
	}

	return b.String(), 0
}

// renderHunkWithLineTracking renders a hunk and updates cursorLine if cursor is within.
// Returns the rendered content and the number of newlines written.
func renderHunkWithLineTracking(m Model, sectionIdx, itemIdx, hunkIdx int, hunk *git.Hunk, folded bool, startLine int, cursorLine *int) (string, int) {
	var b strings.Builder
	lineNum := startLine

	onHunk := m.cursor.Section == sectionIdx &&
		m.cursor.Item == itemIdx &&
		m.cursor.Hunk == hunkIdx &&
		m.cursor.Line == -1

	if onHunk {
		*cursorLine = lineNum
	}

	// Hunk header with fold indicator
	sign := "v"
	if folded {
		sign = ">"
	}
	header := "    " + sign + " " + hunk.Header
	if onHunk {
		b.WriteString(renderCursorLine(m, header))
	} else {
		b.WriteString(m.tokens.DiffHunkHeader.Render(header))
		b.WriteString("\n")
	}
	lineNum++

	// Diff lines (only if not folded)
	if !folded {
		for lineIdx, line := range hunk.Lines {
			onLine := m.cursor.Section == sectionIdx &&
				m.cursor.Item == itemIdx &&
				m.cursor.Hunk == hunkIdx &&
				m.cursor.Line == lineIdx

			if onLine {
				*cursorLine = lineNum
			}

			var lineStr string
			switch line.Op {
			case git.DiffOpAdd:
				lineStr = "      +" + line.Content
				if onLine {
					b.WriteString(renderCursorLine(m, lineStr))
				} else {
					b.WriteString(m.tokens.DiffAdd.Render(lineStr))
					b.WriteString("\n")
				}
			case git.DiffOpDelete:
				lineStr = "      -" + line.Content
				if onLine {
					b.WriteString(renderCursorLine(m, lineStr))
				} else {
					b.WriteString(m.tokens.DiffDelete.Render(lineStr))
					b.WriteString("\n")
				}
			case git.DiffOpContext:
				lineStr = "       " + line.Content
				if onLine {
					b.WriteString(renderCursorLine(m, lineStr))
				} else {
					b.WriteString(m.tokens.DiffContext.Render(lineStr))
					b.WriteString("\n")
				}
			}
			lineNum++
		}
	}

	return b.String(), lineNum - startLine
}

// ensureCursorVisible scrolls the viewport minimally to keep cursor in view.
// Use for normal cursor movement.
func ensureCursorVisible(m *Model, cursorLine int) {
	if cursorLine < m.viewport.YOffset {
		m.viewport.YOffset = cursorLine
	} else if cursorLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = cursorLine - m.viewport.Height + 1
	}
}

// preserveScreenPosition adjusts viewport so cursor stays at same screen row.
// Call this when expanding content (diff toggle) to prevent jarring jumps.
// screenRow is the cursor's position relative to viewport top (cursorLine - yOffset).
func preserveScreenPosition(m *Model, newCursorLine int, screenRow int) {
	m.viewport.YOffset = newCursorLine - screenRow
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}
}
