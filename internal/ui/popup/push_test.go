package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPushPopup_Switches(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{Branch: "main"})

	// Should have all Neogit push switches
	expectedSwitches := []string{
		"force-with-lease", "force", "no-verify", "dry-run",
		"set-upstream", "tags", "follow-tags",
	}
	if len(p.switches) != len(expectedSwitches) {
		t.Errorf("expected %d switches, got %d", len(expectedSwitches), len(p.switches))
	}

	for i, expected := range expectedSwitches {
		if p.switches[i].Label != expected {
			t.Errorf("switch %d: expected %q, got %q", i, expected, p.switches[i].Label)
		}
	}
}

func TestPushPopup_ForceNotPersisted(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{Branch: "main"})

	// force-with-lease and force should not be persisted
	for _, sw := range p.switches {
		if sw.Label == "force-with-lease" || sw.Label == "force" {
			if sw.Persisted {
				t.Errorf("%s should not be persisted", sw.Label)
			}
		}
	}
}

func TestPushPopup_ActionGroups_NotDetached(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{Branch: "main"})

	// When not detached, should have "Push main to" group
	if len(p.groups) < 1 {
		t.Fatal("expected at least 1 action group")
	}

	if p.groups[0].Title != "Push main to" {
		t.Errorf("expected first group 'Push main to', got %q", p.groups[0].Title)
	}
}

func TestPushPopup_ActionGroups_Detached(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{IsDetached: true})

	// When detached, should have "Push" group without branch name
	if len(p.groups) < 1 {
		t.Fatal("expected at least 1 action group")
	}

	if p.groups[0].Title != "Push" {
		t.Errorf("expected first group 'Push', got %q", p.groups[0].Title)
	}
}

func TestPushPopup_DynamicLabels(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{
		Branch:          "main",
		PushRemoteLabel: "origin/main",
		UpstreamLabel:   "upstream/main",
	})

	// First group actions should use resolved labels
	if p.groups[0].Actions[0].Label != "origin/main" {
		t.Errorf("expected 'origin/main', got %q", p.groups[0].Actions[0].Label)
	}
	if p.groups[0].Actions[1].Label != "upstream/main" {
		t.Errorf("expected 'upstream/main', got %q", p.groups[0].Actions[1].Label)
	}
}

func TestPushPopup_FallbackLabels(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{Branch: "main"})

	// Without resolved labels, should fall back to defaults
	if p.groups[0].Actions[0].Label != "pushRemote" {
		t.Errorf("expected 'pushRemote', got %q", p.groups[0].Actions[0].Label)
	}
	if p.groups[0].Actions[1].Label != "@{upstream}" {
		t.Errorf("expected '@{upstream}', got %q", p.groups[0].Actions[1].Label)
	}
}

func TestPushPopup_PushToRemote(t *testing.T) {
	tokens := testTokens()
	p := NewPushPopup(tokens, nil, PushPopupParams{Branch: "main"})
	p.SetSize(80, 24)

	// Press 'p' to push to remote
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	if !p.Done() {
		t.Error("popup should be done after 'p'")
	}

	result := p.Result()
	if result.Action != "p" {
		t.Errorf("expected action 'p', got %q", result.Action)
	}
}
