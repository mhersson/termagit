package refsview

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/git"
)

// deleteBranchCmd deletes a local branch and returns the result.
func deleteBranchCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := repo.DeleteBranch(ctx, name, false)
		return DeleteBranchMsg{Err: err}
	}
}

// refreshRefsCmd reloads refs data after a mutation.
func refreshRefsCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := repo.ListRefs(ctx)
		return RefsRefreshedMsg{Err: err}
	}
}
