package popup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBranchConfigPopup_ConfigItems(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main", []string{"origin"}, "false", "")

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
	p := NewBranchConfigPopup(tokens, nil, "main", []string{"origin"}, "false", "")

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

func TestBranchConfigPopup_RebaseHasChoices(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main", []string{"origin"}, "false", "")

	for _, c := range p.config {
		if c.Label == "branch.main.rebase" {
			if len(c.Choices) < 2 {
				t.Errorf("expected choices for rebase, got %d", len(c.Choices))
			}
			return
		}
	}
	t.Error("rebase config item not found")
}

func TestBranchConfigPopup_AutoSetupMergeHasChoices(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main", []string{"origin"}, "false", "")

	for _, c := range p.config {
		if c.Label == "branch.autoSetupMerge" {
			expected := []string{"always", "true", "false", "inherit", "simple", "default:true"}
			if len(c.Choices) != len(expected) {
				t.Errorf("expected %d choices for autoSetupMerge, got %d", len(expected), len(c.Choices))
			}
			return
		}
	}
	t.Error("autoSetupMerge config item not found")
}

func TestBranchConfigPopup_HasSectionHeadings(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main", []string{"origin"}, "false", "")

	// Should have section headings
	titles := make([]string, 0)
	for _, s := range p.configSections {
		if s.Title != "" {
			titles = append(titles, s.Title)
		}
	}

	expected := []string{"Configure branch", "Configure repository defaults", "Configure branch creation"}
	if len(titles) != len(expected) {
		t.Errorf("expected %d section headings, got %d: %v", len(expected), len(titles), titles)
	}
	for i, e := range expected {
		if i < len(titles) && titles[i] != e {
			t.Errorf("heading %d: expected %q, got %q", i, e, titles[i])
		}
	}
}

func TestBranchConfigPopup_CloseWithQ(t *testing.T) {
	tokens := testTokens()
	p := NewBranchConfigPopup(tokens, nil, "main", []string{"origin"}, "false", "")
	p.SetSize(80, 24)

	// Press 'q' to close
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !p.Done() {
		t.Error("popup should be done after 'q'")
	}
}
