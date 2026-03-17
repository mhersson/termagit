package vim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCursor_New(t *testing.T) {
	c := NewCursor()
	assert.Equal(t, 0, c.Line)
	assert.Equal(t, 0, c.Col)
}

func TestCursor_MoveRight_MovesWithinLine(t *testing.T) {
	b := NewBuffer("hello")
	c := NewCursor()

	c.MoveRight(b)
	assert.Equal(t, 1, c.Col)

	c.MoveRight(b)
	assert.Equal(t, 2, c.Col)
}

func TestCursor_MoveRight_StopsAtLineEnd(t *testing.T) {
	b := NewBuffer("hi")
	c := &Cursor{Line: 0, Col: 1}

	c.MoveRight(b)
	// In normal mode, cursor stops at last char (col 1 for "hi")
	assert.Equal(t, 1, c.Col, "cursor should stop at last character")
}

func TestCursor_MoveRight_StopsAtLineEnd_InsertMode(t *testing.T) {
	b := NewBuffer("hi")
	c := &Cursor{Line: 0, Col: 1}

	c.MoveRightInsert(b)
	assert.Equal(t, 2, c.Col, "insert mode allows cursor past last char")
}

func TestCursor_MoveLeft_MovesWithinLine(t *testing.T) {
	b := NewBuffer("hello")
	c := &Cursor{Line: 0, Col: 3}

	c.MoveLeft(b)
	assert.Equal(t, 2, c.Col)
}

func TestCursor_MoveLeft_StopsAtLineStart(t *testing.T) {
	b := NewBuffer("hello")
	c := &Cursor{Line: 0, Col: 0}

	c.MoveLeft(b)
	assert.Equal(t, 0, c.Col, "cursor should stop at line start")
}

func TestCursor_MoveDown_MovesWithinBuffer(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	c := &Cursor{Line: 0, Col: 0}

	c.MoveDown(b)
	assert.Equal(t, 1, c.Line)
}

func TestCursor_MoveDown_StopsAtBufferEnd(t *testing.T) {
	b := NewBuffer("line1\nline2")
	c := &Cursor{Line: 1, Col: 0}

	c.MoveDown(b)
	assert.Equal(t, 1, c.Line, "cursor should stop at last line")
}

func TestCursor_MoveDown_ClampsColumnToShorterLine(t *testing.T) {
	b := NewBuffer("longer line\nhi")
	c := &Cursor{Line: 0, Col: 10}

	c.MoveDown(b)
	assert.Equal(t, 1, c.Line)
	assert.Equal(t, 1, c.Col, "col should clamp to shorter line")
}

func TestCursor_MoveUp_MovesWithinBuffer(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	c := &Cursor{Line: 2, Col: 0}

	c.MoveUp(b)
	assert.Equal(t, 1, c.Line)
}

func TestCursor_MoveUp_StopsAtBufferStart(t *testing.T) {
	b := NewBuffer("line1\nline2")
	c := &Cursor{Line: 0, Col: 3}

	c.MoveUp(b)
	assert.Equal(t, 0, c.Line, "cursor should stop at first line")
}

func TestCursor_MoveUp_ClampsColumnToShorterLine(t *testing.T) {
	b := NewBuffer("hi\nlonger line")
	c := &Cursor{Line: 1, Col: 10}

	c.MoveUp(b)
	assert.Equal(t, 0, c.Line)
	assert.Equal(t, 1, c.Col, "col should clamp to shorter line")
}

func TestCursor_LineStart_MovesToColumn0(t *testing.T) {
	b := NewBuffer("hello")
	c := &Cursor{Line: 0, Col: 3}

	c.LineStart(b)
	assert.Equal(t, 0, c.Col)
}

func TestCursor_LineEnd_MovesToLastChar(t *testing.T) {
	b := NewBuffer("hello")
	c := &Cursor{Line: 0, Col: 0}

	c.LineEnd(b)
	assert.Equal(t, 4, c.Col, "should be at last character (index 4 for 'hello')")
}

func TestCursor_LineEnd_EmptyLine(t *testing.T) {
	b := NewBuffer("")
	c := NewCursor()

	c.LineEnd(b)
	assert.Equal(t, 0, c.Col, "empty line should stay at 0")
}

func TestCursor_BufferStart_GoesToFirstLine(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	c := &Cursor{Line: 2, Col: 3}

	c.BufferStart(b)
	assert.Equal(t, 0, c.Line)
	assert.Equal(t, 0, c.Col)
}

func TestCursor_BufferEnd_GoesToLastLine(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	c := &Cursor{Line: 0, Col: 0}

	c.BufferEnd(b)
	assert.Equal(t, 2, c.Line)
	assert.Equal(t, 0, c.Col, "G goes to beginning of last line")
}

func TestCursor_Clamp(t *testing.T) {
	b := NewBuffer("hi\nbye")
	c := &Cursor{Line: 5, Col: 10}

	c.Clamp(b)
	assert.Equal(t, 1, c.Line, "line should clamp to valid range")
	assert.Equal(t, 2, c.Col, "col should clamp to line length")
}

func TestCursor_Clamp_EmptyBuffer(t *testing.T) {
	b := NewBuffer("")
	c := &Cursor{Line: 5, Col: 10}

	c.Clamp(b)
	assert.Equal(t, 0, c.Line)
	assert.Equal(t, 0, c.Col)
}
