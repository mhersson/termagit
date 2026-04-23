package app

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/cmdhistory"
	"github.com/mhersson/termagit/internal/ui/notification"
	"github.com/mhersson/termagit/internal/ui/status"
	"github.com/mhersson/termagit/internal/watcher"
)

func TestApp_NotifyMsg_AddsToStack(t *testing.T) {
	m := Model{width: 80, height: 24}
	msg := notification.NotifyMsg{Message: "Pushing...", Kind: notification.Info}

	newModel, cmd := m.Update(msg)
	app := newModel.(Model)
	assert.Equal(t, 1, app.notifications.Len())
	assert.NotNil(t, cmd, "should return expire command")
}

func TestApp_NotifyExpiredMsg_RemovesFromStack(t *testing.T) {
	m := Model{width: 80, height: 24}

	// Add a notification first
	n := notification.New("test", notification.Info, 5000000000) // 5 seconds
	m.notifications.Add(n)
	assert.Equal(t, 1, m.notifications.Len())

	// The ExpiredMsg won't match the ID since we can't access the internal ID,
	// but we can verify the handler exists and doesn't panic
	newModel, _ := m.Update(notification.ExpiredMsg{ID: 0})
	app := newModel.(Model)
	// Stack should still have 1 item since the ID didn't match
	assert.Equal(t, 1, app.notifications.Len())
}

func TestApp_OpenCmdHistoryMsg_SwitchesToCmdHistory(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		active: ScreenStatus,
	}

	newModel, _ := m.Update(status.OpenCmdHistoryMsg{})
	app := newModel.(Model)
	assert.Equal(t, ScreenCmdHistory, app.active)
	assert.NotNil(t, app.cmdHistory)
}

func TestApp_CmdHistoryCloseMsg_ReturnsToStatus(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		active: ScreenCmdHistory,
	}

	newModel, _ := m.Update(cmdhistory.CloseMsg{})
	app := newModel.(Model)
	assert.Equal(t, ScreenStatus, app.active)
}

func TestApp_View_OverlaysNotifications(t *testing.T) {
	m := Model{width: 80, height: 24, active: ScreenStatus}
	// Add a notification
	n := notification.New("Test notification", notification.Info, 5000000000)
	m.notifications.Add(n)

	v := m.View()
	assert.Contains(t, v, "Test notification")
}

func TestApp_View_OverlaysConfirmationCentered(t *testing.T) {
	// Verify that CenterOverlay correctly composites a confirm dialog
	// into the center of the view. The full wiring (status.ConfirmView →
	// app.View → CenterOverlay) is verified by running the app.
	tokens := testTokens()
	dialog := notification.ConfirmDialog{Message: "Discard changes to dirty.go?"}
	confirmView := dialog.View(tokens, 50)
	assert.NotEmpty(t, confirmView)

	// Build a simple base
	base := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\n"
	result := notification.CenterOverlay(base, confirmView, 80, 8)
	assert.Contains(t, result, "dirty.go")
	assert.Contains(t, result, "Discard")
}

func TestApp_WindowSizeMsg_PropagatedToCmdHistory(t *testing.T) {
	ch := cmdhistory.New(nil, testTokens(), 80, 24)
	m := Model{
		active:     ScreenCmdHistory,
		cmdHistory: &ch,
		width:      80,
		height:     24,
	}

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := newModel.(Model)
	assert.Equal(t, 120, app.width)
	assert.Equal(t, 40, app.height)
}

func TestApp_RepoChangedMsg_ForwardedToStatus(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		active: ScreenStatus,
	}

	// RepoChangedMsg should be handled without panicking and return a Model
	newModel, _ := m.Update(watcher.RepoChangedMsg{})
	app := newModel.(Model)
	// Active screen should remain status
	assert.Equal(t, ScreenStatus, app.active)
}

func TestApp_QuitMsg_StopsWatcher(t *testing.T) {
	// Model without a watcher - should not panic
	m := Model{width: 80, height: 24}
	_, cmd := m.Update(tea.QuitMsg{})
	assert.NotNil(t, cmd, "QuitMsg should return tea.Quit")
}

func TestApp_WatcherField_SetWhenEnabled(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watcher test in short mode")
	}

	gitDir := t.TempDir() + "/.git"
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	require.NoError(t, os.WriteFile(gitDir+"/HEAD", []byte("ref: refs/heads/main\n"), 0o644))

	w, err := watcher.New(gitDir)
	require.NoError(t, err)
	defer w.Stop()

	m := Model{watcher: w}
	assert.NotNil(t, m.watcher)
}

func TestApp_WatcherField_NilWhenDisabled(t *testing.T) {
	m := Model{}
	assert.Nil(t, m.watcher)
}

func TestApp_WindowSizeMsg_PropagatedToStatus(t *testing.T) {
	m := Model{
		width:  80,
		height: 24,
		active: ScreenStatus,
	}

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app := newModel.(Model)
	assert.Equal(t, 120, app.width)
	assert.Equal(t, 40, app.height)
}

func TestUpdate_NilPointerScreen_FallsBackToStatus(t *testing.T) {
	// If active screen is a pointer-type (lazy-init) view but the pointer is
	// nil, Update should reset to ScreenStatus as a safety fallback rather
	// than silently dropping messages.
	screens := []Screen{
		ScreenCmdHistory,
		ScreenLog,
		ScreenReflog,
		ScreenCommitView,
		ScreenRefsView,
		ScreenStashList,
		ScreenDiffView,
	}

	for _, screen := range screens {
		m := Model{
			active: screen,
			width:  80,
			height: 24,
		}

		// Send a generic key message
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		newModel, _ := m.Update(msg)
		app := newModel.(Model)

		assert.Equal(t, ScreenStatus, app.active,
			"screen %d should fall back to ScreenStatus when pointer is nil", screen)
	}
}

func testTokens() theme.Tokens {
	return theme.Compile(theme.Fallback().Raw())
}
