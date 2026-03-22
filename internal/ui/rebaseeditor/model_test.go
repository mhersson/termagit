package rebaseeditor

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

// --- Test helpers ---

func testTokens() theme.Tokens {
	raw := theme.RawTokens{
		Normal:       "#ffffff",
		Bold:         "#ffffff",
		Dim:          "#888888",
		Comment:      "#666666",
		Hash:         "#aaaaaa",
		GraphOrange:  "#ff8800",
		RebaseDone:   "#555555",
		SubtleText:   "#666666",
		Cursor:       "#ffffff",
		CursorBg:     "#444444",
		Background:   "#1e1e2e",
		GraphBlue:    "#89b4fa",
		GraphGreen:   "#a6e3a1",
		GraphYellow:  "#f9e2af",
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

func testEntries() []git.TodoEntry {
	return []git.TodoEntry{
		{Action: git.TodoPick, AbbrevHash: "abc1234", Subject: "add feature X"},
		{Action: git.TodoSquash, AbbrevHash: "def5678", Subject: "fix typo"},
		{Action: git.TodoReword, AbbrevHash: "ghi9012", Subject: "update README"},
	}
}

func newTestModel(entries []git.TodoEntry) Model {
	tokens := testTokens()
	m := New(nil, tokens)
	m.entries = entries
	m.loading = false
	m.width = 80
	m.height = 24
	return m
}

// --- Tests ---

func TestRebaseEditor_Init_LoadsTodo(t *testing.T) {
	tokens := testTokens()
	m := New(nil, tokens)

	cmd := m.Init()
	require.NotNil(t, cmd, "Init should return a command")

	// Execute the command — it should return todoLoadedMsg
	msg := cmd()
	loaded, ok := msg.(todoLoadedMsg)
	require.True(t, ok, "should return todoLoadedMsg")
	// With nil repo, should get an error
	require.Error(t, loaded.Err)
}

func TestRebaseEditor_SetAction_Pick(t *testing.T) {
	entries := testEntries()
	entries[0].Action = git.TodoSquash // Start as squash
	m := newTestModel(entries)

	// Press 'p' to set action to pick
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = newModel.(Model)

	assert.Equal(t, git.TodoPick, m.entries[0].Action, "action should be pick")
}

func TestRebaseEditor_SetAction_Squash(t *testing.T) {
	m := newTestModel(testEntries())

	// Press 's' to set action to squash
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(Model)

	assert.Equal(t, git.TodoSquash, m.entries[0].Action, "action should be squash")
}

func TestRebaseEditor_SetAction_Drop_CommentsPrefixed(t *testing.T) {
	m := newTestModel(testEntries())

	// Press 'd' to drop
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = newModel.(Model)

	assert.Equal(t, git.TodoDrop, m.entries[0].Action, "action should be drop")

	// Verify rendering includes comment prefix
	view := m.View()
	assert.Contains(t, view, "#", "dropped entry should contain comment prefix")
}

func TestRebaseEditor_CycleAction_RotatesCorrectly(t *testing.T) {
	m := newTestModel(testEntries())

	// Cycle through actions: pick -> reword -> edit -> squash -> fixup
	actions := []struct {
		key    rune
		expect git.TodoAction
	}{
		{'r', git.TodoReword},
		{'e', git.TodoEdit},
		{'s', git.TodoSquash},
		{'f', git.TodoFixup},
		{'p', git.TodoPick},
	}

	for _, tc := range actions {
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tc.key}})
		m = newModel.(Model)
		assert.Equal(t, tc.expect, m.entries[0].Action, "action should be %s after pressing %c", tc.expect, tc.key)
	}
}

func TestRebaseEditor_MoveDown_SwapsEntries(t *testing.T) {
	m := newTestModel(testEntries())
	assert.Equal(t, 0, m.cursor)

	// gj to move down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)

	assert.Equal(t, 1, m.cursor, "cursor should move to 1")
	assert.Equal(t, "def5678", m.entries[0].AbbrevHash, "first entry should now be def5678")
	assert.Equal(t, "abc1234", m.entries[1].AbbrevHash, "second entry should now be abc1234")
}

