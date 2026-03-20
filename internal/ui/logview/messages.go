package logview

import (
	tea "github.com/charmbracelet/bubbletea"
)

// OpenLogViewMsg requests opening the log view.
type OpenLogViewMsg struct {
	Branch string
}

// CloseLogViewMsg signals that the log view should close.
type CloseLogViewMsg struct{}

// LoadMoreMsg requests loading more commits.
type LoadMoreMsg struct{}

// CommitsLoadedMsg delivers newly loaded commits.
type CommitsLoadedMsg struct {
	Commits []logCommit
	HasMore bool
	Err     error
}

// logCommit is an internal type for loaded commits.
type logCommit struct {
	hash, abbrevHash, subject, authorName string
}

// yankCmd returns a command that yanks text to clipboard.
func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return YankMsg{Text: text}
	}
}

// YankMsg carries text to be yanked to clipboard.
type YankMsg struct {
	Text string
}
