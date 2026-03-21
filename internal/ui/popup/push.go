package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// PushPopupParams holds dynamic context for the push popup.
type PushPopupParams struct {
	Branch          string
	IsDetached      bool
	PushRemoteLabel string // resolved label like "origin/main", falls back to "pushRemote"
	UpstreamLabel   string // resolved label like "origin/main", falls back to "@{upstream}"
}

// NewPushPopup creates the push popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/push/init.lua
func NewPushPopup(tokens theme.Tokens, state *State, params PushPopupParams) Popup {
	p := New("Push", tokens)

	// Switches (from Neogit push popup)
	p.AddSwitchNonPersisted("f", "force-with-lease", "Force with lease", false)
	p.AddSwitchNonPersisted("F", "force", "Force", false)
	p.AddSwitch("h", "no-verify", "Disable hooks", false)
	p.AddSwitch("d", "dry-run", "Dry run", false)
	p.AddSwitch("u", "set-upstream", "Set the upstream before pushing", false)
	p.AddSwitch("T", "tags", "Include all tags", false)
	p.AddSwitch("t", "follow-tags", "Include related annotated tags", false)

	pushLabel := params.PushRemoteLabel
	if pushLabel == "" {
		pushLabel = "pushRemote"
	}
	upstreamLabel := params.UpstreamLabel
	if upstreamLabel == "" {
		upstreamLabel = "@{upstream}"
	}

	if !params.IsDetached {
		p.AddActionGroup("Push "+params.Branch+" to", []Action{
			{Key: "p", Label: pushLabel},
			{Key: "u", Label: upstreamLabel},
			{Key: "e", Label: "elsewhere"},
		})
	}

	// Push group
	p.AddActionGroup("Push", []Action{
		{Key: "o", Label: "another branch"},
		{Key: "r", Label: "explicit refspec"},
		{Key: "m", Label: "matching branches"},
		{Key: "T", Label: "a tag"},
		{Key: "t", Label: "all tags"},
	})

	p.AddActionGroup("Configure", []Action{
		{Key: "C", Label: "Set variables..."},
	})

	if state != nil {
		state.ApplyToPopup("push", &p)
	}

	return p
}
