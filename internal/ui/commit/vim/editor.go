package vim

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mode represents the vim editing mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisualLine
)

// Tokens holds compiled styles for rendering.
type Tokens struct {
	Normal      lipgloss.Style
	CursorBlock lipgloss.Style
	Selection   lipgloss.Style
	Comment     lipgloss.Style // For comment lines (# ...)

	// Diff syntax highlighting
	DiffAdd        lipgloss.Style // Lines starting with +
	DiffDelete     lipgloss.Style // Lines starting with -
	DiffContext    lipgloss.Style // Context lines (space prefix)
	DiffHunkHeader lipgloss.Style // @@ ... @@ lines
	DiffHeader     lipgloss.Style // diff --git, ---, +++ lines
}

// Editor is a vim-like text editor component.
type Editor struct {
	buffer   *Buffer
	cursor   *Cursor
	mode     Mode
	tokens   Tokens
	pending  rune // For operator-pending mode (d, c, y, g)
	selStart int  // Visual line selection start
	selEnd   int  // Visual line selection end

	register       string // unnamed register contents
	registerIsLine bool   // true if register holds a full line

	width, height int
	viewportTop   int // First visible line in the viewport
	xOffset       int // First visible column in the viewport (horizontal scroll)
	diffStartLine int // First line of the diff section (-1 = no diff section)
}

// NewEditor creates a new vim editor with the specified initial mode.
func NewEditor(tokens Tokens, initialMode Mode) *Editor {
	return &Editor{
		buffer:        NewBuffer(""),
		cursor:        NewCursor(),
		mode:          initialMode,
		tokens:        tokens,
		diffStartLine: -1, // No diff section by default
	}
}

// Mode returns the current editing mode.
func (e *Editor) Mode() Mode {
	return e.mode
}

// SetMode sets the editing mode.
func (e *Editor) SetMode(m Mode) {
	e.mode = m
	if m == ModeNormal || m == ModeVisualLine {
		e.cursor.Clamp(e.buffer)
	}
}

// Content returns the full buffer content.
func (e *Editor) Content() string {
	return e.buffer.Content()
}

// SetContent sets the buffer content and resets viewport to top.
func (e *Editor) SetContent(content string) {
	e.buffer.SetContent(content)
	e.cursor.Clamp(e.buffer)
	e.viewportTop = 0 // Reset viewport to show from the beginning
	e.xOffset = 0     // Reset horizontal scroll
	e.diffStartLine = e.findDiffStart()
}

// findDiffStart scans the buffer for the scissors line (">8") and returns the
// line index where the diff section begins, or -1 if there is no scissors line.
func (e *Editor) findDiffStart() int {
	for i := 0; i < e.buffer.LineCount(); i++ {
		line := e.buffer.Line(i)
		if strings.Contains(line, ">8") || strings.Contains(line, "> 8") {
			return i
		}
	}
	return -1
}

// Line returns the current cursor line.
func (e *Editor) Line() int {
	return e.cursor.Line
}

// Col returns the current cursor column.
func (e *Editor) Col() int {
	return e.cursor.Col
}

// XOffset returns the horizontal scroll offset.
func (e *Editor) XOffset() int {
	return e.xOffset
}

// SetCursor sets the cursor position.
func (e *Editor) SetCursor(line, col int) {
	e.cursor.Line = line
	e.cursor.Col = col
	// Reset viewport to show from cursor line when explicitly setting cursor
	e.viewportTop = line
}

// ResetViewport resets the viewport to show from line 0.
func (e *Editor) ResetViewport() {
	e.viewportTop = 0
}

// LineCount returns the number of lines.
func (e *Editor) LineCount() int {
	return e.buffer.LineCount()
}

// LineContent returns the content of the specified line.
func (e *Editor) LineContent(line int) string {
	return e.buffer.Line(line)
}

// SelectionStart returns the start line of visual selection.
func (e *Editor) SelectionStart() int {
	return e.selStart
}

