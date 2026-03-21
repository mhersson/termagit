package reflogview

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

// Model is the reflog view model.
type Model struct {
	entries []git.ReflogEntry
	tokens  theme.Tokens
	keys    KeyMap
	header  string
	cursor  int
	offset  int

	pendingKey string // for "gg" sequence

	width  int
	height int
}

// New creates a new reflog view model.
func New(entries []git.ReflogEntry, tokens theme.Tokens, ref string) Model {
	header := "Reflog for " + ref
	if ref == "" || ref == "HEAD" {
		header = "Reflog for HEAD"
	}

	return Model{
		entries: entries,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		header:  header,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		newM, cmd := m.handleKey(msg)
		return newM, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Handle "gg" sequence
	if m.pendingKey == "g" {
		m.pendingKey = ""
		if msg.String() == "g" {
			m.cursor = 0
			m.offset = 0
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Close), key.Matches(msg, m.keys.CloseEscape):
		return m, func() tea.Msg { return CloseReflogViewMsg{} }

	case key.Matches(msg, m.keys.MoveDown):
		return m.moveDown(1), nil

	case key.Matches(msg, m.keys.MoveUp):
		return m.moveUp(1), nil

	case key.Matches(msg, m.keys.PageDown):
		return m.moveDown(m.visibleLines()), nil

	case key.Matches(msg, m.keys.PageUp):
		return m.moveUp(m.visibleLines()), nil

	case key.Matches(msg, m.keys.HalfPageDown):
		return m.moveDown(m.visibleLines() / 2), nil

	case key.Matches(msg, m.keys.HalfPageUp):
		return m.moveUp(m.visibleLines() / 2), nil

	case key.Matches(msg, m.keys.GoToTop):
		m.pendingKey = "g"
		return m, nil

	case key.Matches(msg, m.keys.GoToBottom):
		return m.goToBottom(), nil

	case key.Matches(msg, m.keys.Yank):
		if len(m.entries) > 0 && m.cursor < len(m.entries) {
			hash := m.entries[m.cursor].Oid[:7]
			return m, yankCmd(hash)
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if len(m.entries) > 0 && m.cursor < len(m.entries) {
			hash := m.entries[m.cursor].Oid
			return m, func() tea.Msg { return OpenCommitViewMsg{Hash: hash} }
		}
		return m, nil
	}

	return m, nil
}

func (m Model) moveDown(n int) Model {
	max := len(m.entries) - 1
	if max < 0 {
		return m
	}

	m.cursor += n
	if m.cursor > max {
		m.cursor = max
	}
	m.ensureVisible()
	return m
}

func (m Model) moveUp(n int) Model {
	m.cursor -= n
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
	return m
}

func (m Model) goToBottom() Model {
	max := len(m.entries) - 1
	if max >= 0 {
		m.cursor = max
		m.ensureVisible()
	}
	return m
}

func (m Model) visibleLines() int {
	// Reserve 2 lines for header
	v := m.height - 2
	if v < 1 {
		return 1
	}
	return v
}

func (m *Model) ensureVisible() {
	vis := m.visibleLines()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
