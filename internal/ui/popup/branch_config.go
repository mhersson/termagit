package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewBranchConfigPopup creates the branch config popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/branch_config/init.lua
func NewBranchConfigPopup(tokens theme.Tokens, state *State, branch string) Popup {
	p := New("Configure branch", tokens)

	// Configure branch section
	p.AddConfig("d", "branch."+branch+".description", "Description", "")
	p.AddConfig("u", "branch."+branch+".merge", "Merge", "")
	p.AddConfig("m", "branch."+branch+".remote", "Remote", "")
	p.AddConfig("r", "branch."+branch+".rebase", "Rebase", "")
	p.AddConfig("p", "branch."+branch+".pushRemote", "Push remote", "")

	// Configure repository defaults section
	p.AddConfig("R", "pull.rebase", "Pull rebase", "")
	p.AddConfig("P", "remote.pushDefault", "Push default", "")
	p.AddConfig("b", "neogit.baseBranch", "Base branch", "")
	p.AddConfig("A", "neogit.askSetPushDefault", "Ask set push default", "")

	// Configure branch creation section
	p.AddConfig("as", "branch.autoSetupMerge", "Auto setup merge", "")
	p.AddConfig("ar", "branch.autoSetupRebase", "Auto setup rebase", "")

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("branch_config", &p)
	}

	return p
}
