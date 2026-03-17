package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDiffPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil)

	// Should have Diff and Show groups
	expectedGroups := []string{"Diff", "", "Show"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestDiffPopup_DiffThis(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'd' to diff this
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if !p.Done() {
		t.Error("popup should be done after 'd'")
	}

	result := p.Result()
	if result.Action != "d" {
		t.Errorf("expected action 'd', got %q", result.Action)
	}
}

func TestDiffPopup_ShowCommit(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'c' to show commit
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !p.Done() {
		t.Error("popup should be done after 'c'")
	}

	result := p.Result()
	if result.Action != "c" {
		t.Errorf("expected action 'c', got %q", result.Action)
	}
}
