package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewRebasePopup creates the rebase popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/rebase/init.lua
// State-dependent: different actions when in-rebase vs not.
func NewRebasePopup(tokens theme.Tokens, state *State, inRebase bool) Popup {
	p := New("Rebase", tokens)

	if inRebase {
		// In-rebase actions
		p.AddActionGroup("", []Action{
			{Key: "r", Label: "Continue"},
			{Key: "s", Label: "Skip"},
			{Key: "e", Label: "Edit"},
			{Key: "a", Label: "Abort"},
		})
	} else {
		// Not in-rebase switches
		p.AddSwitch("k", "keep-empty", "Keep empty commits", false)
		p.AddSwitch("u", "update-refs", "Update branches that point to commits that are being rebased", false)
		p.AddSwitch("d", "committer-date-is-author-date", "Use author date as committer date", false)
		p.AddSwitch("t", "ignore-date", "Use current time as author date", false)
		p.AddSwitch("a", "autosquash", "Autosquash fixup and squash commits", false)
		p.AddSwitch("A", "autostash", "Autostash", true) // enabled by default in Neogit
		p.AddSwitch("i", "interactive", "Interactive", false)
		p.AddSwitch("h", "no-verify", "Disable hooks", false)
		p.AddSwitch("S", "gpg-sign", "Sign using gpg", false)

		// Options
		p.AddOption("r", "rebase-merges", "Rebase merges", "")

		// Rebase <branch> onto
		p.AddActionGroup("Rebase onto", []Action{
			{Key: "p", Label: "pushRemote"},
			{Key: "u", Label: "@{upstream}"},
			{Key: "e", Label: "elsewhere"},
			{Key: "b", Label: "base branch"},
		})

		// Rebase group
		p.AddActionGroup("Rebase", []Action{
			{Key: "i", Label: "interactively"},
			{Key: "s", Label: "a subset"},
		})

		// Modify commits group
		p.AddActionGroup("", []Action{
			{Key: "m", Label: "to modify a commit"},
			{Key: "w", Label: "to reword a commit"},
			{Key: "d", Label: "to remove a commit"},
			{Key: "f", Label: "to autosquash"},
		})
	}

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("rebase", &p)
	}

	return p
}
