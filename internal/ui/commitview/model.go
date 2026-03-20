package commitview

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

// Model is the commit view model for displaying commit details.
type Model struct {
	repo   *git.Repository
	tokens theme.Tokens
	keys   KeyMap

	commitID string   // passed in (hash or ref)
	filter   []string // optional file path filter

	info      *git.LogEntry
	overview  *git.CommitOverview
	signature *git.CommitSignature
	diffs     []git.FileDiff

	viewport viewport.Model
	loading  bool
	ready    bool
	width    int
	height   int
	err      error
}

// New creates a new commit view model.
func New(repo *git.Repository, commitID string, tokens theme.Tokens, filter []string) Model {
	return Model{
		repo:     repo,
		tokens:   tokens,
		keys:     DefaultKeyMap(),
		commitID: commitID,
		filter:   filter,
		loading:  true,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.loadCommitDataCmd()
}

// CommitID returns the current commit ID.
func (m Model) CommitID() string {
	return m.commitID
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Reserve lines for header (we'll calculate properly in view)
	headerHeight := 8 // approximate header height
	m.viewport = viewport.New(width, height-headerHeight)
}

// UpdateCommit changes to a different commit (singleton support).
func (m *Model) UpdateCommit(commitID string, filter []string) tea.Cmd {
	if m.commitID == commitID {
		return nil
	}
	m.commitID = commitID
	m.filter = filter
	m.loading = true
	m.ready = false
	m.info = nil
	m.overview = nil
	m.signature = nil
	m.diffs = nil
	return m.loadCommitDataCmd()
}
