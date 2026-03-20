package commitview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.Fallback().Raw())
}

func TestNew_CreatesModel(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)

	assert.Equal(t, "abc123", m.CommitID())
	assert.True(t, m.loading, "should be loading initially")
	assert.False(t, m.ready, "should not be ready yet")
}

func TestNew_WithFilter(t *testing.T) {
	filter := []string{"path/to/file.go"}
	m := New(nil, "abc123", testTokens(), filter)

	assert.Equal(t, filter, m.filter)
}

func TestModel_SetSize(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
}

func TestModel_Init_ReturnsLoadCommand(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	cmd := m.Init()

	// Init should return a command (the loadCommitDataCmd)
	require.NotNil(t, cmd, "Init should return a command")
}

func TestModel_UpdateCommit_ChangesSingletonCommit(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = false
	m.ready = true

	cmd := m.UpdateCommit("def456", nil)

	assert.Equal(t, "def456", m.CommitID())
	assert.True(t, m.loading, "should start loading new commit")
	require.NotNil(t, cmd, "should return load command")
}

func TestModel_UpdateCommit_SameHash_NoOp(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.loading = false

	cmd := m.UpdateCommit("abc123", nil)

	assert.Nil(t, cmd, "same hash should be no-op")
}

func TestKeyMap_DefaultBindings(t *testing.T) {
	keys := DefaultKeyMap()

	// Verify essential keys are bound
	assert.NotEmpty(t, keys.Close.Keys())
	assert.NotEmpty(t, keys.MoveDown.Keys())
	assert.NotEmpty(t, keys.MoveUp.Keys())
}

func TestCommitDataLoadedMsg_SetsData(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:          "abc123def456",
		Subject:       "Test commit",
		AuthorName:    "Test Author",
		AuthorEmail:   "test@example.com",
		AuthorDate:    "2024-01-01T12:00:00Z",
		CommitterName: "Test Committer",
	}

	overview := &git.CommitOverview{
		Summary: "1 file changed, 10 insertions(+)",
		Files:   []git.CommitOverviewFile{{Path: "test.go", Changes: "10"}},
	}

	msg := CommitDataLoadedMsg{
		Info:     info,
		Overview: overview,
	}

	newM, _ := m.Update(msg)
	model := newM.(Model)

	assert.False(t, model.loading)
	assert.True(t, model.ready)
	assert.Equal(t, info, model.info)
	assert.Equal(t, overview, model.overview)
}

func TestCloseCommitViewMsg_OnQKey(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = false
	m.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(keyMsg)

	// Should emit close message
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseCommitViewMsg)
	assert.True(t, ok, "should emit CloseCommitViewMsg")
}

func TestView_ShowsLoadingWhenLoading(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = true

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestView_RendersCommitHeader(t *testing.T) {
	m := New(nil, "abc123def456789", testTokens(), nil)
	m.SetSize(80, 24)

	// Use proper Update flow to set data and viewport content
	info := &git.LogEntry{
		Hash:        "abc123def456789",
		Subject:     "Test commit subject",
		AuthorName:  "Test Author",
		AuthorEmail: "test@example.com",
		AuthorDate:  "2024-01-01T12:00:00Z",
	}

	msg := CommitDataLoadedMsg{
		Info:     info,
		Overview: &git.CommitOverview{},
	}

	newM, _ := m.Update(msg)
	model := newM.(Model)

	view := model.View()
	// Should contain "Commit <hash>"
	assert.Contains(t, view, "Commit")
	assert.Contains(t, view, "abc123def456789")
}

func TestModel_Done_InitiallyFalse(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	assert.False(t, m.Done())
}

func TestModel_Done_TrueAfterClose(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = false
	m.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.True(t, model.Done())
}

func TestModel_SetOverlayMode(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	assert.False(t, m.overlayMode)

	m.SetOverlayMode(true)
	assert.True(t, m.overlayMode)
}

func TestModel_CursorMovement_Down(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	// Set up data with multiple lines
	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test commit",
		AuthorName:  "Author",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}
	overview := &git.CommitOverview{
		Summary: "1 file changed",
		Files:   []git.CommitOverviewFile{{Path: "test.go", Changes: "10"}},
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: overview}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Cursor should start at 0
	assert.Equal(t, 0, model.cursorLine)

	// Move down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)

	assert.Equal(t, 1, model.cursorLine)
}

