package logview

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/commitview"
)

// Model is the log view model.
type Model struct {
	repo    *git.Repository
	tokens  theme.Tokens
	keys    KeyMap
	opts    git.LogOpts
	remotes []string
	header  string

	commits  []git.LogEntry
	cursor   int
	offset   int // scroll offset
	hasMore  bool
	loading  bool
	expanded []bool // commit detail expansion state

	filterActive bool
	filterInput  textinput.Model
	filterText   string
	filtered     []int // indices into commits

	pendingKey string // for "gg" sequence

	commitView *commitview.Model // overlay (nil = not showing)

	width  int
	height int
}

// New creates a new log view model.
func New(commits []git.LogEntry, repo *git.Repository, tokens theme.Tokens, opts *git.LogOpts, hasMore bool, branch string) Model {
	ti := textinput.New()
	ti.Placeholder = "Filter..."
	ti.CharLimit = 100

	var logOpts git.LogOpts
	if opts != nil {
		logOpts = *opts
	}

	expanded := make([]bool, len(commits))

	return Model{
		repo:     repo,
		tokens:   tokens,
		keys:     DefaultKeyMap(),
		opts:     logOpts,
		commits:  commits,
		hasMore:  hasMore,
		header:   "Commits on " + branch,
		expanded: expanded,

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
		// Update commit view size if active (70% of screen)
		if m.commitView != nil {
			m.commitView.SetSize(m.width, m.height*70/100)
		}
		return m, nil

	case tea.KeyMsg:
		newM, cmd := m.handleKey(msg)
		return newM, cmd

	case commitview.CommitDataLoadedMsg:
		// Forward to commit view if active
		if m.commitView != nil {
			cv := *m.commitView
			newCV, cmd := cv.Update(msg)
			cvModel := newCV.(commitview.Model)
			m.commitView = &cvModel
			return m, cmd
		}
		return m, nil

	case commitview.CloseCommitViewMsg:
		// Handle close from the overlay commit view - don't bubble up to app
		if m.commitView != nil {
			m.commitView = nil
		}
		return m, nil

	case CommitsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, nil
		}
		// Append new commits
		for _, c := range msg.Commits {
			m.commits = append(m.commits, git.LogEntry{
				Hash:            c.hash,
				AbbreviatedHash: c.abbrevHash,
				Subject:         c.subject,
				AuthorName:      c.authorName,
			})
			m.expanded = append(m.expanded, false)
		}
		m.hasMore = msg.HasMore
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Delegate to commit view if active
	if m.commitView != nil {
		return m.handleCommitViewKey(msg)
	}

	// Handle filter mode
	if m.filterActive {
		return m.handleFilterKey(msg)
	}

	// Handle "gg" sequence
	if m.pendingKey == "g" {
		m.pendingKey = ""
		if msg.String() == "g" {
			m.cursor = 0
			m.offset = 0
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Close), key.Matches(msg, m.keys.CloseEscape):
		return m, func() tea.Msg { return CloseLogViewMsg{} }

	case key.Matches(msg, m.keys.MoveDown):
		return m.moveDown(1), nil

	case key.Matches(msg, m.keys.MoveUp):
		return m.moveUp(1), nil

	case key.Matches(msg, m.keys.PageDown):
		return m.moveDown(m.visibleLines()), nil

	case key.Matches(msg, m.keys.PageUp):
		return m.moveUp(m.visibleLines()), nil

	case key.Matches(msg, m.keys.HalfPageDown):
		return m.moveDown(m.visibleLines() / 2), nil

	case key.Matches(msg, m.keys.HalfPageUp):
		return m.moveUp(m.visibleLines() / 2), nil

	case key.Matches(msg, m.keys.GoToTop):
		m.pendingKey = "g"
		return m, nil

	case key.Matches(msg, m.keys.GoToBottom):
		return m.goToBottom(), nil

	case key.Matches(msg, m.keys.LoadMore):
		if m.hasMore && !m.loading {
			m.loading = true
			return m, m.loadMoreCmd()
		}
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		m.filterActive = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Yank):
		if len(m.commits) > 0 && m.cursor < len(m.commits) {
			hash := m.commits[m.cursor].AbbreviatedHash
			return m, yankCmd(hash)
		}
		return m, nil

	case key.Matches(msg, m.keys.ToggleDetail):
		if m.cursor < len(m.expanded) {
			idx := m.cursor
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				idx = m.filtered[m.cursor]
			}
			if idx < len(m.expanded) {
				m.expanded[idx] = !m.expanded[idx]
				// If expanding, scroll down to show details
				if m.expanded[idx] && m.cursor == m.offset {
					// Cursor is at top of view, scroll down slightly to show details
					if m.offset > 0 {
						m.offset--
					}
				}
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		// Open commit view as overlay for the selected commit
		if len(m.commits) > 0 {
			idx := m.cursor
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				idx = m.filtered[m.cursor]
			}
			if idx < len(m.commits) {
				hash := m.commits[idx].Hash
				cv := commitview.New(m.repo, hash, m.tokens, nil)
				cv.SetSize(m.width, m.height*70/100)
				cv.SetOverlayMode(true)
				m.commitView = &cv
				return m, cv.Init()
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
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

func (m Model) handleCommitViewKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	cv := *m.commitView
	newCV, cmd := cv.Update(msg)
	cvModel := newCV.(commitview.Model)
	m.commitView = &cvModel

	if cvModel.Done() {
		m.commitView = nil
		return m, nil // Don't bubble CloseCommitViewMsg
	}
	return m, cmd
}

func (m *Model) applyFilter() {
	// Use filterInput value if filterText not set (for testing)
	text := m.filterText
	if text == "" {
		text = m.filterInput.Value()
	}

	if text == "" {
		m.filtered = nil
		return
	}

	filter := strings.ToLower(text)
	m.filtered = nil

	for i, c := range m.commits {
		if strings.Contains(strings.ToLower(c.Subject), filter) ||
			strings.Contains(strings.ToLower(c.AbbreviatedHash), filter) ||
			strings.Contains(strings.ToLower(c.AuthorName), filter) {
			m.filtered = append(m.filtered, i)
		}
	}

	// Reset cursor to first filtered result
	if len(m.filtered) > 0 {
		m.cursor = 0
	}
}

func (m Model) moveDown(n int) Model {
	max := m.maxCursor()
	if max < 0 {
		return m
	}

	m.cursor += n
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

func (m Model) goToBottom() Model {
	max := m.maxCursor()
	if max >= 0 {
		m.cursor = max
		m.ensureVisible()
	}
	return m
}

func (m Model) maxCursor() int {
	if len(m.filtered) > 0 {
		return len(m.filtered) - 1
	}
	return len(m.commits) - 1
}

func (m Model) visibleLines() int {
	// Reserve 3 lines for header + hint row + filter
	v := m.height - 3
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

func (m Model) loadMoreCmd() tea.Cmd {
	// Will be implemented when wiring to app
	return func() tea.Msg {
		return LoadMoreMsg{}
	}
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetRemotes sets the remote names for ref parsing.
func (m *Model) SetRemotes(remotes []string) {
	m.remotes = remotes
}
