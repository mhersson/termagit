package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRemoteConfigPopup_ConfigItems(t *testing.T) {
	tokens := testTokens()
	p := NewRemoteConfigPopup(tokens, nil, "origin")

	// Should have config items for the remote
	if len(p.config) < 5 {
		t.Errorf("expected at least 5 config items, got %d", len(p.config))
	}

	// Check for expected config items
	expectedConfigs := []string{
		"remote.origin.url",
		"remote.origin.fetch",
		"remote.origin.pushurl",
		"remote.origin.push",
		"remote.origin.tagOpt",
	}

	found := make(map[string]bool)
	for _, c := range p.config {
		found[c.Label] = true
	}

	for _, expected := range expectedConfigs {
		if !found[expected] {
			t.Errorf("expected config item %q not found", expected)
		}
	}
}

func TestRemoteConfigPopup_CloseWithQ(t *testing.T) {
	tokens := testTokens()
	p := NewRemoteConfigPopup(tokens, nil, "origin")
	p.SetSize(80, 24)

	// Press 'q' to close
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !p.Done() {
		t.Error("popup should be done after 'q'")
	}
}
