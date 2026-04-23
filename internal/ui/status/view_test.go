package status

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

// testModel builds a Model with sections for render testing.
func testModel() Model {
	tokens := theme.Compile(theme.RawTokens{})
	m := Model{
		tokens: tokens,
		head: HeadState{
			Branch:    "main",
			AbbrevOid: "abc123d",
			Subject:   "test commit",
		},
		cursor: Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1},
		sections: []Section{
			{
				Kind:  SectionUnstaged,
				Title: "Unstaged changes",
				Items: []Item{
					{Entry: &git.StatusEntry{Path: "file1.go", Unstaged: git.FileStatusModified}},
					{Entry: &git.StatusEntry{Path: "file2.go", Unstaged: git.FileStatusModified}},
				},
			},
			{
				Kind:  SectionStaged,
				Title: "Staged changes",
				Items: []Item{
					{Entry: &git.StatusEntry{Path: "file3.go", Staged: git.FileStatusAdded}},
				},
			},
		},
	}
	return m
}

func TestRenderSectionLineCount_MatchesStringsCount(t *testing.T) {
	m := testModel()
	for i := range m.sections {
		s := &m.sections[i]
		var cursorLine int
		content, lineCount := renderSectionWithLineTracking(m, i, s, 0, &cursorLine)
		actual := strings.Count(content, "\n")
		assert.Equal(t, actual, lineCount,
			"section %q: returned line count %d != strings.Count %d",
			s.Title, lineCount, actual)
	}
}

func TestRenderItemLineCount_MatchesStringsCount(t *testing.T) {
	m := testModel()
	s := &m.sections[0]
	for i := range s.Items {
		item := &s.Items[i]
		var cursorLine int
		content, lineCount := renderItemWithLineTracking(m, 0, i, item, s.Kind, 0, &cursorLine)
		actual := strings.Count(content, "\n")
		assert.Equal(t, actual, lineCount,
			"item %q: returned line count %d != strings.Count %d",
			item.Entry.Path, lineCount, actual)
	}
}

func TestRenderHunkLineCount_MatchesStringsCount(t *testing.T) {
	m := testModel()
	hunk := git.Hunk{
		Header: "@@ -1,3 +1,4 @@",
		Lines: []git.DiffLine{
			{Op: git.DiffOpContext, Content: " line1"},
			{Op: git.DiffOpDelete, Content: " line2"},
			{Op: git.DiffOpAdd, Content: " line2-new"},
			{Op: git.DiffOpContext, Content: " line3"},
		},
	}
	var cursorLine int

	// Unfolded
	content, lineCount := renderHunkWithLineTracking(m, 0, 0, 0, &hunk, false, 0, &cursorLine)
	actual := strings.Count(content, "\n")
	assert.Equal(t, actual, lineCount, "unfolded hunk line count mismatch")

	// Folded
	content, lineCount = renderHunkWithLineTracking(m, 0, 0, 0, &hunk, true, 0, &cursorLine)
	actual = strings.Count(content, "\n")
	assert.Equal(t, actual, lineCount, "folded hunk line count mismatch")
}

func TestRenderItemLineCount_WithExpandedHunks(t *testing.T) {
	m := testModel()
	item := &Item{
		Entry:    &git.StatusEntry{Path: "file.go", Unstaged: git.FileStatusModified},
		Expanded: true,
		Hunks: []git.Hunk{
			{
				Header: "@@ -1,3 +1,4 @@",
				Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: " line1"},
					{Op: git.DiffOpAdd, Content: " added"},
				},
			},
		},
		HunksFolded: []bool{false},
	}
	var cursorLine int
	content, lineCount := renderItemWithLineTracking(m, 0, 0, item, SectionUnstaged, 0, &cursorLine)
	actual := strings.Count(content, "\n")
	assert.Equal(t, actual, lineCount, "expanded item line count mismatch")
}

// testModelWithHints builds a model with hint bar enabled for cursor line tests.
func testModelWithHints() Model {
	m := testModel()
	m.cfg = &config.Config{}
	return m
}

