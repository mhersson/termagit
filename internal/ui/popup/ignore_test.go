package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIgnorePopup_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewIgnorePopup(tokens, nil, "~/.gitignore_global")

	// Should have gitignore actions including global with path
	found := make(map[string]string)
	for _, g := range p.groups {
		for _, a := range g.Actions {
			found[a.Key] = a.Label
		}
	}

	expected := map[string]string{
		"t": "shared at top-level            (.gitignore)",
		"s": "shared in sub-directory        (path/to/.gitignore)",
		"p": "privately for this repository  (.git/info/exclude)",
	}
	for key, label := range expected {
		if found[key] != label {
			t.Errorf("action %q: expected label %q, got %q", key, label, found[key])
		}
	}

	// Global should include the path
	globalLabel, ok := found["g"]
	if !ok {
		t.Fatal("expected global ignore action")
	}
	if globalLabel != "privately for all repositories (~/.gitignore_global)" {
		t.Errorf("expected global label with path, got %q", globalLabel)
	}
}

func TestIgnorePopup_NoGlobalIgnore(t *testing.T) {
	tokens := testTokens()
	p := NewIgnorePopup(tokens, nil, "") // no global ignore path

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
	p := NewIgnorePopup(tokens, nil, "")
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
