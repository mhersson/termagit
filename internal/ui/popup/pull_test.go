package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPullPopup_ConfigItem(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, PullPopupParams{Branch: "main"})

	// Should have config item for branch.main.rebase
	found := false
	for _, c := range p.config {
		if c.Key == "r" && c.Label == "branch.main.rebase" {
			found = true
		}
	}
	if !found {
		t.Error("expected config item 'r' for branch.main.rebase")
	}
}

func TestPullPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, PullPopupParams{Branch: "main"})

	// Should have all Neogit pull switches
	expectedSwitches := []string{"ff-only", "rebase", "autostash", "tags", "force"}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestPullPopup_RebaseNotPersisted(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, PullPopupParams{Branch: "main"})

	// rebase and force should not be persisted
	for _, sw := range p.switches {
		if sw.Label == "rebase" || sw.Label == "force" {
			if sw.Persisted {
				t.Errorf("%s should not be persisted", sw.Label)
			}
		}
	}
}

func TestPullPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, PullPopupParams{Branch: "main"})

	// Should have pull group with branch name
	if len(p.groups) < 1 {
		t.Fatal("expected at least 1 action group")
	}

	if p.groups[0].Title != "Pull into main from" {
		t.Errorf("expected first group 'Pull into main from', got %q", p.groups[0].Title)
	}
}

func TestPullPopup_DetachedHead(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, PullPopupParams{IsDetached: true})

	// When detached, should have "Pull from" (not "Pull into X from")
	if len(p.groups) < 1 {
		t.Fatal("expected at least 1 action group")
	}
	if p.groups[0].Title != "Pull from" {
		t.Errorf("expected 'Pull from', got %q", p.groups[0].Title)
	}

	// Should NOT have config items when detached
	if len(p.config) != 0 {
		t.Errorf("expected 0 config items when detached, got %d", len(p.config))
	}
}

func TestPullPopup_PullFromRemote(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, PullPopupParams{Branch: "main"})
	p.SetSize(80, 24)

	// Press 'p' to pull from remote
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	if !p.Done() {
		t.Error("popup should be done after 'p'")
	}

	result := p.Result()
	if result.Action != "p" {
		t.Errorf("expected action 'p', got %q", result.Action)
	}
}
