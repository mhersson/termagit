package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewCherryPickPopup creates the cherry-pick popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/cherry_pick/init.lua
// State-dependent: different actions when in-progress vs not.
func NewCherryPickPopup(tokens theme.Tokens, state *State, inProgress bool) Popup {
	p := New("Cherry Pick", tokens)

	if inProgress {
		// In-progress actions
		p.AddActionGroup("", []Action{
			{Key: "A", Label: "Continue"},
			{Key: "s", Label: "Skip"},
			{Key: "a", Label: "Abort"},
		})
	} else {
		// Options (not in-progress)
		p.AddOption("m", "mainline", "Mainline parent number", "")
		p.AddOption("s", "strategy", "Strategy", "")

		// Switches (not in-progress)
		p.AddSwitch("F", "ff", "Attempt fast-forward", false)
		p.AddSwitch("x", "reference-in-message", "Add reference to original commit", false)
		p.AddSwitch("e", "edit", "Edit commit message", false)
		p.AddSwitch("s", "signoff", "Add Signed-off-by line", false)
		p.AddSwitch("S", "gpg-sign", "Sign using gpg", false)

		// Apply here
		p.AddActionGroup("Apply here", []Action{
			{Key: "A", Label: "Pick"},
			{Key: "a", Label: "Apply"},
			{Key: "h", Label: "Harvest"},
			{Key: "m", Label: "Squash"},
		})

		// Apply elsewhere
		p.AddActionGroup("Apply elsewhere", []Action{
			{Key: "d", Label: "Donate"},
			{Key: "n", Label: "Spinout"},
			{Key: "s", Label: "Spinoff"},
		})
	}

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("cherry_pick", &p)
	}

	return p
}
