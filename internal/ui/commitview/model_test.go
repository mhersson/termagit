package commitview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.Fallback().Raw())
}

func TestNew_CreatesModel(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)

	assert.Equal(t, "abc123", m.CommitID())
	assert.True(t, m.loading, "should be loading initially")
	assert.False(t, m.ready, "should not be ready yet")
}

func TestNew_WithFilter(t *testing.T) {
	filter := []string{"path/to/file.go"}
	m := New(nil, "abc123", testTokens(), filter)

	assert.Equal(t, filter, m.filter)
}

func TestModel_SetSize(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
}

func TestModel_Init_ReturnsLoadCommand(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	cmd := m.Init()

	// Init should return a command (the loadCommitDataCmd)
	require.NotNil(t, cmd, "Init should return a command")
}

func TestModel_UpdateCommit_ChangesSingletonCommit(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = false
	m.ready = true

	cmd := m.UpdateCommit("def456", nil)

	assert.Equal(t, "def456", m.CommitID())
	assert.True(t, m.loading, "should start loading new commit")
	require.NotNil(t, cmd, "should return load command")
}

func TestModel_UpdateCommit_SameHash_NoOp(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.loading = false

	cmd := m.UpdateCommit("abc123", nil)

	assert.Nil(t, cmd, "same hash should be no-op")
}

func TestKeyMap_DefaultBindings(t *testing.T) {
	keys := DefaultKeyMap()

	// Verify essential keys are bound
	assert.NotEmpty(t, keys.Close.Keys())
	assert.NotEmpty(t, keys.MoveDown.Keys())
	assert.NotEmpty(t, keys.MoveUp.Keys())
}

func TestCommitDataLoadedMsg_SetsData(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)

	info := &git.LogEntry{
		Hash:          "abc123def456",
		Subject:       "Test commit",
		AuthorName:    "Test Author",
		AuthorEmail:   "test@example.com",
		AuthorDate:    "2024-01-01T12:00:00Z",
		CommitterName: "Test Committer",
	}

	overview := &git.CommitOverview{
		Summary: "1 file changed, 10 insertions(+)",
		Files:   []git.CommitOverviewFile{{Path: "test.go", Changes: "10"}},
	}

	msg := CommitDataLoadedMsg{
		Info:     info,
		Overview: overview,
	}

	newM, _ := m.Update(msg)
	model := newM.(Model)

	assert.False(t, model.loading)
	assert.True(t, model.ready)
	assert.Equal(t, info, model.info)
	assert.Equal(t, overview, model.overview)
}

func TestCloseCommitViewMsg_OnQKey(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = false
	m.ready = true

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(keyMsg)

	// Should emit close message
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseCommitViewMsg)
	assert.True(t, ok, "should emit CloseCommitViewMsg")
}

func TestView_ShowsLoadingWhenLoading(t *testing.T) {
	m := New(nil, "abc123", testTokens(), nil)
	m.SetSize(80, 24)
	m.loading = true

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestView_RendersCommitHeader(t *testing.T) {
	m := New(nil, "abc123def456789", testTokens(), nil)
	m.SetSize(80, 24)

	// Use proper Update flow to set data and viewport content
	info := &git.LogEntry{
		Hash:        "abc123def456789",
		Subject:     "Test commit subject",
		AuthorName:  "Test Author",
		AuthorEmail: "test@example.com",
		AuthorDate:  "2024-01-01T12:00:00Z",
	}

	msg := CommitDataLoadedMsg{
		Info:     info,
		Overview: &git.CommitOverview{},
	}

	newM, _ := m.Update(msg)
	model := newM.(Model)

	view := model.View()
	// Should contain "Commit <hash>"
	assert.Contains(t, view, "Commit")
	assert.Contains(t, view, "abc123def456789")
}
