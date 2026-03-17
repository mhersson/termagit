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
