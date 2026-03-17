package popup

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/theme"
)

func TestNewPopup(t *testing.T) {
	tokens := testTokens()
	p := New("Test Popup", tokens)

	if p.title != "Test Popup" {
		t.Errorf("expected title 'Test Popup', got %q", p.title)
	}
	if p.Done() {
		t.Error("new popup should not be done")
	}
}

func TestPopup_AddSwitch(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddSwitch("a", "all", "Stage all", false)

	if len(p.switches) != 1 {
		t.Fatalf("expected 1 switch, got %d", len(p.switches))
	}

	sw := p.switches[0]
	if sw.Key != "a" || sw.Label != "all" || sw.Description != "Stage all" {
		t.Errorf("unexpected switch: %+v", sw)
	}
	if sw.Enabled {
		t.Error("switch should be disabled by default")
	}
}

func TestPopup_ToggleSwitch(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddSwitch("a", "all", "Stage all", false)

	// Toggle with -a
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if !p.switches[0].Enabled {
		t.Error("switch should be enabled after toggle")
	}

	// Toggle again
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if p.switches[0].Enabled {
		t.Error("switch should be disabled after second toggle")
	}
}

func TestPopup_IncompatibleSwitches(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddSwitch("f", "ff-only", "Fast-forward only", false)
	p.AddSwitch("n", "no-ff", "No fast-forward", false)
	p.SetIncompatible("f", "n")

	// Enable ff-only
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	if !p.switches[0].Enabled {
		t.Error("ff-only should be enabled")
	}

	// Enable no-ff (should disable ff-only)
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if p.switches[0].Enabled {
		t.Error("ff-only should be auto-disabled when no-ff is enabled")
	}
	if !p.switches[1].Enabled {
		t.Error("no-ff should be enabled")
	}
}

func TestPopup_AddOption(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddOption("A", "author", "Override author", "")

	if len(p.options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(p.options))
	}

	opt := p.options[0]
	if opt.Key != "A" || opt.Label != "author" || opt.Description != "Override author" {
		t.Errorf("unexpected option: %+v", opt)
	}
}

func TestPopup_AddActionGroup(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit"},
		{Key: "e", Label: "Extend"},
	})

	if len(p.groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(p.groups))
	}

	g := p.groups[0]
	if g.Title != "Create" {
		t.Errorf("expected group title 'Create', got %q", g.Title)
	}
	if len(g.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(g.Actions))
	}
}

func TestPopup_ExecuteAction(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit"},
	})

	// Press 'c' to execute
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if !p.Done() {
		t.Error("popup should be done after action")
	}

	result := p.Result()
	if result.Action != "c" {
		t.Errorf("expected action 'c', got %q", result.Action)
	}
}

func TestPopup_DisabledActionNotExecutable(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit", Disabled: true},
	})

	// Try to press 'c'
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	if p.Done() {
		t.Error("popup should not be done when disabled action is pressed")
	}
}

func TestPopup_CloseWithQ(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !p.Done() {
		t.Error("popup should be done after q")
	}

	result := p.Result()
	if result.Action != "" {
		t.Errorf("expected empty action, got %q", result.Action)
	}
}

func TestPopup_CloseWithEsc(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)

	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if !p.Done() {
		t.Error("popup should be done after escape")
	}
}

func TestPopup_ResultContainsSwitches(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddSwitch("a", "all", "Stage all", false)
	p.AddSwitch("v", "verbose", "Verbose", true) // enabled by default
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit"},
	})

	// Toggle -a on
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Execute action
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	result := p.Result()
	if !result.Switches["all"] {
		t.Error("expected 'all' switch to be enabled in result")
	}
	if !result.Switches["verbose"] {
		t.Error("expected 'verbose' switch to be enabled in result")
	}
}

func TestPopup_View(t *testing.T) {
	tokens := testTokens()
	p := New("Test Popup", tokens)
	p.AddSwitch("a", "all", "Stage all", false)
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit"},
	})

	p.SetSize(80, 24)
	view := p.View()

	// Should contain title
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestPopup_ViewFormat_SwitchRendering(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddSwitch("a", "all", "Stage all modified and deleted files", false)

	p.SetSize(80, 24)
	view := p.View()

	// Should render as: -a Stage all modified and deleted files (--all)
	// The key prefix is always "-" for both switches and options
	if !strings.Contains(view, "-a") {
		t.Error("switch should have -key format")
	}
	if !strings.Contains(view, "Stage all modified and deleted files") {
		t.Error("switch should contain description")
	}
	if !strings.Contains(view, "(--all)") {
		t.Error("switch should have (--flag) format at end")
	}
}

