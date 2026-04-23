package reflogview

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/nav"
	"github.com/mhersson/termagit/internal/ui/shared"
)

// Model is the reflog view model.
type Model struct {
	entries []git.ReflogEntry
	tokens  theme.Tokens
	navKeys nav.NavigationKeys
	popKeys nav.PopupKeys
	header  string
	cursor  nav.Cursor
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
		navKeys: nav.DefaultNavigationKeys(),
		popKeys: nav.DefaultPopupKeys(),
		header:  header,
		cursor:  nav.NewCursor(2),
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
		m.cursor.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.cursor.HandleGG(msg.String()) {
		return m, nil
	}

	max := len(m.entries) - 1
	if handled, cmd := nav.HandleNavigationKey(msg, &m.cursor, m.navKeys, max); handled {
		return m, cmd
	}
	if handled, cmd := nav.HandlePopupKey(msg, m.popKeys, m.currentHash()); handled {
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.navKeys.Close), key.Matches(msg, m.navKeys.CloseEscape):
		return m, func() tea.Msg { return CloseReflogViewMsg{} }

	case key.Matches(msg, m.navKeys.Yank):
		if len(m.entries) > 0 && m.cursor.Pos < len(m.entries) {
			hash := m.entries[m.cursor.Pos].Oid[:7]
			return m, shared.YankCmd(hash)
		}
		return m, nil

	case key.Matches(msg, m.navKeys.Select):
		if len(m.entries) > 0 && m.cursor.Pos < len(m.entries) {
			hash := m.entries[m.cursor.Pos].Oid
			return m, func() tea.Msg { return shared.OpenCommitViewMsg{Hash: hash} }
		}
		return m, nil
	}

	return m, nil
}

func (m Model) currentHash() string {
	if len(m.entries) > 0 && m.cursor.Pos < len(m.entries) {
		return m.entries[m.cursor.Pos].Oid
	}
	return ""
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.cursor.SetSize(width, height)
}
