package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIgnorePopup_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewIgnorePopup(tokens, nil, true) // hasGlobalIgnore = true

	// Should have gitignore actions
	expectedActions := map[string]string{
		"t": "shared at top-level (.gitignore)",
		"s": "shared in sub-directory",
		"p": "privately for this repository",
		"g": "globally for this user",
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

func TestIgnorePopup_NoGlobalIgnore(t *testing.T) {
	tokens := testTokens()
	p := NewIgnorePopup(tokens, nil, false) // hasGlobalIgnore = false

	// Should NOT have global action
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "g" {
				t.Error("global ignore action should not be present when no global ignore file")
			}
		}
	}
}

func TestIgnorePopup_TopLevel(t *testing.T) {
	tokens := testTokens()
	p := NewIgnorePopup(tokens, nil, false)
	p.SetSize(80, 24)

	// Press 't' to add to top-level .gitignore
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	if !p.Done() {
		t.Error("popup should be done after 't'")
	}

	result := p.Result()
	if result.Action != "t" {
		t.Errorf("expected action 't', got %q", result.Action)
	}
}
