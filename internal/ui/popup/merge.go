package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewMergePopup creates the merge popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/merge/init.lua
// State-dependent: different actions when in-merge vs not.
func NewMergePopup(tokens theme.Tokens, state *State, inMerge bool) Popup {
	p := New("Merge", tokens)

	if inMerge {
		// In-merge actions
		p.AddActionGroup("Actions", []Action{
			{Key: "m", Label: "Commit merge"},
			{Key: "a", Label: "Abort merge"},
		})
	} else {
		// Not in-merge switches
		p.AddSwitch("f", "ff-only", "Fast-forward only", false)
		p.AddSwitch("n", "no-ff", "No fast-forward", false)
		p.SetIncompatible("f", "n")

		// Options
		p.AddOption("s", "strategy", "Strategy", "")
		p.AddOption("X", "strategy-option", "Strategy Option", "")

		// Whitespace switches (Neogit uses cli_prefix = "-")
		p.AddSwitch("b", "Xignore-space-change", "Ignore changes in amount of whitespace", false)
		p.AddSwitch("w", "Xignore-all-space", "Ignore whitespace when comparing lines", false)

		p.AddOption("A", "Xdiff-algorithm", "Diff algorithm", "")
		p.AddOption("S", "gpg-sign", "Sign using gpg", "")

		// Actions
		p.AddActionGroup("Actions", []Action{
			{Key: "m", Label: "Merge"},
			{Key: "e", Label: "Merge and edit message"},
			{Key: "n", Label: "Merge but don't commit"},
			{Key: "a", Label: "Absorb"},
		})

		p.AddActionGroup("", []Action{
			{Key: "p", Label: "Preview merge"},
		})

		p.AddActionGroup("", []Action{
			{Key: "s", Label: "Squash merge"},
			{Key: "i", Label: "Dissolve"},
		})
	}

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("merge", &p)
	}

	return p
}
