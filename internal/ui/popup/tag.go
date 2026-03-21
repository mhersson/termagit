package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewTagPopup creates the tag popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/tag/init.lua
func NewTagPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Tag", tokens)

	// Switches
	p.AddSwitchNonPersisted("f", "force", "Force", false)
	p.AddSwitch("a", "annotate", "Annotate", false)
	p.AddSwitch("s", "sign", "Sign", false)

	// Options
	p.AddOption("u", "local-user", "Sign as", "")

	// Create group
	p.AddActionGroup("Create", []Action{
		{Key: "t", Label: "tag"},
		{Key: "r", Label: "release"},
	})

	// Do group
	p.AddActionGroup("Do", []Action{
		{Key: "x", Label: "delete"},
		{Key: "p", Label: "prune"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("tag", &p)
	}

	return p
}
