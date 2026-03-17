package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestResetPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewResetPopup(tokens, nil)

	// Should have Reset and Reset this groups
	expectedGroups := []string{"Reset", "Reset this"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestResetPopup_ResetThisActions(t *testing.T) {
	tokens := testTokens()
	p := NewResetPopup(tokens, nil)

	// Reset this group should have mixed, soft, hard, keep, index, worktree
	expectedActions := map[string]string{
		"m": "mixed (HEAD and index)",
		"s": "soft (HEAD only)",
		"h": "hard (HEAD, index and files)",
		"k": "keep (HEAD and index, keeping uncommitted)",
		"i": "index (only)",
		"w": "worktree (only)",
	}

	resetThis := p.groups[1] // Second group
	for _, a := range resetThis.Actions {
		if expected, ok := expectedActions[a.Key]; ok {
			if a.Label != expected {
				t.Errorf("action %q: expected label %q, got %q", a.Key, expected, a.Label)
			}
			delete(expectedActions, a.Key)
		}
	}

	for key, label := range expectedActions {
		t.Errorf("expected action %q (%s) not found", key, label)
	}
}

func TestResetPopup_HardReset(t *testing.T) {
	tokens := testTokens()
	p := NewResetPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'h' for hard reset
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	if !p.Done() {
		t.Error("popup should be done after 'h'")
	}

	result := p.Result()
	if result.Action != "h" {
		t.Errorf("expected action 'h', got %q", result.Action)
	}
}
