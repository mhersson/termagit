package refsview

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

// confirmMode indicates what type of confirmation is pending.
type confirmMode int

const (
	confirmNone         confirmMode = iota
	confirmDeleteBranch             // Delete a branch
)

// RefsSectionKind classifies a refs section.
type RefsSectionKind int

const (
	RefsSectionLocal  RefsSectionKind = iota
	RefsSectionRemote
	RefsSectionTags
)

// RefsSection represents a foldable section in the refs view.
type RefsSection struct {
	Title      string
	Kind       RefsSectionKind
	Items      []git.RefEntry
	Folded     bool
	RemoteName string // only for remote sections
	RemoteURL  string // only for remote sections
}

// flatRow represents a single navigable row in the flattened view.
type flatRow struct {
	sectionIdx int
	itemIdx    int  // -1 for section headers
	isHeader   bool
}

// Model is the refs view model.
type Model struct {
	repo   *git.Repository
	tokens theme.Tokens
	keys   KeyMap

	sections []RefsSection
	flatRows []flatRow
	cursor   int
	offset   int

	confirmMode confirmMode
	confirmRef  *git.RefEntry

	pendingKey string

	width  int
	height int
}

// New creates a new refs view model.
func New(refs *git.RefsResult, remotes []git.Remote, repo *git.Repository, tokens theme.Tokens) Model {
	sections := buildSections(refs, remotes)

	m := Model{
		repo:     repo,
		tokens:   tokens,
		keys:     DefaultKeyMap(),
		sections: sections,
	}
	m.rebuildFlatRows()
	return m
}

// buildSections creates the section list from refs data.
func buildSections(refs *git.RefsResult, remotes []git.Remote) []RefsSection {
	var sections []RefsSection

	// Local branches section
	sections = append(sections, RefsSection{
		Title: "Branches",
		Kind:  RefsSectionLocal,
		Items: refs.LocalBranches,
	})

	// Remote sections (one per remote, sorted by name)
	remoteURLs := make(map[string]string)
	for _, r := range remotes {
		remoteURLs[r.Name] = r.FetchURL
	}

	var remoteNames []string
	for name := range refs.RemoteBranches {
		remoteNames = append(remoteNames, name)
	}
	sort.Strings(remoteNames)

	for _, name := range remoteNames {
		branches := refs.RemoteBranches[name]
		sections = append(sections, RefsSection{
			Title:      fmt.Sprintf("Remote %s", name),
			Kind:       RefsSectionRemote,
			Items:      branches,
			RemoteName: name,
			RemoteURL:  remoteURLs[name],
		})
	}

	// Tags section
	sections = append(sections, RefsSection{
		Title: "Tags",
		Kind:  RefsSectionTags,
		Items: refs.Tags,
	})

	return sections
}

// rebuildFlatRows recomputes the navigable row list from current section state.
func (m *Model) rebuildFlatRows() {
	m.flatRows = nil
	for si, sec := range m.sections {
		// Section header
		m.flatRows = append(m.flatRows, flatRow{sectionIdx: si, itemIdx: -1, isHeader: true})
		if !sec.Folded {
			for ii := range sec.Items {
				m.flatRows = append(m.flatRows, flatRow{sectionIdx: si, itemIdx: ii})
			}
		}
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

	case DeleteBranchMsg:
		m.confirmMode = confirmNone
		m.confirmRef = nil
		if msg.Err != nil {
			return m, nil
		}
		// Refresh after successful deletion
		return m, m.refreshCmd()

	case RefsRefreshedMsg:
		if msg.Err == nil && m.repo != nil {
			ctx := context.Background()
			refs, err := m.repo.ListRefs(ctx)
			if err == nil {
				remotes, _ := m.repo.ListRemotes(ctx)
				m.sections = buildSections(refs, remotes)
				m.rebuildFlatRows()
				if m.cursor >= len(m.flatRows) {
					m.cursor = len(m.flatRows) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Handle confirmation mode
	if m.confirmMode != confirmNone {
		return m.handleConfirmKey(msg)
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
		return m, func() tea.Msg { return CloseRefsViewMsg{} }

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

	case key.Matches(msg, m.keys.Toggle):
		return m.toggleFold(), nil

	case key.Matches(msg, m.keys.Select):
		ref := m.currentRef()
		if ref != nil {
			return m, func() tea.Msg { return OpenCommitViewMsg{Hash: ref.Oid} }
		}
		return m, nil

	case key.Matches(msg, m.keys.DeleteBranch):
		ref := m.currentRef()
		if ref != nil && ref.Type == git.RefTypeLocalBranch {
			m.confirmMode = confirmDeleteBranch
			m.confirmRef = ref
		}
		return m, nil

	case key.Matches(msg, m.keys.Yank):
		ref := m.currentRef()
		if ref != nil {
			return m, yankCmd(ref.AbbrevOid)
		}
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshCmd()

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
	case key.Matches(msg, m.keys.PullPopup):
		return m, m.openPopupCmd("pull")
	case key.Matches(msg, m.keys.PushPopup):
		return m, m.openPopupCmd("push")
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
	}

	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		if m.confirmMode == confirmDeleteBranch && m.confirmRef != nil {
			return m, deleteBranchCmd(m.repo, m.confirmRef.UnambiguousName)
		}
		m.confirmMode = confirmNone
		m.confirmRef = nil
		return m, nil
	case "n", "esc":
		m.confirmMode = confirmNone
		m.confirmRef = nil
		return m, nil
	}
	return m, nil
}

// openPopupCmd returns a command that emits an OpenPopupMsg.
func (m Model) openPopupCmd(popupType string) tea.Cmd {
	hash := ""
	if ref := m.currentRef(); ref != nil {
		hash = ref.Oid
	}
	return func() tea.Msg {
		return OpenPopupMsg{Type: popupType, Commit: hash}
	}
}

// refreshCmd returns a command to reload refs.
func (m Model) refreshCmd() tea.Cmd {
	return refreshRefsCmd(m.repo)
}

// currentRef returns the ref under the cursor, or nil if on a header.
func (m Model) currentRef() *git.RefEntry {
	if m.cursor < 0 || m.cursor >= len(m.flatRows) {
		return nil
	}
	row := m.flatRows[m.cursor]
	if row.isHeader {
		return nil
	}
	sec := m.sections[row.sectionIdx]
	if row.itemIdx < 0 || row.itemIdx >= len(sec.Items) {
		return nil
	}
	return &sec.Items[row.itemIdx]
}

// toggleFold toggles the fold state of the section under the cursor.
func (m Model) toggleFold() Model {
	if m.cursor < 0 || m.cursor >= len(m.flatRows) {
		return m
	}
	row := m.flatRows[m.cursor]
	si := row.sectionIdx
	m.sections[si].Folded = !m.sections[si].Folded
	m.rebuildFlatRows()
	// Keep cursor in bounds
	if m.cursor >= len(m.flatRows) {
		m.cursor = len(m.flatRows) - 1
	}
	return m
}

func (m Model) moveDown(n int) Model {
	max := len(m.flatRows) - 1
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
	max := len(m.flatRows) - 1
	if max >= 0 {
		m.cursor = max
		m.ensureVisible()
	}
	return m
}

func (m Model) visibleLines() int {
	v := m.height
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

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// ConfirmMessage returns the confirmation message if pending, empty string otherwise.
func (m Model) ConfirmMessage() string {
	switch m.confirmMode {
	case confirmDeleteBranch:
		if m.confirmRef != nil {
			return fmt.Sprintf("Delete branch '%s'?", m.confirmRef.UnambiguousName)
		}
	}
	return ""
}
