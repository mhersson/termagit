package cmdhistory

import "github.com/charmbracelet/bubbles/key"

// toggleFoldKey is the view-specific key binding for toggling entry fold state.
var toggleFoldKey = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "toggle fold"),
)
