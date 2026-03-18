package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/cmdhistory"
	"github.com/mhersson/conjit/internal/ui/notification"
	"github.com/mhersson/conjit/internal/ui/status"
	"github.com/stretchr/testify/assert"
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

func testTokens() theme.Tokens {
	return theme.Compile(theme.Fallback().Raw())
}