// SelectionEnd returns the end line of visual selection.
func (e *Editor) SelectionEnd() int {
	return e.selEnd
}

// Register returns the unnamed register contents.
func (e *Editor) Register() string {
	return e.register
}

// RegisterIsLine returns true if the register holds a full line.
func (e *Editor) RegisterIsLine() bool {
	return e.registerIsLine
}

// HandleKey processes a key press and returns true if the key was handled.
func (e *Editor) HandleKey(msg tea.KeyMsg) bool {
	switch e.mode {
	case ModeInsert:
		return e.handleInsertMode(msg)
	case ModeNormal:
		return e.handleNormalMode(msg)
	case ModeVisualLine:
		return e.handleVisualLineMode(msg)
	}
	return false
}

// handleInsertMode handles keys in insert mode.
func (e *Editor) handleInsertMode(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEscape:
		e.mode = ModeNormal
		e.cursor.Clamp(e.buffer)
		return true

	case tea.KeyBackspace:
		if e.buffer.DeleteBack(e.cursor.Line, e.cursor.Col) {
			if e.cursor.Col > 0 {
				e.cursor.Col--
			} else if e.cursor.Line > 0 {
				// Joined with previous line - cursor moved up
				e.cursor.Line--
				e.cursor.Col = e.buffer.LineLen(e.cursor.Line)
			}
		}
		return true

	case tea.KeyEnter:
		e.buffer.InsertNewline(e.cursor.Line, e.cursor.Col)
		e.cursor.Line++
		e.cursor.Col = 0
		return true

	case tea.KeySpace:
		e.buffer.InsertRune(e.cursor.Line, e.cursor.Col, ' ')
		e.cursor.Col++
		return true

	case tea.KeyTab:
		e.buffer.InsertRune(e.cursor.Line, e.cursor.Col, '\t')
		e.cursor.Col++
		return true

	case tea.KeyRunes:
		for _, r := range msg.Runes {
			switch r {
			case '\n':
				e.buffer.InsertNewline(e.cursor.Line, e.cursor.Col)
				e.cursor.Line++
				e.cursor.Col = 0
			case '\r':
				// Skip carriage returns (Windows \r\n)
			default:
				e.buffer.InsertRune(e.cursor.Line, e.cursor.Col, r)
				e.cursor.Col++
			}
		}
		return true
	}
	return false
}

// handleNormalMode handles keys in normal mode.
func (e *Editor) handleNormalMode(msg tea.KeyMsg) bool {
	// Handle pending operator (d, c, y)
	if e.pending != 0 {
		return e.handlePendingOperator(msg)
	}

	switch msg.Type {
	case tea.KeyEscape:
		e.pending = 0
		return true

	case tea.KeyCtrlF:
		// Page down
		e.pageDown()
		return true

	case tea.KeyCtrlB:
		// Page up
		e.pageUp()
		return true

	case tea.KeyCtrlD:
		// Half page down
		e.halfPageDown()
		return true

	case tea.KeyCtrlU:
		// Half page up
		e.halfPageUp()
		return true

	case tea.KeyRunes:
		if len(msg.Runes) == 0 {
			return false
		}
		return e.handleNormalRune(msg.Runes[0])
	}
	return false
}

// pageDown moves cursor down by a full page.
func (e *Editor) pageDown() {
	if e.height == 0 {
		return
	}
	lines := e.height - 1 // Leave one line overlap for context
	if lines < 1 {
		lines = 1
	}
	e.cursor.Line += lines
	lineCount := e.buffer.LineCount()
	if e.cursor.Line >= lineCount {
		e.cursor.Line = lineCount - 1
	}
	if e.cursor.Line < 0 {
		e.cursor.Line = 0
	}
	e.cursor.Clamp(e.buffer)
}

