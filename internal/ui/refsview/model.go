package refsview

import (
	"context"
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/nav"
	"github.com/mhersson/termagit/internal/ui/shared"
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
	RefsSectionLocal RefsSectionKind = iota
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
	itemIdx    int // -1 for section headers
	isHeader   bool
}

// Model is the refs view model.
type Model struct {
	repo   *git.Repository
	tokens theme.Tokens

	navKeys nav.NavigationKeys
	popKeys nav.PopupKeys

	toggleKey       key.Binding
	deleteBranchKey key.Binding
	refreshKey      key.Binding

	sections []RefsSection
	flatRows []flatRow
	cursor   nav.Cursor

	confirmMode confirmMode
	confirmRef  *git.RefEntry
}

// New creates a new refs view model.
func New(refs *git.RefsResult, remotes []git.Remote, repo *git.Repository, tokens theme.Tokens) Model {
	sections := buildSections(refs, remotes)

	m := Model{
		repo:    repo,
		tokens:  tokens,
		navKeys: nav.DefaultNavigationKeys(),
		popKeys: nav.DefaultPopupKeys(),
		toggleKey: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "fold/unfold"),
		),
		deleteBranchKey: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete branch"),
		),
		refreshKey: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("C-r", "refresh"),
		),
		sections: sections,
		cursor:   nav.NewCursor(0),
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
		m.cursor.SetSize(msg.Width, msg.Height)
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
				if m.cursor.Pos >= len(m.flatRows) {
					m.cursor.Pos = len(m.flatRows) - 1
				}
				if m.cursor.Pos < 0 {
					m.cursor.Pos = 0
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
	if m.cursor.HandleGG(msg.String()) {
		return m, nil
	}

	max := len(m.flatRows) - 1

	if handled, cmd := nav.HandleNavigationKey(msg, &m.cursor, m.navKeys, max); handled {
		return m, cmd
	}

	if handled, cmd := nav.HandlePopupKey(msg, m.popKeys, m.currentRefHash()); handled {
		return m, cmd
	}

	switch {
	case key.Matches(msg, m.navKeys.Close), key.Matches(msg, m.navKeys.CloseEscape):
		return m, func() tea.Msg { return CloseRefsViewMsg{} }

	case key.Matches(msg, m.toggleKey):
		return m.toggleFold(), nil

	case key.Matches(msg, m.navKeys.Select):
		ref := m.currentRef()
		if ref != nil {
			return m, func() tea.Msg { return shared.OpenCommitViewMsg{Hash: ref.Oid} }
		}
		return m, nil

	case key.Matches(msg, m.deleteBranchKey):
		ref := m.currentRef()
		if ref != nil && ref.Type == git.RefTypeLocalBranch {
			m.confirmMode = confirmDeleteBranch
			m.confirmRef = ref
		}
		return m, nil

	case key.Matches(msg, m.navKeys.Yank):
		ref := m.currentRef()
		if ref != nil {
			return m, shared.YankCmd(ref.AbbrevOid)
		}
		return m, nil

	case key.Matches(msg, m.refreshKey):
		return m, m.refreshCmd()
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

func (m Model) currentRefHash() string {
	if ref := m.currentRef(); ref != nil {
		return ref.Oid
	}
	return ""
}

// refreshCmd returns a command to reload refs.
func (m Model) refreshCmd() tea.Cmd {
	return refreshRefsCmd(m.repo)
}

// currentRef returns the ref under the cursor, or nil if on a header.
func (m Model) currentRef() *git.RefEntry {
	if m.cursor.Pos < 0 || m.cursor.Pos >= len(m.flatRows) {
		return nil
	}
	row := m.flatRows[m.cursor.Pos]
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
	if m.cursor.Pos < 0 || m.cursor.Pos >= len(m.flatRows) {
		return m
	}
	row := m.flatRows[m.cursor.Pos]
	si := row.sectionIdx
	m.sections[si].Folded = !m.sections[si].Folded
	m.rebuildFlatRows()
	// Keep cursor in bounds
	if m.cursor.Pos >= len(m.flatRows) {
		m.cursor.Pos = len(m.flatRows) - 1
	}
	return m
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

// SetSize updates the view dimensions.
func (m *Model) SetSize(width, height int) {
	m.cursor.SetSize(width, height)
}
