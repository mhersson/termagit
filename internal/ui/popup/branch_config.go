package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewBranchConfigPopup creates the branch config popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/branch_config/init.lua
// remotes is the list of configured remote names (for pushRemote/pushDefault choices).
// pullRebase is the current value of pull.rebase (local or "false" if unset).
// globalPullRebase is the global pull.rebase value (empty if unset).
func NewBranchConfigPopup(tokens theme.Tokens, state *State, branch string,
	remotes []string, pullRebase, globalPullRebase string,
) Popup {
	p := New("Configure branch", tokens)

	// Configure branch section
	p.AddConfigSection("Configure branch")
	p.AddConfig("d", "branch."+branch+".description", "Description", "")
	p.AddConfig("u", "branch."+branch+".merge", "Merge", "")
	p.AddConfig("m", "branch."+branch+".remote", "Remote", "")
	rebaseChoices := []string{"true", "false"}
	p.AddConfigWithChoices("r", "branch."+branch+".rebase", "Rebase", "",
		append(rebaseChoices, "pull.rebase:"+pullRebase))
	p.AddConfigWithChoices("p", "branch."+branch+".pushRemote", "Push remote", "", remotes)

	// Configure repository defaults section
	p.AddConfigSection("")
	p.AddConfigSection("Configure repository defaults")
	pullRebaseChoices := []string{"true", "false"}
	if globalPullRebase != "" {
		pullRebaseChoices = append(pullRebaseChoices, "global:"+globalPullRebase)
	}
	p.AddConfigWithChoices("R", "pull.rebase", "Pull rebase", "", pullRebaseChoices)
	p.AddConfigWithChoices("P", "remote.pushDefault", "Push default", "", remotes)
	p.AddConfig("b", "neogit.baseBranch", "Base branch", "")
	p.AddConfigWithChoices("A", "neogit.askSetPushDefault", "Ask set push default", "",
		[]string{"ask", "ask-if-unset", "never"})

	// Configure branch creation section
	p.AddConfigSection("")
	p.AddConfigSection("Configure branch creation")
	p.AddConfigWithChoices("as", "branch.autoSetupMerge", "Auto setup merge", "",
		[]string{"always", "true", "false", "inherit", "simple", "default:true"})
	p.AddConfigWithChoices("ar", "branch.autoSetupRebase", "Auto setup rebase", "",
		[]string{"always", "local", "remote", "never", "default:never"})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("branch_config", &p)
	}

	return p
}
