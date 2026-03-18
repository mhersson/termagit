package commitselect

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

// OpenCommitSelectMsg triggers opening the commit select view at the app level.
type OpenCommitSelectMsg struct {
	Commits []git.LogEntry
}

// SelectedMsg is sent when the user selects a commit.
type SelectedMsg struct {
	Hash     string // abbreviated commit hash
	FullHash string // full 40-char commit hash
	Subject  string // commit subject line
}

// AbortedMsg is sent when the user aborts the selection.
type AbortedMsg struct{}

// Model is the commit select view for picking a target commit.
type Model struct {
	commits []git.LogEntry
	tokens  theme.Tokens
	cursor  int
	offset  int // scroll offset for viewport
	width   int
	height  int
	done    bool
	aborted bool
}

// New creates a new commit select model.
func New(commits []git.LogEntry, tokens theme.Tokens, width, height int) Model {
	return Model{
		commits: commits,
		tokens:  tokens,
		width:   width,
		height:  height,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.selectCurrent()
	case tea.KeyEscape:
		return m.abort()
	case tea.KeyUp:
		return m.moveUp(1), nil
	case tea.KeyDown:
		return m.moveDown(1), nil
	case tea.KeyCtrlD:
		return m.moveDown(m.visibleLines() / 2), nil
	case tea.KeyCtrlU:
		return m.moveUp(m.visibleLines() / 2), nil
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			return m.moveDown(1), nil
		case "k":
			return m.moveUp(1), nil
		case "q":
			return m.abort()
		}
	}
	return m, nil
}

func (m Model) selectCurrent() (tea.Model, tea.Cmd) {
	m.done = true
	if len(m.commits) == 0 {
		m.aborted = true
		return m, func() tea.Msg { return AbortedMsg{} }
	}
	entry := m.commits[m.cursor]
	return m, func() tea.Msg {
		return SelectedMsg{Hash: entry.AbbreviatedHash, FullHash: entry.Hash, Subject: entry.Subject}
	}
}

func (m Model) abort() (tea.Model, tea.Cmd) {
	m.done = true
	m.aborted = true
	return m, func() tea.Msg { return AbortedMsg{} }
}

func (m Model) moveDown(n int) Model {
	if len(m.commits) == 0 {
		return m
	}
	m.cursor += n
	max := len(m.commits) - 1
	if m.cursor > max {
		m.cursor = max
	}
	m.ensureVisible()
	return m
}

func (m Model) moveUp(n int) Model {
	m.cursor -= n
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
	return m
}

// visibleLines returns how many commit lines fit in the viewport.
// Reserve 2 lines for header + blank line.
func (m Model) visibleLines() int {
	v := m.height - 2
	if v < 1 {
		return 1
	}
	return v
}

func (m *Model) ensureVisible() {
	vis := m.visibleLines()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(m.tokens.SubtleText.Render("Select a commit with <cr>, or <esc> to abort"))
	b.WriteString("\n\n")

	vis := m.visibleLines()
	end := m.offset + vis
	if end > len(m.commits) {
		end = len(m.commits)
	}

	for i := m.offset; i < end; i++ {
		c := m.commits[i]
		if i == m.cursor {
			// Cursor line: full line with cursor styling
			line := fmt.Sprintf("  %s %s", c.AbbreviatedHash, c.Subject)
			b.WriteString(m.tokens.Cursor.Render(line))
		} else {
			// Normal line: styled hash + plain subject
			b.WriteString("  ")
			b.WriteString(m.tokens.Hash.Render(c.AbbreviatedHash))
			b.WriteString(" ")
			b.WriteString(c.Subject)
		}
		b.WriteString("\n")
	}

	return b.String()
}

// Done returns whether the selection is complete.
func (m Model) Done() bool {
	return m.done
}

// Aborted returns whether the user aborted.
func (m Model) Aborted() bool {
	return m.aborted
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
