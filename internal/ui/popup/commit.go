package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewCommitPopup creates the commit popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/commit/init.lua
func NewCommitPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Commit", tokens)

	// Switches (from Neogit commit popup)
	p.AddSwitch("a", "all", "Stage all modified and deleted files", false)
	p.AddSwitchNonPersisted("e", "allow-empty", "Allow empty commit", false)
	p.AddSwitch("v", "verbose", "Show diff of changes to be committed", false)
	p.AddSwitch("h", "no-verify", "Disable hooks", false)
	p.AddSwitch("R", "reset-author", "Claim authorship and reset author date", false)
	p.AddSwitch("s", "signoff", "Add Signed-off-by line", false)

	// Options (from Neogit commit popup — all use "-" prefix)
	p.AddOptionWithPrefix("-", "A", "author", "Override the author", "")
	p.AddOptionWithPrefix("-", "S", "gpg-sign", "Sign using gpg", "")
	p.AddOptionWithPrefix("-", "C", "reuse-message", "Reuse commit message", "")

	// Action groups (from Neogit commit popup)
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit"},
	})

	p.AddActionGroup("Edit HEAD", []Action{
		{Key: "e", Label: "Extend"},
		{Spacer: true},
		{Key: "a", Label: "Amend"},
		{Spacer: true},
		{Key: "w", Label: "Reword"},
	})

	p.AddActionGroup("Edit", []Action{
		{Key: "f", Label: "Fixup"},
		{Key: "s", Label: "Squash"},
		{Key: "A", Label: "Alter"},
		{Key: "n", Label: "Augment"},
		{Key: "W", Label: "Revise"},
	})

	p.AddActionGroup("Edit and rebase", []Action{
		{Key: "F", Label: "Instant Fixup"},
		{Key: "S", Label: "Instant Squash"},
	})

	p.AddActionGroup("Spread across commits", []Action{
		{Key: "x", Label: "Absorb"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("commit", &p)
	}

	return p
}
