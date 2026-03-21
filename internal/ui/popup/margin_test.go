package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMarginPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewMarginPopup(tokens, nil)

	// "decorate" should be a switch
	foundDecorate := false
	for _, sw := range p.switches {
		if sw.Label == "decorate" {
			foundDecorate = true
		}
	}
	if !foundDecorate {
		t.Error("expected 'decorate' switch")
	}
}

func TestMarginPopup_OrderOption(t *testing.T) {
	tokens := testTokens()
	p := NewMarginPopup(tokens, nil)

	// "o" should be an option (not a switch) for commit ordering
	foundOrder := false
	for _, opt := range p.options {
		if opt.Key == "o" {
			foundOrder = true
		}
	}
	if !foundOrder {
		t.Error("expected 'o' as an option for commit ordering")
	}

	// Should NOT be in switches
	for _, sw := range p.switches {
		if sw.Key == "o" {
			t.Error("'o' should be an option, not a switch")
		}
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
