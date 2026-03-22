package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewWorktreePopup creates the worktree popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/worktree/init.lua
func NewWorktreePopup(tokens theme.Tokens, state *State) Popup {
	p := New("Worktree", tokens)

	// Worktree group
	p.AddActionGroup("Worktree", []Action{
		{Key: "w", Label: "Checkout"},
		{Key: "W", Label: "Create"},
	})

	// Do group
	p.AddActionGroup("Do", []Action{
		{Key: "g", Label: "Goto"},
		{Key: "m", Label: "Move"},
		{Key: "D", Label: "Delete"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("worktree", &p)
	}

	return p
}
