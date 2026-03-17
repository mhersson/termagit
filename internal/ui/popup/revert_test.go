package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRevertPopup_NotInProgress_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRevertPopup(tokens, nil, false)

	// When not in progress, should have Revert group
	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}

	foundRevert := false
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "v" && a.Label == "Commit(s)" {
				foundRevert = true
			}
		}
	}

	if !foundRevert {
		t.Error("expected 'Commit(s)' revert action")
	}
}

func TestRevertPopup_InProgress_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRevertPopup(tokens, nil, true)

	// When in progress, should have continue, skip, abort
	expectedActions := map[string]string{
		"v": "Continue",
		"s": "Skip",
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

func TestRevertPopup_NoEditNotPersisted(t *testing.T) {
	tokens := testTokens()
	p := NewRevertPopup(tokens, nil, false)

	// no-edit should not be persisted
	for _, sw := range p.switches {
		if sw.Label == "no-edit" {
			if sw.Persisted {
				t.Error("no-edit should not be persisted")
			}
			return
		}
	}
}

func TestRevertPopup_RevertCommits(t *testing.T) {
	tokens := testTokens()
	p := NewRevertPopup(tokens, nil, false)
	p.SetSize(80, 24)

	// Press 'v' to revert commits
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	if !p.Done() {
		t.Error("popup should be done after 'v'")
	}

	result := p.Result()
	if result.Action != "v" {
		t.Errorf("expected action 'v', got %q", result.Action)
	}
}
