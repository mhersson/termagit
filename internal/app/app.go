package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/cmdlog"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/theme"
)

// Model is the main application model.
type Model struct {
	cfg    *config.Config
	tokens theme.Tokens
	logger *cmdlog.Logger
	width  int
	height int
}

// New creates a new application model.
func New(cfg *config.Config, tokens theme.Tokens, logger *cmdlog.Logger) Model {
	return Model{
		cfg:    cfg,
		tokens: tokens,
		logger: logger,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View renders the model.
func (m Model) View() string {
	return "conjit initialising..."
}
