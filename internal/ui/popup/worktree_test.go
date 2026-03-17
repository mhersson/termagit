package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWorktreePopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewWorktreePopup(tokens, nil)

	// Should have Worktree and Do groups
	expectedGroups := []string{"Worktree", "Do"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestWorktreePopup_CheckoutWorktree(t *testing.T) {
	tokens := testTokens()
	p := NewWorktreePopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'w' to checkout worktree
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})

	if !p.Done() {
		t.Error("popup should be done after 'w'")
	}

	result := p.Result()
	if result.Action != "w" {
		t.Errorf("expected action 'w', got %q", result.Action)
	}
}

func TestWorktreePopup_CreateWorktree(t *testing.T) {
	tokens := testTokens()
	p := NewWorktreePopup(tokens, nil)
	p.SetSize(80, 24)

	// Press 'W' to create worktree
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}})

	if !p.Done() {
		t.Error("popup should be done after 'W'")
	}

	result := p.Result()
	if result.Action != "W" {
		t.Errorf("expected action 'W', got %q", result.Action)
	}
}
