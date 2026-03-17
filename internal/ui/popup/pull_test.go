package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPullPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, "main")

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
	p := NewPullPopup(tokens, nil, "main")

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
	p := NewPullPopup(tokens, nil, "main")

	// Should have pull group with branch name
	if len(p.groups) < 1 {
		t.Fatal("expected at least 1 action group")
	}

	if p.groups[0].Title != "Pull into main from" {
		t.Errorf("expected first group 'Pull into main from', got %q", p.groups[0].Title)
	}
}

func TestPullPopup_PullFromRemote(t *testing.T) {
	tokens := testTokens()
	p := NewPullPopup(tokens, nil, "main")
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
