package logview

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the log view.
type KeyMap struct {
	MoveDown     key.Binding
	MoveUp       key.Binding
	LoadMore     key.Binding
	Filter       key.Binding
	Yank         key.Binding
	Close        key.Binding
	CloseEscape  key.Binding
	ToggleDetail key.Binding
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
			key.WithHelp("j", "next commit"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k", "prev commit"),
		),
		LoadMore: key.NewBinding(
			key.WithKeys("+"),
			key.WithHelp("+", "load more"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Yank: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank hash"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		CloseEscape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		ToggleDetail: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle details"),
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
