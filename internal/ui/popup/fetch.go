package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// FetchPopupParams holds dynamic context for the fetch popup.
type FetchPopupParams struct {
	PushRemoteLabel string // resolved label, falls back to "pushRemote"
	UpstreamLabel   string // resolved label, falls back to "@{upstream}"
}

// NewFetchPopup creates the fetch popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/fetch/init.lua
func NewFetchPopup(tokens theme.Tokens, state *State, params FetchPopupParams) Popup {
	p := New("Fetch", tokens)

	// Switches
	p.AddSwitch("p", "prune", "Prune deleted branches", false)
	p.AddSwitch("t", "tags", "Fetch all tags", false)
	p.AddSwitchNonPersisted("F", "force", "force", false)

	pushLabel := params.PushRemoteLabel
	if pushLabel == "" {
		pushLabel = "pushRemote"
	}
	upstreamLabel := params.UpstreamLabel
	if upstreamLabel == "" {
		upstreamLabel = "@{upstream}"
	}

	p.AddActionGroup("Fetch from", []Action{
		{Key: "p", Label: pushLabel},
		{Key: "u", Label: upstreamLabel},
		{Key: "e", Label: "elsewhere"},
		{Key: "a", Label: "all remotes"},
	})

	p.AddActionGroup("Fetch", []Action{
		{Key: "o", Label: "another branch"},
		{Key: "r", Label: "explicit refspec"},
		{Key: "m", Label: "submodules"},
	})

	p.AddActionGroup("Configure", []Action{
		{Key: "C", Label: "Set variables..."},
	})

	if state != nil {
		state.ApplyToPopup("fetch", &p)
	}

	return p
}
