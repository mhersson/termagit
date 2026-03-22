package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// NewYankPopup creates the yank popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/yank/init.lua
func NewYankPopup(tokens theme.Tokens, state *State, hasURL, hasTags bool) Popup {
	p := New("Yank", tokens)

	// Yank Commit info actions
	actions := []Action{
		{Key: "Y", Label: "Hash"},
		{Key: "s", Label: "Subject"},
		{Key: "m", Label: "Message (subject and body)"},
		{Key: "b", Label: "Message body"},
	}

	// URL only shown when URL can be determined
	if hasURL {
		actions = append(actions, Action{Key: "u", Label: "URL"})
	}

	actions = append(actions,
		Action{Key: "d", Label: "Diff"},
		Action{Key: "a", Label: "Author"},
	)

	// Tags only shown when tags exist
	if hasTags {
		actions = append(actions, Action{Key: "t", Label: "Tags"})
	}

	p.AddActionGroup("Yank Commit info", actions)

	// Apply saved state if provided
	if state != nil {
		state.ApplyToPopup("yank", &p)
	}

	return p
}
