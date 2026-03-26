package nav

import "github.com/charmbracelet/bubbles/key"

// NavigationKeys contains the standard navigation key bindings shared by all list views.
type NavigationKeys struct {
	MoveDown     key.Binding
	MoveUp       key.Binding
	PageDown     key.Binding
	PageUp       key.Binding
	HalfPageDown key.Binding
	HalfPageUp   key.Binding
	GoToTop      key.Binding
	GoToBottom   key.Binding
	Close        key.Binding
	CloseEscape  key.Binding
	Yank         key.Binding
	Select       key.Binding
}

// DefaultNavigationKeys returns the standard navigation key bindings.
func DefaultNavigationKeys() NavigationKeys {
	return NavigationKeys{
		MoveDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j", "move down"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k", "move up"),
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
			key.WithHelp("enter", "select"),
		),
	}
}

// PopupKeys contains popup trigger key bindings shared by all views.
type PopupKeys struct {
	CherryPickPopup key.Binding
	BranchPopup     key.Binding
	CommitPopup     key.Binding
	DiffPopup       key.Binding
	FetchPopup      key.Binding
	MergePopup      key.Binding
	PullPopup       key.Binding
	PushPopup       key.Binding
	RebasePopup     key.Binding
	RevertPopup     key.Binding
	ResetPopup      key.Binding
	TagPopup        key.Binding
	BisectPopup     key.Binding
	RemotePopup     key.Binding
	WorktreePopup   key.Binding
	OpenCommitLink  key.Binding
}

// DefaultPopupKeys returns the standard popup trigger key bindings.
func DefaultPopupKeys() PopupKeys {
	return PopupKeys{
		CherryPickPopup: key.NewBinding(key.WithKeys("A")),
		BranchPopup:     key.NewBinding(key.WithKeys("b")),
		CommitPopup:     key.NewBinding(key.WithKeys("c")),
		DiffPopup:       key.NewBinding(key.WithKeys("d")),
		FetchPopup:      key.NewBinding(key.WithKeys("f")),
		MergePopup:      key.NewBinding(key.WithKeys("m")),
		PullPopup:       key.NewBinding(key.WithKeys("p")),
		PushPopup:       key.NewBinding(key.WithKeys("P")),
		RebasePopup:     key.NewBinding(key.WithKeys("r")),
		RevertPopup:     key.NewBinding(key.WithKeys("v")),
		ResetPopup:      key.NewBinding(key.WithKeys("X")),
		TagPopup:        key.NewBinding(key.WithKeys("t")),
		BisectPopup:     key.NewBinding(key.WithKeys("B")),
		RemotePopup:     key.NewBinding(key.WithKeys("M")),
		WorktreePopup:   key.NewBinding(key.WithKeys("w")),
		OpenCommitLink:  key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open commit link")),
	}
}
