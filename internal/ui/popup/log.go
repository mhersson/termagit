package popup

import (
	"github.com/mhersson/conjit/internal/theme"
)

// NewLogPopup creates the log popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/log/init.lua
func NewLogPopup(tokens theme.Tokens, state *State) Popup {
	p := New("Log", tokens)

	// Commit Limiting options
	p.AddOption("n", "max-count", "Limit number of commits", "256")
	p.AddOption("A", "author", "Limit to author", "")
	p.AddOption("F", "grep", "Search messages", "")
	p.AddOption("G", "G", "Search changes", "")
	p.AddOption("S", "S", "Search occurrences", "")
	p.AddOption("L", "L", "Trace line evolution", "")
	p.AddOption("s", "since", "Since date", "")
	p.AddOption("u", "until", "Until date", "")

	// Commit Limiting switches
	p.AddSwitch("m", "no-merges", "Omit merges", false)
	p.AddSwitch("p", "first-parent", "First parent", false)
	p.AddSwitch("i", "invert-grep", "Invert grep", false)

	// History Simplification
	p.AddSwitch("D", "simplify-by-decoration", "Simplify by decoration", false)

	// Commit Ordering
	p.AddSwitch("r", "reverse", "Reverse order", false)
	p.AddSwitch("R", "reflog", "Show reflog", false)

	// Formatting switches
	p.AddSwitch("g", "graph", "Show graph", false)
	p.AddSwitch("c", "color", "Show color", false)
	p.AddSwitch("d", "decorate", "Show decorations", true) // enabled by default

	// Log group
	p.AddActionGroup("Log", []Action{
		{Key: "l", Label: "current"},
		{Key: "h", Label: "HEAD"},
		{Key: "u", Label: "related"},
		{Key: "o", Label: "other"},
		{Key: "L", Label: "local branches"},
		{Key: "b", Label: "all branches"},
		{Key: "a", Label: "all references"},
		{Key: "B", Label: "matching branches"},
		{Key: "T", Label: "matching tags"},
		{Key: "m", Label: "merged"},
	})

	// Reflog group
	p.AddActionGroup("Reflog", []Action{
		{Key: "r", Label: "current"},
		{Key: "H", Label: "HEAD"},
		{Key: "O", Label: "other"},
	})

	// Other group
	p.AddActionGroup("Other", []Action{
		{Key: "s", Label: "shortlog"},
	})

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("log", &p)
	}

	return p
}
