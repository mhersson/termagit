package cmdhistory

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/cmdlog"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/nav"
)

// CloseMsg signals that the command history view should be closed.
type CloseMsg struct{}

// Model is the command history view model.
type Model struct {
	entries []cmdlog.Entry
	folded  []bool
	cursor  nav.Cursor
	navKeys nav.NavigationKeys
	tokens  theme.Tokens
}

// New creates a new command history model from the given entries.
func New(entries []cmdlog.Entry, tokens theme.Tokens, width, height int) Model {
	folded := make([]bool, len(entries))
	for i := range folded {
		folded[i] = true // collapsed by default
	}
	c := nav.NewCursor(2) // 2 header rows: title + blank line
	c.SetSize(width, height)
	return Model{
		entries: entries,
		folded:  folded,
		cursor:  c,
		navKeys: nav.DefaultNavigationKeys(),
		tokens:  tokens,
	}
}

// Init returns no initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.cursor.SetSize(width, height)
}

// entryHeight returns the visual line count for entry at idx.
func entryHeight(m *Model, idx int) int {
	if idx < 0 || idx >= len(m.entries) {
		return 0
	}
	if m.folded[idx] {
		return 1
	}
	h := 1 // header row
	output := m.entries[idx].Stdout + m.entries[idx].Stderr
	if output != "" {
		h += countLines(output)
	}
	if m.entries[idx].Error != "" {
		h += countLines(m.entries[idx].Error)
	}
	h++ // trailing blank line
	return h
}

// Update handles messages for the command history view.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.cursor.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.cursor.HandleGG(msg.String()) {
		ensureCursorVisible(&m)
		return m, nil
	}

	max := len(m.entries) - 1
	if handled, cmd := nav.HandleNavigationKey(msg, &m.cursor, m.navKeys, max); handled {
		ensureCursorVisible(&m)
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.navKeys.Close), key.Matches(msg, m.navKeys.CloseEscape):
		return m, func() tea.Msg { return CloseMsg{} }

	case key.Matches(msg, toggleFoldKey):
		if m.cursor.Pos < len(m.folded) {
			m.folded[m.cursor.Pos] = !m.folded[m.cursor.Pos]
		}
		ensureCursorVisible(&m)
		return m, nil
	}

	return m, nil
}

// ensureCursorVisible adjusts the scroll offset so the cursor entry is visible,
// accounting for variable-height entries (expanded entries consume multiple lines).
func ensureCursorVisible(m *Model) {
	vis := m.cursor.VisibleLines()
	if vis <= 0 {
		return
	}

	// If cursor is above offset, snap offset to cursor.
	if m.cursor.Pos < m.cursor.Offset {
		m.cursor.Offset = m.cursor.Pos
		return
	}

	// Walk from Offset to Pos summing visual heights.
	linesUsed := 0
	for i := m.cursor.Offset; i <= m.cursor.Pos && i < len(m.entries); i++ {
		linesUsed += entryHeight(m, i)
	}

	// If total exceeds viewport, increase Offset until it fits.
	for linesUsed > vis && m.cursor.Offset < m.cursor.Pos {
		linesUsed -= entryHeight(m, m.cursor.Offset)
		m.cursor.Offset++
	}
}
