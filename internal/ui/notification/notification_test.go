package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.RawTokens{
		NotificationInfo:    "#7aa2f7",
		NotificationSuccess: "#9ece6a",
		NotificationWarn:    "#ff9e64",
		NotificationError:   "#f7768e",
	})
}

func TestNotification_New_SetsExpiry(t *testing.T) {
	before := time.Now()
	n := New("test", Info, 3*time.Second)
	after := time.Now()

	assert.Equal(t, "test", n.Message)
	assert.Equal(t, Info, n.Kind)
	assert.True(t, n.Expiry.After(before.Add(2*time.Second)))
	assert.True(t, n.Expiry.Before(after.Add(4*time.Second)))
}

func TestNotification_Expired_FalseBeforeExpiry(t *testing.T) {
	n := New("test", Info, 5*time.Second)
	assert.False(t, n.Expired())
}

func TestNotification_Expired_TrueAfterExpiry(t *testing.T) {
	n := Notification{
		Message: "old",
		Kind:    Info,
		Expiry:  time.Now().Add(-1 * time.Second),
	}
	assert.True(t, n.Expired())
}

func TestNotification_DefaultDuration_ByKind(t *testing.T) {
	tests := []struct {
		kind     Kind
		expected time.Duration
	}{
		{Info, 3 * time.Second},
		{Success, 3 * time.Second},
		{Warning, 4 * time.Second},
		{Error, 5 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, DefaultDuration(tt.kind))
		})
	}
}

func TestNotification_View_ContainsMessage(t *testing.T) {
	tokens := testTokens()
	n := New("Push complete", Success, 3*time.Second)
	v := n.View(tokens, 60)
	assert.Contains(t, v, "Push complete")
}

func TestNotification_View_Info_HasIcon(t *testing.T) {
	tokens := testTokens()
	n := New("Fetching...", Info, 3*time.Second)
	v := n.View(tokens, 60)
	assert.Contains(t, v, "ℹ")
}

func TestNotification_View_Success_HasIcon(t *testing.T) {
	tokens := testTokens()
	n := New("Pushed", Success, 3*time.Second)
	v := n.View(tokens, 60)
	assert.Contains(t, v, "✓")
}

func TestNotification_View_Warning_HasIcon(t *testing.T) {
	tokens := testTokens()
	n := New("Diverged", Warning, 4*time.Second)
	v := n.View(tokens, 60)
	assert.Contains(t, v, "⚠")
}

func TestNotification_View_Error_HasIcon(t *testing.T) {
	tokens := testTokens()
	n := New("Failed", Error, 5*time.Second)
	v := n.View(tokens, 60)
	assert.Contains(t, v, "✗")
}

func TestStack_Add_And_View(t *testing.T) {
	tokens := testTokens()
	var s Stack
	s.Add(New("first", Info, 3*time.Second))
	s.Add(New("second", Error, 5*time.Second))

	v := s.View(tokens, 60)
	assert.Contains(t, v, "first")
	assert.Contains(t, v, "second")
}

func TestStack_Remove_Expired(t *testing.T) {
	var s Stack
	s.Add(Notification{
		Message: "expired",
		Kind:    Info,
		Expiry:  time.Now().Add(-1 * time.Second),
	})
	s.Add(New("active", Error, 5*time.Second))

	s.RemoveExpired()
	assert.Equal(t, 1, s.Len())
	assert.Equal(t, "active", s.items[0].Message)
}

func TestStack_Empty_ReturnsEmptyView(t *testing.T) {
	tokens := testTokens()
	var s Stack
	v := s.View(tokens, 60)
	assert.Equal(t, "", v)
}

func TestOverlay_PositionsTopRight(t *testing.T) {
	// Create a 5x10 base screen
	base := strings.Repeat(".........."+"\n", 5)
	overlay := "Hi"
	result := Overlay(base, overlay, 10)
	lines := strings.Split(result, "\n")
	require.NotEmpty(t, lines)
	// The overlay "Hi" should appear in the first line, towards the right
	assert.Contains(t, lines[0], "Hi")
}

func TestOverlay_EmptyOverlay_ReturnsBase(t *testing.T) {
	base := "hello\nworld\n"
	result := Overlay(base, "", 40)
	assert.Equal(t, base, result)
}

func TestNotification_Kind_String(t *testing.T) {
	assert.Equal(t, "info", Info.String())
	assert.Equal(t, "success", Success.String())
	assert.Equal(t, "warning", Warning.String())
	assert.Equal(t, "error", Error.String())
}

func TestNotification_borderColor(t *testing.T) {
	tokens := testTokens()

	n := New("test", Info, 3*time.Second)
	style := n.borderStyle(tokens)
	assert.NotEqual(t, lipgloss.Style{}, style)

	n2 := New("test", Error, 5*time.Second)
	style2 := n2.borderStyle(tokens)
	assert.NotEqual(t, lipgloss.Style{}, style2)
}
