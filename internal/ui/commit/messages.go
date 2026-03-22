package commit

import "github.com/mhersson/termagit/internal/git"

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

// commentCharLoadedMsg delivers the git comment character.
type commentCharLoadedMsg struct {
	Char string
	Err  error
}

// branchLoadedMsg delivers the current branch name.
type branchLoadedMsg struct {
	Branch string
	Err    error
}

// statusLoadedMsg delivers the git status for the template.
type statusLoadedMsg struct {
	Status *git.StatusResult
	Err    error
}

// generateCommitMessageMsg delivers the result of an external commit message generator.
type generateCommitMessageMsg struct {
	Message string
	Err     error
}
