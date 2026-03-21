package status

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/commitview"
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
	PopupYank
	PopupRemoteConfig
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

// branchActionKind identifies which branch action is pending branch selection.
type branchActionKind int

const (
	branchActionNone          branchActionKind = iota
	branchActionCheckout                       // b: checkout branch/revision
	branchActionCheckoutLocal                  // l: checkout local branch
	branchActionCheckoutRecent                 // r: checkout recent branch
	branchActionDelete                         // D: delete branch
	branchActionPushElsewhere                  // Push popup: e
	branchActionPushOther                      // Push popup: o (source branch)
	branchActionRebaseElsewhere                // Rebase popup: e
	branchActionLogOtherBranch                 // Log popup: o
	branchActionBranchConfigure                // Branch popup: C (select branch to configure)
	branchActionMergeBranch                    // Merge popup: m/e/n/a/s/i (select branch to merge)
	branchActionWorktreeCheckout               // Worktree popup: w (select branch for worktree checkout)
	branchActionDonate                         // Cherry-pick popup: d (select branch to donate to)
)

// inputPromptKind identifies which action is pending text input.
type inputPromptKind int

const (
	inputPromptNone             inputPromptKind = iota
	inputPromptNewBranchCheckout                // c: new branch + checkout
	inputPromptNewBranch                        // n: new branch no checkout
	inputPromptSpinOff                          // s: spin-off
	inputPromptSpinOut                          // S: spin-out
	inputPromptRename                           // m: rename current branch
	inputPromptRenameFile                       // R: rename file
	inputPromptPushRefspec                      // Push: explicit refspec
	inputPromptPushTag                          // Push: tag name
	inputPromptReflogRef                        // Log: other reflog ref
	inputPromptWorktreePath                     // Branch: worktree path
	inputPromptTagName                          // Tag: name for new tag
	inputPromptTagRelease                       // Tag: release tag name
	inputPromptTagDelete                        // Tag: tag name to delete
	inputPromptRemoteName                       // Remote: name for new remote
	inputPromptRemoteURL                        // Remote: URL for new remote
	inputPromptRemoteRename                     // Remote: new name for rename
	inputPromptRemoteRemove                     // Remote: name to remove
	inputPromptRemotePrune                      // Remote: name to prune
	inputPromptRemoteConfigure                  // Remote: name to configure
	inputPromptWorktreeCreate                   // Worktree: create path
	inputPromptWorktreeMove                     // Worktree: move destination
	inputPromptWorktreeDelete                   // Worktree: delete path
	inputPromptBisectScript                     // Bisect: script path
	inputPromptStashMessage                     // Stash: push with message
	inputPromptStashRename                      // Stash: rename
	inputPromptStashBranch                      // Stash: branch name
)

// cherryPickActionKind identifies which cherry-pick action is pending commit selection.
type cherryPickActionKind int

const (
	cherryPickActionNone    cherryPickActionKind = iota
	cherryPickActionPick                         // A: pick commits
	cherryPickActionApply                        // a: apply (no commit)
	cherryPickActionHarvest                      // h: harvest
	cherryPickActionSquash                       // m: squash
	cherryPickActionDonate                       // d: donate
	cherryPickActionSpinout                      // n: spinout
	cherryPickActionSpinoff                      // s: spinoff
)

// revertActionKind identifies which revert action is pending commit selection.
type revertActionKind int

const (
	revertActionNone    revertActionKind = iota
	revertActionCommit                   // v: revert commits
	revertActionChanges                  // V: revert changes (no commit)
)

// resetActionKind identifies which reset action is pending commit selection.
type resetActionKind int

const (
	resetActionNone resetActionKind = iota
	resetActionBranch               // b: reset branch to commit
)

// mergeActionKind identifies which merge action variant is active.
type mergeActionKind int

const (
	mergeActionNone     mergeActionKind = iota
	mergeActionMerge                    // m: merge
	mergeActionEdit                     // e: merge + edit message
	mergeActionNoCommit                 // n: merge --no-commit
	mergeActionAbsorb                   // a: absorb
	mergeActionSquash                   // s: squash merge
	mergeActionDissolve                 // i: dissolve
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
	viewport viewport.Model

	loading bool

	// Active popup (nil = none)
	popup     *popup.Popup
	popupKind PopupKind

	// Confirmation state
	confirmMode ConfirmMode
	confirmPath string
	confirmHunk int

	// Cursor restore after stage/unstage/discard
	pendingRestore cursorRestore

	// Hunk-level cursor restore (two-phase: expand file, then place cursor after hunks load)
	pendingHunkRestore hunkRestore

	// Peek file preview state
	peekActive   bool
	peekPath     string
	peekContent  string
	peekViewport viewport.Model

	// Notification bar
	notification string

	// Pending key for multi-key sequences (e.g., "gg")
	pendingKey string

	// Commit view overlay (nil = no commit view)
	commitView *commitview.Model

	// Pending commit special action (waiting for commit select result)
	commitSpecialOpts  git.CommitOpts    // popup switches captured before commit select
	commitSpecialKind  commitSpecialKind // which special action initiated the select

	// Pending rebase special action (waiting for commit select result)
	rebaseSpecialKind rebaseSpecialKind // which rebase action initiated the select
	rebaseSpecialOpts git.RebaseOpts    // popup switches captured before commit select

	// Pending branch action (waiting for branch select result)
	branchActionKind  branchActionKind
	mergeActionKind   mergeActionKind   // which merge variant is pending branch select
	mergeOpts         git.MergeOpts     // merge opts captured before branch select

	// Pending cherry-pick action (waiting for commit select result)
	cherryPickActionKind cherryPickActionKind
	cherryPickOpts       git.CherryPickOpts
	donateHashes         []string // hashes to donate (set after commit select, cleared after branch select)

	// Pending revert action (waiting for commit select result)
	revertActionKind revertActionKind
	revertOpts       git.RevertOpts

	// Pending reset action (waiting for commit select result)
	resetActionKind resetActionKind
	resetMode       git.ResetMode

	// Pending tag options (captured from popup before text input)
	tagOpts git.TagOpts

	// Inline text input prompt (for branch name entry)
	inputPromptKind  inputPromptKind
	inputPromptLabel string
	inputPrompt      textinput.Model

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

// InputPromptView returns the rendered input prompt overlay, or "" if no prompt is active.
func (m Model) InputPromptView(maxWidth int) string {
	if m.inputPromptKind == inputPromptNone {
		return ""
	}

	label := m.inputPromptLabel + m.inputPrompt.View()
	// Pad and style as a dialog box
	d := notification.ConfirmDialog{Message: label}
	return d.View(m.tokens, maxWidth)
}
