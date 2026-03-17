package keys

import "github.com/charmbracelet/bubbles/key"

// GlobalKeys defines application-wide key bindings.
type GlobalKeys struct {
	Help       key.Binding
	CmdHistory key.Binding
	Quit       key.Binding
}

// DefaultGlobalKeys returns the default global key bindings.
func DefaultGlobalKeys() GlobalKeys {
	return GlobalKeys{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		CmdHistory: key.NewBinding(
			key.WithKeys("$"),
			key.WithHelp("$", "command history"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}
}
