package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewFetchPopup creates the fetch popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/fetch/init.lua
func NewFetchPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Fetch", tokens)

	// Switches
	p.AddSwitch("p", "prune", "Prune remote tracking branches no longer on remote", false)
	p.AddSwitch("t", "tags", "Fetch all tags", false)
	p.AddSwitch("F", "force", "Force", false)

	// Fetch from group
	p.AddActionGroup("Fetch from", []Action{
		{Key: "p", Label: "pushRemote"},
		{Key: "u", Label: "@{upstream}"},
		{Key: "e", Label: "elsewhere"},
		{Key: "a", Label: "all remotes"},
	})

	// Fetch group
	p.AddActionGroup("Fetch", []Action{
		{Key: "o", Label: "another branch"},
		{Key: "r", Label: "explicit refspec"},
		{Key: "m", Label: "submodules"},
	})

	// Configure group
	p.AddActionGroup("Configure", []Action{
		{Key: "C", Label: "Set variables..."},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("fetch", &p)
	}

	return p
}
