package branchselect

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

// OpenBranchSelectMsg triggers opening the branch select view at the app level.
type OpenBranchSelectMsg struct {
	Branches []git.Branch
}

// SelectedMsg is sent when the user selects a branch.
type SelectedMsg struct {
	Name string
}

// AbortedMsg is sent when the user aborts the selection.
type AbortedMsg struct{}

// Model is the branch select view for picking a branch.
type Model struct {
	branches []git.Branch
	tokens   theme.Tokens
	cursor   int
	offset   int // scroll offset for viewport
	width    int
	height   int
	done     bool
	aborted  bool

	// Filter
	filterActive bool
	filterInput  textinput.Model
	filterText   string
	filtered     []int // indices into branches
}

// New creates a new branch select model.
func New(branches []git.Branch, tokens theme.Tokens, width, height int) Model {
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.CharLimit = 100

	return Model{
		branches:    branches,
		tokens:      tokens,
		width:       width,
		height:      height,
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
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "j":
			return m.moveDown(1), nil
		case "k":
			return m.moveUp(1), nil
		case "q":
			return m.abort()
		case "/":
			m.filterActive = true
			m.filterInput.Focus()
			return m, textinput.Blink
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

	for i, b := range m.branches {
		if strings.Contains(strings.ToLower(b.Name), filter) {
			m.filtered = append(m.filtered, i)
		}
	}

	if len(m.filtered) > 0 {
		m.cursor = 0
		m.offset = 0
	}
}

func (m Model) selectCurrent() (tea.Model, tea.Cmd) {
	m.done = true
	items := m.visibleBranches()
	if len(items) == 0 {
		m.aborted = true
		return m, func() tea.Msg { return AbortedMsg{} }
	}

	idx := m.cursor
	if idx >= len(items) {
		idx = len(items) - 1
	}

	name := items[idx].Name
	return m, func() tea.Msg {
		return SelectedMsg{Name: name}
	}
}

func (m Model) abort() (tea.Model, tea.Cmd) {
	m.done = true
	m.aborted = true
	return m, func() tea.Msg { return AbortedMsg{} }
}

func (m Model) visibleBranches() []git.Branch {
	if len(m.filtered) > 0 {
		result := make([]git.Branch, len(m.filtered))
		for i, idx := range m.filtered {
			result[i] = m.branches[idx]
		}
		return result
	}
	return m.branches
}

func (m Model) moveDown(n int) Model {
	items := m.visibleBranches()
	if len(items) == 0 {
		return m
	}
	m.cursor += n
	max := len(items) - 1
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

	b.WriteString(m.tokens.SubtleText.Render("Select a branch with <cr>, or <esc> to abort"))
	b.WriteString("\n\n")

	items := m.visibleBranches()
	vis := m.visibleLines()
	end := m.offset + vis
	if end > len(items) {
		end = len(items)
	}

	for i := m.offset; i < end; i++ {
		branch := items[i]
		marker := "  "
		if branch.IsCurrent {
			marker = "* "
		}

		if i == m.cursor {
			line := marker + branch.Name
			b.WriteString(m.tokens.Cursor.Render(line))
		} else {
			b.WriteString(marker)
			if branch.IsCurrent {
				b.WriteString(m.tokens.Branch.Render(branch.Name))
			} else if branch.IsRemote {
				b.WriteString(m.tokens.Remote.Render(branch.Name))
			} else {
				b.WriteString(branch.Name)
			}
		}
		b.WriteString("\n")
	}

	if m.filterActive {
		b.WriteString("\n")
		b.WriteString("/" + m.filterInput.View())
	} else if m.filterText != "" {
		b.WriteString("\n")
		b.WriteString(m.tokens.SubtleText.Render("filter: " + m.filterText))
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
