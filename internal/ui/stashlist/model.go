package stashlist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
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
	keys    KeyMap
	stashes []git.StashEntry
	cursor  int
	offset  int

	confirmMode confirmMode
	confirmIdx  int // stash index being confirmed

	pendingKey string

	width  int
	height int
}

// New creates a new stash list view model.
func New(stashes []git.StashEntry, repo *git.Repository, tokens theme.Tokens) Model {
	return Model{
		repo:    repo,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		stashes: stashes,
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
			if m.cursor >= len(m.stashes) {
				m.cursor = len(m.stashes) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
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
		return m, func() tea.Msg { return CloseStashListMsg{} }

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

	case key.Matches(msg, m.keys.Select):
		if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
			name := m.stashes[m.cursor].Name
			return m, func() tea.Msg { return OpenCommitViewMsg{Hash: name} }
		}
		return m, nil

	case key.Matches(msg, m.keys.Discard):
		if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
			m.confirmMode = confirmDropStash
			m.confirmIdx = m.stashes[m.cursor].Index
		}
		return m, nil

	case key.Matches(msg, m.keys.Yank):
		if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
			return m, yankCmd(m.stashes[m.cursor].Name)
		}
		return m, nil

	// Popup triggers
	case key.Matches(msg, m.keys.CherryPickPopup):
		return m, m.openPopupCmd("cherry-pick")
	case key.Matches(msg, m.keys.BranchPopup):
		return m, m.openPopupCmd("branch")
	case key.Matches(msg, m.keys.CommitPopup):
		return m, m.openPopupCmd("commit")
	case key.Matches(msg, m.keys.DiffPopup):
		return m, m.openPopupCmd("diff")
	case key.Matches(msg, m.keys.FetchPopup):
		return m, m.openPopupCmd("fetch")
	case key.Matches(msg, m.keys.MergePopup):
		return m, m.openPopupCmd("merge")
	case key.Matches(msg, m.keys.PullPopup):
		return m, m.openPopupCmd("pull")
	case key.Matches(msg, m.keys.PushPopup):
		return m, m.openPopupCmd("push")
	case key.Matches(msg, m.keys.RebasePopup):
		return m, m.openPopupCmd("rebase")
	case key.Matches(msg, m.keys.RevertPopup):
		return m, m.openPopupCmd("revert")
	case key.Matches(msg, m.keys.ResetPopup):
		return m, m.openPopupCmd("reset")
	case key.Matches(msg, m.keys.TagPopup):
		return m, m.openPopupCmd("tag")
	case key.Matches(msg, m.keys.BisectPopup):
		return m, m.openPopupCmd("bisect")
	case key.Matches(msg, m.keys.RemotePopup):
		return m, m.openPopupCmd("remote")
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

// openPopupCmd returns a command that emits an OpenPopupMsg.
func (m Model) openPopupCmd(popupType string) tea.Cmd {
	name := ""
	if len(m.stashes) > 0 && m.cursor < len(m.stashes) {
		name = m.stashes[m.cursor].Name
	}
	return func() tea.Msg {
		return OpenPopupMsg{Type: popupType, Commit: name}
	}
}

func (m Model) moveDown(n int) Model {
	max := len(m.stashes) - 1
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
	max := len(m.stashes) - 1
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

// ConfirmMessage returns the confirmation message if pending.
func (m Model) ConfirmMessage() string {
	if m.confirmMode == confirmDropStash {
		return fmt.Sprintf("Drop stash@{%d}?", m.confirmIdx)
	}
	return ""
}
