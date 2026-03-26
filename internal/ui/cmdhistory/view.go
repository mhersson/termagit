package cmdhistory

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// commandMask contains flags stripped from displayed commands (matching Neogit).
var commandMask = []string{
	" --no-pager",
	" --literal-pathspecs",
	" --no-optional-locks",
	" -c core.preloadindex=true",
	" -c color.ui=always",
	" -c diff.noprefix=false",
}

// View renders the command history view.
func (m Model) View() string {
	if m.cursor.Width == 0 || m.cursor.Height == 0 {
		return ""
	}

	if len(m.entries) == 0 {
		return "No commands recorded"
	}

	var b strings.Builder

	// Title
	b.WriteString(m.tokens.SectionHeader.Render("Git Command History"))
	b.WriteString("\n\n")

	maxLines := m.cursor.VisibleLines()
	linesUsed := 0

	for i := m.cursor.Offset; i < len(m.entries) && linesUsed < maxLines; i++ {
		entry := m.entries[i]
		isErr := entry.ExitCode != 0
		isCursor := i == m.cursor.Pos

		// Exit code
		code := fmt.Sprintf("%3d", entry.ExitCode)
		if isErr {
			code = m.tokens.NotificationError.Render(code)
		} else {
			code = m.tokens.ChangeAdded.Render(code)
		}

		// Command (strip internal flags)
		cmd := entry.Command
		for _, mask := range commandMask {
			cmd = strings.ReplaceAll(cmd, mask, "")
		}

		// Duration
		var timeStr string
		if entry.DurationMs >= 1000 {
			timeStr = fmt.Sprintf("(%d ms)", entry.DurationMs)
		} else {
			timeStr = fmt.Sprintf("(%3.3f ms)", float64(entry.DurationMs))
		}
		timeStr = m.tokens.SubtleText.Render(timeStr)

		// Stdio indicator
		var stdio string
		if isErr {
			lines := countLines(entry.Stderr)
			stdio = fmt.Sprintf("[stderr %3d]", lines)
		} else {
			lines := countLines(entry.Stdout)
			stdio = fmt.Sprintf("[stdout %3d]", lines)
		}
		stdio = m.tokens.SubtleText.Render(stdio)

		// Build the row
		row := fmt.Sprintf("%s %s  %s %s", code, cmd, timeStr, stdio)

		if isCursor {
			row = m.tokens.Cursor.Render(ansi.Strip(row))
		}

		b.WriteString(row)
		b.WriteString("\n")
		linesUsed++

		// If expanded, show output
		if !m.folded[i] {
			output := entry.Stdout + entry.Stderr
			if output != "" {
				for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
					if linesUsed >= maxLines {
						break
					}
					b.WriteString(m.tokens.SubtleText.Render("  | " + line))
					b.WriteString("\n")
					linesUsed++
				}
			}
			if entry.Error != "" {
				for _, line := range strings.Split(strings.TrimRight(entry.Error, "\n"), "\n") {
					if linesUsed >= maxLines {
						break
					}
					b.WriteString(m.tokens.NotificationError.Render("  ERR " + line))
					b.WriteString("\n")
					linesUsed++
				}
			}
			if linesUsed < maxLines {
				b.WriteString("\n")
				linesUsed++
			}
		}
	}

	return b.String()
}

// countLines counts non-empty lines in a string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	return len(lines)
}
