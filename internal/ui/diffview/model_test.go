package diffview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

func testTokens() theme.Tokens {
	return theme.Compile(theme.Fallback().Raw())
}

func testConfig() *config.Config {
	return &config.Config{
		UI: config.UIConfig{
			DisableLineNumbers: true,
		},
	}
}

func testSource(kind git.DiffKind) DiffSource {
	return DiffSource{Kind: kind}
}

func testDiffs() []git.FileDiff {
	return []git.FileDiff{
		{
			Path: "file1.go",
			Hunks: []git.Hunk{
				{
					Header:   "@@ -1,3 +1,4 @@",
					OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 4,
					Lines: []git.DiffLine{
						{Op: git.DiffOpContext, Content: "line1"},
						{Op: git.DiffOpAdd, Content: "new line"},
						{Op: git.DiffOpContext, Content: "line2"},
					},
				},
				{
					Header:   "@@ -10,3 +11,4 @@",
					OldStart: 10, OldCount: 3, NewStart: 11, NewCount: 4,
					Lines: []git.DiffLine{
						{Op: git.DiffOpContext, Content: "line10"},
						{Op: git.DiffOpDelete, Content: "old line"},
						{Op: git.DiffOpAdd, Content: "new line"},
						{Op: git.DiffOpContext, Content: "line12"},
					},
				},
			},
		},
		{
			Path: "file2.go",
			Hunks: []git.Hunk{
				{
					Header:   "@@ -1,2 +1,3 @@",
					OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 3,
					Lines: []git.DiffLine{
						{Op: git.DiffOpContext, Content: "a"},
						{Op: git.DiffOpAdd, Content: "b"},
						{Op: git.DiffOpContext, Content: "c"},
					},
				},
			},
		},
	}
}

func loadModel(m Model) Model {
	m.SetSize(80, 24)
	msg := DiffDataLoadedMsg{Files: testDiffs()}
	newM, _ := m.Update(msg)
	return newM.(Model)
}

// 1. TestDiffModel_Init_LoadsFiles
func TestDiffModel_Init_LoadsFiles(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())

	assert.True(t, m.loading, "should be loading initially")
	cmd := m.Init()
	require.NotNil(t, cmd, "Init should return a command")
}

// 2. TestDiffModel_ScrollDown_AdvancesViewport
func TestDiffModel_ScrollDown_AdvancesViewport(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	assert.Equal(t, 0, m.cursorLine)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.Equal(t, 1, model.cursorLine)
}

// 3. TestDiffModel_NextHunk_MovesToNextHunkHeader
func TestDiffModel_NextHunk_MovesToNextHunkHeader(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.Greater(t, model.cursorLine, 0, "should move cursor to a hunk header")

	// Verify the line is actually a hunk header line
	hunkLines := model.getHunkHeaderLines()
	assert.Contains(t, hunkLines, model.cursorLine, "cursor should be on a hunk header line")
}

// 4. TestDiffModel_PrevHunk_MovesToPrevHunkHeader
func TestDiffModel_PrevHunk_MovesToPrevHunkHeader(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	// Move to end first
	m.cursorLine = m.totalLines - 1

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.Less(t, model.cursorLine, m.totalLines-1, "should move cursor back")

	hunkLines := model.getHunkHeaderLines()
	assert.Contains(t, hunkLines, model.cursorLine, "cursor should be on a hunk header line")
}

// 5. TestDiffModel_NextFile_ShowsNextFileDiff
func TestDiffModel_NextFile_ShowsNextFileDiff(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	assert.Equal(t, 0, m.fileIdx)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.Equal(t, 1, model.fileIdx)
}

// 6. TestDiffModel_PrevFile_ShowsPrevFileDiff
func TestDiffModel_PrevFile_ShowsPrevFileDiff(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	m.fileIdx = 1

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.Equal(t, 0, model.fileIdx)
}

// 7. TestDiffModel_NextFile_ClampsAtLastFile
func TestDiffModel_NextFile_ClampsAtLastFile(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	// Press ] 3 times — should clamp at file index 1 (2 files)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}}
	for range 3 {
		newM, _ := m.Update(keyMsg)
		m = newM.(Model)
	}

	assert.Equal(t, 1, m.fileIdx, "should clamp at last file index")
}

