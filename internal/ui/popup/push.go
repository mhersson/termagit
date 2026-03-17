package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewPushPopup creates the push popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/push/init.lua
func NewPushPopup(tokens theme.Tokens, state *State, branch string, isDetached bool) Popup {
	p := New("Push", tokens)

	// Switches (from Neogit push popup)
	p.AddSwitchNonPersisted("f", "force-with-lease", "Force with lease", false)
	p.AddSwitchNonPersisted("F", "force", "Force", false)
	p.AddSwitch("h", "no-verify", "Disable hooks", false)
	p.AddSwitch("d", "dry-run", "Dry run", false)
	p.AddSwitch("u", "set-upstream", "Set the upstream before pushing", false)
	p.AddSwitch("T", "tags", "Include all tags", false)
	p.AddSwitch("t", "follow-tags", "Include related annotated tags", false)

	if !isDetached {
		// Push <branch> to
		p.AddActionGroup("Push "+branch+" to", []Action{
			{Key: "p", Label: "pushRemote"},
			{Key: "u", Label: "@{upstream}"},
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

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("push", &p)
	}

	return p
}
