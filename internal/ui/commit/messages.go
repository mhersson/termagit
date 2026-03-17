package commit

import "github.com/mhersson/conjit/internal/git"

// OpenCommitEditorMsg triggers opening the commit editor.
type OpenCommitEditorMsg struct {
	Opts   git.CommitOpts
	Action string // "commit", "amend", "reword", "extend", etc.
}

// CommitEditorDoneMsg signals commit completed.
type CommitEditorDoneMsg struct {
	Hash string
	Err  error
}

// CommitEditorAbortMsg signals user aborted.
type CommitEditorAbortMsg struct{}

// commitHistoryLoadedMsg delivers commit history for cycling.
type commitHistoryLoadedMsg struct {
	Messages []string
	Err      error
}

// stagedDiffLoadedMsg delivers staged diff for preview.
type stagedDiffLoadedMsg struct {
	Diff []git.FileDiff
	Err  error
}
