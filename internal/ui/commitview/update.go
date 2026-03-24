package commitview

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case CommitDataLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.ready = true
		m.info = msg.Info
		m.overview = msg.Overview
		m.signature = msg.Signature
		m.diffs = msg.Diffs
		m.xOffset = 0
		m.cursorCol = 0
		// Build viewport content and calculate total lines
		content := m.renderContent()
		m.totalLines = strings.Count(content, "\n")
		if m.totalLines > 0 && !strings.HasSuffix(content, "\n") {
			m.totalLines++ // Count the last line if it doesn't end with newline
		}
		m.maxLineWidth = maxVisibleWidth(content)
		m.viewport.SetContent(content)
		m.cursorLine = 0
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			// Only allow close while loading
			if key.Matches(msg, m.keys.Close) || key.Matches(msg, m.keys.CloseEscape) {
				m.done = true
				return m, func() tea.Msg { return CloseCommitViewMsg{} }
			}
			return m, nil
		}
		return m.handleKey(msg)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Handle pending bracket sequences ([c / ]c)
	if m.pendingBracket != "" {
		bracket := m.pendingBracket
		m.pendingBracket = ""
		if msg.String() == "c" {
			if bracket == "]" {
				m.viewport.YOffset++
				if m.viewport.YOffset > m.totalLines-m.viewport.Height {
					m.viewport.YOffset = m.totalLines - m.viewport.Height
				}
				if m.viewport.YOffset < 0 {
					m.viewport.YOffset = 0
				}
			} else {
				m.viewport.YOffset--
				if m.viewport.YOffset < 0 {
					m.viewport.YOffset = 0
				}
			}
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Close), key.Matches(msg, m.keys.CloseEscape):
		m.done = true
		return m, func() tea.Msg { return CloseCommitViewMsg{} }

	case key.Matches(msg, m.keys.MoveDown):
		if m.cursorLine < m.totalLines-1 {
			m.cursorLine++
		}
		// Ensure cursor visible
		if m.cursorLine >= m.viewport.YOffset+m.viewport.Height {
			m.viewport.YOffset = m.cursorLine - m.viewport.Height + 1
		}
		return m, nil

	case key.Matches(msg, m.keys.MoveUp):
		if m.cursorLine > 0 {
			m.cursorLine--
		}
		// Ensure cursor visible
		if m.cursorLine < m.viewport.YOffset {
			m.viewport.YOffset = m.cursorLine
		}
		return m, nil

	case key.Matches(msg, m.keys.PageDown):
		m.viewport.PageDown()
		m.ensureCursorInViewport()
		return m, nil

	case key.Matches(msg, m.keys.PageUp):
		m.viewport.PageUp()
		m.ensureCursorInViewport()
		return m, nil

	case key.Matches(msg, m.keys.HalfPageDown):
		m.viewport.HalfPageDown()
		m.ensureCursorInViewport()
		return m, nil

	case key.Matches(msg, m.keys.HalfPageUp):
		m.viewport.HalfPageUp()
		m.ensureCursorInViewport()
		return m, nil

	// Hunk navigation
	case key.Matches(msg, m.keys.NextHunkHeader):
		m.cursorLine = m.findNextHunkHeader(m.cursorLine)
		m.ensureCursorVisible()
		return m, nil

	case key.Matches(msg, m.keys.PrevHunkHeader):
		m.cursorLine = m.findPrevHunkHeader(m.cursorLine)
		m.ensureCursorVisible()
		return m, nil

	// Horizontal cursor movement
	case key.Matches(msg, m.keys.ScrollRight):
		m.cursorCol++
		m = m.ensureHorizontalCursorVisible()
		return m, nil

	case key.Matches(msg, m.keys.ScrollLeft):
		if m.cursorCol > 0 {
			m.cursorCol--
		}
		m = m.ensureHorizontalCursorVisible()
		return m, nil

	case key.Matches(msg, m.keys.ScrollStart):
		m.cursorCol = 0
		m.xOffset = 0
		return m, nil

	case key.Matches(msg, m.keys.ScrollEnd):
		// Move cursor to end of widest visible content
		m.cursorCol = m.maxLineWidth
		if m.cursorCol > 0 {
			m.cursorCol-- // Position on last char, not past it
		}
		m = m.ensureHorizontalCursorVisible()
		return m, nil

	// Two-key scroll sequences: ]c / [c
	case msg.String() == "]":
		m.pendingBracket = "]"
		return m, nil

	case msg.String() == "[":
		m.pendingBracket = "["
		return m, nil

	// Yank
	case key.Matches(msg, m.keys.YankSelected):
		if m.info != nil {
			return m, yankCmd(m.info.Hash)
		}
		return m, nil

	// Actions
	case key.Matches(msg, m.keys.OpenFileInWorktree):
		filePath := m.getCurrentFilePath()
		if filePath != "" {
			return m, func() tea.Msg {
				return OpenFileMsg{Path: filePath}
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.OpenCommitLink):
		if m.info != nil {
			return m, func() tea.Msg {
				// URL would be constructed from repo remote + commit hash
				return OpenURLMsg{URL: m.info.Hash}
			}
		}
		return m, nil

	// Popup triggers
	case key.Matches(msg, m.keys.CherryPickPopup):
		return m, m.openPopupCmd("cherry-pick")

	case key.Matches(msg, m.keys.BranchPopup):
		return m, m.openPopupCmd("branch")

	case key.Matches(msg, m.keys.BisectPopup):
		return m, m.openPopupCmd("bisect")

	case key.Matches(msg, m.keys.CommitPopup):
		return m, m.openPopupCmd("commit")

	case key.Matches(msg, m.keys.DiffPopup):
		return m, m.openPopupCmd("diff")

	case key.Matches(msg, m.keys.PushPopup):
		return m, m.openPopupCmd("push")

	case key.Matches(msg, m.keys.RevertPopup):
		return m, m.openPopupCmd("revert")

	case key.Matches(msg, m.keys.RebasePopup):
		return m, m.openPopupCmd("rebase")

	case key.Matches(msg, m.keys.ResetPopup):
		return m, m.openPopupCmd("reset")

	case key.Matches(msg, m.keys.TagPopup):
		return m, m.openPopupCmd("tag")
	}

	return m, nil
}

