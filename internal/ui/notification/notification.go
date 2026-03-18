package notification

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/conjit/internal/theme"
)

// Kind represents the severity of a notification.
type Kind int

const (
	Info Kind = iota
	Success
	Warning
	Error
)

// String returns the string representation of a Kind.
func (k Kind) String() string {
	switch k {
	case Info:
		return "info"
	case Success:
		return "success"
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// icon returns the icon character for this kind.
func (k Kind) icon() string {
	switch k {
	case Info:
		return "ℹ"
	case Success:
		return "✓"
	case Warning:
		return "⚠"
	case Error:
		return "✗"
	default:
		return "•"
	}
}

// Notification represents an ephemeral message shown to the user.
type Notification struct {
	Message string
	Kind    Kind
	Expiry  time.Time
	id      int64
}

// ExpiredMsg is sent when a notification's lifetime has elapsed.
type ExpiredMsg struct {
	ID int64
}

// NotifyMsg is sent from any view to the app to show a notification.
type NotifyMsg struct {
	Message string
	Kind    Kind
}

var nextID int64

// New creates a notification that expires after the given duration.
func New(msg string, kind Kind, d time.Duration) Notification {
	nextID++
	return Notification{
		Message: msg,
		Kind:    kind,
		Expiry:  time.Now().Add(d),
		id:      nextID,
	}
}

// DefaultDuration returns the auto-dismiss duration for a notification kind.
func DefaultDuration(kind Kind) time.Duration {
	switch kind {
	case Info, Success:
		return 3 * time.Second
	case Warning:
		return 4 * time.Second
	case Error:
		return 5 * time.Second
	default:
		return 3 * time.Second
	}
}

// Expired returns true if the notification has passed its expiry time.
func (n Notification) Expired() bool {
	return time.Now().After(n.Expiry)
}

// ExpireCmd returns a tea.Cmd that sends ExpiredMsg after the remaining lifetime.
func (n Notification) ExpireCmd() tea.Cmd {
	remaining := time.Until(n.Expiry)
	if remaining <= 0 {
		return func() tea.Msg { return ExpiredMsg{ID: n.id} }
	}
	id := n.id
	return tea.Tick(remaining, func(time.Time) tea.Msg {
		return ExpiredMsg{ID: id}
	})
}

// borderStyle returns the lipgloss border style colored by notification kind.
func (n Notification) borderStyle(tokens theme.Tokens) lipgloss.Style {
	var color lipgloss.TerminalColor
	switch n.Kind {
	case Info:
		color = tokens.NotificationInfo.GetForeground()
	case Success:
		color = tokens.NotificationSuccess.GetForeground()
	case Warning:
		color = tokens.NotificationWarn.GetForeground()
	case Error:
		color = tokens.NotificationError.GetForeground()
	default:
		color = tokens.NotificationInfo.GetForeground()
	}

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(0, 1)
}

// View renders the notification as a bordered box.
func (n Notification) View(tokens theme.Tokens, maxWidth int) string {
	icon := n.Kind.icon()

	var textStyle lipgloss.Style
	switch n.Kind {
	case Info:
		textStyle = tokens.NotificationInfo
	case Success:
		textStyle = tokens.NotificationSuccess
	case Warning:
		textStyle = tokens.NotificationWarn
	case Error:
		textStyle = tokens.NotificationError
	}

	content := textStyle.Render(icon + " " + n.Message)
	box := n.borderStyle(tokens)

	// Limit width
	innerMax := maxWidth - 4 // border + padding
	if innerMax > 0 {
		box = box.MaxWidth(maxWidth)
		_ = innerMax
	}

	return box.Render(content)
}

// Stack holds a list of active notifications.
type Stack struct {
	items []Notification
}

// Add appends a notification to the stack.
func (s *Stack) Add(n Notification) {
	s.items = append(s.items, n)
}

// RemoveExpired removes all expired notifications from the stack.
func (s *Stack) RemoveExpired() {
	alive := s.items[:0]
	for _, n := range s.items {
		if !n.Expired() {
			alive = append(alive, n)
		}
	}
	s.items = alive
}

// RemoveByID removes the notification with the given ID.
func (s *Stack) RemoveByID(id int64) {
	for i, n := range s.items {
		if n.id == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return
		}
	}
}

// Len returns the number of active notifications.
func (s *Stack) Len() int {
	return len(s.items)
}

// View renders all notifications stacked vertically.
func (s *Stack) View(tokens theme.Tokens, maxWidth int) string {
	if len(s.items) == 0 {
		return ""
	}
	var parts []string
	for _, n := range s.items {
		parts = append(parts, n.View(tokens, maxWidth))
	}
	return strings.Join(parts, "\n")
}

// ExpireCmds returns tea.Cmds for all active notifications.
func (s *Stack) ExpireCmds() []tea.Cmd {
	var cmds []tea.Cmd
	for _, n := range s.items {
		cmds = append(cmds, n.ExpireCmd())
	}
	return cmds
}

// Overlay composites a notification block onto the upper-right corner
// of the base screen output.
func Overlay(base, overlay string, width int) string {
	if overlay == "" {
		return base
	}

	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, ol := range overlayLines {
		if i >= len(baseLines) {
			break
		}

		olPlain := stripAnsi(ol)
		olLen := len([]rune(olPlain))
		basePlain := stripAnsi(baseLines[i])
		baseLen := len([]rune(basePlain))

		startCol := width - olLen
		if startCol < 0 {
			startCol = 0
		}

		if baseLen < startCol {
			// Pad the base line to reach startCol
			padding := strings.Repeat(" ", startCol-baseLen)
			baseLines[i] = baseLines[i] + padding + ol
		} else {
			// Overwrite the right portion of the base line
			runes := []rune(basePlain)
			left := string(runes[:startCol])
			baseLines[i] = left + ol
		}
	}

	return strings.Join(baseLines, "\n")
}

// CenterOverlay composites an overlay block onto the center of the base screen.
func CenterOverlay(base, overlay string, width, height int) string {
	if overlay == "" {
		return base
	}

	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	// Vertical centering
	startRow := (height - len(overlayLines)) / 2
	if startRow < 0 {
		startRow = 0
	}

	for i, ol := range overlayLines {
		row := startRow + i
		if row >= len(baseLines) {
			break
		}

		olPlain := stripAnsi(ol)
		olLen := len([]rune(olPlain))

		// Horizontal centering
		startCol := (width - olLen) / 2
		if startCol < 0 {
			startCol = 0
		}

		basePlain := stripAnsi(baseLines[row])
		baseLen := len([]rune(basePlain))

		if baseLen < startCol {
			padding := strings.Repeat(" ", startCol-baseLen)
			baseLines[row] = baseLines[row] + padding + ol
		} else {
			runes := []rune(basePlain)
			left := string(runes[:startCol])
			endCol := startCol + olLen
			var right string
			if endCol < baseLen {
				right = string(runes[endCol:])
			}
			baseLines[row] = left + ol + right
		}
	}

	return strings.Join(baseLines, "\n")
}

// ConfirmDialog represents a confirmation prompt shown as a centered overlay.
type ConfirmDialog struct {
	Message string
}

// View renders the confirmation dialog as a bordered box with icon and key hints.
func (d ConfirmDialog) View(tokens theme.Tokens, maxWidth int) string {
	icon := "⚠"
	content := tokens.ConfirmText.Render(icon+" "+d.Message) +
		"  " +
		tokens.ConfirmKey.Render("y") +
		tokens.ConfirmText.Render("/") +
		tokens.ConfirmKey.Render("N")

	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(tokens.ConfirmBorder.GetForeground()).
		Padding(0, 1)

	innerMax := maxWidth - 4
	if innerMax > 0 {
		box = box.MaxWidth(maxWidth)
		_ = innerMax
	}

	return box.Render(content)
}

// stripAnsi removes ANSI escape sequences for length calculation.
func stripAnsi(s string) string {
	var out strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
