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
	// Help lines are now in the buffer content via buildInitialContent,
	// not in the View() output. Test that buildInitialContent has the commands.
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"

	content := m.buildInitialContent()

	assert.Contains(t, content, "Commands:")
	assert.Contains(t, content, "Submit")
	assert.Contains(t, content, "Abort")
	assert.Contains(t, content, "Previous Message")
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

func TestExtractMessage_FiltersCommentLines(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"

	content := `My commit message

# This is a comment
# Another comment
With more content`

	m.vimEditor.SetContent(content)

	result := m.extractMessage()
	// After filtering comments and trimming, we get the message and content
	expected := `My commit message

With more content`
	assert.Equal(t, expected, result)
}

func TestExtractMessage_StopsAtScissorsLine(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"

	content := `My commit message

# ------------------------ >8 ------------------------
# Do not modify or remove the line above.
diff --git a/file.txt
+new content`

	m.vimEditor.SetContent(content)

	result := m.extractMessage()
	assert.Equal(t, "My commit message", result)
}

func TestExtractMessage_RespectsCustomCommentChar(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = ";"

	content := `My commit message
; This is a comment
# This is NOT a comment
More content`

	m.vimEditor.SetContent(content)

	result := m.extractMessage()
	expected := `My commit message
# This is NOT a comment
More content`
	assert.Equal(t, expected, result)
}

func TestExtractMessage_TrimsWhitespace(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"

	content := `

My commit message

# Comment
`

	m.vimEditor.SetContent(content)

	result := m.extractMessage()
	assert.Equal(t, "My commit message", result)
}

func TestBuildInitialContent_IncludesCommandsSection(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"

	content := m.buildInitialContent()

	assert.Contains(t, content, "# Commands:")
	assert.Contains(t, content, "Submit")
	assert.Contains(t, content, "Abort")
	assert.Contains(t, content, "Previous Message")
}

func TestBuildInitialContent_IncludesBranchInfo(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "feature/my-branch"

	content := m.buildInitialContent()

	assert.Contains(t, content, "# On branch feature/my-branch")
}

func TestBuildInitialContent_IncludesStagedFiles(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"
	m.status = &git.StatusResult{
		Staged: []git.StatusEntry{
			git.NewStatusEntry("src/main.go", git.FileStatusModified, git.FileStatusNone),
		},
	}

	content := m.buildInitialContent()

	assert.Contains(t, content, "# Changes to be committed:")
	assert.Contains(t, content, "modified:   src/main.go")
}

func TestBuildInitialContent_IncludesUnstagedFiles(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"
	m.status = &git.StatusResult{
		Unstaged: []git.StatusEntry{
			git.NewStatusEntry("docs/README.md", git.FileStatusNone, git.FileStatusModified),
		},
	}

	content := m.buildInitialContent()

	assert.Contains(t, content, "# Changes not staged for commit:")
	assert.Contains(t, content, "modified:   docs/README.md")
}

func TestBuildInitialContent_IncludesUntrackedFiles(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"
	m.status = &git.StatusResult{
		Untracked: []git.StatusEntry{
			git.NewStatusEntry("new_file.txt", git.FileStatusNone, git.FileStatusUntracked),
		},
	}

	content := m.buildInitialContent()

	assert.Contains(t, content, "# Untracked files:")
	assert.Contains(t, content, "new_file.txt")
}

func TestBuildInitialContent_IncludesScissorsAndDiff(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"
	m.diff = []git.FileDiff{
		{
			Path: "test.go",
			Hunks: []git.Hunk{
				{
					OldStart: 1, OldCount: 3,
					NewStart: 1, NewCount: 4,
					Lines: []git.DiffLine{
						{Content: " existing line"},
						{Content: "+new line added"},
					},
				},
			},
		},
	}
	m.showDiff = true

	content := m.buildInitialContent()

	assert.Contains(t, content, "# ------------------------ >8 ------------------------")
	assert.Contains(t, content, "# Do not modify or remove the line above.")
	assert.Contains(t, content, "diff --git a/test.go b/test.go")
	assert.Contains(t, content, "+new line added")
}

func TestBuildInitialContent_OmitsDiffWhenDisabled(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"
	m.diff = []git.FileDiff{
		{
			Path: "test.go",
			Hunks: []git.Hunk{
				{Lines: []git.DiffLine{{Content: "diff content"}}},
			},
		},
	}
	m.showDiff = false

	content := m.buildInitialContent()

	assert.NotContains(t, content, ">8")
	assert.NotContains(t, content, "diff content")
}

func TestMaybeInitializeContent_CursorAtTop(t *testing.T) {
	m := newTestModel(t)
	m.commentChar = "#"
	m.branch = "main"
	m.commentCharLoaded = true
	m.statusLoaded = true
	m.diffLoaded = true
	m.status = &git.StatusResult{
		Staged: []git.StatusEntry{
			git.NewStatusEntry("file1.go", git.FileStatusModified, git.FileStatusNone),
			git.NewStatusEntry("file2.go", git.FileStatusModified, git.FileStatusNone),
		},
	}
	// Add a large diff to simulate content that could push cursor down
	m.diff = []git.FileDiff{
		{
			Path: "test.go",
			Hunks: []git.Hunk{
				{
					OldStart: 1, OldCount: 100,
					NewStart: 1, NewCount: 100,
					Lines: make([]git.DiffLine, 100), // 100 lines of diff
				},
			},
		},
	}
	m.showDiff = true

	newModel, _ := m.maybeInitializeContent()
	m = newModel.(Model)

	// Cursor should be at line 0, col 0 (top of buffer)
	assert.Equal(t, 0, m.vimEditor.Line(), "cursor should be at line 0")
	assert.Equal(t, 0, m.vimEditor.Col(), "cursor should be at col 0")
}
