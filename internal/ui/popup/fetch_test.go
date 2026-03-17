package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFetchPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewFetchPopup(tokens, nil)

	// Should have prune, tags, force switches
	expectedSwitches := []string{"prune", "tags", "force"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestFetchPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewFetchPopup(tokens, nil)

	// Should have Fetch from, Fetch, Configure groups
	expectedGroups := []string{"Fetch from", "Fetch", "Configure"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestFetchPopup_FetchAllRemotes(t *testing.T) {
	tokens := testTokens()
	p := NewFetchPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'a' to fetch all remotes
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if !p.Done() {
		t.Error("popup should be done after 'a'")
	}

	result := p.Result()
	if result.Action != "a" {
		t.Errorf("expected action 'a', got %q", result.Action)
	}
}
