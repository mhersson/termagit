package stashlist

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
)

// CloseStashListMsg signals that the stash list view should close.
type CloseStashListMsg struct{}

// YankMsg carries text to be yanked to clipboard.
type YankMsg struct {
	Text string
}

// OpenCommitViewMsg requests opening the commit view for a stash.
type OpenCommitViewMsg struct {
	Hash string
}

// OpenPopupMsg requests opening a popup from the stash list view.
type OpenPopupMsg struct {
	Type   string // popup type
	Commit string // stash name for context
}

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

// yankCmd returns a command that yanks text to clipboard.
func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return YankMsg{Text: text}
	}
}
