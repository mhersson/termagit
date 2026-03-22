package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewCherryPickPopup creates the cherry-pick popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/cherry_pick/init.lua
// State-dependent: different actions when in-progress vs not.
func NewCherryPickPopup(tokens theme.Tokens, state *State, inProgress bool) Popup {
	p := New("Cherry Pick", tokens)

	if inProgress {
		// In-progress actions
		p.AddActionGroup("Cherry Pick", []Action{
			{Key: "A", Label: "continue"},
			{Key: "s", Label: "skip"},
			{Key: "a", Label: "abort"},
		})
	} else {
		// Options (not in-progress)
		p.AddOptionWithPrefix("-", "m", "mainline", "Replay merge relative to parent", "")
		p.AddOptionWithChoices("s", "strategy", "Strategy", "",
			[]string{"octopus", "ours", "resolve", "subtree", "recursive"})

		// Switches (not in-progress)
		p.AddSwitch("F", "ff", "Attempt fast-forward", true) // enabled by default in Neogit
		p.AddSwitch("x", "x", "Reference cherry in commit message", false)
		p.AddSwitch("e", "edit", "Edit commit messages", false)
		p.SetIncompatible("F", "e") // ff and edit are incompatible
		p.AddSwitch("s", "signoff", "Add Signed-off-by lines", false)
		p.AddOptionWithPrefix("-", "S", "gpg-sign", "Sign using gpg", "")

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
