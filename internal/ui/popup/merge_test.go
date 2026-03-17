package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMergePopup_NotInMerge_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewMergePopup(tokens, nil, false)

	// Should have ff-only, no-ff, squash, edit switches
	expectedSwitches := []string{"ff-only", "no-ff"}
	if len(p.switches) < len(expectedSwitches) {
		t.Errorf("expected at least %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestMergePopup_NotInMerge_IncompatibleSwitches(t *testing.T) {
	tokens := testTokens()
	p := NewMergePopup(tokens, nil, false)
	p.SetSize(80, 24)

	// Enable ff-only
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	if !p.switches[0].Enabled {
		t.Error("ff-only should be enabled")
	}

	// Enable no-ff (should disable ff-only)
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if p.switches[0].Enabled {
		t.Error("ff-only should be disabled when no-ff is enabled")
	}
	if !p.switches[1].Enabled {
		t.Error("no-ff should be enabled")
	}
}

func TestMergePopup_InMerge_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewMergePopup(tokens, nil, true)

	// When in merge, should have commit and abort actions
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	foundCommit := false
	foundAbort := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "m" && a.Label == "Commit merge" {
				foundCommit = true
			}
			if a.Key == "a" && a.Label == "Abort merge" {
				foundAbort = true
			}
		}
	}

	if !foundCommit {
		t.Error("expected 'Commit merge' action in merge state")
	}
	if !foundAbort {
		t.Error("expected 'Abort merge' action in merge state")
	}
}

func TestMergePopup_NotInMerge_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewMergePopup(tokens, nil, false)

	// When not in merge, should have merge actions
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	foundMerge := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "m" && a.Label == "Merge" {
				foundMerge = true
			}
		}
	}

	if !foundMerge {
		t.Error("expected 'Merge' action when not in merge")
	}
}
