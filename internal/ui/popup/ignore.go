package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewIgnorePopup creates the ignore popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/ignore/init.lua
func NewIgnorePopup(tokens theme.Tokens, state *State, hasGlobalIgnore bool) Popup {
	p := New("Ignore", tokens)

	// Gitignore actions
	actions := []Action{
		{Key: "t", Label: "shared at top-level (.gitignore)"},
		{Key: "s", Label: "shared in sub-directory"},
		{Key: "p", Label: "privately for this repository"},
	}

	// Only show global option if global ignore file exists
	if hasGlobalIgnore {
		actions = append(actions, Action{Key: "g", Label: "globally for this user"})
	}

	p.AddActionGroup("Gitignore", actions)

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("ignore", &p)
	}

	return p
}
