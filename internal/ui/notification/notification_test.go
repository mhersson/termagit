package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.RawTokens{
		NotificationInfo:    "#7aa2f7",
		NotificationSuccess: "#9ece6a",
		NotificationWarn:    "#ff9e64",
		NotificationError:   "#f7768e",
		ConfirmBorder:       "#ff9e64",
		ConfirmText:         "#a9b1d6",
		ConfirmKey:          "#e0af68",
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

// --- ConfirmDialog tests ---

func TestConfirmDialog_View_ContainsMessage(t *testing.T) {
	tokens := testTokens()
	d := ConfirmDialog{Message: "Discard changes to main.go?"}
	v := d.View(tokens, 60)
	assert.Contains(t, v, "Discard changes to main.go?")
}

func TestConfirmDialog_View_ContainsKeys(t *testing.T) {
	tokens := testTokens()
	d := ConfirmDialog{Message: "Discard?"}
	v := d.View(tokens, 60)
	assert.Contains(t, v, "y")
	assert.Contains(t, v, "N")
}

func TestConfirmDialog_View_HasBorder(t *testing.T) {
	tokens := testTokens()
	d := ConfirmDialog{Message: "Discard?"}
	v := d.View(tokens, 60)
	// Rounded border uses "╭" at top-left
	assert.Contains(t, v, "╭")
}

func TestConfirmDialog_View_HasIcon(t *testing.T) {
	tokens := testTokens()
	d := ConfirmDialog{Message: "Discard?"}
	v := d.View(tokens, 60)
	assert.Contains(t, v, "⚠")
}

// --- CenterOverlay tests ---

func TestCenterOverlay_PlacesCentered(t *testing.T) {
	// 10 lines x 40 cols base
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, strings.Repeat(".", 40))
	}
	base := strings.Join(lines, "\n")
	overlay := "Hello"

	result := CenterOverlay(base, overlay, 40, 10)
	resultLines := strings.Split(result, "\n")

	// Overlay should be at vertical middle (line 4 or 5 for 10 lines)
	found := false
	for i, l := range resultLines {
		if strings.Contains(l, "Hello") {
			// Should be roughly centered vertically
			assert.True(t, i >= 3 && i <= 6, "expected overlay near vertical center, got line %d", i)
			found = true
			break
		}
	}
	assert.True(t, found, "overlay text not found in result")
}

func TestCenterOverlay_EmptyOverlay_ReturnsBase(t *testing.T) {
	base := "hello\nworld\n"
	result := CenterOverlay(base, "", 40, 5)
	assert.Equal(t, base, result)
}

func TestCenterOverlay_MultilineOverlay(t *testing.T) {
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, strings.Repeat(".", 40))
	}
	base := strings.Join(lines, "\n")
	overlay := "line1\nline2\nline3"

	result := CenterOverlay(base, overlay, 40, 10)
	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line2")
	assert.Contains(t, result, "line3")
}

func TestNotification_New_SequentialUniqueIDs(t *testing.T) {
	// Bubble Tea is single-threaded, so IDs only need to be unique sequentially.
	const count = 100
	seen := make(map[int64]bool, count)
	for i := 0; i < count; i++ {
		n := New("test", Info, time.Second)
		assert.False(t, seen[n.id], "duplicate notification ID: %d", n.id)
		seen[n.id] = true
	}
	assert.Equal(t, count, len(seen))
}

func TestConfirmDialog_View_FitsLongMessage(t *testing.T) {
	tokens := testTokens()
	longPath := "internal/very/deep/nested/directory/structure/with/long/filename_test.go"
	msg := "Discard changes to " + longPath + "?"
	d := ConfirmDialog{Message: msg}
	// maxWidth large enough that content should NOT wrap
	v := d.View(tokens, 200)

	lines := strings.Split(v, "\n")
	// A non-wrapping bordered box should be exactly 3 lines: top border, content, bottom border
	assert.Equal(t, 3, len(lines), "long message should render on a single content line, got %d lines:\n%s", len(lines), v)
	assert.Contains(t, v, longPath, "full path should be visible")
}

func TestConfirmDialog_View_CapsAtMaxWidth(t *testing.T) {
	tokens := testTokens()
	longPath := "internal/very/deep/nested/directory/structure/with/long/filename_test.go"
	msg := "Discard changes to " + longPath + "?"
	d := ConfirmDialog{Message: msg}
	// maxWidth too small for content — output should be truncated to maxWidth
	v := d.View(tokens, 40)

	lines := strings.Split(v, "\n")
	for _, line := range lines {
		assert.LessOrEqual(t, lipgloss.Width(line), 40, "line should not exceed maxWidth")
	}
	// Full path should NOT be visible since it's truncated
	assert.NotContains(t, v, "filename_test.go", "truncated dialog should not show the full path")
}

func TestNotification_View_FitsLongMessage(t *testing.T) {
	tokens := testTokens()
	longMsg := "Successfully pushed refs/heads/feature/very-long-branch-name-for-testing to origin"
	n := New(longMsg, Success, 3*time.Second)
	v := n.View(tokens, 200)

	lines := strings.Split(v, "\n")
	assert.Equal(t, 3, len(lines), "long notification should render on a single content line, got %d lines:\n%s", len(lines), v)
	assert.Contains(t, v, longMsg)
}

func TestOverlay_PreservesBaseANSICodes(t *testing.T) {
	// Base line with ANSI styling (bold "Hello")
	styled := "\x1b[1mHello\x1b[0m world!!"
	base := styled + "\n" + "second line..."
	overlay := "XX"
	result := Overlay(base, overlay, 14)
	lines := strings.Split(result, "\n")
	// The left portion should still contain the ANSI bold code
	assert.Contains(t, lines[0], "\x1b[1m")
}

func TestCenterOverlay_PreservesBaseANSICodes(t *testing.T) {
	styled := "\x1b[1mHello\x1b[0m world!!"
	base := styled + "\n" + styled + "\n" + styled
	overlay := "XX"
	result := CenterOverlay(base, overlay, 14, 3)
	lines := strings.Split(result, "\n")
	// The overlaid line should still contain ANSI codes in the preserved portions
	assert.Contains(t, lines[1], "\x1b[1m")
}

// --- InputDialog tests ---

func TestInputDialog_View_ContainsMessage(t *testing.T) {
	tokens := testTokens()
	d := InputDialog{Message: "Create and checkout branch: my-branch"}
	v := d.View(tokens, 60)
	assert.Contains(t, v, "Create and checkout branch: my-branch")
}

func TestInputDialog_View_DoesNotContainYN(t *testing.T) {
	tokens := testTokens()
	d := InputDialog{Message: "Create branch: "}
	v := d.View(tokens, 60)
	// Must NOT have the y/N confirmation hint
	assert.NotContains(t, v, "y/")
	assert.NotContains(t, v, "/N")
}

func TestInputDialog_View_HasBorder(t *testing.T) {
	tokens := testTokens()
	d := InputDialog{Message: "Create branch: "}
	v := d.View(tokens, 60)
	assert.Contains(t, v, "╭")
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
