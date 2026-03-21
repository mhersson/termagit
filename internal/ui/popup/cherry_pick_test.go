package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCherryPickPopup_NotInProgress_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, false)

	// When not in progress, should have Apply here group
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	foundPick := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "A" && a.Label == "Pick" {
				foundPick = true
			}
		}
	}

	if !foundPick {
		t.Error("expected 'Pick' action")
	}
}

func TestCherryPickPopup_InProgress_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, true)

	// When in progress, should have continue, skip, abort
	expectedActions := map[string]string{
		"A": "continue",
		"s": "skip",
		"a": "abort",
	}

	for _, g := range p.groups {
		for _, a := range g.Actions {
			if expected, ok := expectedActions[a.Key]; ok {
				if a.Label != expected {
					t.Errorf("action %q: expected label %q, got %q", a.Key, expected, a.Label)
				}
				delete(expectedActions, a.Key)
			}
		}
	}

	for key, label := range expectedActions {
		t.Errorf("expected action %q (%s) not found", key, label)
	}
}

func TestCherryPickPopup_XSwitchLabel(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, false)

	// Neogit uses -x with cli_prefix="-", label is just "x"
	var found bool
	for _, sw := range p.switches {
		if sw.Key == "x" {
			if sw.Label != "x" {
				t.Errorf("switch -x label: expected %q, got %q", "x", sw.Label)
			}
			found = true
		}
	}

	if !found {
		t.Error("expected switch with key 'x'")
	}
}

func TestCherryPickPopup_FFEnabledByDefault(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, false)

	for _, sw := range p.switches {
		if sw.Label == "ff" {
			if !sw.Enabled {
				t.Error("'ff' switch should be enabled by default")
			}
			return
		}
	}
	t.Error("'ff' switch not found")
}

func TestCherryPickPopup_FFEditIncompatible(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, false)
	p.SetSize(80, 24)

	// Enable edit (should disable ff since they're incompatible)
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	var ffEnabled, editEnabled bool
	for _, sw := range p.switches {
		if sw.Label == "ff" {
			ffEnabled = sw.Enabled
		}
		if sw.Label == "edit" {
			editEnabled = sw.Enabled
		}
	}

	if !editEnabled {
		t.Error("edit should be enabled")
	}
	if ffEnabled {
		t.Error("ff should be disabled when edit is enabled (incompatible)")
	}
}

func TestCherryPickPopup_InProgress_GroupHeading(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, true)

	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}
	if p.groups[0].Title != "Cherry Pick" {
		t.Errorf("in-progress group heading: expected %q, got %q", "Cherry Pick", p.groups[0].Title)
	}
}

func TestCherryPickPopup_Pick(t *testing.T) {
	tokens := testTokens()
	p := NewCherryPickPopup(tokens, nil, false)
	p.SetSize(80, 24)

	// Press 'A' to pick
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})

	if !p.Done() {
		t.Error("popup should be done after 'A'")
	}

	result := p.Result()
	if result.Action != "A" {
		t.Errorf("expected action 'A', got %q", result.Action)
	}
}
