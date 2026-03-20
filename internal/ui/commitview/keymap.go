package commitview

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines key bindings for the commit view.
type KeyMap struct {
	// Navigation
	MoveDown       key.Binding
	MoveUp         key.Binding
	PageDown       key.Binding
	PageUp         key.Binding
	HalfPageDown   key.Binding
	HalfPageUp     key.Binding
	PrevHunkHeader key.Binding
	NextHunkHeader key.Binding
	ScrollUp       key.Binding
	ScrollDown     key.Binding

	// Actions
	OpenFileInWorktree key.Binding
	OpenCommitLink     key.Binding
	YankSelected       key.Binding

	// Close
	Close       key.Binding
	CloseEscape key.Binding

	// Popup triggers (for future phases)
	CherryPickPopup key.Binding
	BranchPopup     key.Binding
	CommitPopup     key.Binding
	DiffPopup       key.Binding
	PushPopup       key.Binding
	RevertPopup     key.Binding
	ResetPopup      key.Binding
	BisectPopup     key.Binding
	TagPopup        key.Binding
}

// DefaultKeyMap returns the default key bindings for commit view.
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
		PrevHunkHeader: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "previous hunk"),
		),
		NextHunkHeader: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next hunk"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("[c"),
			key.WithHelp("[c", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("]c"),
			key.WithHelp("]c", "scroll down"),
		),

		OpenFileInWorktree: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "open file in worktree"),
		),
		OpenCommitLink: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open commit link"),
		),
		YankSelected: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank commit hash"),
		),

		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		CloseEscape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),

		// Popup triggers
		CherryPickPopup: key.NewBinding(key.WithKeys("A")),
		BranchPopup:     key.NewBinding(key.WithKeys("b")),
		CommitPopup:     key.NewBinding(key.WithKeys("c")),
		DiffPopup:       key.NewBinding(key.WithKeys("d")),
		PushPopup:       key.NewBinding(key.WithKeys("P")),
		RevertPopup:     key.NewBinding(key.WithKeys("r")),
		ResetPopup:      key.NewBinding(key.WithKeys("X")),
		BisectPopup:     key.NewBinding(key.WithKeys("B")),
		TagPopup:        key.NewBinding(key.WithKeys("t")),
	}
}
