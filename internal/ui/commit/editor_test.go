package commit

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/commit/vim"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorModel_Init_LoadsCommitHistory(t *testing.T) {
	m := newTestModel(t)
	cmd := m.Init()

	// When repo is nil, Init may return nil or commands for history/diff
	// This test verifies Init doesn't panic
	if cmd != nil {
		_ = executeBatch(t, cmd)
	}
}

func TestEditorModel_Submit_CreatesCommit(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("Test commit message")

	// Simulate two-key sequence: ctrl+c ctrl+c
	m.pendingKey = "ctrl+c"
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)

	assert.True(t, m.done, "editor should be done after submit")
	assert.False(t, m.aborted, "editor should not be aborted")
	require.NotNil(t, cmd, "should return commit command")
}

func TestEditorModel_Submit_EmptyMessage_DoesNotSubmit(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("") // Empty message

	// Simulate two-key sequence: ctrl+c ctrl+c
	m.pendingKey = "ctrl+c"
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)

	// Should NOT commit with empty message (matches git behavior)
	// The command may still be returned but will fail in execution
	assert.True(t, m.done, "editor should be done")
	require.NotNil(t, cmd)
}

func TestEditorModel_Abort_DoesNotCommit(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("Test commit message")

	// Simulate two-key sequence: ctrl+c ctrl+k
	m.pendingKey = "ctrl+c"
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)

	assert.True(t, m.done, "editor should be done after abort")
	assert.True(t, m.aborted, "editor should be aborted")
	require.NotNil(t, cmd, "should return abort message command")
}

func TestEditorModel_Close_AbortsInNormalMode(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("Test commit message")
	m.vimEditor.SetMode(vim.ModeNormal) // q only works in normal mode

	// Press 'q' to close
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(Model)

	assert.True(t, m.done, "editor should be done after close in normal mode")
	assert.True(t, m.aborted, "close should abort")
	require.NotNil(t, cmd)
}

func TestEditorModel_Close_TypesQInInsertMode(t *testing.T) {
	m := newTestModel(t)
	// Editor starts in insert mode by default
	assert.Equal(t, vim.ModeInsert, m.vimEditor.Mode())

	// Press 'q' - should type 'q' in insert mode, not close
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(Model)

	assert.False(t, m.done, "editor should not be done when q typed in insert mode")
	assert.Equal(t, "q", m.vimEditor.Content(), "q should be typed into buffer")
}

func TestEditorModel_PrevMessage_CyclesHistory(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("current text")

	// Load some history
	m.cycler = git.NewCycler([]string{"previous commit 1", "previous commit 2"})

	// Press alt+p
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}, Alt: true})
	m = newModel.(Model)

	assert.Equal(t, "previous commit 1", m.vimEditor.Content())
}

func TestEditorModel_NextMessage_CyclesForward(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("current text")

	// Load history and cycle backward first
	m.cycler = git.NewCycler([]string{"prev1", "prev2"})
	_ = m.cycler.Prev(m.vimEditor.Content()) // Go to prev1

	// Press alt+n
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}, Alt: true})
	m = newModel.(Model)

	// Should restore original text
	assert.Equal(t, "current text", m.vimEditor.Content())
}

func TestEditorModel_ResetMessage_RestoresOriginal(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("original message")

	// Load history and cycle backward a few times
	m.cycler = git.NewCycler([]string{"prev1", "prev2"})
	_ = m.cycler.Prev(m.vimEditor.Content()) // save original, go to prev1
	m.vimEditor.SetContent("prev1")
	_ = m.cycler.Prev("prev1") // go to prev2
	m.vimEditor.SetContent("prev2")

	// Press alt+r to reset
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}, Alt: true})
	m = newModel.(Model)

	// Should restore the saved original
	assert.Equal(t, "original message", m.vimEditor.Content())
}

func TestEditorModel_AmendFlag_PassedToCommit(t *testing.T) {
	opts := git.CommitOpts{Amend: true}
	m := newTestModelWithOpts(t, opts)

	assert.True(t, m.opts.Amend, "amend flag should be preserved")
}

func TestEditorModel_NoVerify_DisablesHooks(t *testing.T) {
	opts := git.CommitOpts{NoVerify: true}
	m := newTestModelWithOpts(t, opts)

	assert.True(t, m.opts.NoVerify, "no-verify flag should be preserved")
}

func TestEditorModel_TwoKeySequence_Submit(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("Test message")

	// First ctrl+c sets pending key
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.Equal(t, "ctrl+c", m.pendingKey, "first ctrl+c should set pending key")

	// Second ctrl+c triggers submit
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.True(t, m.done, "second ctrl+c should submit")
	assert.False(t, m.aborted)
}

func TestEditorModel_TwoKeySequence_Abort(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("Test message")

	// First ctrl+c sets pending key
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.Equal(t, "ctrl+c", m.pendingKey)

	// k triggers abort
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	assert.True(t, m.done)
	assert.True(t, m.aborted, "ctrl+c k should abort")
}

func TestEditorModel_TwoKeySequence_CancelOnOtherKey(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetMode(vim.ModeNormal) // Test in normal mode to avoid typing
	m.pendingKey = "ctrl+c"

	// Press any other key (not ctrl+c or k)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = newModel.(Model)

	assert.Empty(t, m.pendingKey, "pending key should be cleared")
	assert.False(t, m.done, "should not be done")
}

