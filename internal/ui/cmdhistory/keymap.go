package cmdhistory

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the command history view.
type KeyMap struct {
	MoveDown    key.Binding
	MoveUp      key.Binding
	ToggleFold  key.Binding
	Close       key.Binding
	CloseEscape key.Binding
}

// DefaultKeyMap returns the default key bindings matching Neogit.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		MoveDown: key.NewBinding(
			key.WithKeys("j", "ctrl+j"),
			key.WithHelp("j", "next entry"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k", "ctrl+k"),
			key.WithHelp("k", "prev entry"),
		),
		ToggleFold: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle fold"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		CloseEscape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
	}
}
