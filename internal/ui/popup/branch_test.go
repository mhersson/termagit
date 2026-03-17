package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBranchPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewBranchPopup(tokens, nil, "main", true)

	// Should have recurse-submodules switch
	if len(p.switches) != 1 {
		t.Errorf("expected 1 switch, got %d", len(p.switches))
	}

	if p.switches[0].Label != "recurse-submodules" {
		t.Errorf("expected 'recurse-submodules', got %q", p.switches[0].Label)
	}
}

func TestBranchPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewBranchPopup(tokens, nil, "main", true)

	// Should have Checkout, Create, Do groups (plus unlabeled group)
	expectedGroups := []string{"Checkout", "", "Create", "Do"}
	if len(p.groups) != len(expectedGroups) {
		t.Errorf("expected %d groups, got %d", len(expectedGroups), len(p.groups))
	}

	for i, expected := range expectedGroups {
		if p.groups[i].Title != expected {
			t.Errorf("group %d: expected %q, got %q", i, expected, p.groups[i].Title)
		}
	}
}

func TestBranchPopup_CheckoutAction(t *testing.T) {
	tokens := testTokens()
	p := NewBranchPopup(tokens, nil, "main", true)
	p.SetSize(80, 24)

	// Press 'b' to checkout branch
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	if !p.Done() {
		t.Error("popup should be done after 'b'")
	}

	result := p.Result()
	if result.Action != "b" {
		t.Errorf("expected action 'b', got %q", result.Action)
	}
}

func TestBranchPopup_ConfigItems_WithBranch(t *testing.T) {
	tokens := testTokens()
	p := NewBranchPopup(tokens, nil, "main", true)

	// When on a branch, should have config items
	if len(p.config) == 0 {
		t.Error("expected config items when on a branch")
	}
}

func TestBranchPopup_ConfigItems_NoBranch(t *testing.T) {
	tokens := testTokens()
	p := NewBranchPopup(tokens, nil, "", false)

	// When not on a branch, should have no config items
	if len(p.config) != 0 {
		t.Errorf("expected no config items when not on a branch, got %d", len(p.config))
	}
}
