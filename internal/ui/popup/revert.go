package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewRevertPopup creates the revert popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/revert/init.lua
// State-dependent: different actions when in-progress vs not.
func NewRevertPopup(tokens theme.Tokens, state *State, inProgress, hasHunk bool) Popup {
	p := New("Revert", tokens)

	if inProgress {
		// In-progress actions — no options or switches
		p.AddActionGroup("Revert", []Action{
			{Key: "v", Label: "continue"},
			{Key: "s", Label: "skip"},
			{Key: "a", Label: "abort"},
		})
	} else {
		// Options (only when not in-progress)
		p.AddOptionWithPrefix("-", "m", "mainline", "Replay merge relative to parent", "")

		// Switches (not in-progress)
		p.AddSwitch("e", "edit", "Edit commit messages", true) // enabled by default in Neogit
		p.AddSwitchNonPersisted("E", "no-edit", "Don't edit commit messages", false)
		p.SetIncompatible("e", "E")
		p.AddSwitch("s", "signoff", "Add Signed-off-by lines", false)

		p.AddOptionWithChoices("s", "strategy", "Strategy", "",
			[]string{"octopus", "ours", "resolve", "subtree", "recursive"})
		p.AddOptionWithPrefix("-", "S", "gpg-sign", "Sign using gpg", "")

		// Revert actions
		revertActions := []Action{
			{Key: "v", Label: "Commit(s)"},
			{Key: "V", Label: "Changes"},
		}
		if hasHunk {
			revertActions = append(revertActions, Action{Key: "h", Label: "Hunk"})
		}
		p.AddActionGroup("Revert", revertActions)
	}

	if state != nil {
		state.ApplyToPopup("revert", &p)
	}

	return p
}
