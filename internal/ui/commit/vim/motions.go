package vim

import "unicode"

// WordForward moves the cursor to the start of the next word (w motion).
func (c *Cursor) WordForward(b *Buffer) {
	line := b.Line(c.Line)
	col := c.Col

	// First, skip to end of current word (if in a word)
	for col < len(line) && !unicode.IsSpace(rune(line[col])) {
		col++
	}

	// Skip whitespace
	for col < len(line) && unicode.IsSpace(rune(line[col])) {
		col++
	}

	// If we reached end of line, try next line
	if col >= len(line) {
		if c.Line < b.LineCount()-1 {
			c.Line++
			c.Col = 0
			// Skip leading whitespace on new line
			newLine := b.Line(c.Line)
			for c.Col < len(newLine) && unicode.IsSpace(rune(newLine[c.Col])) {
				c.Col++
			}
		} else {
			// At last line, go to last char
			if len(line) > 0 {
				c.Col = len(line) - 1
			}
		}
		return
	}

	c.Col = col
}

// WordBackward moves the cursor to the start of the previous word (b motion).
func (c *Cursor) WordBackward(b *Buffer) {
	line := b.Line(c.Line)
	col := c.Col

	// If at start of line, go to previous line
	if col == 0 {
		if c.Line > 0 {
			c.Line--
			line = b.Line(c.Line)
			col = len(line)
			// Skip trailing whitespace
			for col > 0 && unicode.IsSpace(rune(line[col-1])) {
				col--
			}
			// Skip to start of word
			for col > 0 && !unicode.IsSpace(rune(line[col-1])) {
				col--
			}
			c.Col = col
		}
		return
	}

	// Skip whitespace going backwards
	for col > 0 && unicode.IsSpace(rune(line[col-1])) {
		col--
	}

	// Skip to start of word
	for col > 0 && !unicode.IsSpace(rune(line[col-1])) {
		col--
	}

	c.Col = col
}

// WordEnd moves the cursor to the end of the current/next word (e motion).
func (c *Cursor) WordEnd(b *Buffer) {
	line := b.Line(c.Line)
	col := c.Col

	// Move at least one character
	if col < len(line) {
		col++
	}

	// Skip whitespace
	for col < len(line) && unicode.IsSpace(rune(line[col])) {
		col++
	}

	// If we hit end of line, try next line
	if col >= len(line) {
		if c.Line < b.LineCount()-1 {
			c.Line++
			line = b.Line(c.Line)
			col = 0
			// Skip leading whitespace
			for col < len(line) && unicode.IsSpace(rune(line[col])) {
				col++
			}
		}
	}

	// Move to end of word
	for col < len(line) && !unicode.IsSpace(rune(line[col])) {
		col++
	}
	col-- // Back up to last char of word

	if col < 0 {
		col = 0
	}
	c.Col = col
}

// DeleteWord deletes from cursor to start of next word and returns deleted text.
func (c *Cursor) DeleteWord(b *Buffer) string {
	startCol := c.Col
	line := b.Line(c.Line)

	// Find end position (same logic as WordForward)
	endCol := startCol

	// Skip current word
	for endCol < len(line) && !unicode.IsSpace(rune(line[endCol])) {
		endCol++
	}

	// Include trailing whitespace (for dw behavior)
	for endCol < len(line) && unicode.IsSpace(rune(line[endCol])) {
		endCol++
	}

	deleted := line[startCol:endCol]
	b.DeleteRange(c.Line, startCol, c.Line, endCol)

	return deleted
}

// DeleteToLineEnd deletes from cursor to end of line and returns deleted text.
func (c *Cursor) DeleteToLineEnd(b *Buffer) string {
	line := b.Line(c.Line)
	if c.Col >= len(line) {
		return ""
	}
	deleted := line[c.Col:]
	b.DeleteRange(c.Line, c.Col, c.Line, len(line))
	return deleted
}