// pageUp moves cursor up by a full page.
func (e *Editor) pageUp() {
	if e.height == 0 {
		return
	}
	lines := e.height - 1 // Leave one line overlap for context
	if lines < 1 {
		lines = 1
	}
	e.cursor.Line -= lines
	if e.cursor.Line < 0 {
		e.cursor.Line = 0
	}
	e.cursor.Clamp(e.buffer)
}

// halfPageDown moves cursor down by half a page.
func (e *Editor) halfPageDown() {
	if e.height == 0 {
		return
	}
	lines := e.height / 2
	if lines < 1 {
		lines = 1
	}
	e.cursor.Line += lines
	lineCount := e.buffer.LineCount()
	if e.cursor.Line >= lineCount {
		e.cursor.Line = lineCount - 1
	}
	if e.cursor.Line < 0 {
		e.cursor.Line = 0
	}
	e.cursor.Clamp(e.buffer)
}

// halfPageUp moves cursor up by half a page.
func (e *Editor) halfPageUp() {
	if e.height == 0 {
		return
	}
	lines := e.height / 2
	if lines < 1 {
		lines = 1
	}
	e.cursor.Line -= lines
	if e.cursor.Line < 0 {
		e.cursor.Line = 0
	}
	e.cursor.Clamp(e.buffer)
}

// handleNormalRune handles a single rune in normal mode.
func (e *Editor) handleNormalRune(r rune) bool {
	switch r {
	// Mode switches
	case 'i':
		e.mode = ModeInsert
		return true
	case 'a':
		e.cursor.MoveRightInsert(e.buffer)
		e.mode = ModeInsert
		return true
	case 'A':
		e.cursor.Col = e.buffer.LineLen(e.cursor.Line)
		e.mode = ModeInsert
		return true
	case 'o':
		e.buffer.InsertLineBelow(e.cursor.Line, "")
		e.cursor.Line++
		e.cursor.Col = 0
		e.mode = ModeInsert
		return true
	case 'O':
		e.buffer.InsertLineAbove(e.cursor.Line, "")
		e.cursor.Col = 0
		e.mode = ModeInsert
		return true

	// Movement
	case 'h':
		e.cursor.MoveLeft(e.buffer)
		return true
	case 'l':
		e.cursor.MoveRight(e.buffer)
		return true
	case 'j':
		e.cursor.MoveDown(e.buffer)
		return true
	case 'k':
		e.cursor.MoveUp(e.buffer)
		return true
	case '0':
		e.cursor.LineStart(e.buffer)
		return true
	case '$':
		e.cursor.LineEnd(e.buffer)
		return true
	case 'w':
		e.cursor.WordForward(e.buffer)
		return true
	case 'b':
		e.cursor.WordBackward(e.buffer)
		return true
	case 'e':
		e.cursor.WordEnd(e.buffer)
		return true
	case 'g':
		e.pending = 'g'
		return true
	case 'G':
		e.cursor.BufferEnd(e.buffer)
		return true

	// Visual mode
	case 'V':
		e.mode = ModeVisualLine
		e.selStart = e.cursor.Line
		e.selEnd = e.cursor.Line
		return true

	// Operators
	case 'd':
		e.pending = 'd'
		return true
	case 'c':
		e.pending = 'c'
		return true
	case 'y':
		e.pending = 'y'
		return true
	case 'x':
		e.deleteChar()
		return true
	case 'p':
		e.pasteAfter()
		return true
	case 'P':
		e.pasteBefore()
		return true
	case 'J':
		if e.cursor.Line < e.buffer.LineCount()-1 {
			joinCol := e.buffer.LineLen(e.cursor.Line)
			e.buffer.JoinLine(e.cursor.Line)
			e.cursor.Col = joinCol
		}
		return true

	case 'D':
		// D = delete to end of line (same as d$)
		deleted := e.cursor.DeleteToLineEnd(e.buffer)
		e.register = deleted
		e.registerIsLine = false
		e.cursor.Clamp(e.buffer)
		return true
	case 'C':
		// C = change to end of line (same as c$)
		deleted := e.cursor.DeleteToLineEnd(e.buffer)
		e.register = deleted
		e.registerIsLine = false
		e.mode = ModeInsert
		return true
	}
	return false
}

