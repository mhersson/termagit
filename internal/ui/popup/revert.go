package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewRevertPopup creates the revert popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/revert/init.lua
// State-dependent: different actions when in-progress vs not.
func NewRevertPopup(tokens theme.Tokens, state *State, inProgress bool) Popup {
	p := New("Revert", tokens)

	// Options (always shown)
	p.AddOption("m", "mainline", "Mainline parent number", "")
	p.AddOption("s", "strategy", "Strategy", "")
	p.AddOption("S", "gpg-sign", "Sign using gpg", "")

	if inProgress {
		// In-progress actions
		p.AddActionGroup("", []Action{
			{Key: "v", Label: "Continue"},
			{Key: "s", Label: "Skip"},
			{Key: "a", Label: "Abort"},
		})
	} else {
		// Switches (not in-progress)
		p.AddSwitch("e", "edit", "Edit commit message", false)
		p.AddSwitchNonPersisted("E", "no-edit", "Don't edit commit message", false)
		p.SetIncompatible("e", "E")
		p.AddSwitch("s", "signoff", "Add Signed-off-by line", false)

		// Revert actions
		p.AddActionGroup("Revert", []Action{
			{Key: "v", Label: "Commit(s)"},
			{Key: "V", Label: "Changes"},
			// "h" (Hunk) only shown when hunk is under cursor - omitted for now
		})
	}

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("revert", &p)
	}

	return p
}
