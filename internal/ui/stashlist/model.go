package stashlist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/nav"
	"github.com/mhersson/termagit/internal/ui/shared"
)

// confirmMode indicates what type of confirmation is pending.
type confirmMode int

const (
	confirmNone     confirmMode = iota
	confirmDropStash
)

// Model is the stash list view model.
type Model struct {
	repo    *git.Repository
	tokens  theme.Tokens
	navKeys nav.NavigationKeys
	popKeys nav.PopupKeys
	Discard key.Binding
	stashes []git.StashEntry
	cursor  nav.Cursor

	confirmMode confirmMode
	confirmIdx  int // stash index being confirmed
}

// New creates a new stash list view model.
func New(stashes []git.StashEntry, repo *git.Repository, tokens theme.Tokens) Model {
	return Model{
		repo:    repo,
		tokens:  tokens,
		navKeys: nav.DefaultNavigationKeys(),
		popKeys: nav.DefaultPopupKeys(),
		Discard: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "drop stash"),
		),
		stashes: stashes,
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

	case StashDroppedMsg:
		m.confirmMode = confirmNone
		if msg.Err != nil {
			return m, nil
		}
		return m, refreshStashesCmd(m.repo)

	case StashesRefreshedMsg:
		if msg.Err == nil {
			m.stashes = msg.Stashes
			if m.cursor.Pos >= len(m.stashes) {
				m.cursor.Pos = len(m.stashes) - 1
			}
			if m.cursor.Pos < 0 {
				m.cursor.Pos = 0
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Handle confirmation mode
	if m.confirmMode != confirmNone {
		return m.handleConfirmKey(msg)
	}

	// Handle "gg" sequence
	if m.cursor.HandleGG(msg.String()) {
		return m, nil
	}

	// Navigation keys
	max := len(m.stashes) - 1
	if handled, cmd := nav.HandleNavigationKey(msg, &m.cursor, m.navKeys, max); handled {
		return m, cmd
	}

	// Popup trigger keys
	if handled, cmd := nav.HandlePopupKey(msg, m.popKeys, m.currentStashName()); handled {
		return m, cmd
	}

	// View-specific keys
	switch {
	case key.Matches(msg, m.navKeys.Close), key.Matches(msg, m.navKeys.CloseEscape):
		return m, func() tea.Msg { return CloseStashListMsg{} }

	case key.Matches(msg, m.navKeys.Select):
		if len(m.stashes) > 0 && m.cursor.Pos < len(m.stashes) {
			name := m.stashes[m.cursor.Pos].Name
			return m, func() tea.Msg { return shared.OpenCommitViewMsg{Hash: name} }
		}
		return m, nil

	case key.Matches(msg, m.Discard):
		if len(m.stashes) > 0 && m.cursor.Pos < len(m.stashes) {
			m.confirmMode = confirmDropStash
			m.confirmIdx = m.stashes[m.cursor.Pos].Index
		}
		return m, nil

	case key.Matches(msg, m.navKeys.Yank):
		if len(m.stashes) > 0 && m.cursor.Pos < len(m.stashes) {
			return m, shared.YankCmd(m.stashes[m.cursor.Pos].Name)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.confirmMode == confirmDropStash {
			return m, dropStashCmd(m.repo, m.confirmIdx)
		}
		m.confirmMode = confirmNone
		return m, nil
	case "n", "esc":
		m.confirmMode = confirmNone
		return m, nil
	}
	return m, nil
}

func (m Model) currentStashName() string {
	if len(m.stashes) > 0 && m.cursor.Pos < len(m.stashes) {
		return m.stashes[m.cursor.Pos].Name
	}
	return ""
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.cursor.SetSize(width, height)
}

// ConfirmMessage returns the confirmation message if pending.
func (m Model) ConfirmMessage() string {
	if m.confirmMode == confirmDropStash {
		return fmt.Sprintf("Drop stash@{%d}?", m.confirmIdx)
	}
	return ""
}
