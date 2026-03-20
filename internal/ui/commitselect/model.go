package commitselect

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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
	Hash     string   // abbreviated commit hash (backward compat)
	FullHash string   // full 40-char commit hash (backward compat)
	Subject  string   // commit subject line (backward compat)
	Hashes   []string // all selected full hashes (for multi-select)
	Subjects []string // all selected subjects (for multi-select)
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

	// Multi-select / visual mode
	visualMode  bool
	visualStart int
	selected    []bool
	multiSelect bool

	// Filter
	filterActive bool
	filterInput  textinput.Model
	filterText   string
	filtered     []int // indices into commits
}

// New creates a new commit select model.
func New(commits []git.LogEntry, tokens theme.Tokens, width, height int) Model {
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.CharLimit = 100

	return Model{
		commits:     commits,
		tokens:      tokens,
		width:       width,
		height:      height,
		selected:    make([]bool, len(commits)),
		filterInput: ti,
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
	// Handle filter mode first
	if m.filterActive {
		return m.handleFilterKey(msg)
	}

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
	case tea.KeySpace:
		return m.toggleSelection()
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			return m.moveDown(1), nil
		case "k":
			return m.moveUp(1), nil
		case "q":
			return m.abort()
		case "v":
			return m.toggleVisualMode()
		case "/":
			m.filterActive = true
			m.filterInput.Focus()
			return m, textinput.Blink
		case "y":
			return m.yankSelected()
		}
	}
	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.filterActive = false
		m.filterText = ""
		m.filtered = nil
		m.filterInput.SetValue("")
		return m, nil
	case tea.KeyEnter:
		m.filterActive = false
		m.filterText = m.filterInput.Value()
		m.applyFilter()
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	return m, cmd
}

func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filtered = nil
		return
	}

	filter := strings.ToLower(m.filterText)
	m.filtered = nil

	for i, c := range m.commits {
		if strings.Contains(strings.ToLower(c.Subject), filter) ||
			strings.Contains(strings.ToLower(c.AbbreviatedHash), filter) ||
			strings.Contains(strings.ToLower(c.AuthorName), filter) {
			m.filtered = append(m.filtered, i)
		}
	}

	// Reset cursor
	if len(m.filtered) > 0 {
		m.cursor = 0
	}
}

func (m Model) toggleVisualMode() (tea.Model, tea.Cmd) {
	m.visualMode = !m.visualMode
	if m.visualMode {
		m.visualStart = m.cursor
	}
	return m, nil
}

func (m Model) toggleSelection() (tea.Model, tea.Cmd) {
	if !m.multiSelect {
		m.multiSelect = true
	}

	idx := m.cursor
	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		idx = m.filtered[m.cursor]
	}

	if idx < len(m.selected) {
		m.selected[idx] = !m.selected[idx]
	}

	// Move down after toggle
	m = m.moveDown(1)
	return m, nil
}

func (m Model) yankSelected() (tea.Model, tea.Cmd) {
	var hashes []string
	for i, sel := range m.selected {
		if sel {
			hashes = append(hashes, m.commits[i].Hash)
		}
	}
	if len(hashes) == 0 && m.cursor < len(m.commits) {
		hashes = []string{m.commits[m.cursor].Hash}
	}

	// Return yank command (handled by app)
	hashStr := strings.Join(hashes, "\n")
	return m, func() tea.Msg {
		// OSC 52 clipboard would be handled at app level
		return YankMsg{Text: hashStr}
	}
}

// YankMsg is sent when hashes should be copied to clipboard.
type YankMsg struct {
	Text string
}

func (m Model) selectCurrent() (tea.Model, tea.Cmd) {
	m.done = true
	if len(m.commits) == 0 {
		m.aborted = true
		return m, func() tea.Msg { return AbortedMsg{} }
	}

	// Collect selected commits
	var hashes []string
	var subjects []string

	// If in visual mode, select the range
	if m.visualMode {
		start, end := m.visualStart, m.cursor
		if start > end {
			start, end = end, start
		}
		for i := start; i <= end; i++ {
			idx := i
			if len(m.filtered) > 0 && i < len(m.filtered) {
				idx = m.filtered[i]
			}
			if idx < len(m.commits) {
				hashes = append(hashes, m.commits[idx].Hash)
				subjects = append(subjects, m.commits[idx].Subject)
			}
		}
	} else if m.multiSelect {
		// Use selected array
		for i, sel := range m.selected {
			if sel {
				hashes = append(hashes, m.commits[i].Hash)
				subjects = append(subjects, m.commits[i].Subject)
			}
		}
	}

	// Fallback to single selection
	if len(hashes) == 0 {
		idx := m.cursor
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			idx = m.filtered[m.cursor]
		}
		if idx < len(m.commits) {
			hashes = []string{m.commits[idx].Hash}
			subjects = []string{m.commits[idx].Subject}
		}
	}

	entry := m.commits[m.cursor]
	return m, func() tea.Msg {
		return SelectedMsg{
			Hash:     entry.AbbreviatedHash,
			FullHash: entry.Hash,
			Subject:  entry.Subject,
			Hashes:   hashes,
			Subjects: subjects,
		}
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
