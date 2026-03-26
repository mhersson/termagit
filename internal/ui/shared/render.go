package shared

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// PadRight pads a string with spaces to the given width in runes.
func PadRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// PadToHeight ensures the string has at least height lines.
func PadToHeight(s string, height int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines[:height], "\n")
}

// MaxVisibleWidth returns the maximum visible width across all lines in content.
func MaxVisibleWidth(content string) int {
	maxW := 0
	for _, line := range strings.Split(content, "\n") {
		w := ansi.StringWidth(line)
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

// TruncateString truncates a string to maxRunes runes.
func TruncateString(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}
