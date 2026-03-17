package vim

// Cursor represents a position in the buffer.
type Cursor struct {
	Line int
	Col  int
}

// NewCursor creates a new cursor at position (0, 0).
func NewCursor() *Cursor {
	return &Cursor{Line: 0, Col: 0}
}

// MoveRight moves the cursor one character to the right.
// In normal mode, stops at the last character of the line.
func (c *Cursor) MoveRight(b *Buffer) {
	lineLen := b.LineLen(c.Line)
	if lineLen == 0 {
		return
	}
	// Normal mode: cursor can only go to last char (len-1)
	maxCol := lineLen - 1
	if c.Col < maxCol {
		c.Col++
	}
}

// MoveRightInsert moves the cursor one character to the right for insert mode.
// Allows cursor to be past the last character.
func (c *Cursor) MoveRightInsert(b *Buffer) {
	lineLen := b.LineLen(c.Line)
	if c.Col < lineLen {
		c.Col++
	}
}

// MoveLeft moves the cursor one character to the left.
func (c *Cursor) MoveLeft(b *Buffer) {
	if c.Col > 0 {
		c.Col--
	}
}

// MoveDown moves the cursor one line down.
func (c *Cursor) MoveDown(b *Buffer) {
	if c.Line < b.LineCount()-1 {
		c.Line++
		c.clampCol(b)
	}
}

// MoveUp moves the cursor one line up.
func (c *Cursor) MoveUp(b *Buffer) {
	if c.Line > 0 {
		c.Line--
		c.clampCol(b)
	}
}

// LineStart moves the cursor to the start of the line (column 0).
func (c *Cursor) LineStart(b *Buffer) {
	c.Col = 0
}

// LineEnd moves the cursor to the last character of the line.
func (c *Cursor) LineEnd(b *Buffer) {
	lineLen := b.LineLen(c.Line)
	if lineLen > 0 {
		c.Col = lineLen - 1
	} else {
		c.Col = 0
	}
}

// BufferStart moves the cursor to the start of the buffer (gg).
func (c *Cursor) BufferStart(b *Buffer) {
	c.Line = 0
	c.Col = 0
}

// BufferEnd moves the cursor to the last line (G).
func (c *Cursor) BufferEnd(b *Buffer) {
	c.Line = b.LineCount() - 1
	if c.Line < 0 {
		c.Line = 0
	}
	c.Col = 0
}

// Clamp ensures the cursor is within valid bounds.
func (c *Cursor) Clamp(b *Buffer) {
	// Clamp line
	if c.Line < 0 {
		c.Line = 0
	} else if c.Line >= b.LineCount() {
		c.Line = b.LineCount() - 1
	}
	if c.Line < 0 {
		c.Line = 0
	}
	// Clamp column
	c.clampCol(b)
}

// clampCol ensures the column is within valid bounds for the current line.
func (c *Cursor) clampCol(b *Buffer) {
	lineLen := b.LineLen(c.Line)
	if lineLen == 0 {
		c.Col = 0
		return
	}
	// In normal mode, max col is lineLen-1
	maxCol := lineLen - 1
	if c.Col > maxCol {
		c.Col = maxCol
	}
	if c.Col < 0 {
		c.Col = 0
	}
}

// ClampInsert ensures the cursor is within valid bounds for insert mode.
func (c *Cursor) ClampInsert(b *Buffer) {
	// Clamp line
	if c.Line < 0 {
		c.Line = 0
	} else if c.Line >= b.LineCount() {
		c.Line = b.LineCount() - 1
	}
	if c.Line < 0 {
		c.Line = 0
	}
	// In insert mode, col can be at lineLen (past last char)
	lineLen := b.LineLen(c.Line)
	if c.Col > lineLen {
		c.Col = lineLen
	}
	if c.Col < 0 {
		c.Col = 0
	}
}
