package status

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/popup"
)

// Tokens is an alias for theme.Tokens used in this package.
type Tokens = theme.Tokens

// ConfirmMode indicates what type of confirmation is pending.
type ConfirmMode int

const (
	ConfirmNone        ConfirmMode = iota // No confirmation pending
	ConfirmDiscard                        // Discard changes to a file
	ConfirmDiscardHunk                    // Discard a single hunk
	ConfirmUntrack                        // Untrack a file
)

// SectionKind is the type of a status buffer section.
// All 12 Neogit sections are defined here.
type SectionKind int

const (
	SectionSequencer        SectionKind = iota // cherry-pick / revert in progress
	SectionRebase                              // rebase in progress
	SectionBisect                              // bisect in progress
	SectionUntracked                           // untracked files
	SectionUnstaged                            // unstaged changes
	SectionStaged                              // staged changes
	SectionStashes                             // stashes
	SectionUnmergedUpstream                    // "Unmerged into"
	SectionUnpushedPushRemote                  // "Unpushed to"
	SectionRecentCommits                       // "Recent Commits"
	SectionUnpulledUpstream                    // "Unpulled from" (upstream)
	SectionUnpulledPushRemote                  // "Unpulled from" (push remote)
)

// HeadState holds everything needed to render the HEAD bar.
type HeadState struct {
	Branch    string
	Oid       string // full hash
	AbbrevOid string // 7 chars
	Subject   string
	Detached  bool

	UpstreamBranch  string
	UpstreamRemote  string
	UpstreamOid     string
	UpstreamSubject string

	PushBranch  string
	PushRemote  string
	PushOid     string
	PushSubject string

	Tag         string
	TagOid      string
	TagDistance int
}

// Section represents a section in the status buffer.
type Section struct {
	Kind   SectionKind
	Title  string // exact Neogit title string
	Folded bool
	Hidden bool // when true, section is not rendered at all
	Items  []Item
}

// Item represents an item within a section.
type Item struct {
	// File items (Untracked/Unstaged/Staged)
	Entry        *git.StatusEntry
	Expanded     bool
	Hunks        []git.Hunk
	HunksFolded  []bool // tracks fold state per hunk (true = folded, lines hidden)
	HunksLoading bool

	// Stash items
	Stash *git.StashEntry

	// Commit items (Recent/Unpulled/Unmerged)
	Commit *git.LogEntry

	// Sequencer/Rebase items
	Action        string
	ActionHash    string
	ActionSubject string
	ActionDone    bool
	ActionStopped bool // current position marker in rebase
}

// Cursor tracks the current position in the status buffer.
type Cursor struct {
	Section int // index into sections slice (only visible sections)
	Item    int // index into section's items (-1 = on section header)
	Hunk    int // index into item's Hunks (-1 = on file line)
	Line    int // index into hunk's Lines (-1 = on hunk header)
}

// Model is the status buffer model.
type Model struct {
	repo   *git.Repository
	cfg    *config.Config
	tokens Tokens
	keys   KeyMap
	width  int
	height int

	// Repo state (populated by loadStatusCmd)
	head     HeadState
	sections []Section
	cursor   Cursor
	viewport viewport.Model //nolint:unused // Phase 4

	loading     bool
	lastRefresh time.Time //nolint:unused // Phase 4

	// Active popup (nil = none)
	popup *popup.Popup

	// Confirmation state
	confirmMode ConfirmMode //nolint:unused // Phase 4 - used in update.go
	confirmPath string      //nolint:unused // Phase 4 - used in view.go
	confirmHunk int         //nolint:unused // Phase 4 - hunk index for ConfirmDiscardHunk

	// Peek file preview state
	peekActive   bool            //nolint:unused // Phase 4 - used in view.go
	peekPath     string          //nolint:unused // Phase 4 - used in view.go
	peekContent  string          //nolint:unused // Phase 4 - used in view.go
	peekViewport viewport.Model  //nolint:unused // Phase 4 - used in view.go

	// Notification bar
	notification string
	notifyExpiry time.Time //nolint:unused // Phase 4

	// Pending key for multi-key sequences (e.g., "gg")
	pendingKey string

	err error
}

// New creates a new status buffer model.
func New(repo *git.Repository, cfg *config.Config, tokens Tokens, keys KeyMap) Model {
	return Model{
		repo:   repo,
		cfg:    cfg,
		tokens: tokens,
		keys:   keys,
		cursor: Cursor{
			Section: 0,
			Item:    -1, // start on section header
			Hunk:    -1,
			Line:    -1,
		},
		loading: true,
	}
}

// Init initializes the model and starts loading status.
func (m Model) Init() tea.Cmd {
	if m.repo == nil {
		return nil
	}
	return loadStatusCmd(m.repo, m.cfg)
}

// Update handles messages - implementation in update.go.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return update(m, msg)
}

// View renders the model - implementation in view.go.
func (m Model) View() string {
	return view(m)
}
