package vim

import "strings"

// Buffer holds line-based text content for the vim editor.
type Buffer struct {
	lines []string
}

// NewBuffer creates a new Buffer with the given content.
func NewBuffer(content string) *Buffer {
	var lines []string
	if content == "" {
		lines = []string{""}
	} else {
		lines = strings.Split(content, "\n")
	}
	return &Buffer{lines: lines}
}

// LineCount returns the number of lines in the buffer.
func (b *Buffer) LineCount() int {
	return len(b.lines)
}

// Line returns the content of the given line, or empty string if out of bounds.
func (b *Buffer) Line(lineNum int) string {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return ""
	}
	return b.lines[lineNum]
}

// LineLen returns the length of the given line.
func (b *Buffer) LineLen(lineNum int) int {
	return len(b.Line(lineNum))
}

// InsertRune inserts a rune at the given position.
func (b *Buffer) InsertRune(lineNum, col int, r rune) {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return
	}
	line := b.lines[lineNum]
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}
	b.lines[lineNum] = line[:col] + string(r) + line[col:]
}

// DeleteBack deletes the character before the cursor position.
// Returns true if deletion occurred.
func (b *Buffer) DeleteBack(lineNum, col int) bool {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return false
	}

	// At buffer start - nothing to delete
	if lineNum == 0 && col == 0 {
		return false
	}

	// At line start - join with previous line
	if col == 0 {
		prevLine := b.lines[lineNum-1]
		currLine := b.lines[lineNum]
		b.lines[lineNum-1] = prevLine + currLine
		b.lines = append(b.lines[:lineNum], b.lines[lineNum+1:]...)
		return true
	}

	// Delete character before cursor
	line := b.lines[lineNum]
	if col > len(line) {
		col = len(line)
	}
	b.lines[lineNum] = line[:col-1] + line[col:]
	return true
}

// InsertNewline splits the current line at the cursor position.
func (b *Buffer) InsertNewline(lineNum, col int) {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return
	}
	line := b.lines[lineNum]
	if col < 0 {
		col = 0
	}
	if col > len(line) {
		col = len(line)
	}

	before := line[:col]
	after := line[col:]

	b.lines[lineNum] = before
	// Insert new line after current
	newLines := make([]string, len(b.lines)+1)
	copy(newLines, b.lines[:lineNum+1])
	newLines[lineNum+1] = after
	copy(newLines[lineNum+2:], b.lines[lineNum+1:])
	b.lines = newLines
}

// DeleteLine deletes the given line. If it's the only line, leaves an empty line.
func (b *Buffer) DeleteLine(lineNum int) {
	if lineNum < 0 || lineNum >= len(b.lines) {
		return
	}
	if len(b.lines) == 1 {
		b.lines[0] = ""
		return
	}
	b.lines = append(b.lines[:lineNum], b.lines[lineNum+1:]...)
}

// DeleteLines deletes lines from startLine to endLine inclusive.
func (b *Buffer) DeleteLines(startLine, endLine int) {
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(b.lines) {
		endLine = len(b.lines) - 1
	}
	if startLine > endLine {
		return
	}
	b.lines = append(b.lines[:startLine], b.lines[endLine+1:]...)
	if len(b.lines) == 0 {
		b.lines = []string{""}
	}
}

// DeleteRange deletes text from (startLine, startCol) to (endLine, endCol).
func (b *Buffer) DeleteRange(startLine, startCol, endLine, endCol int) {
	if startLine < 0 || startLine >= len(b.lines) {
		return
	}
	if startLine == endLine {
		// Single line deletion
		line := b.lines[startLine]
		if startCol < 0 {
			startCol = 0
		}
		if endCol > len(line) {
			endCol = len(line)
		}
		b.lines[startLine] = line[:startCol] + line[endCol:]
		return
	}
	// Multi-line deletion would join start and end lines
	// For simplicity, handle the common cases
	startPart := b.lines[startLine][:startCol]
	endPart := ""
	if endLine < len(b.lines) {
		if endCol <= len(b.lines[endLine]) {
			endPart = b.lines[endLine][endCol:]
		}
	}
	b.lines[startLine] = startPart + endPart
	b.DeleteLines(startLine+1, endLine)
}

// JoinLine joins the given line with the next line, separated by a space.
// If the next line is empty (or whitespace-only), no space is added.
// No-op if lineNum is the last line or out of bounds.
func (b *Buffer) JoinLine(lineNum int) {
	if lineNum < 0 || lineNum >= len(b.lines)-1 {
		return
	}
	nextLine := strings.TrimLeft(b.lines[lineNum+1], " \t")
	if nextLine == "" {
		// Just remove the empty next line
	} else {
		b.lines[lineNum] = b.lines[lineNum] + " " + nextLine
	}
	b.lines = append(b.lines[:lineNum+1], b.lines[lineNum+2:]...)
}

// Content returns the full buffer content as a string.
func (b *Buffer) Content() string {
	return strings.Join(b.lines, "\n")
}

// SetContent replaces the buffer content.
func (b *Buffer) SetContent(content string) {
	if content == "" {
		b.lines = []string{""}
	} else {
		b.lines = strings.Split(content, "\n")
	}
}

// InsertLineBelow inserts a new line below the given line.
func (b *Buffer) InsertLineBelow(lineNum int, text string) {
	if lineNum < 0 {
		lineNum = 0
	}
	if lineNum >= len(b.lines) {
		lineNum = len(b.lines) - 1
	}
	newLines := make([]string, len(b.lines)+1)
	copy(newLines, b.lines[:lineNum+1])
	newLines[lineNum+1] = text
	copy(newLines[lineNum+2:], b.lines[lineNum+1:])
	b.lines = newLines
}

// InsertLineAbove inserts a new line above the given line.
func (b *Buffer) InsertLineAbove(lineNum int, text string) {
	if lineNum < 0 {
		lineNum = 0
	}
	if lineNum > len(b.lines) {
		lineNum = len(b.lines)
	}
	newLines := make([]string, len(b.lines)+1)
	copy(newLines, b.lines[:lineNum])
	newLines[lineNum] = text
	copy(newLines[lineNum+1:], b.lines[lineNum:])
	b.lines = newLines
}
