package nav

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/ui/shared"
)

// HandleNavigationKey processes standard navigation keys.
// Returns (handled, cmd). Caller passes max (highest valid cursor position).
func HandleNavigationKey(msg tea.KeyMsg, c *Cursor, keys NavigationKeys, max int) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.MoveDown):
		c.MoveDown(1, max)
		return true, nil
	case key.Matches(msg, keys.MoveUp):
		c.MoveUp(1)
		return true, nil
	case key.Matches(msg, keys.PageDown):
		c.MoveDown(c.VisibleLines(), max)
		return true, nil
	case key.Matches(msg, keys.PageUp):
		c.MoveUp(c.VisibleLines())
		return true, nil
	case key.Matches(msg, keys.HalfPageDown):
		c.MoveDown(c.VisibleLines()/2, max)
		return true, nil
	case key.Matches(msg, keys.HalfPageUp):
		c.MoveUp(c.VisibleLines() / 2)
		return true, nil
	case key.Matches(msg, keys.GoToTop):
		c.PendingKey = "g"
		return true, nil
	case key.Matches(msg, keys.GoToBottom):
		c.GoToBottom(max)
		return true, nil
	}
	return false, nil
}

// HandlePopupKey processes popup trigger keys.
// commitHash is the hash/identifier of the item under cursor.
// Returns (handled, cmd).
func HandlePopupKey(msg tea.KeyMsg, keys PopupKeys, commitHash string) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.CherryPickPopup):
		return true, shared.OpenPopupCmd("cherry-pick", commitHash)
	case key.Matches(msg, keys.BranchPopup):
		return true, shared.OpenPopupCmd("branch", commitHash)
	case key.Matches(msg, keys.CommitPopup):
		return true, shared.OpenPopupCmd("commit", commitHash)
	case key.Matches(msg, keys.DiffPopup):
		return true, shared.OpenPopupCmd("diff", commitHash)
	case key.Matches(msg, keys.FetchPopup):
		return true, shared.OpenPopupCmd("fetch", commitHash)
	case key.Matches(msg, keys.MergePopup):
		return true, shared.OpenPopupCmd("merge", commitHash)
	case key.Matches(msg, keys.PullPopup):
		return true, shared.OpenPopupCmd("pull", commitHash)
	case key.Matches(msg, keys.PushPopup):
		return true, shared.OpenPopupCmd("push", commitHash)
	case key.Matches(msg, keys.RebasePopup):
		return true, shared.OpenPopupCmd("rebase", commitHash)
	case key.Matches(msg, keys.RevertPopup):
		return true, shared.OpenPopupCmd("revert", commitHash)
	case key.Matches(msg, keys.ResetPopup):
		return true, shared.OpenPopupCmd("reset", commitHash)
	case key.Matches(msg, keys.TagPopup):
		return true, shared.OpenPopupCmd("tag", commitHash)
	case key.Matches(msg, keys.BisectPopup):
		return true, shared.OpenPopupCmd("bisect", commitHash)
	case key.Matches(msg, keys.RemotePopup):
		return true, shared.OpenPopupCmd("remote", commitHash)
	case key.Matches(msg, keys.WorktreePopup):
		return true, shared.OpenPopupCmd("worktree", commitHash)
	case key.Matches(msg, keys.OpenCommitLink):
		if commitHash != "" {
			return true, func() tea.Msg { return shared.OpenCommitLinkMsg{Hash: commitHash} }
		}
		return true, nil
	}
	return false, nil
}