func TestComputeCursorLine_MatchesRenderContent(t *testing.T) {
	m := testModelWithHints()

	// Test cursor on each section header
	for i := range m.sections {
		m.cursor = Cursor{Section: i, Item: -1, Hunk: -1, Line: -1}
		_, expected := renderContent(m)
		actual := computeCursorLine(m)
		assert.Equal(t, expected, actual,
			"section header %d: computeCursorLine=%d, renderContent=%d", i, actual, expected)
	}

	// Test cursor on each item
	for i, s := range m.sections {
		for j := range s.Items {
			m.cursor = Cursor{Section: i, Item: j, Hunk: -1, Line: -1}
			_, expected := renderContent(m)
			actual := computeCursorLine(m)
			assert.Equal(t, expected, actual,
				"section %d item %d: computeCursorLine=%d, renderContent=%d", i, j, actual, expected)
		}
	}
}

func TestComputeCursorLine_WithFoldedSections(t *testing.T) {
	m := testModelWithHints()
	// Fold the first section
	m.sections[0].Folded = true
	// Cursor on second section header
	m.cursor = Cursor{Section: 1, Item: -1, Hunk: -1, Line: -1}

	_, expected := renderContent(m)
	actual := computeCursorLine(m)
	assert.Equal(t, expected, actual, "folded section cursor mismatch")
}

func TestComputeCursorLine_WithExpandedHunks(t *testing.T) {
	m := testModelWithHints()
	// Expand item and add hunks
	m.sections[0].Items[0].Expanded = true
	m.sections[0].Items[0].Hunks = []git.Hunk{
		{
			Header: "@@ -1,3 +1,4 @@",
			Lines: []git.DiffLine{
				{Op: git.DiffOpContext, Content: " ctx1"},
				{Op: git.DiffOpAdd, Content: " add1"},
				{Op: git.DiffOpDelete, Content: " del1"},
			},
		},
	}
	m.sections[0].Items[0].HunksFolded = []bool{false}

	// Cursor on hunk header
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: -1}
	_, expected := renderContent(m)
	actual := computeCursorLine(m)
	assert.Equal(t, expected, actual, "hunk header cursor mismatch")

	// Cursor on diff line
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: 1}
	_, expected = renderContent(m)
	actual = computeCursorLine(m)
	assert.Equal(t, expected, actual, "diff line cursor mismatch")

	// Cursor on second item (after expanded hunks)
	m.cursor = Cursor{Section: 0, Item: 1, Hunk: -1, Line: -1}
	_, expected = renderContent(m)
	actual = computeCursorLine(m)
	assert.Equal(t, expected, actual, "item after hunks cursor mismatch")
}

func TestComputeCursorLine_WithMergeAndPush(t *testing.T) {
	m := testModelWithHints()
	m.head.UpstreamBranch = "main"
	m.head.UpstreamRemote = "origin"
	m.head.PushBranch = "main"
	m.head.PushRemote = "origin"
	m.head.Tag = "v1.0"

	// Cursor on first section header
	m.cursor = Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}
	_, expected := renderContent(m)
	actual := computeCursorLine(m)
	assert.Equal(t, expected, actual, "cursor with all head lines mismatch")
}

func TestComputeCursorLine_HintBarDisabled(t *testing.T) {
	m := testModel()
	m.cfg = &config.Config{UI: config.UIConfig{DisableHint: true}}

	m.cursor = Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}
	_, expected := renderContent(m)
	actual := computeCursorLine(m)
	assert.Equal(t, expected, actual, "no hint bar cursor mismatch")
}

func TestContentCache_InvalidatedOnToggle(t *testing.T) {
	m := testModel()
	m.contentDirty = true
	m.ensureContent()
	require.False(t, m.contentDirty)

	m.invalidateContent()
	assert.True(t, m.contentDirty)
}

func TestContentCache_NotInvalidatedOnCursorMove(t *testing.T) {
	m := testModel()
	m.contentDirty = true
	m.ensureContent()

	// Move cursor — should not dirty the cache
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}
	assert.False(t, m.contentDirty, "cursor move should not invalidate content")
	assert.NotEmpty(t, m.cachedBaseContent, "cache should still have content")
}

// === Visual Mode Selection Tests ===

