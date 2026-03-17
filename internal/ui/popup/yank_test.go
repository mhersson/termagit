package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestYankPopup_ActionGroups(t *testing.T) {
	tokens := testTokens()
	p := NewYankPopup(tokens, nil, true, true) // hasURL=true, hasTags=true

	// Should have Yank Commit info group
	if len(p.groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(p.groups))
	}

	if p.groups[0].Title != "Yank Commit info" {
		t.Errorf("expected group title 'Yank Commit info', got %q", p.groups[0].Title)
	}
}

func TestYankPopup_Actions(t *testing.T) {
	tokens := testTokens()
	p := NewYankPopup(tokens, nil, true, true)

	// Should have all yank actions
	expectedActions := map[string]string{
		"Y": "Hash",
		"s": "Subject",
		"m": "Message",
		"b": "Message body",
		"u": "URL",
		"d": "Diff",
		"a": "Author",
		"t": "Tags",
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

func TestYankPopup_NoURLOrTags(t *testing.T) {
	tokens := testTokens()
	p := NewYankPopup(tokens, nil, false, false)

	// Should NOT have URL and Tags actions
	for _, g := range p.groups {
		for _, a := range g.Actions {
			if a.Key == "u" {
				t.Error("URL action should not be present when hasURL=false")
			}
			if a.Key == "t" {
				t.Error("Tags action should not be present when hasTags=false")
			}
		}
	}
}

func TestYankPopup_YankHash(t *testing.T) {
	tokens := testTokens()
	p := NewYankPopup(tokens, nil, false, false)
	p.SetSize(80, 24)

	// Press 'Y' to yank hash
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	if !p.Done() {
		t.Error("popup should be done after 'Y'")
	}

	result := p.Result()
	if result.Action != "Y" {
		t.Errorf("expected action 'Y', got %q", result.Action)
	}
}
