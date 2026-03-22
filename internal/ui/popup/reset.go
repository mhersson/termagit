package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewResetPopup creates the reset popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/reset/init.lua
func NewResetPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Reset", tokens)

	// Reset group
	p.AddActionGroup("Reset", []Action{
		{Key: "f", Label: "file"},
		{Key: "b", Label: "branch"},
	})

	// Reset this group — padded labels for alignment (matches Neogit)
	p.AddActionGroup("Reset this", []Action{
		{Key: "m", Label: "mixed    (HEAD and index)"},
		{Key: "s", Label: "soft     (HEAD only)"},
		{Key: "h", Label: "hard     (HEAD, index and files)"},
		{Key: "k", Label: "keep     (HEAD and index, keeping uncommitted)"},
		{Key: "i", Label: "index    (only)"},
		{Key: "w", Label: "worktree (only)"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("reset", &p)
	}

	return p
}
