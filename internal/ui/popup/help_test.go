package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	keys := DefaultHelpKeys()
	p := NewHelpPopup(tokens, keys)

	// Should have Commands, Applying changes, Essential commands groups
	expectedGroups := []string{"Commands", "Applying changes", "Essential commands"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestHelpPopup_CommandsGroup(t *testing.T) {
	tokens := testTokens()
	keys := DefaultHelpKeys()
	p := NewHelpPopup(tokens, keys)

	// Commands group should have popup keys
	commands := p.groups[0]
	if len(commands.Actions) == 0 {
		t.Error("Commands group should have actions")
	}

	// Check for some expected popup keys
	expectedKeys := map[string]bool{
		"c": false, // Commit
		"b": false, // Branch
		"P": false, // Push
		"p": false, // Pull
	}

	for _, a := range commands.Actions {
		if _, ok := expectedKeys[a.Key]; ok {
			expectedKeys[a.Key] = true
		}
	}

	for key, found := range expectedKeys {
		if !found {
			t.Errorf("expected popup key %q in Commands group", key)
		}
	}
}

func TestHelpPopup_ApplyingChangesGroup(t *testing.T) {
	tokens := testTokens()
	keys := DefaultHelpKeys()
	p := NewHelpPopup(tokens, keys)

	// Applying changes group should have stage/unstage/discard
	applyingChanges := p.groups[1]
	if len(applyingChanges.Actions) == 0 {
		t.Error("Applying changes group should have actions")
	}

	expectedActions := map[string]bool{
		"s": false, // Stage
		"u": false, // Unstage
		"x": false, // Discard
	}

	for _, a := range applyingChanges.Actions {
		if _, ok := expectedActions[a.Key]; ok {
			expectedActions[a.Key] = true
		}
	}

	for key, found := range expectedActions {
		if !found {
			t.Errorf("expected key %q in Applying changes group", key)
		}
	}
}

func TestHelpPopup_CloseWithQ(t *testing.T) {
	tokens := testTokens()
	keys := DefaultHelpKeys()
	p := NewHelpPopup(tokens, keys)
	p.SetSize(80, 24)

	// Press 'q' to close
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !p.Done() {
		t.Error("popup should be done after 'q'")
	}

	result := p.Result()
	if result.Action != "" {
		t.Errorf("expected empty action on close, got %q", result.Action)
	}
}

// DefaultHelpKeys returns default key bindings for testing.
func DefaultHelpKeys() HelpKeys {
	return HelpKeys{
		// Popups
		CommitPopup:     "c",
		BranchPopup:     "b",
		PushPopup:       "P",
		PullPopup:       "p",
		FetchPopup:      "f",
		MergePopup:      "m",
		RebasePopup:     "r",
		RevertPopup:     "v",
		CherryPickPopup: "A",
		ResetPopup:      "X",
		StashPopup:      "Z",
		TagPopup:        "t",
		RemotePopup:     "M",
		WorktreePopup:   "w",
		BisectPopup:     "B",
		IgnorePopup:     "i",
		DiffPopup:       "d",
		LogPopup:        "l",
		MarginPopup:     "L",

		// Actions
		Stage:   "s",
		Unstage: "u",
		Discard: "x",

		// Navigation
		MoveDown:     "j",
		MoveUp:       "k",
		Close:        "q",
		Refresh:      "C-r",
		NextSection:  "C-n",
		PrevSection:  "C-p",
		ToggleFold:   "tab",
	}
}