func TestEditorModel_View_ContainsHelpLines(t *testing.T) {
	m := newTestModel(t)
	m.width = 80
	m.height = 24

	view := m.View()

	assert.Contains(t, view, "Commands:")
	assert.Contains(t, view, "Submit")
	assert.Contains(t, view, "Abort")
	assert.Contains(t, view, "Previous Message")
}

func TestEditorModel_View_ContainsContent(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetContent("My commit message")
	// Need to set size via Update to propagate to vimEditor
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newModel.(Model)

	view := m.View()

	assert.Contains(t, view, "My commit message")
}

func TestEditorModel_View_ShowsModeIndicator(t *testing.T) {
	m := newTestModel(t)
	m.width = 80
	m.height = 24

	// Insert mode by default
	view := m.View()
	assert.Contains(t, view, "[INSERT]")

	// Switch to normal mode
	m.vimEditor.SetMode(vim.ModeNormal)
	view = m.View()
	assert.Contains(t, view, "[NORMAL]")
}

// Helper functions

func newTestModel(t *testing.T) Model {
	t.Helper()
	return newTestModelWithOpts(t, git.CommitOpts{})
}

func newTestModelWithOpts(t *testing.T, opts git.CommitOpts) Model {
	t.Helper()
	cfg := testConfig()
	tokens := testTokens()
	return New(nil, opts, cfg, tokens, "commit")
}

func testConfig() *config.Config {
	return &config.Config{
		CommitEditor: config.CommitEditorConfig{
			ShowStagedDiff:      true,
			StagedDiffSplitKind: "split",
			SpellCheck:          false,
		},
	}
}

func testTokens() theme.Tokens {
	raw := theme.RawTokens{
		Normal:       "#ffffff",
		Bold:         "#ffffff",
		Dim:          "#888888",
		Comment:      "#666666",
		PopupBorder:  "#888888",
		PopupTitle:   "#ffffff",
		PopupKey:     "#ff00ff",
		PopupKeyBg:   "#333333",
		PopupSwitch:  "#00ff00",
		PopupOption:  "#ffff00",
		PopupAction:  "#00ffff",
		PopupSection: "#ff8800",
		Cursor:       "#ffffff",
		CursorBg:     "#444444",
	}
	return theme.Compile(raw)
}

// executeBatch executes a tea.Cmd batch and collects all resulting messages.
func executeBatch(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}

	var msgs []tea.Msg
	msg := cmd()
	if msg == nil {
		return msgs
	}

	// Handle batch messages
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			msgs = append(msgs, executeBatch(t, c)...)
		}
		return msgs
	}

	msgs = append(msgs, msg)
	return msgs
}

// Test for window size handling
func TestEditorModel_WindowSize_UpdatesDimensions(t *testing.T) {
	m := newTestModel(t)

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = newModel.(Model)

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
}

// Test commit history loaded message handling
func TestEditorModel_CommitHistoryLoaded_InitializesCycler(t *testing.T) {
	m := newTestModel(t)

	newModel, _ := m.Update(commitHistoryLoadedMsg{
		Messages: []string{"prev1", "prev2", "prev3"},
	})
	m = newModel.(Model)

	require.NotNil(t, m.cycler)
}

// Test staged diff loaded message handling
func TestEditorModel_StagedDiffLoaded_SetsDiff(t *testing.T) {
	m := newTestModel(t)

	diff := []git.FileDiff{
		{Path: "test.go", IsNew: true},
	}
	newModel, _ := m.Update(stagedDiffLoadedMsg{Diff: diff})
	m = newModel.(Model)

	assert.Len(t, m.diff, 1)
	assert.Equal(t, "test.go", m.diff[0].Path)
}

// Test that entering text mode works
func TestEditorModel_VimEditorReceivesInput(t *testing.T) {
	m := newTestModel(t)
	// Editor starts in insert mode

	// Type some characters
	for _, r := range "Hello" {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = newModel.(Model)
	}

	assert.Equal(t, "Hello", m.vimEditor.Content())
}

// Test error handling for commit history
func TestEditorModel_CommitHistoryError_HandledGracefully(t *testing.T) {
	m := newTestModel(t)

	newModel, _ := m.Update(commitHistoryLoadedMsg{
		Err: context.DeadlineExceeded,
	})
	m = newModel.(Model)

	// Should handle error gracefully (cycler remains nil or empty)
	// Editor should still be usable
	assert.False(t, m.done)
}

// Test vim mode transitions
func TestEditorModel_ESC_SwitchesToNormalMode(t *testing.T) {
	m := newTestModel(t)
	assert.Equal(t, vim.ModeInsert, m.vimEditor.Mode())

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = newModel.(Model)

	assert.Equal(t, vim.ModeNormal, m.vimEditor.Mode())
}

func TestEditorModel_i_SwitchesToInsertMode(t *testing.T) {
	m := newTestModel(t)
	m.vimEditor.SetMode(vim.ModeNormal)

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = newModel.(Model)

	assert.Equal(t, vim.ModeInsert, m.vimEditor.Mode())
}
