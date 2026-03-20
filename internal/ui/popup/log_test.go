package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLogPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewLogPopup(tokens, nil)

	// Should have many switches for commit limiting, ordering, formatting
	if len(p.switches) < 5 {
		t.Errorf("expected at least 5 switches, got %d", len(p.switches))
	}

	// Check for some key switches
	found := make(map[string]bool)
	for _, sw := range p.switches {
		found[sw.Label] = true
	}

	expected := []string{"no-merges", "first-parent", "graph"}
	for _, e := range expected {
		if !found[e] {
			t.Errorf("expected switch %q not found", e)
		}
	}
}

func TestLogPopup_Options(t *testing.T) {
	tokens := testTokens()
	p := NewLogPopup(tokens, nil)

	// Should have max-count and other options
	if len(p.options) < 1 {
		t.Errorf("expected at least 1 option, got %d", len(p.options))
	}

	// Check for max-count
	found := false
	for _, opt := range p.options {
		if opt.Label == "max-count" {
			found = true
			if opt.Value != "256" {
				t.Errorf("expected max-count default value '256', got %q", opt.Value)
			}
			break
		}
	}
	if !found {
		t.Error("expected max-count option")
	}
}

func TestLogPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewLogPopup(tokens, nil)

	// Should have Log, Reflog, Other groups
	expectedGroups := []string{"Log", "Reflog", "Other"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestLogPopup_LogCurrent(t *testing.T) {
	tokens := testTokens()
	p := NewLogPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'l' to log current
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	if !p.Done() {
		t.Error("popup should be done after 'l'")
	}

	result := p.Result()
	if result.Action != "l" {
		t.Errorf("expected action 'l', got %q", result.Action)
	}
}

func TestLogPopup_GraphDisabled(t *testing.T) {
	tokens := testTokens()
	p := NewLogPopup(tokens, nil)

	// Graph should be disabled by default
	for _, sw := range p.switches {
		if sw.Label == "graph" {
			if sw.Enabled {
				t.Error("graph switch should be disabled by default")
			}
			return
		}
	}
	t.Error("graph switch not found")
}
