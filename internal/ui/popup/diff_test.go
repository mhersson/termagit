package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDiffPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil, true, false)

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
	p := NewDiffPopup(tokens, nil, true, false)
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
	p := NewDiffPopup(tokens, nil, true, false)
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

func TestDiffPopup_NoItem_DiffThisDisabled(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil, false, false)

	// When no item selected, "d" and "h" actions should be disabled
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "d" && !a.Disabled {
				t.Error("'d' action should be disabled when no item selected")
			}
		}
	}
}

func TestDiffPopup_CommitSelected_ThisToHead(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil, true, true)

	// When a commit is selected, "h" should not be disabled
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "h" && a.Disabled {
				t.Error("'h' action should not be disabled when commit is selected")
			}
		}
	}
}

func TestDiffPopup_NoCommit_ThisToHeadDisabled(t *testing.T) {
	tokens := testTokens()
	p := NewDiffPopup(tokens, nil, true, false) // has item but not a commit

	// "h" should be disabled
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "h" && !a.Disabled {
				t.Error("'h' action should be disabled when no commit selected")
			}
		}
	}
}
