package stashlist

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
)

// dropStashCmd drops a stash entry by index.
func dropStashCmd(repo *git.Repository, index int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := repo.StashDrop(ctx, index)
		return StashDroppedMsg{Index: index, Err: err}
	}
}

// refreshStashesCmd reloads the stash list.
func refreshStashesCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		stashes, err := repo.ListStashes(ctx)
		return StashesRefreshedMsg{Stashes: stashes, Err: err}
	}
}
