package refsview

import tea "github.com/charmbracelet/bubbletea"

// CloseRefsViewMsg signals that the refs view should close.
type CloseRefsViewMsg struct{}

// YankMsg carries text to be yanked to clipboard.
type YankMsg struct {
	Text string
}

// OpenCommitViewMsg requests opening the commit view for a specific hash.
type OpenCommitViewMsg struct {
	Hash string
}

// OpenPopupMsg requests opening a popup from the refs view.
type OpenPopupMsg struct {
	Type   string // popup type: "commit", "branch", etc.
	Commit string // commit hash for context
}

// DeleteBranchMsg carries the result of a branch deletion attempt.
type DeleteBranchMsg struct {
	Err error
}

// RefsRefreshedMsg carries refreshed refs data after a mutation.
type RefsRefreshedMsg struct {
	Err error
}

// yankCmd returns a command that yanks text to clipboard.
func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return YankMsg{Text: text}
	}
}
