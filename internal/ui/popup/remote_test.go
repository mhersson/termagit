package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRemotePopup_ConfigItems(t *testing.T) {
	tokens := testTokens()
	p := NewRemotePopup(tokens, nil, "origin")

	// Should have config items for origin
	if len(p.config) == 0 {
		t.Error("expected config items for remote")
	}

	// Check for url config
	found := false
	for _, c := range p.config {
		if c.Label == "remote.origin.url" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected remote.origin.url config item")
	}
}

func TestRemotePopup_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewRemotePopup(tokens, nil, "origin")

	// Should have Add, Rename, Remove, Configure actions
	expectedActions := map[string]string{
		"a": "Add",
		"r": "Rename",
		"x": "Remove",
		"C": "Configure...",
		"p": "Prune stale branches",
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

func TestRemotePopup_FetchSwitch(t *testing.T) {
	tokens := testTokens()
	p := NewRemotePopup(tokens, nil, "origin")

	// Should have "f" switch for "Fetch after add", enabled by default
	var found bool
	for _, sw := range p.switches {
		if sw.Key == "f" && sw.Label == "f" {
			if !sw.Enabled {
				t.Error("'f' switch should be enabled by default")
			}
			found = true
		}
	}
	if !found {
		t.Error("expected switch -f for 'Fetch after add'")
	}
}

func TestRemotePopup_GroupHeadings(t *testing.T) {
	tokens := testTokens()
	p := NewRemotePopup(tokens, nil, "origin")

	if len(p.groups) < 2 {
		t.Fatalf("expected at least 2 action groups, got %d", len(p.groups))
	}
	if p.groups[0].Title != "Actions" {
		t.Errorf("first group heading: expected %q, got %q", "Actions", p.groups[0].Title)
	}
}

func TestRemotePopup_TwoActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewRemotePopup(tokens, nil, "origin")

	if len(p.groups) != 2 {
		t.Errorf("expected 2 action groups, got %d", len(p.groups))
	}
}

func TestRemotePopup_AddRemote(t *testing.T) {
	tokens := testTokens()
	p := NewRemotePopup(tokens, nil, "origin")
	p.SetSize(80, 24)

	// Press 'a' to add remote
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if !p.Done() {
		t.Error("popup should be done after 'a'")
	}

	result := p.Result()
	if result.Action != "a" {
		t.Errorf("expected action 'a', got %q", result.Action)
	}
}
