package refsview

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the refs view.
type KeyMap struct {
	MoveDown     key.Binding
	MoveUp       key.Binding
	Close        key.Binding
	CloseEscape  key.Binding
	Toggle       key.Binding // tab - fold/unfold section
	Select       key.Binding // enter - open commit view
	DeleteBranch key.Binding // x - delete branch
	Yank         key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageDown key.Binding
	HalfPageUp   key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding
	Refresh      key.Binding

	// Popup triggers (matching Neogit refs_view mappings)
	CherryPickPopup key.Binding
	BranchPopup     key.Binding
	CommitPopup     key.Binding
	DiffPopup       key.Binding
	FetchPopup      key.Binding
	PullPopup       key.Binding
	PushPopup       key.Binding
	RebasePopup     key.Binding
	RevertPopup     key.Binding
	ResetPopup      key.Binding
	TagPopup        key.Binding
	BisectPopup     key.Binding
	RemotePopup     key.Binding
}

// DefaultKeyMap returns the default key bindings matching Neogit.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		MoveDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j", "next ref"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k", "prev ref"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		CloseEscape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "fold/unfold"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open commit"),
		),
		DeleteBranch: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete branch"),
		),
		Yank: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank hash"),
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
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("C-r", "refresh"),
		),

		// Popup triggers
		CherryPickPopup: key.NewBinding(key.WithKeys("A")),
		BranchPopup:     key.NewBinding(key.WithKeys("b")),
		CommitPopup:     key.NewBinding(key.WithKeys("c")),
		DiffPopup:       key.NewBinding(key.WithKeys("d")),
		FetchPopup:      key.NewBinding(key.WithKeys("f")),
		PullPopup:       key.NewBinding(key.WithKeys("p")),
		PushPopup:       key.NewBinding(key.WithKeys("P")),
		RebasePopup:     key.NewBinding(key.WithKeys("r")),
		RevertPopup:     key.NewBinding(key.WithKeys("v")),
		ResetPopup:      key.NewBinding(key.WithKeys("X")),
		TagPopup:        key.NewBinding(key.WithKeys("t")),
		BisectPopup:     key.NewBinding(key.WithKeys("B")),
		RemotePopup:     key.NewBinding(key.WithKeys("M")),
	}
}
