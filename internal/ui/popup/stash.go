package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewStashPopup creates the stash popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/stash/init.lua
func NewStashPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Stash", tokens)

	// Switches
	p.AddSwitch("u", "include-untracked", "Also save untracked files", false)
	p.AddSwitch("a", "all", "Also save untracked and ignored files", false)
	p.SetIncompatible("u", "a")

	// Stash group
	p.AddActionGroup("Stash", []Action{
		{Key: "z", Label: "both"},
		{Key: "i", Label: "index"},
		{Key: "w", Label: "worktree"},
		{Key: "x", Label: "keeping index"},
		{Key: "P", Label: "push"},
	})

	// Snapshot group
	p.AddActionGroup("Snapshot", []Action{
		{Key: "Z", Label: "both"},
		{Key: "I", Label: "index"},
		{Key: "W", Label: "worktree"},
		{Key: "r", Label: "to wip ref"},
	})

	// Use group
	p.AddActionGroup("Use", []Action{
		{Key: "p", Label: "pop"},
		{Key: "a", Label: "apply"},
		{Key: "d", Label: "drop"},
	})

	// Inspect group
	p.AddActionGroup("Inspect", []Action{
		{Key: "l", Label: "List"},
		{Key: "v", Label: "Show"},
	})

	// Transform group
	p.AddActionGroup("Transform", []Action{
		{Key: "b", Label: "Branch"},
		{Key: "B", Label: "Branch here"},
		{Key: "m", Label: "Rename"},
		{Key: "f", Label: "Format patch"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("stash", &p)
	}

	return p
}
