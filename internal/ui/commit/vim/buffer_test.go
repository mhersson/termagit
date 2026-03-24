package vim

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuffer_NewBuffer_Empty(t *testing.T) {
	b := NewBuffer("")
	assert.Equal(t, 1, b.LineCount(), "empty buffer should have 1 line")
	assert.Equal(t, "", b.Line(0), "first line should be empty")
}

func TestBuffer_NewBuffer_WithContent(t *testing.T) {
	b := NewBuffer("hello\nworld")
	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "hello", b.Line(0))
	assert.Equal(t, "world", b.Line(1))
}

func TestBuffer_InsertRune(t *testing.T) {
	b := NewBuffer("")
	b.InsertRune(0, 0, 'H')
	b.InsertRune(0, 1, 'i')

	assert.Equal(t, "Hi", b.Line(0))
}

func TestBuffer_InsertRune_MiddleOfLine(t *testing.T) {
	b := NewBuffer("Hllo")
	b.InsertRune(0, 1, 'e')

	assert.Equal(t, "Hello", b.Line(0))
}

func TestBuffer_InsertRune_MultipleLinesPreserved(t *testing.T) {
	b := NewBuffer("line1\nline2")
	b.InsertRune(0, 5, '!')

	assert.Equal(t, "line1!", b.Line(0))
	assert.Equal(t, "line2", b.Line(1))
}

func TestBuffer_DeleteBack(t *testing.T) {
	b := NewBuffer("Hello")
	deleted := b.DeleteBack(0, 5)

	assert.Equal(t, "Hell", b.Line(0))
	assert.True(t, deleted, "should return true when deletion occurred")
}

func TestBuffer_DeleteBack_AtLineStart_JoinsWithPrevious(t *testing.T) {
	b := NewBuffer("Hello\nWorld")
	deleted := b.DeleteBack(1, 0)

	assert.True(t, deleted)
	assert.Equal(t, 1, b.LineCount())
	assert.Equal(t, "HelloWorld", b.Line(0))
}

func TestBuffer_DeleteBack_AtBufferStart_ReturnsFalse(t *testing.T) {
	b := NewBuffer("Hello")
	deleted := b.DeleteBack(0, 0)

	assert.False(t, deleted, "should return false at buffer start")
	assert.Equal(t, "Hello", b.Line(0))
}

func TestBuffer_InsertNewline(t *testing.T) {
	b := NewBuffer("HelloWorld")
	b.InsertNewline(0, 5)

	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "Hello", b.Line(0))
	assert.Equal(t, "World", b.Line(1))
}

func TestBuffer_InsertNewline_AtEnd(t *testing.T) {
	b := NewBuffer("Hello")
	b.InsertNewline(0, 5)

	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "Hello", b.Line(0))
	assert.Equal(t, "", b.Line(1))
}

func TestBuffer_DeleteLine(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	b.DeleteLine(1)

	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "line1", b.Line(0))
	assert.Equal(t, "line3", b.Line(1))
}

func TestBuffer_DeleteLine_OnlyLine_LeavesEmptyLine(t *testing.T) {
	b := NewBuffer("only line")
	b.DeleteLine(0)

	assert.Equal(t, 1, b.LineCount())
	assert.Equal(t, "", b.Line(0))
}

func TestBuffer_DeleteRange_SingleLine(t *testing.T) {
	b := NewBuffer("Hello World")
	b.DeleteRange(0, 0, 0, 6) // Delete "Hello "

	assert.Equal(t, "World", b.Line(0))
}

func TestBuffer_DeleteRange_MultipleLinesFromTo(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3\nline4")
	b.DeleteLines(1, 2) // Delete line2 and line3

	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "line1", b.Line(0))
	assert.Equal(t, "line4", b.Line(1))
}

func TestBuffer_Content(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	assert.Equal(t, "line1\nline2\nline3", b.Content())
}

func TestBuffer_Content_EmptyBuffer(t *testing.T) {
	b := NewBuffer("")
	assert.Equal(t, "", b.Content())
}

func TestBuffer_SetContent(t *testing.T) {
	b := NewBuffer("old content")
	b.SetContent("new\ncontent")

	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "new", b.Line(0))
	assert.Equal(t, "content", b.Line(1))
}

func TestBuffer_LineLen(t *testing.T) {
	b := NewBuffer("hello\nworld!")
	assert.Equal(t, 5, b.LineLen(0))
	assert.Equal(t, 6, b.LineLen(1))
}

func TestBuffer_Line_OutOfBounds_ReturnsEmpty(t *testing.T) {
	b := NewBuffer("hello")
	assert.Equal(t, "", b.Line(100))
	assert.Equal(t, "", b.Line(-1))
}

func TestBuffer_InsertLineBelow(t *testing.T) {
	b := NewBuffer("line1\nline3")
	b.InsertLineBelow(0, "line2")

	require.Equal(t, 3, b.LineCount())
	assert.Equal(t, "line1", b.Line(0))
	assert.Equal(t, "line2", b.Line(1))
	assert.Equal(t, "line3", b.Line(2))
}

func TestBuffer_InsertLineAbove(t *testing.T) {
	b := NewBuffer("line2\nline3")
	b.InsertLineAbove(0, "line1")

	require.Equal(t, 3, b.LineCount())
	assert.Equal(t, "line1", b.Line(0))
	assert.Equal(t, "line2", b.Line(1))
	assert.Equal(t, "line3", b.Line(2))
}

func TestBuffer_JoinLine_JoinsWithNextLine(t *testing.T) {
	b := NewBuffer("hello\nworld")
	b.JoinLine(0)

	assert.Equal(t, 1, b.LineCount())
	assert.Equal(t, "hello world", b.Line(0))
}

func TestBuffer_JoinLine_LastLine_NoOp(t *testing.T) {
	b := NewBuffer("hello")
	b.JoinLine(0)

	assert.Equal(t, 1, b.LineCount())
	assert.Equal(t, "hello", b.Line(0))
}

func TestBuffer_JoinLine_EmptyNextLine(t *testing.T) {
	b := NewBuffer("hello\n")
	b.JoinLine(0)

	assert.Equal(t, 1, b.LineCount())
	assert.Equal(t, "hello", b.Line(0))
}

func TestBuffer_JoinLine_PreservesRemaining(t *testing.T) {
	b := NewBuffer("line1\nline2\nline3")
	b.JoinLine(0)

	assert.Equal(t, 2, b.LineCount())
	assert.Equal(t, "line1 line2", b.Line(0))
	assert.Equal(t, "line3", b.Line(1))
}

func TestBuffer_JoinLine_TrimsLeadingWhitespace(t *testing.T) {
	b := NewBuffer("hello\n   world")
	b.JoinLine(0)

	assert.Equal(t, 1, b.LineCount())
	assert.Equal(t, "hello world", b.Line(0))
}
