package nav

// Cursor provides cursor position, scrolling, and viewport tracking.
// Embed in view models for shared navigation behavior.
type Cursor struct {
	Pos        int    // cursor position
	Offset     int    // scroll offset
	PendingKey string // for "gg" sequence
	Width      int
	Height     int
	headerRows int // lines reserved for header (subtracted from Height for visible lines)
}

// NewCursor creates a Cursor with the given number of header rows reserved.
func NewCursor(headerRows int) Cursor {
	return Cursor{headerRows: headerRows}
}

// SetSize updates the dimensions.
func (c *Cursor) SetSize(w, h int) {
	c.Width = w
	c.Height = h
}

// VisibleLines returns how many item lines fit in the viewport.
func (c *Cursor) VisibleLines() int {
	v := c.Height - c.headerRows
	if v < 1 {
		return 1
	}
	return v
}

// MoveDown moves the cursor down by n, clamping to max.
func (c *Cursor) MoveDown(n, max int) {
	if max < 0 {
		return
	}
	c.Pos += n
	if c.Pos > max {
		c.Pos = max
	}
	c.EnsureVisible()
}

// MoveUp moves the cursor up by n, clamping to 0.
func (c *Cursor) MoveUp(n int) {
	c.Pos -= n
	if c.Pos < 0 {
		c.Pos = 0
	}
	c.EnsureVisible()
}

// GoToTop moves the cursor to position 0.
func (c *Cursor) GoToTop() {
	c.Pos = 0
	c.Offset = 0
}

// GoToBottom moves the cursor to max.
func (c *Cursor) GoToBottom(max int) {
	if max >= 0 {
		c.Pos = max
		c.EnsureVisible()
	}
}

// EnsureVisible adjusts the scroll offset so the cursor is in the viewport.
func (c *Cursor) EnsureVisible() {
	vis := c.VisibleLines()
	if c.Pos < c.Offset {
		c.Offset = c.Pos
	}
	if c.Pos >= c.Offset+vis {
		c.Offset = c.Pos - vis + 1
	}
}

// HandleGG processes the "gg" two-key sequence.
// Call this at the top of handleKey. Returns true if the key was consumed.
func (c *Cursor) HandleGG(keyStr string) bool {
	if c.PendingKey == "g" {
		c.PendingKey = ""
		if keyStr == "g" {
			c.GoToTop()
			return true
		}
		return false
	}
	return false
}
