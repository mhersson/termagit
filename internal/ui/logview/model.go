package logview

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/graph"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/commitview"
)

// displayRow represents a single rendered line in the log view.
type displayRow struct {
	commitIdx  int          // Index into m.commits (-1 for graph-only connector rows)
	graphCells graph.Row    // Graph cells for this row (nil when graph disabled)
}

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

	graphEnabled bool
	displayRows  []displayRow

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

	m := Model{
		repo:         repo,
		tokens:       tokens,
		keys:         DefaultKeyMap(),
		opts:         logOpts,
		commits:      commits,
		hasMore:      hasMore,
		header:       "Commits on " + branch,
		expanded:     expanded,
		graphEnabled: logOpts.Graph,
		filterInput:  ti,
	}
	m.computeDisplayRows()
	return m
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
				ParentHashes:    c.parentHashes,
			})
			m.expanded = append(m.expanded, false)
		}
		m.hasMore = msg.HasMore
		// Recompute display rows (graph needs all commits for correct topology)
		m.computeDisplayRows()
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

	// Popup triggers
	case key.Matches(msg, m.keys.CherryPickPopup):
		return m, m.openPopupCmd("cherry-pick")
	case key.Matches(msg, m.keys.BranchPopup):
		return m, m.openPopupCmd("branch")
	case key.Matches(msg, m.keys.CommitPopup):
		return m, m.openPopupCmd("commit")
	case key.Matches(msg, m.keys.DiffPopup):
		return m, m.openPopupCmd("diff")
	case key.Matches(msg, m.keys.FetchPopup):
		return m, m.openPopupCmd("fetch")
	case key.Matches(msg, m.keys.MergePopup):
		return m, m.openPopupCmd("merge")
	case key.Matches(msg, m.keys.PullPopup):
		return m, m.openPopupCmd("pull")
	case key.Matches(msg, m.keys.RebasePopup):
		return m, m.openPopupCmd("rebase")
	case key.Matches(msg, m.keys.RevertPopup):
		return m, m.openPopupCmd("revert")
	case key.Matches(msg, m.keys.ResetPopup):
		return m, m.openPopupCmd("reset")
	case key.Matches(msg, m.keys.TagPopup):
		return m, m.openPopupCmd("tag")
	case key.Matches(msg, m.keys.BisectPopup):
		return m, m.openPopupCmd("bisect")
	case key.Matches(msg, m.keys.RemotePopup):
		return m, m.openPopupCmd("remote")
	case key.Matches(msg, m.keys.WorktreePopup):
		return m, m.openPopupCmd("worktree")
	case key.Matches(msg, m.keys.OpenCommitLink):
		return m, m.openCommitLinkCmd()
	}

	return m, nil
}

// openPopupCmd returns a command that emits an OpenPopupMsg for the given popup type.
func (m Model) openPopupCmd(popupType string) tea.Cmd {
	hash := ""
	if len(m.commits) > 0 {
		idx := m.cursor
		if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
			idx = m.filtered[m.cursor]
		}
		if idx < len(m.commits) {
			hash = m.commits[idx].Hash
		}
	}
	return func() tea.Msg {
		return OpenPopupMsg{Type: popupType, Commit: hash}
	}
}

// openCommitLinkCmd returns a command to open the commit URL in a browser.
func (m Model) openCommitLinkCmd() tea.Cmd {
	if len(m.commits) == 0 {
		return nil
	}
	idx := m.cursor
	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		idx = m.filtered[m.cursor]
	}
	if idx >= len(m.commits) {
		return nil
	}
	hash := m.commits[idx].Hash
	return func() tea.Msg {
		return OpenCommitLinkMsg{Hash: hash}
	}
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

// computeDisplayRows builds the display row list from commits and graph data.
func (m *Model) computeDisplayRows() {
	if !m.graphEnabled || len(m.commits) == 0 {
		// No graph — one display row per commit
		m.displayRows = make([]displayRow, len(m.commits))
		for i := range m.commits {
			m.displayRows[i] = displayRow{commitIdx: i}
		}
		return
	}

	// Convert commits to graph input
	inputs := make([]graph.CommitInput, len(m.commits))
	for i, c := range m.commits {
		var parents []string
		if c.ParentHashes != "" {
			parents = strings.Fields(c.ParentHashes)
		}
		inputs[i] = graph.CommitInput{OID: c.Hash, Parents: parents}
	}

	graphRows := graph.Build(inputs)

	// Build display rows by matching graph rows to commits via OID
	// Build a map from OID -> commit index for lookup
	oidToIdx := make(map[string]int, len(m.commits))
	for i, c := range m.commits {
		oidToIdx[c.Hash] = i
	}

	m.displayRows = make([]displayRow, 0, len(graphRows))
	for _, gr := range graphRows {
		dr := displayRow{
			commitIdx:  -1,
			graphCells: gr,
		}
		if len(gr) > 0 && gr[0].OID != "" {
			if idx, ok := oidToIdx[gr[0].OID]; ok {
				dr.commitIdx = idx
			}
		}
		m.displayRows = append(m.displayRows, dr)
	}
}

func (m Model) moveDown(n int) Model {
	max := m.maxCursor()
	if max < 0 {
		return m
	}

	for i := 0; i < n; i++ {
		next := m.cursor + 1
		// Skip connector rows (commitIdx == -1)
		for next <= max && m.graphEnabled && len(m.displayRows) > 0 && m.displayRows[next].commitIdx < 0 {
			next++
		}
		if next > max {
			break
		}
		m.cursor = next
	}
	m.ensureVisible()
	return m
}

func (m Model) moveUp(n int) Model {
	for i := 0; i < n; i++ {
		next := m.cursor - 1
		// Skip connector rows (commitIdx == -1)
		for next >= 0 && m.graphEnabled && len(m.displayRows) > 0 && m.displayRows[next].commitIdx < 0 {
			next--
		}
		if next < 0 {
			break
		}
		m.cursor = next
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
	if m.graphEnabled && len(m.displayRows) > 0 {
		// Find the last display row with a commit
		for i := len(m.displayRows) - 1; i >= 0; i-- {
			if m.displayRows[i].commitIdx >= 0 {
				return i
			}
		}
		return -1
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
