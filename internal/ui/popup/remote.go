package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewRemotePopup creates the remote popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/remote/init.lua
func NewRemotePopup(tokens theme.Tokens, state *State, remoteName string) Popup {
	p := New("Remote", tokens)

	// Switch: fetch after add (enabled by default, cli_prefix = "-")
	p.AddSwitch("f", "f", "Fetch after add", true)

	// Config items (for the specified remote)
	if remoteName != "" {
		p.AddConfig("u", "remote."+remoteName+".url", "URL", "")
		p.AddConfig("U", "remote."+remoteName+".fetch", "Fetch refspec", "")
		p.AddConfig("s", "remote."+remoteName+".pushurl", "Push URL", "")
		p.AddConfig("S", "remote."+remoteName+".push", "Push refspec", "")
		p.AddConfigWithChoices("O", "remote."+remoteName+".tagOpt", "Tag option", "",
			[]string{"--no-tags", "--tags"})
	}

	// Actions group 1
	p.AddActionGroup("Actions", []Action{
		{Key: "a", Label: "Add"},
		{Key: "r", Label: "Rename"},
		{Key: "x", Label: "Remove"},
	})

	// Actions group 2
	p.AddActionGroup("", []Action{
		{Key: "C", Label: "Configure..."},
		{Key: "p", Label: "Prune stale branches"},
		{Key: "P", Label: "Prune stale refspecs"},
		{Key: "b", Label: "Update default branch"},
		{Key: "z", Label: "Unshallow remote"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("remote", &p)
	}

	return p
}
