package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// PullPopupParams holds dynamic context for the pull popup.
type PullPopupParams struct {
	Branch          string
	IsDetached      bool
	PushRemoteLabel string // resolved label, falls back to "pushRemote"
	UpstreamLabel   string // resolved label, falls back to "@{upstream}"
}

// NewPullPopup creates the pull popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/pull/init.lua
func NewPullPopup(tokens theme.Tokens, state *State, params PullPopupParams) Popup {
	p := New("Pull", tokens)

	if !params.IsDetached && params.Branch != "" {
		p.AddConfig("r", "branch."+params.Branch+".rebase", "branch rebase", "")
	}

	// Switches (from Neogit pull popup)
	p.AddSwitch("f", "ff-only", "Fast-forward only", false)
	p.AddSwitchNonPersisted("r", "rebase", "Rebase local commits", false)
	p.AddSwitch("a", "autostash", "Autostash", false)
	p.AddSwitch("t", "tags", "Fetch tags", false)
	p.AddSwitchNonPersisted("F", "force", "Force", false)

	pushLabel := params.PushRemoteLabel
	if pushLabel == "" {
		pushLabel = "pushRemote"
	}
	upstreamLabel := params.UpstreamLabel
	if upstreamLabel == "" {
		upstreamLabel = "@{upstream}"
	}

	if params.IsDetached {
		p.AddActionGroup("Pull from", []Action{
			{Key: "p", Label: pushLabel},
			{Key: "u", Label: upstreamLabel},
			{Key: "e", Label: "elsewhere"},
		})
	} else {
		p.AddActionGroup("Pull into "+params.Branch+" from", []Action{
			{Key: "p", Label: pushLabel},
			{Key: "u", Label: upstreamLabel},
			{Key: "e", Label: "elsewhere"},
		})
	}

	p.AddActionGroup("Configure", []Action{
		{Key: "C", Label: "Set variables..."},
	})

	if state != nil {
		state.ApplyToPopup("pull", &p)
	}

	return p
}
