package refsview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/shared"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.RawTokens{})
}

func testRefs() *git.RefsResult {
	return &git.RefsResult{
		LocalBranches: []git.RefEntry{
			{
				Name: "main", UnambiguousName: "main",
				Type: git.RefTypeLocalBranch, Head: true,
				Oid: "abc123def456", AbbrevOid: "abc123d",
				Subject:      "Initial commit",
				UpstreamName: "origin/main", UpstreamStatus: "=",
			},
			{
				Name: "feature/x", UnambiguousName: "feature/x",
				Type: git.RefTypeLocalBranch,
				Oid:  "def456abc123", AbbrevOid: "def456a",
				Subject: "Add feature x",
			},
		},
		RemoteBranches: map[string][]git.RefEntry{
			"origin": {
				{
					Name: "main", UnambiguousName: "origin/main",
					Type: git.RefTypeRemoteBranch, Remote: "origin",
					Oid: "abc123def456", AbbrevOid: "abc123d",
					Subject: "Initial commit",
				},
			},
		},
		Tags: []git.RefEntry{
			{
				Name: "v1.0", UnambiguousName: "tags/v1.0",
				Type: git.RefTypeTag,
				Oid:  "ghi789abc123", AbbrevOid: "ghi789a",
				Subject: "Release v1.0",
			},
		},
	}
}

func testRemotes() []git.Remote {
	return []git.Remote{
		{Name: "origin", FetchURL: "https://github.com/test/repo.git"},
	}
}

func TestRefsModel_Init_LoadsAllSections(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	// Should have 3 sections: Branches, Remote origin, Tags
	assert.Len(t, m.sections, 3)
	assert.Equal(t, "Branches", m.sections[0].Title)
	assert.Equal(t, "Remote origin", m.sections[1].Title)
	assert.Equal(t, "Tags", m.sections[2].Title)
}

func TestRefsModel_LocalBranches_Populated(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	assert.Len(t, m.sections[0].Items, 2)
	assert.Equal(t, "main", m.sections[0].Items[0].Name)
	assert.Equal(t, "feature/x", m.sections[0].Items[1].Name)
}

func TestRefsModel_RemoteBranches_Populated(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	assert.Equal(t, RefsSectionRemote, m.sections[1].Kind)
	assert.Len(t, m.sections[1].Items, 1)
	assert.Equal(t, "origin", m.sections[1].RemoteName)
	assert.Equal(t, "https://github.com/test/repo.git", m.sections[1].RemoteURL)
}

func TestRefsModel_Tags_Populated(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	assert.Equal(t, RefsSectionTags, m.sections[2].Kind)
	assert.Len(t, m.sections[2].Items, 1)
	assert.Equal(t, "v1.0", m.sections[2].Items[0].Name)
}

func TestRefsModel_HeadBranch_MarkedAsCurrent(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	// Move cursor to first ref (past the first section header)
	m.cursor.Pos = 1 // first item in first section

	ref := m.currentRef()
	require.NotNil(t, ref)
	assert.True(t, ref.Head)
	assert.Equal(t, "main", ref.Name)
}

func TestRefsModel_FoldToggle_HidesItems(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	// Initially sections[0] has 2 items = 3 flat rows for that section (1 header + 2 items)
	initialRows := len(m.flatRows)

	// Cursor on first section header (index 0)
	m.cursor.Pos = 0
	m = m.toggleFold()

	assert.True(t, m.sections[0].Folded)
	// Should have fewer flat rows now (removed 2 items)
	assert.Equal(t, initialRows-2, len(m.flatRows))

	// Toggle back
	m = m.toggleFold()
	assert.False(t, m.sections[0].Folded)
	assert.Equal(t, initialRows, len(m.flatRows))
}

func TestRefsModel_DeleteBranch_RequiresConfirm(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.SetSize(80, 24)

	// Move to feature/x (second item in first section = flatRow index 2)
	m.cursor.Pos = 2

	// Press x
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Equal(t, confirmDeleteBranch, m.confirmMode)
	require.NotNil(t, m.confirmRef)
	assert.Equal(t, "feature/x", m.confirmRef.Name)
}

func TestRefsModel_DeleteBranch_CancelOnN(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.Pos = 2 // feature/x

	// Press x then n
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	assert.Equal(t, confirmDeleteBranch, m.confirmMode)

	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, confirmNone, m.confirmMode)
	assert.Nil(t, m.confirmRef)
}

func TestRefsModel_Close_SendsCloseMsg(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseRefsViewMsg)
	assert.True(t, ok)
}

func TestRefsModel_Select_OpensCommitView(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.Pos = 1 // first ref (main)

	_, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	require.NotNil(t, cmd)
	msg := cmd()
	cvMsg, ok := msg.(shared.OpenCommitViewMsg)
	assert.True(t, ok)
	assert.Equal(t, "abc123def456", cvMsg.Hash)
}

func TestRefsView_HeadBranch_HasAtPrefix(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.SetSize(120, 24)

	view := m.View()

	assert.Contains(t, view, "@ ")
	assert.Contains(t, view, "main")
}

func TestRefsView_UpstreamStatus_ColourCoded(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.SetSize(120, 24)

	view := m.View()

	// The upstream name should be present in the view
	assert.Contains(t, view, "origin/main")
}

func TestRefsView_SectionHeaders_FoldableWithSigns(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.SetSize(120, 24)

	view := m.View()

	// Section headers should show counts
	assert.Contains(t, view, "(2)") // Branches has 2 items
	assert.Contains(t, view, "(1)") // Remote origin and Tags each have 1 item
}

func TestRefsView_RemoteSection_ShowsURL(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())
	m.cursor.SetSize(120, 24)

	view := m.View()

	assert.Contains(t, view, "https://github.com/test/repo.git")
}

func TestRefsModel_Navigation_MovesCorrectly(t *testing.T) {
	m := New(testRefs(), testRemotes(), nil, testTokens())

	assert.Equal(t, 0, m.cursor.Pos)

	// Move down
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, m.cursor.Pos)

	// Move up
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor.Pos)

	// Can't go above 0
	m, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, m.cursor.Pos)
}
