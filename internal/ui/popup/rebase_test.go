package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRebasePopup_NotInRebase_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false, "main", "")

	// Should have multiple switches
	if len(p.switches) < 5 {
		t.Errorf("expected at least 5 switches, got %d", len(p.switches))
	}

	// Check for some key switches
	found := make(map[string]bool)
	for _, sw := range p.switches {
		found[sw.Label] = true
	}

	expected := []string{"keep-empty", "autosquash", "autostash", "interactive", "no-verify"}
	for _, e := range expected {
		if !found[e] {
			t.Errorf("expected switch %q not found", e)
		}
	}
}

func TestRebasePopup_InRebase_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, true, "", "")

	// When in rebase, should have continue, skip, edit, abort actions
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	expectedActions := map[string]string{
		"r": "Continue",
		"s": "Skip",
		"e": "Edit",
		"a": "Abort",
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

func TestRebasePopup_NotInRebase_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false, "main", "")

	// When not in rebase, should have rebase actions
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	foundInteractive := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "i" && a.Label == "interactively" {
				foundInteractive = true
			}
		}
	}

	if !foundInteractive {
		t.Error("expected 'interactively' action when not in rebase")
	}
}

func TestRebasePopup_InteractiveAction(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false, "main", "")
	p.SetSize(80, 24)

	// Press 'i' for interactive rebase
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if !p.Done() {
		t.Error("popup should be done after 'i'")
	}

	result := p.Result()
	if result.Action != "i" {
		t.Errorf("expected action 'i', got %q", result.Action)
	}
}

func TestRebasePopup_BranchNameInHeading(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, false, "feature", "main")

	found := false
	for _, g := range p.groups {
		if g.Title == "Rebase feature onto" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected group heading 'Rebase feature onto', got groups: %v", groupTitles(p))
	}
}

func TestRebasePopup_InRebase_GroupHeading(t *testing.T) {
	tokens := testTokens()
	p := NewRebasePopup(tokens, nil, true, "", "")

	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}
	if p.groups[0].Title != "Actions" {
		t.Errorf("in-rebase group heading: expected %q, got %q", "Actions", p.groups[0].Title)
	}
}

func TestRebasePopup_BaseBranchConditional(t *testing.T) {
	tokens := testTokens()

	// With base branch same as current - should not show "b" action
	p := NewRebasePopup(tokens, nil, false, "main", "main")
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "b" {
				t.Error("'b' action should not appear when base branch == current branch")
			}
		}
	}

	// With different base branch - should show "b" action
	p = NewRebasePopup(tokens, nil, false, "feature", "main")
	foundB := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "b" && a.Label == "main" {
				foundB = true
			}
		}
	}
	if !foundB {
		t.Error("expected 'b' action labeled 'main' when base branch differs from current")
	}

	// With empty base branch - should not show "b" action
	p = NewRebasePopup(tokens, nil, false, "feature", "")
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "b" {
				t.Error("'b' action should not appear when base branch is empty")
			}
		}
	}
}

func groupTitles(p Popup) []string {
	var titles []string
	for _, g := range p.groups {
		titles = append(titles, g.Title)
	}
	return titles
}

func TestRebasePopup_OpenRebaseEditorMsg(t *testing.T) {
	// OpenRebaseEditorMsg must be exported so the app layer can receive it.
	// Verify the type exists and is usable as a tea.Msg.
	var msg tea.Msg = OpenRebaseEditorMsg{}
	if _, ok := msg.(OpenRebaseEditorMsg); !ok {
		t.Error("OpenRebaseEditorMsg should be usable as tea.Msg")
	}
}
