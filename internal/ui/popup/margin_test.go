package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMarginPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewMarginPopup(tokens, nil)

	// Should have order and decorate switches
	expectedSwitches := []string{"order", "decorate"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}
}

func TestMarginPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewMarginPopup(tokens, nil)

	// Should have Refresh and Margin groups
	expectedGroups := []string{"Refresh", "Margin"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestMarginPopup_DecorateEnabled(t *testing.T) {
	tokens := testTokens()
	p := NewMarginPopup(tokens, nil)

	// Decorate should be enabled by default
	for _, sw := range p.switches {
		if sw.Label == "decorate" {
			if !sw.Enabled {
				t.Error("decorate switch should be enabled by default")
			}
			return
		}
	}
	t.Error("decorate switch not found")
}

func TestMarginPopup_RefreshBuffer(t *testing.T) {
	tokens := testTokens()
	p := NewMarginPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'g' to refresh buffer
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	if !p.Done() {
		t.Error("popup should be done after 'g'")
	}

	result := p.Result()
	if result.Action != "g" {
		t.Errorf("expected action 'g', got %q", result.Action)
	}
}
