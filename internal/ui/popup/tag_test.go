package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTagPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewTagPopup(tokens, nil)

	// Should have force, annotate, sign switches
	expectedSwitches := []string{"force", "annotate", "sign"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestTagPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewTagPopup(tokens, nil)

	// Should have Create, Do groups
	expectedGroups := []string{"Create", "Do"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestTagPopup_CreateTag(t *testing.T) {
	tokens := testTokens()
	p := NewTagPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 't' to create tag
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	if !p.Done() {
		t.Error("popup should be done after 't'")
	}

	result := p.Result()
	if result.Action != "t" {
		t.Errorf("expected action 't', got %q", result.Action)
	}
}