// handlePendingOperator handles keys when an operator is pending (d, c, g).
func (e *Editor) handlePendingOperator(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes || len(msg.Runes) == 0 {
		if msg.Type == tea.KeyEscape {
			e.pending = 0
			return true
		}
		return false
	}

	r := msg.Runes[0]
	op := e.pending
	e.pending = 0

	switch op {
	case 'g':
		if r == 'g' {
			e.cursor.BufferStart(e.buffer)
			return true
		}
	case 'd':
		return e.handleDeleteOperator(r)
	case 'c':
		return e.handleChangeOperator(r)
	case 'y':
		return e.handleYankOperator(r)
	}
	return false
}

// handleDeleteOperator handles d<motion> operations.
func (e *Editor) handleDeleteOperator(r rune) bool {
	switch r {
	case 'd':
		// dd = delete line
		e.register = e.buffer.Line(e.cursor.Line)
		e.registerIsLine = true
		e.buffer.DeleteLine(e.cursor.Line)
		e.cursor.Clamp(e.buffer)
		return true
	case 'w':
		// dw = delete word
		deleted := e.cursor.DeleteWord(e.buffer)
		e.register = deleted
		e.registerIsLine = false
		e.cursor.Clamp(e.buffer)
		return true
	case '$':
		// d$ = delete to end of line
		deleted := e.cursor.DeleteToLineEnd(e.buffer)
		e.register = deleted
		e.registerIsLine = false
		e.cursor.Clamp(e.buffer)
		return true
	}
	return false
}

// handleChangeOperator handles c<motion> operations.
func (e *Editor) handleChangeOperator(r rune) bool {
	switch r {
	case 'c':
		// cc = change line
		line := e.cursor.Line
		e.register = e.buffer.Line(line)
		e.registerIsLine = true
		e.buffer.DeleteLine(line)
		if e.buffer.LineCount() == 1 && e.buffer.Line(0) == "" {
			// Buffer is empty, stay at line 0
		} else if line >= e.buffer.LineCount() {
			// Deleted last line, stay at end
			e.cursor.Line = e.buffer.LineCount() - 1
		}
		e.cursor.Col = 0
		e.mode = ModeInsert
		return true
	case 'w':
		// cw = change word
		deleted := e.cursor.DeleteWord(e.buffer)
		e.register = deleted
		e.registerIsLine = false
		e.mode = ModeInsert
		return true
	}
	return false
}

// handleVisualLineMode handles keys in visual line mode.
func (e *Editor) handleVisualLineMode(msg tea.KeyMsg) bool {
	switch msg.Type {
	case tea.KeyEscape:
		e.mode = ModeNormal
		return true

	case tea.KeyRunes:
		if len(msg.Runes) == 0 {
			return false
		}
		return e.handleVisualLineRune(msg.Runes[0])
	}
	return false
}

// handleVisualLineRune handles a single rune in visual line mode.
func (e *Editor) handleVisualLineRune(r rune) bool {
	switch r {
	case 'j':
		e.cursor.MoveDown(e.buffer)
		e.selEnd = e.cursor.Line
		return true
	case 'k':
		e.cursor.MoveUp(e.buffer)
		if e.cursor.Line < e.selStart {
			e.selStart = e.cursor.Line
		} else {
			e.selEnd = e.cursor.Line
		}
		return true
	case 'd':
		e.deleteVisualSelection()
		e.mode = ModeNormal
		return true
	case 'c':
		e.deleteVisualSelection()
		e.mode = ModeInsert
		return true
	case 'y':
		e.yankVisualSelection()
		e.mode = ModeNormal
		return true
	}
	return false
}

