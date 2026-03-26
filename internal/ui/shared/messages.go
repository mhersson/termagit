package shared

import tea "github.com/charmbracelet/bubbletea"

// YankMsg requests copying text to the clipboard.
type YankMsg struct{ Text string }

// OpenPopupMsg requests opening a popup of the given type.
type OpenPopupMsg struct {
	Type   string
	Commit string
}

// OpenCommitViewMsg requests opening the commit view for a hash.
type OpenCommitViewMsg struct{ Hash string }

// OpenCommitLinkMsg requests opening a commit URL in a browser.
type OpenCommitLinkMsg struct{ Hash string }

// YankCmd returns a command that emits YankMsg.
func YankCmd(text string) tea.Cmd {
	return func() tea.Msg { return YankMsg{Text: text} }
}

// OpenPopupCmd returns a command that emits OpenPopupMsg.
func OpenPopupCmd(popupType, commit string) tea.Cmd {
	return func() tea.Msg { return OpenPopupMsg{Type: popupType, Commit: commit} }
}
