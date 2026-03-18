package rebaseeditor

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

// Model is the rebase editor model.
type Model struct {
	repo   *git.Repository
	tokens theme.Tokens
	keys   KeyMap

	entries    []git.TodoEntry
	cursor     int
	base       string         // base commit for the rebase (used on submit)
	rebaseOpts git.RebaseOpts // options from the popup (switches)

	// Two-key sequence state
	pendingKey string // "ctrl+c" for <c-c><c-c> / <c-c><c-k>
	pendingG   bool   // true after 'g' pressed (for gk/gj)
	pendingZ   bool   // true after 'Z' pressed (for ZZ/ZQ)
	pendingOSU bool   // true after '[' pressed (for [c)
	pendingOSD bool   // true after ']' pressed (for ]c)

	loading bool
	done    bool
	aborted bool
	err     error

	width  int
	height int
}

// New creates a new rebase editor model.
func New(repo *git.Repository, tokens theme.Tokens) Model {
	return Model{
		repo:    repo,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		loading: true,
	}
}

// NewWithEntries creates a rebase editor pre-loaded with entries (for interactive rebase).
func NewWithEntries(repo *git.Repository, tokens theme.Tokens, entries []git.TodoEntry, base string, opts git.RebaseOpts) Model {
	return Model{
		repo:       repo,
		tokens:     tokens,
		keys:       DefaultKeyMap(),
		entries:    entries,
		base:       base,
		rebaseOpts: opts,
		loading:    false,
	}
}

// Init initializes the model and starts loading the rebase todo.
func (m Model) Init() tea.Cmd {
	return loadTodoCmd(m.repo)
}

// SetSize sets the editor dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
