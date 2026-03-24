package diffview

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

// DiffSource describes what is being diffed.
type DiffSource struct {
	Kind   git.DiffKind
	Path   string // for single-file diffs
	Commit string // for commit diffs (full hash)
	Range  string // for range diffs e.g. "main..feature"
	Stash  string // for stash diffs e.g. "stash@{0}"
}

// Model is the diff view model for displaying diffs.
type Model struct {
	repo   *git.Repository
	cfg    *config.Config
	tokens theme.Tokens
	keys   KeyMap
	source DiffSource

	viewport viewport.Model
	loading  bool

	files   []git.FileDiff
	fileIdx int
	hunkIdx int // -1 = none highlighted

	header string             // e.g. "Staged changes"
	stats  *git.CommitOverview // stat block (nil for staged/unstaged)

	cursorLine   int
	cursorCol    int
	totalLines   int
	xOffset      int
	maxLineWidth int
	pendingKey   string // for "gg" sequence

	width  int
	height int
	err    error
	done   bool
}

// New creates a new diff view model.
func New(repo *git.Repository, source DiffSource, cfg *config.Config, tokens theme.Tokens) Model {
	header := headerForSource(source)
	return Model{
		repo:    repo,
		cfg:     cfg,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		source:  source,
		header:  header,
		loading: true,
		hunkIdx: -1,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.loadDiffCmd()
}

// Source returns the diff source.
func (m Model) Source() DiffSource {
	return m.source
}

// Done returns whether the view should be closed.
func (m Model) Done() bool {
	return m.done
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport = viewport.New(width, height)
}

// headerForSource derives the header text from a DiffSource.
func headerForSource(source DiffSource) string {
	switch source.Kind {
	case git.DiffStaged:
		return "Staged changes"
	case git.DiffUnstaged:
		return "Unstaged changes"
	case git.DiffCommit:
		h := source.Commit
		if len(h) > 7 {
			h = h[:7]
		}
		return "Commit " + h
	case git.DiffRange:
		return "Changes " + source.Range
	case git.DiffStash:
		return "Stash " + source.Stash
	default:
		return "Diff"
	}
}
