package logview

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/graph"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/commitview"
	"github.com/mhersson/termagit/internal/ui/nav"
	"github.com/mhersson/termagit/internal/ui/shared"
)

// commitSearchEntry holds pre-lowercased fields for fast case-insensitive filtering.
type commitSearchEntry struct {
	subject string
	hash    string
	author  string
}

// displayRow represents a single rendered line in the log view.
type displayRow struct {
	commitIdx  int       // Index into m.commits (-1 for graph-only connector rows)
	graphCells graph.Row // Graph cells for this row (nil when graph disabled)
}

// KeyMap has view-specific keys only.
type KeyMap struct {
	LoadMore     key.Binding
	Filter       key.Binding
	ToggleDetail key.Binding
}

// Model is the log view model.
type Model struct {
	repo    *git.Repository
	tokens  theme.Tokens
	navKeys nav.NavigationKeys
	popKeys nav.PopupKeys
	keys    KeyMap
	opts    git.LogOpts
	header  string

	commits     []git.LogEntry
	searchCache []commitSearchEntry // pre-lowered fields for filtering
	cursor      nav.Cursor
	hasMore     bool
	loading     bool
	expanded    []bool // commit detail expansion state

	filterActive bool
	filterInput  textinput.Model
	filterText   string
	filtered     []int // indices into commits

	commitView *commitview.Model // overlay (nil = not showing)

	graphEnabled bool
	displayRows  []displayRow
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
	cache := buildSearchCache(commits)

	m := Model{
		repo:    repo,
		tokens:  tokens,
		navKeys: nav.DefaultNavigationKeys(),
		popKeys: nav.DefaultPopupKeys(),
		keys: KeyMap{
			LoadMore:     key.NewBinding(key.WithKeys("+"), key.WithHelp("+", "load more")),
			Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
			ToggleDetail: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle details")),
		},
		opts:         logOpts,
		commits:      commits,
		searchCache:  cache,
		hasMore:      hasMore,
		header:       "Commits on " + branch,
		expanded:     expanded,
		graphEnabled: logOpts.Graph,
		filterInput:  ti,
		cursor:       nav.NewCursor(3),
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
		m.cursor.SetSize(msg.Width, msg.Height)
		if m.commitView != nil {
			m.commitView.SetSize(msg.Width, msg.Height*70/100)
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case commitview.CommitDataLoadedMsg:
		if m.commitView != nil {
			cv := *m.commitView
			newCV, cmd := cv.Update(msg)
			cvModel := newCV.(commitview.Model)
			m.commitView = &cvModel
			return m, cmd
		}
		return m, nil

	case commitview.CloseCommitViewMsg:
		if m.commitView != nil {
			m.commitView = nil
		}
		return m, nil

	case CommitsLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			return m, nil
		}
		for _, c := range msg.Commits {
			m.commits = append(m.commits, git.LogEntry{
				Hash:            c.hash,
				AbbreviatedHash: c.abbrevHash,
				Subject:         c.subject,
				AuthorName:      c.authorName,
				ParentHashes:    c.parentHashes,
			})
			m.searchCache = append(m.searchCache, commitSearchEntry{
				subject: strings.ToLower(c.subject),
				hash:    strings.ToLower(c.abbrevHash),
				author:  strings.ToLower(c.authorName),
			})
			m.expanded = append(m.expanded, false)
		}
		m.hasMore = msg.HasMore
		m.computeDisplayRows()
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.commitView != nil {
		return m.handleCommitViewKey(msg)
	}

	if m.filterActive {
		return m.handleFilterKey(msg)
	}

	if m.cursor.HandleGG(msg.String()) {
		return m, nil
	}

	// Navigation (custom due to graph row skipping)
	max := m.maxCursor()
	switch {
	case key.Matches(msg, m.navKeys.MoveDown):
		return m.moveDown(1), nil
	case key.Matches(msg, m.navKeys.MoveUp):
		return m.moveUp(1), nil
	case key.Matches(msg, m.navKeys.PageDown):
		return m.moveDown(m.cursor.VisibleLines()), nil
	case key.Matches(msg, m.navKeys.PageUp):
		return m.moveUp(m.cursor.VisibleLines()), nil
	case key.Matches(msg, m.navKeys.HalfPageDown):
		return m.moveDown(m.cursor.VisibleLines() / 2), nil
	case key.Matches(msg, m.navKeys.HalfPageUp):
		return m.moveUp(m.cursor.VisibleLines() / 2), nil
	case key.Matches(msg, m.navKeys.GoToTop):
		m.cursor.PendingKey = "g"
		return m, nil
	case key.Matches(msg, m.navKeys.GoToBottom):
		m.cursor.GoToBottom(max)
		return m, nil
	}

	// Popup triggers
	if handled, cmd := nav.HandlePopupKey(msg, m.popKeys, m.currentHash()); handled {
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.navKeys.Close), key.Matches(msg, m.navKeys.CloseEscape):
		return m, func() tea.Msg { return CloseLogViewMsg{} }

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

	case key.Matches(msg, m.navKeys.Yank):
		if len(m.commits) > 0 && m.cursor.Pos < len(m.commits) {
			hash := m.commits[m.cursor.Pos].AbbreviatedHash
			return m, shared.YankCmd(hash)
		}
		return m, nil

	case key.Matches(msg, m.keys.ToggleDetail):
		if m.cursor.Pos < len(m.expanded) {
			idx := m.cursor.Pos
			if len(m.filtered) > 0 && m.cursor.Pos < len(m.filtered) {
				idx = m.filtered[m.cursor.Pos]
			}
			if idx < len(m.expanded) {
				m.expanded[idx] = !m.expanded[idx]
				if m.expanded[idx] && m.cursor.Pos == m.cursor.Offset {
					if m.cursor.Offset > 0 {
						m.cursor.Offset--
					}
				}
			}
		}
		return m, nil

	case key.Matches(msg, m.navKeys.Select):
		if len(m.commits) > 0 {
			idx := m.cursor.Pos
			if len(m.filtered) > 0 && m.cursor.Pos < len(m.filtered) {
				idx = m.filtered[m.cursor.Pos]
			}
			if idx < len(m.commits) {
				hash := m.commits[idx].Hash
				cv := commitview.New(m.repo, hash, m.tokens, nil)
				cv.SetSize(m.cursor.Width, m.cursor.Height*70/100)
				cv.SetOverlayMode(true)
				m.commitView = &cv
				return m, cv.Init()
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) currentHash() string {
	if len(m.commits) == 0 {
		return ""
	}
	idx := m.cursor.Pos
	if len(m.filtered) > 0 && m.cursor.Pos < len(m.filtered) {
		idx = m.filtered[m.cursor.Pos]
	}
	if idx < len(m.commits) {
		return m.commits[idx].Hash
	}
	return ""
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
		return m, nil
	}
	return m, cmd
}

// buildSearchCache creates pre-lowercased entries for fast case-insensitive filtering.
func buildSearchCache(commits []git.LogEntry) []commitSearchEntry {
	cache := make([]commitSearchEntry, len(commits))
	for i, c := range commits {
		cache[i] = commitSearchEntry{
			subject: strings.ToLower(c.Subject),
			hash:    strings.ToLower(c.AbbreviatedHash),
			author:  strings.ToLower(c.AuthorName),
		}
	}
	return cache
}

func (m *Model) applyFilter() {
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

	for i := range m.commits {
		if i < len(m.searchCache) &&
			(strings.Contains(m.searchCache[i].subject, filter) ||
				strings.Contains(m.searchCache[i].hash, filter) ||
				strings.Contains(m.searchCache[i].author, filter)) {
			m.filtered = append(m.filtered, i)
		}
	}

	if len(m.filtered) > 0 {
		m.cursor.Pos = 0
	}
}

// computeDisplayRows builds the display row list from commits and graph data.
func (m *Model) computeDisplayRows() {
	if !m.graphEnabled || len(m.commits) == 0 {
		m.displayRows = make([]displayRow, len(m.commits))
		for i := range m.commits {
			m.displayRows[i] = displayRow{commitIdx: i}
		}
		return
	}

	inputs := make([]graph.CommitInput, len(m.commits))
	for i, c := range m.commits {
		var parents []string
		if c.ParentHashes != "" {
			parents = strings.Fields(c.ParentHashes)
		}
		inputs[i] = graph.CommitInput{OID: c.Hash, Parents: parents}
	}

	graphRows, err := graph.Build(inputs)
	if err != nil {
		// Render a placeholder row instead of crashing
		m.displayRows = []displayRow{{
			commitIdx:  -1,
			graphCells: []graph.Cell{{Text: "[graph error: " + err.Error() + "]"}},
		}}
		return
	}

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

// Graph-aware moveDown: skips connector rows.
func (m Model) moveDown(n int) Model {
	max := m.maxCursor()
	if max < 0 {
		return m
	}

	for range n {
		next := m.cursor.Pos + 1
		for next <= max && m.graphEnabled && len(m.displayRows) > 0 && m.displayRows[next].commitIdx < 0 {
			next++
		}
		if next > max {
			break
		}
		m.cursor.Pos = next
	}
	m.cursor.EnsureVisible()
	return m
}

// Graph-aware moveUp: skips connector rows.
func (m Model) moveUp(n int) Model {
	for range n {
		next := m.cursor.Pos - 1
		for next >= 0 && m.graphEnabled && len(m.displayRows) > 0 && m.displayRows[next].commitIdx < 0 {
			next--
		}
		if next < 0 {
			break
		}
		m.cursor.Pos = next
	}
	m.cursor.EnsureVisible()
	return m
}

func (m Model) maxCursor() int {
	if len(m.filtered) > 0 {
		return len(m.filtered) - 1
	}
	if m.graphEnabled && len(m.displayRows) > 0 {
		for i := len(m.displayRows) - 1; i >= 0; i-- {
			if m.displayRows[i].commitIdx >= 0 {
				return i
			}
		}
		return -1
	}
	return len(m.commits) - 1
}

func (m Model) loadMoreCmd() tea.Cmd {
	return func() tea.Msg {
		return LoadMoreMsg{}
	}
}

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.cursor.SetSize(width, height)
}
