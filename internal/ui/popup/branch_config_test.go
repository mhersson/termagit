package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBranchConfigPopup_ConfigItems(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main")

	// Should have config items for the branch
	if len(p.config) < 5 {
		t.Errorf("expected at least 5 config items, got %d", len(p.config))
	}

	// Check for expected config items
	expectedConfigs := []string{
		"branch.main.description",
		"branch.main.merge",
		"branch.main.remote",
		"branch.main.rebase",
		"branch.main.pushRemote",
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

func TestBranchConfigPopup_RepositoryDefaults(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main")

	// Should have repository default config items
	expectedRepoConfigs := []string{
		"pull.rebase",
		"remote.pushDefault",
	}

	found := make(map[string]bool)
	for _, c := range p.config {
		found[c.Label] = true
	}

	for _, expected := range expectedRepoConfigs {
		if !found[expected] {
			t.Errorf("expected repository default config %q not found", expected)
		}
	}
}

func TestBranchConfigPopup_CloseWithQ(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main")
	p.SetSize(80, 24)

	// Press 'q' to close
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !p.Done() {
		t.Error("popup should be done after 'q'")
	}
}