func TestModel_CursorMovement_Up(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test commit",
		AuthorName:  "Author",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Move down first
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)
	assert.Equal(t, 1, model.cursorLine)

	// Move back up
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)

	assert.Equal(t, 0, model.cursorLine)
}

func TestModel_CursorMovement_StaysInBounds(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test",
		AuthorName:  "A",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Move up at boundary should stay at 0
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)
	assert.Equal(t, 0, model.cursorLine)
}

func TestView_OverlayMode_HasTopBorder(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.SetOverlayMode(true)

	info := &git.LogEntry{
		Hash:        "abc123def456789",
		Subject:     "Test commit",
		AuthorName:  "Test Author",
		AuthorEmail: "test@example.com",
		AuthorDate:  "2024-01-01T12:00:00Z",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	view := model.View()

	// Should start with the border character in overlay mode
	assert.Contains(t, view, "─", "overlay mode should have top border")
}

func TestView_NonOverlayMode_NoTopBorder(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	// overlayMode is false by default

	info := &git.LogEntry{
		Hash:        "abc123def456789",
		Subject:     "Test commit",
		AuthorName:  "Test Author",
		AuthorEmail: "test@example.com",
		AuthorDate:  "2024-01-01T12:00:00Z",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	view := model.View()
	lines := make([]byte, 0)
	for i := 0; i < len(view) && view[i] != '\n'; i++ {
		lines = append(lines, view[i])
	}
	firstLine := string(lines)

	// First line should be "Commit ..." not a border
	assert.Contains(t, firstLine, "Commit", "non-overlay mode should start with Commit header")
}

// Keymap tests for correct bindings (Neogit compatibility)
func TestKeyMap_RevertPopup_UsesV(t *testing.T) {
	keys := DefaultKeyMap()
	assert.Contains(t, keys.RevertPopup.Keys(), "v", "RevertPopup should use 'v' key (Neogit standard)")
}

func TestKeyMap_RebasePopup_UsesR(t *testing.T) {
	keys := DefaultKeyMap()
	assert.Contains(t, keys.RebasePopup.Keys(), "r", "RebasePopup should use 'r' key (Neogit standard)")
}

// YankSelected handler tests
func TestYankSelected_EmitsYankMsg(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:        "abc123def456789",
		Subject:     "Test commit",
		AuthorName:  "Test Author",
		AuthorEmail: "test@example.com",
		AuthorDate:  "2024-01-01T12:00:00Z",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Press Y to yank
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}}
	_, cmd := model.Update(keyMsg)

	require.NotNil(t, cmd, "Y key should return a command")
	result := cmd()
	yankMsg, ok := result.(YankMsg)
	assert.True(t, ok, "should emit YankMsg")
	assert.Equal(t, "abc123def456789", yankMsg.Text, "should yank full commit hash")
}