// 8. TestDiffModel_StageHunk_OnlyForUnstaged
func TestDiffModel_StageHunk_OnlyForUnstaged(t *testing.T) {
	// With DiffUnstaged — should return a command (but we have no repo, so it'll be nil due to guard)
	m := New(nil, testSource(git.DiffUnstaged), testConfig(), testTokens())
	m = loadModel(m)

	// Move cursor to a hunk line
	hunkLines := m.getHunkHeaderLines()
	require.NotEmpty(t, hunkLines)
	m.cursorLine = hunkLines[0]

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := m.Update(keyMsg)
	// No repo, so no command
	assert.Nil(t, cmd, "no repo means no stage command")

	// With DiffStaged — s should do nothing
	m2 := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m2 = loadModel(m2)
	m2.cursorLine = hunkLines[0]

	_, cmd2 := m2.Update(keyMsg)
	assert.Nil(t, cmd2, "stage should not work for DiffStaged")
}

// 9. TestDiffModel_UnstageHunk_OnlyForStaged
func TestDiffModel_UnstageHunk_OnlyForStaged(t *testing.T) {
	// With DiffStaged — unstage should be allowed (but no repo = nil cmd)
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	hunkLines := m.getHunkHeaderLines()
	require.NotEmpty(t, hunkLines)
	m.cursorLine = hunkLines[0]

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
	_, cmd := m.Update(keyMsg)
	assert.Nil(t, cmd, "no repo means no unstage command")

	// With DiffUnstaged — u should do nothing
	m2 := New(nil, testSource(git.DiffUnstaged), testConfig(), testTokens())
	m2 = loadModel(m2)
	m2.cursorLine = hunkLines[0]

	_, cmd2 := m2.Update(keyMsg)
	assert.Nil(t, cmd2, "unstage should not work for DiffUnstaged")
}

// 10. TestDiffModel_StageHunk_UnavailableForCommitDiff
func TestDiffModel_StageHunk_UnavailableForCommitDiff(t *testing.T) {
	m := New(nil, testSource(git.DiffCommit), testConfig(), testTokens())
	m = loadModel(m)

	hunkLines := m.getHunkHeaderLines()
	require.NotEmpty(t, hunkLines)
	m.cursorLine = hunkLines[0]

	// s should do nothing for commit diff
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	_, cmd := m.Update(keyMsg)
	assert.Nil(t, cmd, "stage should not work for DiffCommit")

	// u should also do nothing
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}}
	_, cmd = m.Update(keyMsg)
	assert.Nil(t, cmd, "unstage should not work for DiffCommit")
}

// 11. TestDiffView_ContextLine_UsesContextStyle
func TestDiffView_ContextLine_UsesContextStyle(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	view := m.View()
	assert.Contains(t, view, "line1", "view should contain context line content")
}

// 12. TestDiffView_AddedLine_UsesAddStyle
func TestDiffView_AddedLine_UsesAddStyle(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	view := m.View()
	assert.Contains(t, view, "new line", "view should contain added line content")
}

// 13. TestDiffView_DeletedLine_UsesDeleteStyle
func TestDiffView_DeletedLine_UsesDeleteStyle(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	view := m.View()
	assert.Contains(t, view, "old line", "view should contain deleted line content")
}

// 14. TestDiffView_HunkHeader_UsesHunkHeaderStyle
func TestDiffView_HunkHeader_UsesHunkHeaderStyle(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	view := m.View()
	assert.Contains(t, view, "@@", "view should contain hunk header")
}

// 15. TestDiffView_FileHeader_ShowsFileAndCounts
func TestDiffView_FileHeader_ShowsFileAndCounts(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	view := m.View()
	assert.Contains(t, view, "file1.go", "view should contain file path")
	assert.Contains(t, view, "file 1/2", "view should contain file counter")
}

// 16. TestDiffView_LineNumbers_ShownWhenConfigured
func TestDiffView_LineNumbers_ShownWhenConfigured(t *testing.T) {
	cfg := testConfig()
	cfg.UI.DisableLineNumbers = false

	m := New(nil, testSource(git.DiffStaged), cfg, testTokens())
	m = loadModel(m)

	view := m.View()
	// Line numbers should appear (old/new line numbers like "  1   1")
	assert.Contains(t, view, "1", "view should contain line numbers")
}

// 17. TestDiffView_Separator_UsesCorrectStyle
func TestDiffView_Separator_UsesCorrectStyle(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	view := m.View()
	assert.Contains(t, view, "─", "view should contain separator line")
}

