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
		"A": "Continue",
		"s": "Skip",
		"a": "Abort",
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

	// The -x switch must be labeled "reference-in-message" per PHASE_6 / Neogit
	var found bool
	for _, sw := range p.switches {
		if sw.Key == "x" {
			if sw.Label != "reference-in-message" {
				t.Errorf("switch -x label: expected %q, got %q", "reference-in-message", sw.Label)
			}
			found = true
		}
	}

	if !found {
		t.Error("expected switch with key 'x'")
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
