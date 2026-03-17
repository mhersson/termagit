package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/cmdlog"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/status"
)

// Screen represents the active screen.
type Screen int

const (
	ScreenStatus Screen = iota
	ScreenLog
	ScreenReflog
	ScreenCommitView
	ScreenRefsView
	ScreenStashList
	ScreenDiffView
	ScreenRebaseEditor
	ScreenCmdHistory
)

// SwitchScreenMsg is sent to switch to a different screen.
type SwitchScreenMsg struct {
	Screen     Screen
	CommitHash string // for ScreenCommitView
}

// Model is the main application model.
type Model struct {
	repo   *git.Repository
	cfg    *config.Config
	tokens theme.Tokens
	logger *cmdlog.Logger

	active Screen
	status status.Model

	width  int
	height int
}

// New creates a new application model.
func New(repo *git.Repository, cfg *config.Config, tokens theme.Tokens, logger *cmdlog.Logger) Model {
	keys := status.DefaultKeyMap()
	return Model{
		repo:   repo,
		cfg:    cfg,
		tokens: tokens,
		logger: logger,
		active: ScreenStatus,
		status: status.New(repo, cfg, tokens, keys),
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.status.Init()
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to active screen
		switch m.active {
		case ScreenStatus:
			var cmd tea.Cmd
			newStatus, cmd := m.status.Update(msg)
			m.status = newStatus.(status.Model)
			return m, cmd
		}
		return m, nil

	case SwitchScreenMsg:
		m.active = msg.Screen
		// Additional screen initialization will be added in future phases
		return m, nil
	}

	// Delegate to active screen
	switch m.active {
	case ScreenStatus:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		return m, cmd
	}

	return m, nil
}

// View renders the model.
func (m Model) View() string {
	switch m.active {
	case ScreenStatus:
		return m.status.View()
	default:
		return "Unknown screen"
	}
}