// deleteChar deletes the character under the cursor (x).
func (e *Editor) deleteChar() {
	line := e.buffer.Line(e.cursor.Line)
	if e.cursor.Col < len(line) {
		runes := []rune(line)
		if e.cursor.Col < len(runes) {
			e.register = string(runes[e.cursor.Col])
			e.registerIsLine = false
		}
		e.buffer.DeleteRange(e.cursor.Line, e.cursor.Col, e.cursor.Line, e.cursor.Col+1)
		e.cursor.Clamp(e.buffer)
	}
}

// deleteVisualSelection deletes the visually selected lines.
func (e *Editor) deleteVisualSelection() {
	start := e.selStart
	end := e.selEnd
	if start > end {
		start, end = end, start
	}
	// Capture lines into register before deletion
	var lines []string
	for i := start; i <= end; i++ {
		lines = append(lines, e.buffer.Line(i))
	}
	e.register = strings.Join(lines, "\n")
	e.registerIsLine = true

	e.buffer.DeleteLines(start, end)
	e.cursor.Line = start
	e.cursor.Clamp(e.buffer)
}

// yankVisualSelection yanks the visually selected lines without deleting.
func (e *Editor) yankVisualSelection() {
	start := e.selStart
	end := e.selEnd
	if start > end {
		start, end = end, start
	}
	var lines []string
	for i := start; i <= end; i++ {
		lines = append(lines, e.buffer.Line(i))
	}
	e.register = strings.Join(lines, "\n")
	e.registerIsLine = true
}

// handleYankOperator handles y<motion> operations.
func (e *Editor) handleYankOperator(r rune) bool {
	switch r {
	case 'y':
		// yy = yank line
		e.register = e.buffer.Line(e.cursor.Line)
		e.registerIsLine = true
		return true
	case 'w':
		// yw = yank word (non-destructive)
		e.register = e.yankWord()
		e.registerIsLine = false
		return true
	case '$':
		// y$ = yank to end of line
		line := e.buffer.Line(e.cursor.Line)
		if e.cursor.Col < len(line) {
			e.register = line[e.cursor.Col:]
		}
		e.registerIsLine = false
		return true
	}
	return false
}

// yankWord returns the text from cursor to the start of the next word without modifying the buffer.
func (e *Editor) yankWord() string {
	line := e.buffer.Line(e.cursor.Line)
	col := e.cursor.Col
	endCol := col
	for endCol < len(line) && !unicode.IsSpace(rune(line[endCol])) {
		endCol++
	}
	for endCol < len(line) && unicode.IsSpace(rune(line[endCol])) {
		endCol++
	}
	return line[col:endCol]
}

// pasteAfter pastes register contents after/below the cursor.
func (e *Editor) pasteAfter() {
	if e.register == "" {
		return
	}
	if e.registerIsLine {
		e.buffer.InsertLineBelow(e.cursor.Line, e.register)
		e.cursor.Line++
		e.cursor.Col = 0
	} else {
		insertCol := e.cursor.Col + 1
		lineLen := e.buffer.LineLen(e.cursor.Line)
		if insertCol > lineLen {
			insertCol = lineLen
		}
		for _, r := range e.register {
			e.buffer.InsertRune(e.cursor.Line, insertCol, r)
			insertCol++
		}
		e.cursor.Col = insertCol - 1
		e.cursor.Clamp(e.buffer)
	}
}

// pasteBefore pastes register contents before/above the cursor.
func (e *Editor) pasteBefore() {
	if e.register == "" {
		return
	}
	if e.registerIsLine {
		e.buffer.InsertLineAbove(e.cursor.Line, e.register)
		e.cursor.Col = 0
	} else {
		insertCol := e.cursor.Col
		for _, r := range e.register {
			e.buffer.InsertRune(e.cursor.Line, insertCol, r)
			insertCol++
		}
		e.cursor.Col = insertCol - 1
		e.cursor.Clamp(e.buffer)
	}
}

// SetSize sets the editor dimensions.
func (e *Editor) SetSize(width, height int) {
	e.width = width
	e.height = height
}