func TestRebaseEditor_MoveUp_SwapsEntries(t *testing.T) {
	m := newTestModel(testEntries())
	m.cursor = 1 // Start at second entry

	// gk to move up
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)

	assert.Equal(t, 0, m.cursor, "cursor should move to 0")
	assert.Equal(t, "def5678", m.entries[0].AbbrevHash, "first entry should now be def5678")
	assert.Equal(t, "abc1234", m.entries[1].AbbrevHash, "second entry should now be abc1234")
}

func TestRebaseEditor_MoveDown_ClampsAtEnd(t *testing.T) {
	m := newTestModel(testEntries())
	m.cursor = len(m.entries) - 1 // At the last entry

	// gj should not move past the end
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)

	assert.Equal(t, len(m.entries)-1, m.cursor, "cursor should stay at end")
}

func TestRebaseEditor_MoveUp_ClampsAtStart(t *testing.T) {
	m := newTestModel(testEntries())
	m.cursor = 0

	// gk should not move past the start
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)

	assert.Equal(t, 0, m.cursor, "cursor should stay at 0")
}

func TestRebaseEditor_InsertBreak_AddsEntry(t *testing.T) {
	m := newTestModel(testEntries())
	originalLen := len(m.entries)

	// Press 'b' to insert break
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = newModel.(Model)

	assert.Equal(t, originalLen+1, len(m.entries), "should have one more entry")
	assert.Equal(t, git.TodoBreak, m.entries[1].Action, "inserted entry should be break")
	assert.Equal(t, 1, m.cursor, "cursor should be on the new break entry")
}

func TestRebaseEditor_InsertExec_AddsEntry(t *testing.T) {
	m := newTestModel(testEntries())
	originalLen := len(m.entries)

	// Press 'x' to activate exec prompt
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = newModel.(Model)
	assert.True(t, m.execActive, "exec prompt should be active")
	assert.Equal(t, originalLen, len(m.entries), "no entry added yet")

	// Type the command
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = newModel.(Model)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(Model)

	// Press Enter to confirm
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = newModel.(Model)

	assert.False(t, m.execActive, "exec prompt should be closed")
	assert.Equal(t, originalLen+1, len(m.entries), "should have one more entry")
	assert.Equal(t, git.TodoExec, m.entries[1].Action, "inserted entry should be exec")
	assert.Equal(t, "ls", m.entries[1].Subject, "exec command should be 'ls'")
	assert.Equal(t, 1, m.cursor, "cursor should be on the new exec entry")
}

func TestRebaseEditor_Submit_WritesAndContinues(t *testing.T) {
	m := newTestModel(testEntries())

	// Simulate <c-c><c-c>
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.Equal(t, "ctrl+c", m.pendingKey, "first ctrl+c should set pending key")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.True(t, m.done, "should be done after submit")
	assert.False(t, m.aborted, "should not be aborted")
	require.NotNil(t, cmd, "should return submit command")
}

func TestRebaseEditor_Abort_CallsAbort(t *testing.T) {
	m := newTestModel(testEntries())

	// Simulate <c-c><c-k>
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)

	assert.True(t, m.done, "should be done after abort")
	assert.True(t, m.aborted, "should be aborted")
	require.NotNil(t, cmd, "should return abort command")
}

func TestRebaseView_CurrentEntry_HasCursorPrefix(t *testing.T) {
	m := newTestModel(testEntries())
	m.cursor = 0

	view := m.View()
	lines := strings.Split(view, "\n")

	// Find the first entry line (after header bar)
	foundCursor := false
	foundNonCursor := false
	for _, line := range lines {
		if strings.HasPrefix(line, "> ") {
			foundCursor = true
		}
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "def5678") {
			foundNonCursor = true
		}
	}
	assert.True(t, foundCursor, "should find cursor prefix '> '")
	assert.True(t, foundNonCursor, "should find non-cursor prefix '  '")
}

