package popup

import (
	"fmt"

	"github.com/mhersson/termagit/internal/theme"
)

// NewIgnorePopup creates the ignore popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/ignore/init.lua
// globalIgnorePath is the path from core.excludesFile; empty means not configured.
func NewIgnorePopup(tokens theme.Tokens, state *State, globalIgnorePath string) Popup {
	p := New("Ignore", tokens)

	// Gitignore actions — labels match Neogit's formatting with path examples
	actions := []Action{
		{Key: "t", Label: "shared at top-level            (.gitignore)"},
		{Key: "s", Label: "shared in sub-directory        (path/to/.gitignore)"},
		{Key: "p", Label: "privately for this repository  (.git/info/exclude)"},
	}

	// Only show global option if global ignore file is configured
	if globalIgnorePath != "" {
		label := fmt.Sprintf("privately for all repositories (%s)", globalIgnorePath)
		actions = append(actions, Action{Key: "g", Label: label})
	}

	p.AddActionGroup("Gitignore", actions)

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("ignore", &p)
	}

	return p
}