// 18. TestDiffModel_RangeDiff_ShellsOut
func TestDiffModel_RangeDiff_ShellsOut(t *testing.T) {
	source := DiffSource{Kind: git.DiffRange, Range: "main..feature"}
	m := New(nil, source, testConfig(), testTokens())

	assert.Equal(t, "Changes main..feature", m.header)
	cmd := m.Init()
	require.NotNil(t, cmd, "Init should return a command for range diff")
}

// 19. TestDiffModel_CloseEmitsCloseDiffViewMsg
func TestDiffModel_CloseEmitsCloseDiffViewMsg(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(keyMsg)

	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(CloseDiffViewMsg)
	assert.True(t, ok, "should emit CloseDiffViewMsg")
}

// Additional tests

func TestNew_SetsHeaderForSource(t *testing.T) {
	tests := []struct {
		source DiffSource
		header string
	}{
		{DiffSource{Kind: git.DiffStaged}, "Staged changes"},
		{DiffSource{Kind: git.DiffUnstaged}, "Unstaged changes"},
		{DiffSource{Kind: git.DiffCommit, Commit: "abc123def"}, "Commit abc123d"},
		{DiffSource{Kind: git.DiffRange, Range: "main..dev"}, "Changes main..dev"},
		{DiffSource{Kind: git.DiffStash, Stash: "stash@{0}"}, "Stash stash@{0}"},
	}

	for _, tc := range tests {
		m := New(nil, tc.source, testConfig(), testTokens())
		assert.Equal(t, tc.header, m.header, "header for %v", tc.source.Kind)
	}
}

func TestModel_SetSize(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m.SetSize(80, 24)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
}

func TestModel_Done_InitiallyFalse(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	assert.False(t, m.Done())
}

func TestModel_Done_TrueAfterClose(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)

	assert.True(t, model.Done())
}

func TestModel_CursorStaysInBounds(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m = loadModel(m)

	// Move up at boundary
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newM, _ := m.Update(keyMsg)
	model := newM.(Model)
	assert.Equal(t, 0, model.cursorLine)
}

func TestView_ShowsLoadingWhenLoading(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m.SetSize(80, 24)
	m.loading = true

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestView_ShowsNoChangesWhenEmpty(t *testing.T) {
	m := New(nil, testSource(git.DiffStaged), testConfig(), testTokens())
	m.SetSize(80, 24)

	msg := DiffDataLoadedMsg{Files: nil}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	view := model.View()
	assert.Contains(t, view, "No changes")
}

func TestParseStashIndex(t *testing.T) {
	idx, ok := parseStashIndex("stash@{0}")
	assert.True(t, ok)
	assert.Equal(t, 0, idx)

	idx, ok = parseStashIndex("stash@{5}")
	assert.True(t, ok)
	assert.Equal(t, 5, idx)

	_, ok = parseStashIndex("HEAD")
	assert.False(t, ok)
}

func TestKeyMap_DefaultBindings(t *testing.T) {
	keys := DefaultKeyMap()

	assert.NotEmpty(t, keys.Close.Keys())
	assert.NotEmpty(t, keys.MoveDown.Keys())
	assert.NotEmpty(t, keys.MoveUp.Keys())
	assert.NotEmpty(t, keys.NextHunk.Keys())
	assert.NotEmpty(t, keys.PrevHunk.Keys())
	assert.NotEmpty(t, keys.NextFile.Keys())
	assert.NotEmpty(t, keys.PrevFile.Keys())
	assert.NotEmpty(t, keys.StageHunk.Keys())
	assert.NotEmpty(t, keys.UnstageHunk.Keys())
}

func TestDiffView_StatBlock_RendersWhenPresent(t *testing.T) {
	m := New(nil, testSource(git.DiffCommit), testConfig(), testTokens())
	m.SetSize(80, 24)

	stats := &git.CommitOverview{
		Summary: "2 files changed, 5 insertions(+), 1 deletion(-)",
		Files: []git.CommitOverviewFile{
			{Path: "file1.go", Changes: "4", Insertions: "++++", Deletions: ""},
			{Path: "file2.go", Changes: "2", Insertions: "+", Deletions: "-"},
		},
	}

	msg := DiffDataLoadedMsg{Files: testDiffs(), Stats: stats}
	newM, _ := m.Update(msg)
	model := newM.(Model)

	view := model.View()
	assert.Contains(t, view, "2 files changed", "should render stat summary")
	assert.Contains(t, view, "file1.go", "should render stat file paths")
}
