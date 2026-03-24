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
	GoToTop        key.Binding
	GoToBottom     key.Binding
	PrevHunkHeader key.Binding
	NextHunkHeader key.Binding

	// Horizontal scroll
	ScrollLeft  key.Binding
	ScrollRight key.Binding
	ScrollStart key.Binding
	ScrollEnd   key.Binding

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
	RebasePopup     key.Binding
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
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		PrevHunkHeader: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "previous hunk"),
		),
		NextHunkHeader: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next hunk"),
		),
		ScrollLeft: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "scroll left"),
		),
		ScrollRight: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "scroll right"),
		),
		ScrollStart: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "scroll to start"),
		),
		ScrollEnd: key.NewBinding(
			key.WithKeys("$"),
			key.WithHelp("$", "scroll to end"),
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
		RevertPopup:     key.NewBinding(key.WithKeys("v")),
		RebasePopup:     key.NewBinding(key.WithKeys("r")),
		ResetPopup:      key.NewBinding(key.WithKeys("X")),
		BisectPopup:     key.NewBinding(key.WithKeys("B")),
		TagPopup:        key.NewBinding(key.WithKeys("t")),
	}
}
