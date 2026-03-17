package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRebasePopup_NotInRebase_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false)

	// Should have multiple switches
	if len(p.switches) < 5 {
		t.Errorf("expected at least 5 switches, got %d", len(p.switches))
	}

	// Check for some key switches
	found := make(map[string]bool)
	for _, sw := range p.switches {
		found[sw.Label] = true
	}

	expected := []string{"keep-empty", "autosquash", "autostash", "interactive", "no-verify"}
	for _, e := range expected {
		if !found[e] {
			t.Errorf("expected switch %q not found", e)
		}
	}
}

func TestRebasePopup_InRebase_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, true)

	// When in rebase, should have continue, skip, edit, abort actions
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	expectedActions := map[string]string{
		"r": "Continue",
		"s": "Skip",
		"e": "Edit",
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

func TestRebasePopup_NotInRebase_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false)

	// When not in rebase, should have rebase actions
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	foundInteractive := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "i" && a.Label == "interactively" {
				foundInteractive = true
			}
		}
	}

	if !foundInteractive {
		t.Error("expected 'interactively' action when not in rebase")
	}
}

func TestRebasePopup_InteractiveAction(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false)
	p.SetSize(80, 24)

	// Press 'i' for interactive rebase
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if !p.Done() {
		t.Error("popup should be done after 'i'")
	}

	result := p.Result()
	if result.Action != "i" {
		t.Errorf("expected action 'i', got %q", result.Action)
	}
}
