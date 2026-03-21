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
		"v": "continue",
		"s": "skip",
		"a": "abort",
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

func TestRevertPopup_OptionsOnlyWhenNotInProgress(t *testing.T) {
	tokens := testTokens()

	// Not in progress: should have options
	p := NewRevertPopup(tokens, nil, false)
	if len(p.options) == 0 {
		t.Error("expected options when not in progress")
	}

	// In progress: should NOT have options
	p = NewRevertPopup(tokens, nil, true)
	if len(p.options) != 0 {
		t.Errorf("expected 0 options when in progress, got %d", len(p.options))
	}
}

func TestRevertPopup_EditEnabledByDefault(t *testing.T) {
	tokens := testTokens()
	p := NewRevertPopup(tokens, nil, false)

	for _, sw := range p.switches {
		if sw.Label == "edit" {
			if !sw.Enabled {
				t.Error("'edit' switch should be enabled by default")
			}
			return
		}
	}
	t.Error("'edit' switch not found")
}

func TestRevertPopup_GroupHeading(t *testing.T) {
	tokens := testTokens()
	p := NewRevertPopup(tokens, nil, false)

	if len(p.groups) == 0 {
		t.Fatal("expected action groups")
	}
	if p.groups[0].Title != "Revert" {
		t.Errorf("group heading: expected %q, got %q", "Revert", p.groups[0].Title)
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
