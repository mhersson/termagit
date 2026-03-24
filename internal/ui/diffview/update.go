package diffview

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case DiffDataLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.files = msg.Files
		m.stats = msg.Stats
		m.fileIdx = 0
		m.hunkIdx = -1
		m.xOffset = 0
		m.cursorCol = 0
		// Build viewport content
		content := m.renderContent()
		m.totalLines = countLines(content)
		m.maxLineWidth = maxVisibleWidth(content)
		m.viewport.SetContent(content)
		m.cursorLine = 0
		return m, nil

	case HunkStagedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		// Reload diffs after stage/unstage
		m.loading = true
		return m, m.loadDiffCmd()

	case tea.KeyMsg:
		if m.loading {
			if key.Matches(msg, m.keys.Close) || key.Matches(msg, m.keys.CloseEscape) {
				m.done = true
				return m, func() tea.Msg { return CloseDiffViewMsg{} }
			}
			return m, nil
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Close), key.Matches(msg, m.keys.CloseEscape):
		m.done = true
		return m, func() tea.Msg { return CloseDiffViewMsg{} }

	case key.Matches(msg, m.keys.MoveDown):
		if m.cursorLine < m.totalLines-1 {
			m.cursorLine++
		}
		m.ensureCursorVisible()
		return m, nil

	case key.Matches(msg, m.keys.MoveUp):
		if m.cursorLine > 0 {
			m.cursorLine--
		}
		m.ensureCursorVisible()
		return m, nil

	case key.Matches(msg, m.keys.PageDown):
		m.viewport.PageDown()
		return m, nil

	case key.Matches(msg, m.keys.PageUp):
		m.viewport.PageUp()
		return m, nil

	case key.Matches(msg, m.keys.HalfPageDown):
		m.viewport.HalfPageDown()
		return m, nil

	case key.Matches(msg, m.keys.HalfPageUp):
		m.viewport.HalfPageUp()
		return m, nil

	// Hunk navigation
	case key.Matches(msg, m.keys.NextHunk), key.Matches(msg, m.keys.NextHunkHeader):
		m.cursorLine = m.findNextHunkHeader(m.cursorLine)
		m.ensureCursorVisible()
		return m, nil

	case key.Matches(msg, m.keys.PrevHunk), key.Matches(msg, m.keys.PrevHunkHeader):
		m.cursorLine = m.findPrevHunkHeader(m.cursorLine)
		m.ensureCursorVisible()
		return m, nil

	// File navigation
	case key.Matches(msg, m.keys.NextFile):
		if m.fileIdx < len(m.files)-1 {
			m.fileIdx++
			m.scrollToFile(m.fileIdx)
		}
		return m, nil

	case key.Matches(msg, m.keys.PrevFile):
		if m.fileIdx > 0 {
			m.fileIdx--
			m.scrollToFile(m.fileIdx)
		}
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

	// Stage/unstage hunk
	case key.Matches(msg, m.keys.StageHunk):
		if m.source.Kind != git.DiffUnstaged || m.repo == nil {
			return m, nil
		}
		fileIdx, hunkIdx := m.currentFileHunk()
		if fileIdx >= 0 && hunkIdx >= 0 && fileIdx < len(m.files) && hunkIdx < len(m.files[fileIdx].Hunks) {
			return m, stageHunkCmd(m.repo, m.files[fileIdx].Path, &m.files[fileIdx].Hunks[hunkIdx])
		}
		return m, nil

	case key.Matches(msg, m.keys.UnstageHunk):
		if m.source.Kind != git.DiffStaged || m.repo == nil {
			return m, nil
		}
		fileIdx, hunkIdx := m.currentFileHunk()
		if fileIdx >= 0 && hunkIdx >= 0 && fileIdx < len(m.files) && hunkIdx < len(m.files[fileIdx].Hunks) {
			return m, unstageHunkCmd(m.repo, m.files[fileIdx].Path, &m.files[fileIdx].Hunks[hunkIdx])
		}
		return m, nil
	}

	return m, nil
}

// ensureCursorVisible adjusts viewport to keep cursor in view.
func (m *Model) ensureCursorVisible() {
	if m.cursorLine < m.viewport.YOffset {
		m.viewport.YOffset = m.cursorLine
	} else if m.cursorLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.YOffset = m.cursorLine - m.viewport.Height + 1
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
	if len(hunkLines) > 0 {
		return hunkLines[0]
	}
	return from
}

// getHunkHeaderLines returns line numbers of all hunk headers in the content.
func (m Model) getHunkHeaderLines() []int {
	var lines []int
	lineNum := 0

	// Header line
	lineNum++

	// Stat block
	if m.stats != nil && len(m.stats.Files) > 0 {
		lineNum++ // summary
		lineNum += len(m.stats.Files)
		lineNum++ // empty line
	}

	// Files
	for _, diff := range m.files {
		lineNum++ // file header
		lineNum++ // separator
		for _, hunk := range diff.Hunks {
			lines = append(lines, lineNum) // hunk header line
			lineNum++                      // hunk header
			lineNum += len(hunk.Lines)     // hunk content
		}
	}

	return lines
}

// getFileHeaderLines returns line numbers of all file headers in the content.
func (m Model) getFileHeaderLines() []int {
	var lines []int
	lineNum := 0

	// Header line
	lineNum++

	// Stat block
	if m.stats != nil && len(m.stats.Files) > 0 {
		lineNum++ // summary
		lineNum += len(m.stats.Files)
		lineNum++ // empty line
	}

	// Files
	for _, diff := range m.files {
		lines = append(lines, lineNum) // file header line
		lineNum++                      // file header
		lineNum++                      // separator
		for _, hunk := range diff.Hunks {
			lineNum++              // hunk header
			lineNum += len(hunk.Lines) // hunk content
		}
	}

	return lines
}

// scrollToFile scrolls the viewport to show the given file index.
func (m *Model) scrollToFile(fileIdx int) {
	fileLines := m.getFileHeaderLines()
	if fileIdx < len(fileLines) {
		m.cursorLine = fileLines[fileIdx]
		m.ensureCursorVisible()
	}
}

// currentFileHunk determines which file and hunk the cursor is currently on.
func (m Model) currentFileHunk() (int, int) {
	lineNum := 0

	// Header line
	lineNum++

	// Stat block
	if m.stats != nil && len(m.stats.Files) > 0 {
		lineNum++ // summary
		lineNum += len(m.stats.Files)
		lineNum++ // empty line
	}

	// Walk through files and hunks
	for fi, diff := range m.files {
		lineNum++ // file header
		lineNum++ // separator
		for hi, hunk := range diff.Hunks {
			hunkStart := lineNum
			lineNum++                  // hunk header
			lineNum += len(hunk.Lines) // hunk content
			if m.cursorLine >= hunkStart && m.cursorLine < lineNum {
				return fi, hi
			}
		}
	}

	return -1, -1
}

// countLines counts the number of lines in content.
func countLines(content string) int {
	if content == "" {
		return 0
	}
	n := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		n++
	}
	return n
}