// openPopupCmd returns a command that emits OpenPopupMsg.
func (m Model) openPopupCmd(popupType string) tea.Cmd {
	commitHash := ""
	if m.info != nil {
		commitHash = m.info.Hash
	}
	return func() tea.Msg {
		return OpenPopupMsg{Type: popupType, Commit: commitHash}
	}
}

// yankCmd returns a command that emits YankMsg.
func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return YankMsg{Text: text}
	}
}

// ensureCursorVisible adjusts viewport to keep cursor in view.
func (m *Model) ensureCursorVisible() {
	if m.cursorLine < m.viewport.YOffset {
		m.viewport.YOffset = m.cursorLine
	} else if m.cursorLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = m.cursorLine - m.viewport.Height + 1
	}
}

// ensureCursorInViewport clamps cursorLine to stay within the visible viewport.
// Called after viewport scrolling (ctrl-d/f/u/b) to keep cursor on-screen.
func (m *Model) ensureCursorInViewport() {
	if m.cursorLine < m.viewport.YOffset {
		m.cursorLine = m.viewport.YOffset
	} else if m.cursorLine >= m.viewport.YOffset+m.viewport.Height {
		m.cursorLine = m.viewport.YOffset + m.viewport.Height - 1
	}
}

// ensureHorizontalCursorVisible adjusts xOffset to keep cursor column in view.
func (m Model) ensureHorizontalCursorVisible() Model {
	// Scroll left if cursor is before viewport
	if m.cursorCol < m.xOffset {
		m.xOffset = m.cursorCol
	}
	// Scroll right if cursor is past viewport
	if m.width > 0 && m.cursorCol >= m.xOffset+m.width {
		m.xOffset = m.cursorCol - m.width + 1
	}
	return m
}

// findNextHunkHeader finds the next hunk header line after the current cursor position.
func (m Model) findNextHunkHeader(from int) int {
	hunkLines := m.getHunkHeaderLines()
	for _, line := range hunkLines {
		if line > from {
			return line
		}
	}
	// Wrap or stay at end
	if len(hunkLines) > 0 {
		return hunkLines[len(hunkLines)-1]
	}
	return from
}

// findPrevHunkHeader finds the previous hunk header line before the current cursor position.
func (m Model) findPrevHunkHeader(from int) int {
	hunkLines := m.getHunkHeaderLines()
	for i := len(hunkLines) - 1; i >= 0; i-- {
		if hunkLines[i] < from {
			return hunkLines[i]
		}
	}
	// Wrap or stay at start
	if len(hunkLines) > 0 {
		return hunkLines[0]
	}
	return from
}

// getHunkHeaderLines returns line numbers of all hunk headers in the content.
func (m Model) getHunkHeaderLines() []int {
	var lines []int
	lineNum := 0

	// Skip header lines (same structure as renderContent)
	lineNum++ // Commit header
	lineNum++ // Author
	lineNum++ // AuthorDate

	// Committer lines (if different)
	if m.info != nil && m.info.CommitterName != "" && m.info.CommitterName != m.info.AuthorName {
		lineNum++ // Committer
		lineNum++ // CommitDate
	}

	lineNum++ // Blank line
	lineNum++ // Subject

	// Body lines
	if m.info != nil && m.info.Body != "" {
		lineNum++ // Blank line
		bodyLines := strings.Split(m.info.Body, "\n")
		lineNum += len(bodyLines)
	}

	// File overview
	if m.overview != nil && len(m.overview.Files) > 0 {
		lineNum++ // Blank line
		lineNum++ // Summary
		lineNum += len(m.overview.Files)
	}

	// Diffs - find hunk headers
	if len(m.diffs) > 0 {
		lineNum++ // Blank line before diffs
		for _, diff := range m.diffs {
			lineNum++ // File header
			for _, hunk := range diff.Hunks {
				lines = append(lines, lineNum) // This is a hunk header
				lineNum++                      // Hunk header line
				lineNum += len(hunk.Lines)     // Hunk content lines
			}
		}
	}

	return lines
}

// getCurrentFilePath returns the file path at the current cursor position.
func (m Model) getCurrentFilePath() string {
	if len(m.diffs) == 0 {
		return ""
	}
	// Simple implementation: return first diff file path
	// A more complete implementation would track which file the cursor is in
	return m.diffs[0].Path
}
