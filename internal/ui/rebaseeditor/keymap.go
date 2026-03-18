package rebaseeditor

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the rebase editor.
// Matches Neogit's config.lua rebase_editor section exactly.
type KeyMap struct {
	Pick       key.Binding
	Reword     key.Binding
	Edit       key.Binding
	Squash     key.Binding
	Fixup      key.Binding
	Execute    key.Binding
	Drop       key.Binding
	Break      key.Binding
	Close      key.Binding
	OpenCommit key.Binding

	// Two-key sequences: gk/gj (handled via pendingG state in update)
	MoveUp   key.Binding
	MoveDown key.Binding

	// Two-key sequences: <c-c><c-c> and <c-c><c-k> (handled via pendingKey state)
	Submit key.Binding
	Abort  key.Binding

	// Vim-style aliases
	SubmitZZ key.Binding
	AbortZQ  key.Binding

	// Scroll bindings
	OpenOrScrollUp   key.Binding
	OpenOrScrollDown key.Binding
}

// DefaultKeyMap returns the default key bindings matching Neogit's rebase_editor exactly.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Pick: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pick"),
		),
		Reword: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reword"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Squash: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "squash"),
		),
		Fixup: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fixup"),
		),
		Execute: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "exec"),
		),
		Drop: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "drop"),
		),
		Break: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "break"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		OpenCommit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("<cr>", "open commit"),
		),
		// MoveUp: gk or <M-up> — gk handled as two-key sequence (g then k)
		MoveUp: key.NewBinding(
			key.WithKeys("alt+up"),
			key.WithHelp("gk", "move up"),
		),
		// MoveDown: gj or <M-down> — gj handled as two-key sequence (g then j)
		MoveDown: key.NewBinding(
			key.WithKeys("alt+down"),
			key.WithHelp("gj", "move down"),
		),
		// Submit/Abort: <c-c><c-c> and <c-c><c-k> are two-key sequences
		Submit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("<c-c><c-c>", "submit"),
		),
		Abort: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("<c-c><c-k>", "abort"),
		),
		SubmitZZ: key.NewBinding(
			key.WithKeys("Z"),
			key.WithHelp("ZZ", "submit"),
		),
		AbortZQ: key.NewBinding(
			key.WithKeys("Z"),
			key.WithHelp("ZQ", "abort"),
		),
		OpenOrScrollUp: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[c", "open/scroll up"),
		),
		OpenOrScrollDown: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]c", "open/scroll down"),
		),
	}
}