func TestRebaseView_ActionLabels_ColourCoded(t *testing.T) {
	entries := []git.TodoEntry{
		{Action: git.TodoPick, AbbrevHash: "abc1234", Subject: "normal commit"},
		{Action: git.TodoPick, AbbrevHash: "def5678", Subject: "done commit", Done: true},
	}
	m := newTestModel(entries)

	view := m.View()
	// The view should contain the entry content (exact ANSI codes are hard to test,
	// but we verify the text is present)
	assert.Contains(t, view, "abc1234", "should render hash")
	assert.Contains(t, view, "normal commit", "should render subject")
	assert.Contains(t, view, "def5678", "should render done entry hash")
	assert.Contains(t, view, "done commit", "should render done entry subject")
}

func TestRebaseView_HelpBlock_AppendsAtBottom(t *testing.T) {
	m := newTestModel(testEntries())

	view := m.View()

	assert.Contains(t, view, "Commands:", "help block should contain 'Commands:'")
	assert.Contains(t, view, "pick   = use commit", "help should describe pick")
	assert.Contains(t, view, "reword = use commit, but edit the commit message", "help should describe reword")
	assert.Contains(t, view, "These lines can be re-ordered", "help should contain reorder note")
	assert.Contains(t, view, "THAT COMMIT WILL BE LOST", "help should contain warning")
}

func TestRebaseView_HelpBlock_KeysFromConfig(t *testing.T) {
	m := newTestModel(testEntries())

	view := m.View()

	// Help block should show the configured key labels from KeyMap
	keys := m.keys
	assert.Contains(t, view, keys.Pick.Help().Key, "help should show pick key")
	assert.Contains(t, view, keys.Reword.Help().Key, "help should show reword key")
	assert.Contains(t, view, keys.Edit.Help().Key, "help should show edit key")
	assert.Contains(t, view, keys.Squash.Help().Key, "help should show squash key")
	assert.Contains(t, view, keys.Fixup.Help().Key, "help should show fixup key")
	assert.Contains(t, view, keys.Execute.Help().Key, "help should show exec key")
	assert.Contains(t, view, keys.Drop.Help().Key, "help should show drop key")
	assert.Contains(t, view, keys.Submit.Help().Key, "help should show submit key")
	assert.Contains(t, view, keys.Abort.Help().Key, "help should show abort key")
	assert.Contains(t, view, keys.MoveUp.Help().Key, "help should show move up key")
	assert.Contains(t, view, keys.MoveDown.Help().Key, "help should show move down key")
}

// --- Additional edge case tests ---

func TestRebaseEditor_ZZ_Submits(t *testing.T) {
	m := newTestModel(testEntries())

	// ZZ = submit
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = newModel.(Model)
	assert.True(t, m.pendingZ, "first Z should set pendingZ")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = newModel.(Model)
	assert.True(t, m.done, "ZZ should submit")
	assert.False(t, m.aborted)
	require.NotNil(t, cmd)
}

func TestRebaseEditor_ZQ_Aborts(t *testing.T) {
	m := newTestModel(testEntries())

	// ZQ = abort
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	m = newModel.(Model)

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Q'}})
	m = newModel.(Model)
	assert.True(t, m.done, "ZQ should abort")
	assert.True(t, m.aborted)
	require.NotNil(t, cmd)
}

func TestRebaseEditor_Close_Q_Aborts(t *testing.T) {
	m := newTestModel(testEntries())

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = newModel.(Model)

	assert.True(t, m.done, "q should close")
	assert.True(t, m.aborted, "close should abort")
	require.NotNil(t, cmd)
}

