package rebaseeditor

import "github.com/mhersson/conjit/internal/git"

// OpenRebaseEditorMsg triggers opening the rebase editor with pre-loaded entries.
// Used for new interactive rebases where we generate the todo from the commit range.
type OpenRebaseEditorMsg struct {
	Entries    []git.TodoEntry
	Base       string         // base commit for the rebase
	RebaseOpts git.RebaseOpts // popup switches
}

// RebaseEditorDoneMsg signals the rebase editor completed successfully.
type RebaseEditorDoneMsg struct {
	Err error
}

// RebaseEditorAbortMsg signals the user aborted the rebase editor.
type RebaseEditorAbortMsg struct{}

// todoLoadedMsg delivers loaded rebase todo entries.
type todoLoadedMsg struct {
	Entries []git.TodoEntry
	Err     error
}

// rebaseSubmitResultMsg delivers the result of submitting the rebase.
type rebaseSubmitResultMsg struct {
	Err error
}

// rebaseAbortResultMsg delivers the result of aborting the rebase.
type rebaseAbortResultMsg struct {
	Err error
}
