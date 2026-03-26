package stashlist

import "github.com/mhersson/termagit/internal/git"

// CloseStashListMsg signals that the stash list view should close.
type CloseStashListMsg struct{}

// StashDroppedMsg carries the result of a stash drop.
type StashDroppedMsg struct {
	Index int
	Err   error
}

// StashesRefreshedMsg carries refreshed stash data.
type StashesRefreshedMsg struct {
	Stashes []git.StashEntry
	Err     error
}
