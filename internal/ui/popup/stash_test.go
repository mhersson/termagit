package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStashPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewStashPopup(tokens, nil)

	// Should have include-untracked and all switches
	expectedSwitches := []string{"include-untracked", "all"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestStashPopup_IncompatibleSwitches(t *testing.T) {
	tokens := testTokens()
	p := NewStashPopup(tokens, nil)
	p.SetSize(80, 24)

	// Enable include-untracked
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	if !p.switches[0].Enabled {
		t.Error("include-untracked should be enabled")
	}

	// Enable all (should disable include-untracked)
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if p.switches[0].Enabled {
		t.Error("include-untracked should be disabled when all is enabled")
	}
	if !p.switches[1].Enabled {
		t.Error("all should be enabled")
	}
}

func TestStashPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewStashPopup(tokens, nil)

	// Should have Stash, Snapshot, Use, Inspect, Transform groups
	expectedGroups := []string{"Stash", "Snapshot", "Use", "Inspect", "Transform"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestStashPopup_StashBoth(t *testing.T) {
	tokens := testTokens()
	p := NewStashPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'z' to stash both
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	if !p.Done() {
		t.Error("popup should be done after 'z'")
	}

	result := p.Result()
	if result.Action != "z" {
		t.Errorf("expected action 'z', got %q", result.Action)
	}
}
