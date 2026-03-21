package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewBranchPopup creates the branch popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/branch/init.lua
func NewBranchPopup(tokens theme.Tokens, state *State, branch string, showConfig, hasUpstream bool) Popup {
	p := New("Branch", tokens)

	// Config items (only when on a branch)
	if showConfig && branch != "" {
		p.AddConfig("d", "branch."+branch+".description", "Branch description", "")
		p.AddConfig("u", "branch."+branch+".merge", "Merge branch", "")
		p.AddConfig("m", "branch."+branch+".remote", "Remote", "")
		p.AddConfig("R", "branch."+branch+".rebase", "Rebase", "")
		p.AddConfig("p", "branch."+branch+".pushRemote", "Push remote", "")
	}

	// Switches
	p.AddSwitch("r", "recurse-submodules", "Recurse submodules when checking out an existing branch", false)

	// Checkout group
	p.AddActionGroup("Checkout", []Action{
		{Key: "b", Label: "branch/revision"},
		{Key: "l", Label: "local branch"},
		{Key: "r", Label: "recent branch"},
	})

	// Unlabeled group (new branch options)
	p.AddActionGroup("", []Action{
		{Key: "c", Label: "new branch"},
		{Key: "s", Label: "new spin-off"},
		{Key: "w", Label: "new worktree"},
	})

	// Create group
	p.AddActionGroup("Create", []Action{
		{Key: "n", Label: "new branch"},
		{Key: "S", Label: "new spin-out"},
		{Key: "W", Label: "new worktree"},
	})

	// Do group
	doActions := []Action{
		{Key: "C", Label: "Configure..."},
		{Key: "m", Label: "rename"},
		{Key: "X", Label: "reset"},
		{Key: "D", Label: "delete"},
	}
	if hasUpstream {
		doActions = append(doActions, Action{Key: "o", Label: "pull request"})
	}
	p.AddActionGroup("Do", doActions)

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("branch", &p)
	}

	return p
}