func testModelWithHunks() Model {
	tokens := theme.Compile(theme.RawTokens{})
	m := Model{
		tokens: tokens,
		cfg:    &config.Config{UI: config.UIConfig{DisableHint: true}},
		head: HeadState{
			Branch:    "main",
			AbbrevOid: "abc123d",
			Subject:   "test commit",
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: 0, Line: 0},
		sections: []Section{
			{
				Kind:  SectionUnstaged,
				Title: "Unstaged changes",
				Items: []Item{
					{
						Entry:    &git.StatusEntry{Path: "file.go", Unstaged: git.FileStatusModified},
						Expanded: true,
						Hunks: []git.Hunk{
							{
								Header: "@@ -1,5 +1,5 @@",
								Lines: []git.DiffLine{
									{Op: git.DiffOpContext, Content: "line1"},
									{Op: git.DiffOpDelete, Content: "line2"},
									{Op: git.DiffOpAdd, Content: "line2-new"},
									{Op: git.DiffOpContext, Content: "line3"},
									{Op: git.DiffOpContext, Content: "line4"},
								},
							},
						},
						HunksFolded: []bool{false},
					},
				},
			},
		},
	}
	return m
}

func TestVisualSelectionRange_SamePosition(t *testing.T) {
	// When anchor == cursor, range should be exactly one line.
	anchor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 2}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 2}
	start, end := visualSelectionRange(anchor, cursor)
	assert.Equal(t, 2, start, "start should be 2")
	assert.Equal(t, 2, end, "end should be 2")
}

func TestVisualSelectionRange_ForwardSelection(t *testing.T) {
	// Anchor before cursor: start=anchor.Line, end=cursor.Line.
	anchor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 1}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 3}
	start, end := visualSelectionRange(anchor, cursor)
	assert.Equal(t, 1, start, "start should be anchor line")
	assert.Equal(t, 3, end, "end should be cursor line")
}

func TestVisualSelectionRange_BackwardSelection(t *testing.T) {
	// Anchor after cursor: order should be normalised so start <= end.
	anchor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 4}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 1}
	start, end := visualSelectionRange(anchor, cursor)
	assert.Equal(t, 1, start, "start should be min(anchor, cursor)")
	assert.Equal(t, 4, end, "end should be max(anchor, cursor)")
}

func TestVisualHighlighting_SelectionApplied(t *testing.T) {
	// When visualMode is true and lines are within the selection range, the
	// rendering code must exercise the selection path without panicking.
	// We verify both that:
	// 1. The hunk renders successfully with visual mode active.
	// 2. The cursor line tracking still matches computeCursorLine.
	m := testModelWithHunks()
	m.visualMode = true
	// Select lines 1..3 within the hunk (0-indexed within hunk lines)
	m.visualAnchor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: 1}
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: 3}

	// Render the hunk directly — must not panic and must return non-empty content.
	hunk := m.sections[0].Items[0].Hunks[0]
	var cl int
	content, lineCount := renderHunkWithLineTracking(m, 0, 0, 0, &hunk, false, 0, &cl)
	assert.NotEmpty(t, content, "hunk content should not be empty")
	// lineCount must account for header + 5 diff lines
	assert.Equal(t, 6, lineCount, "6 lines: 1 header + 5 diff lines")

	// Cursor is at hunk line 3 (the 4th line after the header), so cl should be 4
	// (startLine=0, header=line 0, diff lines 0..4 are lines 1..5).
	assert.Equal(t, 4, cl, "cursor on hunk line 3 → visual line 4 (0+1+3)")

	// renderContent with visual mode must also return the same cursor line.
	m.cfg = &config.Config{UI: config.UIConfig{DisableHint: true}}
	m.head = HeadState{Branch: "main", AbbrevOid: "abc123d", Subject: "test"}
	_, contentCursorLine := renderContent(m)
	computedCursorLine := computeCursorLine(m)
	assert.Equal(t, computedCursorLine, contentCursorLine,
		"renderContent and computeCursorLine must agree in visual mode")
}

func TestVisualHighlighting_CursorLineIsStillCursor(t *testing.T) {
	// The cursor line itself should be rendered as the cursor, not just as a
	// selection-highlighted line — computeCursorLine must still return the
	// correct cursor position in visual mode.
	m := testModelWithHunks()
	m.visualMode = true
	m.visualAnchor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: 0}
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: 2}

	_, cursorLine := renderContent(m)
	expected := computeCursorLine(m)
	assert.Equal(t, expected, cursorLine,
		"cursor line in visual mode should match computeCursorLine")
}
