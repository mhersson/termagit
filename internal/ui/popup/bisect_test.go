package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBisectPopup_NotInProgress_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, false, false)

	// Should have no-checkout and first-parent switches
	expectedSwitches := []string{"no-checkout", "first-parent"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestBisectPopup_NotInProgress_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, false, false)

	// Should have Start and Scripted actions
	expectedActions := map[string]string{
		"B": "Start",
		"S": "Scripted",
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

func TestBisectPopup_InProgress_NotFinished_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, true, false)

	// Should have Bad, Good, Skip, Reset, Run script actions
	expectedActions := map[string]string{
		"b": "Bad",
		"g": "Good",
		"s": "Skip",
		"r": "Reset",
		"S": "Run script",
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

func TestBisectPopup_InProgress_Finished_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, true, true)

	// When finished, should only have Reset action
	foundReset := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "r" && a.Label == "Reset" {
				foundReset = true
			}
		}
	}

	if !foundReset {
		t.Error("expected 'Reset' action when bisect is finished")
	}
}

func TestBisectPopup_NotInProgress_GroupHeading(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, false, false)

	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}
	if p.groups[0].Title != "Bisect" {
		t.Errorf("not-in-progress group heading: expected %q, got %q", "Bisect", p.groups[0].Title)
	}
}

func TestBisectPopup_InProgress_GroupHeading(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, true, false)

	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}
	if p.groups[0].Title != "Actions" {
		t.Errorf("in-progress group heading: expected %q, got %q", "Actions", p.groups[0].Title)
	}
}

func TestBisectPopup_Start(t *testing.T) {
	tokens := testTokens()
	p := NewBisectPopup(tokens, nil, false, false)
	p.SetSize(80, 24)

	// Press 'B' to start bisect
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})

	if !p.Done() {
		t.Error("popup should be done after 'B'")
	}

	result := p.Result()
	if result.Action != "B" {
		t.Errorf("expected action 'B', got %q", result.Action)
	}
}
