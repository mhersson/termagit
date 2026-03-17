package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewBisectPopup creates the bisect popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/bisect/init.lua
// State-dependent: different actions based on in-progress and finished state.
func NewBisectPopup(tokens theme.Tokens, state *State, inProgress, finished bool) Popup {
	p := New("Bisect", tokens)

	if !inProgress {
		// Not in-progress switches
		p.AddSwitch("r", "no-checkout", "Don't checkout the commit", false)
		p.AddSwitch("p", "first-parent", "Follow only the first parent commit upon seeing a merge commit", false)

		// Not in-progress actions
		p.AddActionGroup("", []Action{
			{Key: "B", Label: "Start"},
			{Key: "S", Label: "Scripted"},
		})
	} else if finished {
		// In-progress and finished
		p.AddActionGroup("", []Action{
			{Key: "r", Label: "Reset"},
		})
	} else {
		// In-progress but not finished
		p.AddActionGroup("", []Action{
			{Key: "b", Label: "Bad"},
			{Key: "g", Label: "Good"},
			{Key: "s", Label: "Skip"},
			{Key: "r", Label: "Reset"},
			{Key: "S", Label: "Run script"},
		})
	}

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("bisect", &p)
	}

	return p
}
