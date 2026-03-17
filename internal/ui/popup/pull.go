package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewPullPopup creates the pull popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/pull/init.lua
func NewPullPopup(tokens theme.Tokens, state *State, branch string) Popup {
	p := New("Pull", tokens)

	// Config item for branch.rebase could go here
	// p.AddConfig("r", "branch."+branch+".rebase", "branch rebase", "")

	// Switches (from Neogit pull popup)
	p.AddSwitch("f", "ff-only", "Fast-forward only", false)
	p.AddSwitchNonPersisted("r", "rebase", "Rebase local commits", false)
	p.AddSwitch("a", "autostash", "Autostash", false)
	p.AddSwitch("t", "tags", "Fetch tags", false)
	p.AddSwitchNonPersisted("F", "force", "Force", false)

	// Pull into <branch> from
	p.AddActionGroup("Pull into "+branch+" from", []Action{
		{Key: "p", Label: "pushRemote"},
		{Key: "u", Label: "@{upstream}"},
		{Key: "e", Label: "elsewhere"},
	})

	p.AddActionGroup("Configure", []Action{
		{Key: "C", Label: "Set variables..."},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("pull", &p)
	}

	return p
}