func TestRebaseEditor_ActionOnBreakEntry_NoOp(t *testing.T) {
	entries := []git.TodoEntry{
		{Action: git.TodoBreak},
	}
	m := newTestModel(entries)

	// Press 'p' on a break entry — should be a no-op
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = newModel.(Model)

	assert.Equal(t, git.TodoBreak, m.entries[0].Action, "break should not change action")
}

func TestRebaseEditor_ActionOnExecEntry_NoOp(t *testing.T) {
	entries := []git.TodoEntry{
		{Action: git.TodoExec, Subject: "make test"},
	}
	m := newTestModel(entries)

	// Press 's' on an exec entry — should be a no-op
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = newModel.(Model)

	assert.Equal(t, git.TodoExec, m.entries[0].Action, "exec should not change action")
}

func TestRebaseEditor_CursorNavigation(t *testing.T) {
	m := newTestModel(testEntries())
	assert.Equal(t, 0, m.cursor)

	// j moves down
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)

	// j again
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor)

	// j at end — clamped
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = newModel.(Model)
	assert.Equal(t, 2, m.cursor, "should clamp at end")

	// k moves up
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = newModel.(Model)
	assert.Equal(t, 1, m.cursor)
}

func TestRebaseEditor_PendingKeyCancel(t *testing.T) {
	m := newTestModel(testEntries())

	// ctrl+c then some other key — should cancel pending
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.Equal(t, "ctrl+c", m.pendingKey)

	// Press 'x' — not a valid second key, should cancel
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = newModel.(Model)
	assert.Empty(t, m.pendingKey, "pending key should be cleared")
	assert.False(t, m.done, "should not be done")
}

func TestRebaseEditor_EmptyEntries_NoActions(t *testing.T) {
	m := newTestModel(nil)

	// Press action keys — should not panic
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = newModel.(Model)
	assert.Empty(t, m.entries)
}

func TestRebaseEditor_WindowSizeMsg(t *testing.T) {
	m := newTestModel(testEntries())

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newModel.(Model)

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)
}

func TestRebaseEditor_OpenScrollDown_OpensCommitView(t *testing.T) {
	entries := []git.TodoEntry{
		{Action: git.TodoPick, AbbrevHash: "abc1234", Hash: "abc1234full", Subject: "add feature X"},
	}
	m := newTestModel(entries)

	// ] then c
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = newModel.(Model)
	assert.True(t, m.pendingOSD, "should set pendingOSD after ]")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(Model)

	require.NotNil(t, cmd, "]c should return a command")
	msg := cmd()
	cvMsg, ok := msg.(OpenCommitViewMsg)
	assert.True(t, ok, "expected OpenCommitViewMsg, got %T", msg)
	if ok {
		assert.Equal(t, "abc1234full", cvMsg.Hash)
	}
}

func TestRebaseEditor_OpenScrollUp_OpensCommitView(t *testing.T) {
	entries := []git.TodoEntry{
		{Action: git.TodoPick, AbbrevHash: "abc1234", Hash: "abc1234full", Subject: "add feature X"},
	}
	m := newTestModel(entries)

	// [ then c
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = newModel.(Model)
	assert.True(t, m.pendingOSU, "should set pendingOSU after [")

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = newModel.(Model)

	require.NotNil(t, cmd, "[c should return a command")
	msg := cmd()
	cvMsg, ok := msg.(OpenCommitViewMsg)
	assert.True(t, ok, "expected OpenCommitViewMsg, got %T", msg)
	if ok {
		assert.Equal(t, "abc1234full", cvMsg.Hash)
	}
}

func TestRebaseEditor_OpenScrollDown_NoHash_Noop(t *testing.T) {
	entries := []git.TodoEntry{
		{Action: git.TodoBreak},
	}
	m := newTestModel(entries)

	// ] then c on entry without hash
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = newModel.(Model)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	_ = newModel.(Model)

	assert.Nil(t, cmd, "]c on entry without hash should be nil")
}
