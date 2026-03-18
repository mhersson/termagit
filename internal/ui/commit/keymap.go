package commit

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the commit editor.
// Matches Neogit's config.lua commit_editor section exactly.
type KeyMap struct {
	Close           key.Binding // q
	Submit          key.Binding // <c-c><c-c> (two-key sequence handled in update)
	Abort           key.Binding // <c-c><c-k> (two-key sequence handled in update)
	PrevMessage     key.Binding // <m-p>
	NextMessage     key.Binding // <m-n>
	ResetMessage    key.Binding // <m-r>
	GenerateMessage key.Binding // <c-g>
}

// DefaultKeyMap returns the default key bindings matching Neogit exactly.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		// Note: Submit (<c-c><c-c>) and Abort (<c-c><c-k>) are two-key sequences.
		// The first ctrl+c is handled as a pending key, and the second key
		// determines the action. These bindings are just for reference/help.
		Submit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("<c-c><c-c>", "submit"),
		),
		Abort: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("<c-c><c-k>", "abort"),
		),
		PrevMessage: key.NewBinding(
			key.WithKeys("alt+p"),
			key.WithHelp("<m-p>", "previous message"),
		),
		NextMessage: key.NewBinding(
			key.WithKeys("alt+n"),
			key.WithHelp("<m-n>", "next message"),
		),
		ResetMessage: key.NewBinding(
			key.WithKeys("alt+r"),
			key.WithHelp("<m-r>", "reset message"),
		),
		GenerateMessage: key.NewBinding(
			key.WithKeys("ctrl+g"),
			key.WithHelp("<c-g>", "generate message"),
		),
	}
}
