package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewLogPopup creates the log popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/log/init.lua
func NewLogPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Log", tokens)

	// Commit Limiting (options use "-" prefix, matching Neogit)
	p.AddOptionWithPrefix("-", "n", "max-count", "Limit number of commits", "256")
	p.AddOptionWithPrefix("-", "A", "author", "Limit to author", "")
	p.AddOptionWithPrefix("-", "F", "grep", "Search messages", "")
	p.AddOptionWithPrefix("-", "G", "G", "Search changes", "")
	p.AddOptionWithPrefix("-", "S", "S", "Search occurrences", "")
	p.AddOptionWithPrefix("-", "L", "L", "Trace line evolution", "")
	p.AddOptionWithPrefix("-", "s", "since", "Limit to commits since", "")
	p.AddOptionWithPrefix("-", "u", "until", "Limit to commits until", "")
	p.AddSwitchWithPrefix("=", "m", "no-merges", "Omit merges", false)
	p.AddSwitchWithPrefix("=", "p", "first-parent", "First parent", false)
	p.AddSwitch("i", "invert-grep", "Invert search messages", false)

	// History Simplification
	p.AddSwitch("D", "simplify-by-decoration", "Simplify by decoration", false)
	p.AddSwitch("f", "follow", "Follow renames when showing single-file log", false)

	// Commit Ordering
	p.AddSwitch("r", "reverse", "Reverse order", false)
	p.AddOptionWithChoices("o", "topo", "Order commits by", "",
		[]string{"topo", "author-date", "date"})
	p.AddSwitchWithPrefix("=", "R", "reflog", "List reflog", false)

	// Formatting
	p.AddSwitch("g", "graph", "Show graph", false)
	p.AddSwitch("c", "color", "Show graph in color", false)
	p.AddSwitch("d", "decorate", "Show refnames", true) // enabled by default
	p.AddSwitchWithPrefix("=", "S", "show-signature", "Show signatures", false)

	// Incompatible switches (matching Neogit)
	p.SetIncompatible("r", "g") // reverse ↔ graph
	p.SetIncompatible("r", "c") // reverse ↔ color

	// Log group
	p.AddActionGroup("Log", []Action{
		{Key: "l", Label: "current"},
		{Key: "h", Label: "HEAD"},
		{Key: "u", Label: "related"},
		{Key: "o", Label: "other"},
		{Spacer: true},
		{Key: "L", Label: "local branches"},
		{Key: "b", Label: "all branches"},
		{Key: "a", Label: "all references"},
		{Key: "B", Label: "matching branches", Disabled: true},
		{Key: "T", Label: "matching tags", Disabled: true},
		{Key: "m", Label: "merged", Disabled: true},
	})

	// Reflog group
	p.AddActionGroup("Reflog", []Action{
		{Key: "r", Label: "current"},
		{Key: "H", Label: "HEAD"},
		{Key: "O", Label: "other"},
	})

	// Other group
	p.AddActionGroup("Other", []Action{
		{Key: "s", Label: "shortlog", Disabled: true},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("log", &p)
	}

	return p
}
