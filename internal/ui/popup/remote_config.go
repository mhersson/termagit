package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewRemoteConfigPopup creates the remote config popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/remote_config/init.lua
func NewRemoteConfigPopup(tokens theme.Tokens, state *State, remote string) Popup {
	p := New("Configure remote", tokens)

	// Configure remote section
	p.AddConfig("u", "remote."+remote+".url", "URL", "")
	p.AddConfig("U", "remote."+remote+".fetch", "Fetch refspec", "")
	p.AddConfig("s", "remote."+remote+".pushurl", "Push URL", "")
	p.AddConfig("S", "remote."+remote+".push", "Push refspec", "")
	p.AddConfigWithChoices("O", "remote."+remote+".tagOpt", "Tag option", "",
		[]string{"--no-tags", "--tags"})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("remote_config", &p)
	}

	return p
}
