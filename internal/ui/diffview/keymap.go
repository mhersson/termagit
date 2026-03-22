package diffview

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings for the diff view.
type KeyMap struct {
	// Navigation
	MoveDown     key.Binding
	MoveUp       key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageDown key.Binding
	HalfPageUp   key.Binding

	// Hunk navigation
	NextHunk       key.Binding
	PrevHunk       key.Binding
	NextHunkHeader key.Binding
	PrevHunkHeader key.Binding

	// File navigation
	NextFile key.Binding
	PrevFile key.Binding

	// Actions
	StageHunk   key.Binding
	UnstageHunk key.Binding

	// Toggle fold
	Toggle key.Binding

	// Close
	Close       key.Binding
	CloseEscape key.Binding
}

// DefaultKeyMap returns the default key bindings for the diff view.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		MoveDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "move down"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "move up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f", "pgdown"),
			key.WithHelp("C-f/PgDn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b", "pgup"),
			key.WithHelp("C-b/PgUp", "page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "half page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "half page up"),
		),

		NextHunk: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next hunk"),
		),
		PrevHunk: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "previous hunk"),
		),
		NextHunkHeader: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next hunk header"),
		),
		PrevHunkHeader: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "previous hunk header"),
		),

		NextFile: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next file"),
		),
		PrevFile: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous file"),
		),

		StageHunk: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stage hunk"),
		),
		UnstageHunk: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unstage hunk"),
		),

		Toggle: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle"),
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
