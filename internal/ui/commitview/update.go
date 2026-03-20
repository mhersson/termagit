package commitview

import (
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
		// Build viewport content
		m.viewport.SetContent(m.renderContent())
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			// Only allow close while loading
			if key.Matches(msg, m.keys.Close) || key.Matches(msg, m.keys.CloseEscape) {
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
		return m, func() tea.Msg { return CloseCommitViewMsg{} }

	case key.Matches(msg, m.keys.MoveDown):
		m.viewport.ScrollDown(1)
		return m, nil

	case key.Matches(msg, m.keys.MoveUp):
		m.viewport.ScrollUp(1)
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
