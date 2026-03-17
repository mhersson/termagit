package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommitPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)

	// Should have all Neogit commit switches
	expectedSwitches := []string{"all", "allow-empty", "verbose", "no-verify", "reset-author", "signoff"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestCommitPopup_Options(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)

	// Should have all Neogit commit options
	expectedOptions := []string{"author", "gpg-sign", "reuse-message"}
	if len(p.options) != len(expectedOptions) {
		t.Errorf("expected %d options, got %d", len(expectedOptions), len(p.options))
	}

	for i, expected := range expectedOptions {
		if p.options[i].Label != expected {
			t.Errorf("option %d: expected %q, got %q", i, expected, p.options[i].Label)
		}
	}
}

func TestCommitPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)

	// Should have the correct action groups
	expectedGroups := []string{"Create", "Edit HEAD", "Edit", "Edit and rebase", "Spread across commits"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestCommitPopup_CreateAction(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'c' to commit
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !p.Done() {
		t.Error("popup should be done after 'c'")
	}

	result := p.Result()
	if result.Action != "c" {
		t.Errorf("expected action 'c', got %q", result.Action)
	}
}

func TestCommitPopup_AmendAction(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'a' to amend
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if !p.Done() {
		t.Error("popup should be done after 'a'")
	}

	result := p.Result()
	if result.Action != "a" {
		t.Errorf("expected action 'a', got %q", result.Action)
	}
}

func TestCommitPopup_AllowEmptyNotPersisted(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)

	// allow-empty should not be persisted
	for _, sw := range p.switches {
		if sw.Label == "allow-empty" {
			if sw.Persisted {
				t.Error("allow-empty should not be persisted")
			}
			return
		}
	}
	t.Error("allow-empty switch not found")
}

func TestCommitPopup_SwitchKeys(t *testing.T) {
	tokens := testTokens()
	p := NewCommitPopup(tokens, nil)

	// Verify switch keys match Neogit
	expectedKeys := map[string]string{
		"all":          "a",
		"allow-empty":  "e",
		"verbose":      "v",
		"no-verify":    "h",
		"reset-author": "R",
		"signoff":      "s",
	}

	for _, sw := range p.switches {
		if expected, ok := expectedKeys[sw.Label]; ok {
			if sw.Key != expected {
				t.Errorf("switch %q: expected key %q, got %q", sw.Label, expected, sw.Key)
			}
		}
	}
}
