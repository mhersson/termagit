package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewDiffPopup creates the diff popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/diff/init.lua
func NewDiffPopup(tokens theme.Tokens, state *State, hasItem, commitSelected bool) Popup {
	p := New("Diff", tokens)

	// Diff group — "d" and "h" are conditional on selected item
	p.AddActionGroup("Diff", []Action{
		{Key: "d", Label: "this", Disabled: !hasItem},
		{Key: "h", Label: "this..HEAD", Disabled: !commitSelected},
		{Key: "r", Label: "range"},
		{Key: "p", Label: "paths", Disabled: true},
	})

	// Unlabeled group
	p.AddActionGroup("", []Action{
		{Key: "u", Label: "unstaged"},
		{Key: "s", Label: "staged"},
		{Key: "w", Label: "worktree"},
	})

	// Show group
	p.AddActionGroup("Show", []Action{
		{Key: "c", Label: "Commit"},
		{Key: "t", Label: "Stash"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("diff", &p)
	}

	return p
}
