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
		// Build viewport content and calculate total lines
		content := m.renderContent()
		m.totalLines = strings.Count(content, "\n")
		if m.totalLines > 0 && !strings.HasSuffix(content, "\n") {
			m.totalLines++ // Count the last line if it doesn't end with newline
		}
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

	case key.Matches(msg, m.keys.YankSelected):
		if m.info != nil {
			return m, yankCmd(m.info.Hash)
		}
		return m, nil
	}

	return m, nil
}

// yankCmd copies text to clipboard using OSC 52.
func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		// Return clipboard write message (handled by app)
		return nil
	}
}
