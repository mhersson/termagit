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

// confirmDiscardMsg is sent when the user presses x on a file or hunk.
//
//nolint:unused // Phase 4 - used in update.go
type confirmDiscardMsg struct {
	path    string
	isHunk  bool
	hunkIdx int
}

// confirmResultMsg is sent when user confirms or cancels a confirmation prompt.
//
//nolint:unused // Phase 4 - used in update.go
type confirmResultMsg struct {
	confirmed bool
}

// peekFileMsg is sent when file content is loaded for peek preview.
//
//nolint:unused // Phase 4 - used in update.go
type peekFileMsg struct {
	path    string
	content string
	err     error
}

// closePeekMsg is sent when the peek pane should be closed.
//
//nolint:unused // Phase 4 - used in update.go
type closePeekMsg struct{}

// OpenCmdHistoryMsg is sent when the user presses $ to open command history.
type OpenCmdHistoryMsg struct{}

// OpenLogViewMsg is sent to open the log view with the given commits.
type OpenLogViewMsg struct {
	Commits []git.LogEntry
	HasMore bool
	Branch  string
}

// OpenReflogViewMsg is sent to open the reflog view with the given entries.
type OpenReflogViewMsg struct {
	Entries []git.ReflogEntry
	Ref     string
}
