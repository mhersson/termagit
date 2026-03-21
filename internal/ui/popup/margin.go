package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewMarginPopup creates the margin popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/margin/init.lua
func NewMarginPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Margin", tokens)

	// Order option (cycles through topo/author-date/date)
	p.AddOption("o", "order", "Order commits by", "")

	// Switches
	p.AddSwitch("d", "decorate", "Show refnames", true) // enabled by default

	// Refresh group
	p.AddActionGroup("Refresh", []Action{
		{Key: "g", Label: "buffer"},
	})

	// Margin group
	p.AddActionGroup("Margin", []Action{
		{Key: "L", Label: "toggle visibility"},
		{Key: "l", Label: "cycle style"},
		{Key: "d", Label: "toggle details"},
		{Key: "x", Label: "toggle shortstat"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("margin", &p)
	}

	return p
}
