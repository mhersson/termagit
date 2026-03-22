package rebaseeditor

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
)

// loadTodoCmd loads the rebase todo entries from the repository.
func loadTodoCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return todoLoadedMsg{Err: git.ErrNoRebaseInProgress}
		}
		state, err := repo.ReadRebaseTodo()
		if err != nil {
			return todoLoadedMsg{Err: err}
		}
		// Filter to only pending entries (not done) for editing
		var pending []git.TodoEntry
		for _, e := range state.Entries {
			if !e.Done {
				pending = append(pending, e)
			}
		}
		return todoLoadedMsg{Entries: pending, Err: nil}
	}
}

// submitRebaseCmd writes the modified entries and executes the rebase.
// If base is non-empty, uses RebaseWithTodo (new interactive rebase).
// Otherwise, writes the todo and continues an in-progress rebase.
func submitRebaseCmd(repo *git.Repository, entries []git.TodoEntry, base string, opts git.RebaseOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return rebaseSubmitResultMsg{Err: git.ErrNoRebaseInProgress}
		}
		ctx := context.Background()

		if base != "" {
			// New interactive rebase: run git rebase -i with the edited todo
			err := repo.RebaseWithTodo(ctx, base, entries, opts)
			return rebaseSubmitResultMsg{Err: err}
		}

		// Editing an in-progress rebase: write todo and continue
		if err := repo.WriteRebaseTodo(entries); err != nil {
			return rebaseSubmitResultMsg{Err: err}
		}
		err := repo.RebaseContinue(ctx)
		return rebaseSubmitResultMsg{Err: err}
	}
}

// abortRebaseCmd aborts the in-progress rebase.
func abortRebaseCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return rebaseAbortResultMsg{Err: git.ErrNoRebaseInProgress}
		}
		ctx := context.Background()
		err := repo.RebaseAbort(ctx)
		return rebaseAbortResultMsg{Err: err}
	}
}
