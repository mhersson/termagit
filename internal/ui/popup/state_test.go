package popup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestState_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	state := NewState()

	// Set some switch values
	state.SetSwitch("commit", "all", true)
	state.SetSwitch("commit", "verbose", false)
	state.SetSwitch("push", "tags", true)

	// Save
	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file exists
	stateFile := filepath.Join(dir, "conjit", "popup_state.toml")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Fatalf("state file not created at %s", stateFile)
	}

	// Load into new state
	state2 := NewState()
	if err := state2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify values
	if !state2.GetSwitch("commit", "all") {
		t.Error("expected commit.all to be true")
	}
	if state2.GetSwitch("commit", "verbose") {
		t.Error("expected commit.verbose to be false")
	}
	if !state2.GetSwitch("push", "tags") {
		t.Error("expected push.tags to be true")
	}
}

func TestState_LoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	state := NewState()
	err := state.Load()

	// Should not error on missing file
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
}

func TestState_Options(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	state := NewState()
	state.SetOption("log", "max-count", "256")
	state.SetOption("commit", "author", "Test Author")

	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	state2 := NewState()
	if err := state2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if v := state2.GetOption("log", "max-count"); v != "256" {
		t.Errorf("expected max-count=256, got %q", v)
	}
	if v := state2.GetOption("commit", "author"); v != "Test Author" {
		t.Errorf("expected author='Test Author', got %q", v)
	}
}

func TestState_NonPersistedNotSaved(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	state := NewState()

	// These are the switches that should NOT be persisted
	nonPersisted := []struct {
		popup, label string
	}{
		{"push", "force-with-lease"},
		{"push", "force"},
		{"pull", "rebase"},
		{"commit", "allow-empty"},
		{"revert", "no-edit"},
	}

	// Set them all
	for _, np := range nonPersisted {
		state.SetSwitch(np.popup, np.label, true)
	}

	// Mark them as non-persisted
	for _, np := range nonPersisted {
		state.MarkNonPersisted(np.popup, np.label)
	}

	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load into new state
	state2 := NewState()
	if err := state2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// None of these should be true
	for _, np := range nonPersisted {
		if state2.GetSwitch(np.popup, np.label) {
			t.Errorf("non-persisted switch %s.%s should not be loaded", np.popup, np.label)
		}
	}
}

func TestState_DefaultFalseForUnset(t *testing.T) {
	state := NewState()

	// Unset switches should return false
	if state.GetSwitch("nonexistent", "switch") {
		t.Error("unset switch should return false")
	}

	// Unset options should return empty string
	if v := state.GetOption("nonexistent", "option"); v != "" {
		t.Errorf("unset option should return empty string, got %q", v)
	}
}

func TestState_ApplyToPopup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	state := NewState()
	state.SetSwitch("commit", "all", true)
	state.SetSwitch("commit", "verbose", true)
	state.SetOption("commit", "author", "Test Author")

	tokens := testTokens()
	p := New("Commit", tokens)
	p.AddSwitch("a", "all", "Stage all", false)
	p.AddSwitch("v", "verbose", "Verbose", false)
	p.AddOption("A", "author", "Override author", "")

	// Apply state to popup
	state.ApplyToPopup("commit", &p)

	// Check switches were applied
	if !p.switches[0].Enabled {
		t.Error("expected 'all' switch to be enabled after ApplyToPopup")
	}
	if !p.switches[1].Enabled {
		t.Error("expected 'verbose' switch to be enabled after ApplyToPopup")
	}

	// Check options were applied
	if p.options[0].Value != "Test Author" {
		t.Errorf("expected author='Test Author', got %q", p.options[0].Value)
	}
}

func TestState_SaveFromPopup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	tokens := testTokens()
	p := New("Commit", tokens)
	p.AddSwitch("a", "all", "Stage all", true)
	p.AddSwitch("v", "verbose", "Verbose", false)
	p.AddOption("A", "author", "Override author", "My Author")

	state := NewState()
	state.SaveFromPopup("commit", &p)

	if err := state.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	state2 := NewState()
	if err := state2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !state2.GetSwitch("commit", "all") {
		t.Error("expected 'all' switch to be saved")
	}
	if state2.GetSwitch("commit", "verbose") {
		t.Error("expected 'verbose' switch to not be saved (was false)")
	}
	if v := state2.GetOption("commit", "author"); v != "My Author" {
		t.Errorf("expected author='My Author', got %q", v)
	}
}
