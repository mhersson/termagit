package diffview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
)

// OpenDiffViewMsg triggers opening the diff view at the app level.
type OpenDiffViewMsg struct {
	Source DiffSource
}

// CloseDiffViewMsg is sent when the diff view should be closed.
type CloseDiffViewMsg struct{}

// DiffDataLoadedMsg is sent when diff data has been loaded.
type DiffDataLoadedMsg struct {
	Files []git.FileDiff
	Stats *git.CommitOverview
	Err   error
}

// HunkStagedMsg is sent after a hunk has been staged or unstaged.
type HunkStagedMsg struct {
	Err error
}

// Ensure messages implement tea.Msg.
var (
	_ tea.Msg = OpenDiffViewMsg{}
	_ tea.Msg = CloseDiffViewMsg{}
	_ tea.Msg = DiffDataLoadedMsg{}
	_ tea.Msg = HunkStagedMsg{}
)
