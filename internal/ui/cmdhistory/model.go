package cmdhistory

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/cmdlog"
	"github.com/mhersson/termagit/internal/theme"
)

// CloseMsg signals that the command history view should be closed.
type CloseMsg struct{}

// Model is the command history view model.
type Model struct {
	entries []cmdlog.Entry
	folded  []bool
	cursor  int
	keys    KeyMap
	tokens  theme.Tokens
	width   int
	height  int
}

// New creates a new command history model from the given entries.
func New(entries []cmdlog.Entry, tokens theme.Tokens, width, height int) Model {
	folded := make([]bool, len(entries))
	for i := range folded {
		folded[i] = true // collapsed by default
	}
	return Model{
		entries: entries,
		folded:  folded,
		cursor:  0,
		keys:    DefaultKeyMap(),
		tokens:  tokens,
		width:   width,
		height:  height,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages for the command history view.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Close), key.Matches(msg, m.keys.CloseEscape):
			return m, func() tea.Msg { return CloseMsg{} }

		case key.Matches(msg, m.keys.MoveDown):
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, m.keys.MoveUp):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, m.keys.ToggleFold):
			if m.cursor < len(m.folded) {
				m.folded[m.cursor] = !m.folded[m.cursor]
			}
			return m, nil
		}
	}

	return m, nil
}
