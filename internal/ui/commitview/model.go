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

	// Cursor and overlay mode fields
	cursorLine  int  // current cursor position (0-indexed line)
	totalLines  int  // total navigable lines
	overlayMode bool // true when rendered as split overlay
	done        bool // true when view should close
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
	// Use full height for viewport - content is self-contained
	m.viewport = viewport.New(width, height)
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
	m.cursorLine = 0
	m.totalLines = 0
	m.done = false
	return m.loadCommitDataCmd()
}

// Done returns whether the view should be closed.
func (m Model) Done() bool {
	return m.done
}

// SetOverlayMode sets whether the view is rendered as a split overlay.
func (m *Model) SetOverlayMode(overlay bool) {
	m.overlayMode = overlay
}
