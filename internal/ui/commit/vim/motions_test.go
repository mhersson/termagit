package vim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCursor_WordForward_MovesToNextWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 0}

	c.WordForward(b)
	assert.Equal(t, 6, c.Col, "should move to 'world'")
}

func TestCursor_WordForward_MiddleOfWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 2}

	c.WordForward(b)
	assert.Equal(t, 6, c.Col, "should move to next word 'world'")
}

func TestCursor_WordForward_AtEndOfLine(t *testing.T) {
	b := NewBuffer("hello\nworld")
	c := &Cursor{Line: 0, Col: 4}

	c.WordForward(b)
	assert.Equal(t, 1, c.Line, "should move to next line")
	assert.Equal(t, 0, c.Col, "should be at start of next word")
}

func TestCursor_WordForward_StopsAtBufferEnd(t *testing.T) {
	b := NewBuffer("hello")
	c := &Cursor{Line: 0, Col: 3}

	c.WordForward(b)
	assert.Equal(t, 0, c.Line)
	assert.Equal(t, 4, c.Col, "should stop at last char")
}

func TestCursor_WordForward_SkipsWhitespace(t *testing.T) {
	b := NewBuffer("hello   world")
	c := &Cursor{Line: 0, Col: 0}

	c.WordForward(b)
	assert.Equal(t, 8, c.Col, "should skip multiple spaces to 'world'")
}

func TestCursor_WordBackward_MovesToPrevWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 8}

	c.WordBackward(b)
	assert.Equal(t, 6, c.Col, "should move to start of 'world'")
}

func TestCursor_WordBackward_AtStartOfWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 6}

	c.WordBackward(b)
	assert.Equal(t, 0, c.Col, "should move to start of 'hello'")
}

func TestCursor_WordBackward_AcrossLines(t *testing.T) {
	b := NewBuffer("hello\nworld")
	c := &Cursor{Line: 1, Col: 0}

	c.WordBackward(b)
	assert.Equal(t, 0, c.Line, "should move to previous line")
	assert.Equal(t, 0, c.Col, "should be at start of 'hello'")
}

func TestCursor_WordBackward_StopsAtBufferStart(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 0}

	c.WordBackward(b)
	assert.Equal(t, 0, c.Line)
	assert.Equal(t, 0, c.Col, "should stay at buffer start")
}

func TestCursor_WordEnd_MovesToEndOfWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 0}

	c.WordEnd(b)
	assert.Equal(t, 4, c.Col, "should move to end of 'hello'")
}

func TestCursor_WordEnd_AtEndOfWord_MovesToNextWordEnd(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 4}

	c.WordEnd(b)
	assert.Equal(t, 10, c.Col, "should move to end of 'world'")
}

func TestCursor_WordEnd_AcrossLines(t *testing.T) {
	b := NewBuffer("hi\nworld")
	c := &Cursor{Line: 0, Col: 1}

	c.WordEnd(b)
	assert.Equal(t, 1, c.Line, "should move to next line")
	assert.Equal(t, 4, c.Col, "should be at end of 'world'")
}

func TestCursor_FindWordBoundary_EmptyLine(t *testing.T) {
	b := NewBuffer("")
	c := &Cursor{Line: 0, Col: 0}

	c.WordForward(b)
	assert.Equal(t, 0, c.Col, "should stay at 0 for empty line")
}

func TestCursor_DeleteWord_ReturnsDeletedText(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 0}

	deleted := c.DeleteWord(b)
	assert.Equal(t, "hello ", deleted, "should delete word plus trailing space")
	assert.Equal(t, "world", b.Line(0))
	assert.Equal(t, 0, c.Col, "cursor should stay at position")
}

func TestCursor_DeleteWord_MiddleOfWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 2}

	deleted := c.DeleteWord(b)
	assert.Equal(t, "llo ", deleted)
	assert.Equal(t, "heworld", b.Line(0))
}

func TestCursor_DeleteWord_LastWord(t *testing.T) {
	b := NewBuffer("hello world")
	c := &Cursor{Line: 0, Col: 6}

	deleted := c.DeleteWord(b)
	assert.Equal(t, "world", deleted)
	assert.Equal(t, "hello ", b.Line(0))
}
