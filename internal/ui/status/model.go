package status

import (
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/notification"
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

// PopupKind identifies which popup is active.
type PopupKind int

const (
	PopupNone PopupKind = iota
	PopupCommit
	PopupBranch
	PopupPush
	PopupPull
	PopupFetch
	PopupMerge
	PopupRebase
	PopupRevert
	PopupCherryPick
	PopupReset
	PopupStash
	PopupTag
	PopupRemote
	PopupWorktree
	PopupBisect
	PopupIgnore
	PopupDiff
	PopupLog
	PopupMargin
	PopupHelp
)

// commitSpecialKind identifies which special commit action is pending commit selection.
type commitSpecialKind int

const (
	commitSpecialNone         commitSpecialKind = iota
	commitSpecialFixup                          // f: --fixup=<sha>, no editor
	commitSpecialSquash                         // s: --squash=<sha>, no editor
	commitSpecialAugment                        // n: --squash=<sha>, with editor
	commitSpecialAlter                          // A: --fixup=amend:<sha>, with editor
	commitSpecialRevise                         // W: --fixup=reword:<sha>, with editor
	commitSpecialInstantFixup                   // F: --fixup=<sha> + autosquash
	commitSpecialInstantSquash                  // S: --squash=<sha> + autosquash
)

// rebaseSpecialKind identifies which rebase action is pending commit selection.
type rebaseSpecialKind int

const (
	rebaseSpecialNone        rebaseSpecialKind = iota
	rebaseSpecialInteractive                   // i: interactive rebase from selected commit
	rebaseSpecialSubset                        // s: rebase a subset
	rebaseSpecialModify                        // m: modify a commit (edit)
	rebaseSpecialReword                        // w: reword a commit
	rebaseSpecialDrop                          // d: drop a commit
)

// cursorRestore holds info to restore cursor position after a status reload.
type cursorRestore struct {
	active      bool
	path        string      // file path to find
	sectionKind SectionKind // which section to look in after the operation
	itemIndex   int         // original item index for clamping
	hunk        int         // hunk index (-1 = on file)
}

// hunkRestore holds info to place the cursor on a hunk once hunks finish loading.
type hunkRestore struct {
	active     bool
	sectionIdx int
	itemIdx    int
	hunkIdx    int // target hunk index (clamped to available hunks)
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
	popup     *popup.Popup
	popupKind PopupKind

	// Confirmation state
	confirmMode ConfirmMode //nolint:unused // Phase 4 - used in update.go
	confirmPath string      //nolint:unused // Phase 4 - used in view.go
	confirmHunk int         //nolint:unused // Phase 4 - hunk index for ConfirmDiscardHunk

	// Cursor restore after stage/unstage/discard
	pendingRestore cursorRestore

	// Hunk-level cursor restore (two-phase: expand file, then place cursor after hunks load)
	pendingHunkRestore hunkRestore

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

	// Pending commit special action (waiting for commit select result)
	commitSpecialOpts  git.CommitOpts    // popup switches captured before commit select
	commitSpecialKind  commitSpecialKind // which special action initiated the select

	// Pending rebase special action (waiting for commit select result)
	rebaseSpecialKind rebaseSpecialKind // which rebase action initiated the select
	rebaseSpecialOpts git.RebaseOpts    // popup switches captured before commit select

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

// ConfirmView returns the rendered confirmation dialog if a confirmation is
// pending, or an empty string if no confirmation is active.
func (m Model) ConfirmView(maxWidth int) string {
	var msg string
	switch m.confirmMode {
	case ConfirmNone:
		return ""
	case ConfirmDiscard:
		msg = "Discard changes to " + m.confirmPath + "?"
	case ConfirmDiscardHunk:
		msg = "Discard hunk in " + m.confirmPath + "?"
	case ConfirmUntrack:
		msg = "Untrack " + m.confirmPath + "?"
	default:
		return ""
	}

	d := notification.ConfirmDialog{Message: msg}
	return d.View(m.tokens, maxWidth)
}
