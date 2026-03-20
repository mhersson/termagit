package reflogview

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the reflog view.
type KeyMap struct {
	MoveDown     key.Binding
	MoveUp       key.Binding
	Close        key.Binding
	CloseEscape  key.Binding
	Yank         key.Binding
	Select       key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageDown key.Binding
	HalfPageUp   key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding
}

// DefaultKeyMap returns the default key bindings matching Neogit.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		MoveDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j", "next entry"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k", "prev entry"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		CloseEscape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		Yank: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank hash"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open commit"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f", "pgdown"),
			key.WithHelp("C-f", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b", "pgup"),
			key.WithHelp("C-b", "page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "half page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "half page up"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
	}
}