// Popup trigger tests
func TestPopupTriggers_EmitOpenPopupMsg(t *testing.T) {
	tests := []struct {
		key        string
		popupType  string
		keyBinding func(KeyMap) []string
	}{
		{"A", "cherry-pick", func(k KeyMap) []string { return k.CherryPickPopup.Keys() }},
		{"b", "branch", func(k KeyMap) []string { return k.BranchPopup.Keys() }},
		{"B", "bisect", func(k KeyMap) []string { return k.BisectPopup.Keys() }},
		{"c", "commit", func(k KeyMap) []string { return k.CommitPopup.Keys() }},
		{"d", "diff", func(k KeyMap) []string { return k.DiffPopup.Keys() }},
		{"P", "push", func(k KeyMap) []string { return k.PushPopup.Keys() }},
		{"v", "revert", func(k KeyMap) []string { return k.RevertPopup.Keys() }},
		{"r", "rebase", func(k KeyMap) []string { return k.RebasePopup.Keys() }},
		{"X", "reset", func(k KeyMap) []string { return k.ResetPopup.Keys() }},
		{"t", "tag", func(k KeyMap) []string { return k.TagPopup.Keys() }},
	}

	for _, tc := range tests {
		t.Run(tc.popupType, func(t *testing.T) {
			m := New(nil, "abc123def456", testTokens(), nil)
			m.SetSize(80, 24)

			info := &git.LogEntry{
				Hash:        "abc123def456",
				Subject:     "Test",
				AuthorName:  "A",
				AuthorEmail: "a@b.com",
				AuthorDate:  "2024-01-01",
			}
			msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
			newM, _ := m.Update(msg)
			model := newM.(Model)

			// Press the popup key
			var keyMsg tea.KeyMsg
			if len(tc.key) == 1 {
				keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}
			}
			_, cmd := model.Update(keyMsg)

			require.NotNil(t, cmd, "%s key should return a command", tc.key)
			result := cmd()
			popupMsg, ok := result.(OpenPopupMsg)
			assert.True(t, ok, "should emit OpenPopupMsg for %s", tc.popupType)
			assert.Equal(t, tc.popupType, popupMsg.Type, "popup type should be %s", tc.popupType)
			assert.Equal(t, "abc123def456", popupMsg.Commit, "should include commit hash")
		})
	}
}

// Hunk navigation tests
func TestNextHunkHeader_MovesToNextHunk(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test",
		AuthorName:  "A",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}
	diffs := []git.FileDiff{
		{
			Path: "test.go",
			Hunks: []git.Hunk{
				{Header: "@@ -1,3 +1,4 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "line1"}}},
				{Header: "@@ -10,3 +11,4 @@", Lines: []git.DiffLine{{Op: git.DiffOpAdd, Content: "new line"}}},
			},
		},
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}, Diffs: diffs}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Start at top
	assert.Equal(t, 0, model.cursorLine)

	// Press } to go to next hunk
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'}'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)

	// Should have moved to a hunk header line (exact line depends on content structure)
	assert.Greater(t, model.cursorLine, 0, "should move cursor to hunk header")
}

func TestPrevHunkHeader_MovesToPreviousHunk(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test",
		AuthorName:  "A",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}
	diffs := []git.FileDiff{
		{
			Path: "test.go",
			Hunks: []git.Hunk{
				{Header: "@@ -1,3 +1,4 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "line1"}}},
				{Header: "@@ -10,3 +11,4 @@", Lines: []git.DiffLine{{Op: git.DiffOpAdd, Content: "new line"}}},
			},
		},
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}, Diffs: diffs}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Move to end first
	model.cursorLine = model.totalLines - 1

	// Press { to go to previous hunk
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'{'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)

	// Should have moved to a hunk header line
	assert.Less(t, model.cursorLine, model.totalLines-1, "should move cursor to previous hunk header")
}

// Scroll tests
func TestScrollDown_MovesViewport(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 10) // Small height to enable scrolling

	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test commit with long body",
		Body:        "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10",
		AuthorName:  "A",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	initialOffset := model.viewport.YOffset

	// Press ]c to scroll down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']', 'c'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)

	assert.GreaterOrEqual(t, model.viewport.YOffset, initialOffset, "viewport should scroll down")
}

func TestScrollUp_MovesViewport(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 10)

	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "Test",
		Body:        "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\nLine 6\nLine 7\nLine 8\nLine 9\nLine 10",
		AuthorName:  "A",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}

	msg := CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	// Scroll down first
	model.viewport.YOffset = 5

	// Press [c to scroll up
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'[', 'c'}}
	newM, _ = model.Update(keyMsg)
	model = newM.(Model)

	assert.LessOrEqual(t, model.viewport.YOffset, 5, "viewport should scroll up")
}
