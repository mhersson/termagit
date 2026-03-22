package status

import (
	"github.com/mhersson/conjit/internal/git"
)

// statusLoadedMsg is sent when status loading completes.
type statusLoadedMsg struct {
	head     HeadState
	sections []Section
	err      error
}

// hunksLoadedMsg is sent when hunks for an item are loaded.
type hunksLoadedMsg struct {
	sectionIdx int
	itemIdx    int
	hunks      []git.Hunk
	err        error
}

// operationDoneMsg is sent when a git operation completes.
type operationDoneMsg struct {
	err error
	op  string // e.g., "Push", "Pull", "Fetch"
}

// notificationExpiredMsg is sent when a notification should be cleared.
type notificationExpiredMsg struct{}

// repoChangedMsg is sent by the file watcher when the repo changes.
type repoChangedMsg struct{}

// peekFileMsg is sent when file content is loaded for peek preview.
type peekFileMsg struct {
	path    string
	content string
	err     error
}

// closePeekMsg is sent when the peek pane should be closed.
type closePeekMsg struct{}

// branchesLoadedMsg is sent when branch listing completes.
type branchesLoadedMsg struct {
	branches []git.Branch
	err      error
}

// OpenCmdHistoryMsg is sent when the user presses $ to open command history.
type OpenCmdHistoryMsg struct{}

// OpenLogViewMsg is sent to open the log view with the given commits.
type OpenLogViewMsg struct {
	Commits []git.LogEntry
	HasMore bool
	Branch  string
	Opts    *git.LogOpts
}

// OpenReflogViewMsg is sent to open the reflog view with the given entries.
type OpenReflogViewMsg struct {
	Entries []git.ReflogEntry
	Ref     string
}

// remoteConfigLoadedMsg is sent when remote config values are loaded for the remote config popup.
type remoteConfigLoadedMsg struct {
	remote string
	values map[string]string // config key -> value
	err    error
}

// branchConfigLoadedMsg is sent when branch config values are loaded for the branch config popup.
type branchConfigLoadedMsg struct {
	branch          string
	values          map[string]string // config key -> value
	remotes         []string          // configured remote names
	pullRebase      string            // local pull.rebase value
	globalPullRebase string           // global pull.rebase value
	err             error
}
