package commitview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
)

// OpenCommitViewMsg triggers opening the commit view at the app level.
type OpenCommitViewMsg struct {
	CommitID string   // commit hash or ref
	Filter   []string // optional file path filter
}

// CloseCommitViewMsg is sent when the commit view should be closed.
type CloseCommitViewMsg struct{}

// CommitDataLoadedMsg is sent when commit data has been loaded.
type CommitDataLoadedMsg struct {
	Info      *git.LogEntry
	Overview  *git.CommitOverview
	Signature *git.CommitSignature
	Diffs     []git.FileDiff
	Err       error
}

// ScrollCommitViewMsg requests scrolling the commit view.
type ScrollCommitViewMsg struct {
	Direction int // -1 up, +1 down
}

// Ensure messages implement tea.Msg
var (
	_ tea.Msg = OpenCommitViewMsg{}
	_ tea.Msg = CloseCommitViewMsg{}
	_ tea.Msg = CommitDataLoadedMsg{}
	_ tea.Msg = ScrollCommitViewMsg{}
)
