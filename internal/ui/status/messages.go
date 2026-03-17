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