func TestPopup_ViewFormat_OptionRendering(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddOption("A", "author", "Override the author", "")

	p.SetSize(80, 24)
	view := p.View()

	// Should render as: -A Override the author (--author=)
	// The key prefix is "-" (not "=" as was before)
	if !strings.Contains(view, "-A") {
		t.Error("option should have -key format (not =key)")
	}
	if !strings.Contains(view, "Override the author") {
		t.Error("option should contain description")
	}
	if !strings.Contains(view, "(--author=)") {
		t.Error("option should have (--option=) format at end")
	}
}

func TestPopup_ViewFormat_OptionWithValue(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddOption("A", "author", "Override the author", "John Doe")

	p.SetSize(80, 24)
	view := p.View()

	// Option with value should render as: -A Override the author (--author=John Doe)
	if !strings.Contains(view, "(--author=John Doe)") {
		t.Error("option with value should show value after =")
	}
}

func TestPopup_ViewFormat_NoTitle(t *testing.T) {
	tokens := testTokens()
	p := New("Test Popup", tokens)
	p.AddSwitch("a", "all", "Stage all", false)

	p.SetSize(80, 24)
	view := p.View()

	// Neogit popups don't have a centered title at top
	// The view should start with Arguments section, not a title
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("view should have content")
	}

	// First non-empty line should be "Arguments", not the title
	firstLine := strings.TrimSpace(lines[0])
	if strings.Contains(firstLine, "Test Popup") {
		t.Error("popup should not have centered title at top")
	}
}

func TestPopup_ViewFormat_ArgumentsSection(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddSwitch("a", "all", "Stage all", false)
	p.AddOption("A", "author", "Override", "")

	p.SetSize(80, 24)
	view := p.View()

	// Both switches and options should be under "Arguments" heading
	if !strings.Contains(view, "Arguments") {
		t.Error("popup should have Arguments section header")
	}
}

func TestPopup_ViewFormat_ActionGrid(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddActionGroup("Create", []Action{
		{Key: "c", Label: "Commit"},
	})
	p.AddActionGroup("Edit HEAD", []Action{
		{Key: "e", Label: "Extend"},
	})

	p.SetSize(80, 24)
	view := p.View()

	// Action groups should be rendered as columns (side by side)
	// Both "Create" and "Edit HEAD" should be on the same line
	lines := strings.Split(view, "\n")
	foundBothOnSameLine := false
	for _, line := range lines {
		if strings.Contains(line, "Create") && strings.Contains(line, "Edit HEAD") {
			foundBothOnSameLine = true
			break
		}
	}
	if !foundBothOnSameLine {
		t.Error("action group headers should be on the same line (grid layout)")
	}
}

func TestPopup_ConfigItems(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddConfig("d", "branch.main.description", "Description", "My branch")

	if len(p.config) != 1 {
		t.Fatalf("expected 1 config item, got %d", len(p.config))
	}

	cfg := p.config[0]
	if cfg.Key != "d" || cfg.Label != "branch.main.description" {
		t.Errorf("unexpected config: %+v", cfg)
	}
	if cfg.Value != "My branch" {
		t.Errorf("expected value 'My branch', got %q", cfg.Value)
	}
}

func TestPopup_Spacer(t *testing.T) {
	tokens := testTokens()
	p := New("Test", tokens)
	p.AddActionGroup("Edit HEAD", []Action{
		{Key: "e", Label: "Extend"},
		{Spacer: true},
		{Key: "a", Label: "Amend"},
	})

	if len(p.groups[0].Actions) != 3 {
		t.Errorf("expected 3 items (including spacer), got %d", len(p.groups[0].Actions))
	}
	if !p.groups[0].Actions[1].Spacer {
		t.Error("second item should be a spacer")
	}
}

func testTokens() theme.Tokens {
	raw := theme.RawTokens{
		Normal:       "#ffffff",
		PopupBorder:  "#888888",
		PopupTitle:   "#ffffff",
		PopupKey:     "#ff00ff",
		PopupKeyBg:   "#333333",
		PopupSwitch:  "#00ff00",
		PopupOption:  "#ffff00",
		PopupAction:  "#00ffff",
		PopupSection: "#ff8800",
	}
	return theme.Compile(raw)
}
