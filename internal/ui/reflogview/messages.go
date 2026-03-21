package reflogview

import tea "github.com/charmbracelet/bubbletea"

// OpenReflogViewMsg requests opening the reflog view.
type OpenReflogViewMsg struct {
	Ref string
}

// CloseReflogViewMsg signals that the reflog view should close.
type CloseReflogViewMsg struct{}

// YankMsg carries text to be yanked to clipboard.
type YankMsg struct {
	Text string
}

// OpenCommitViewMsg requests opening the commit view for a specific hash.
type OpenCommitViewMsg struct {
	Hash string
}

// OpenPopupMsg requests opening a popup from the reflog view.
type OpenPopupMsg struct {
	Type   string // popup type: "commit", "branch", etc.
	Commit string // commit hash for context
}

// OpenCommitLinkMsg requests opening a commit URL in the browser.
type OpenCommitLinkMsg struct {
	Hash string
}

// yankCmd returns a command that yanks text to clipboard.
func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		return YankMsg{Text: text}
	}
}
