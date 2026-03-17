package status

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/conjit/internal/git"
)

func TestSectionKind_AllTwelveValues(t *testing.T) {
	// All 12 Neogit sections must be defined
	expectedKinds := []SectionKind{
		SectionSequencer,        // cherry-pick / revert
		SectionRebase,           // rebase in progress
		SectionBisect,           // bisect in progress
		SectionUntracked,        // untracked files
		SectionUnstaged,         // unstaged changes
		SectionStaged,           // staged changes
		SectionStashes,          // stashes
		SectionUnmergedUpstream, // "Unmerged into"
		SectionUnpushedPushRemote, // "Unpushed to"
		SectionRecentCommits,    // "Recent Commits"
		SectionUnpulledUpstream, // "Unpulled from" (upstream)
		SectionUnpulledPushRemote, // "Unpulled from" (push remote)
	}

	if len(expectedKinds) != 12 {
		t.Errorf("expected 12 section kinds, got %d", len(expectedKinds))
	}

	// Verify they are distinct values (iota increments correctly)
	seen := make(map[SectionKind]bool)
	for _, k := range expectedKinds {
		if seen[k] {
			t.Errorf("duplicate section kind value: %d", k)
		}
		seen[k] = true
	}
}

func TestNew_InitializesWithDefaultCursor(t *testing.T) {
	m := New(nil, nil, Tokens{}, KeyMap{})

	// Default cursor should be on first section header
	if m.cursor.Section != 0 {
		t.Errorf("expected cursor.Section=0, got %d", m.cursor.Section)
	}
	if m.cursor.Item != -1 {
		t.Errorf("expected cursor.Item=-1 (section header), got %d", m.cursor.Item)
	}
	if m.cursor.Hunk != -1 {
		t.Errorf("expected cursor.Hunk=-1, got %d", m.cursor.Hunk)
	}
}

func TestHeadState_Fields(t *testing.T) {
	// HeadState must have all required fields
	h := HeadState{
		Branch:          "main",
		Oid:             "abc123def456abc123def456abc123def456abc1",
		AbbrevOid:       "abc123d",
		Subject:         "add config loader",
		Detached:        false,
		UpstreamBranch:  "main",
		UpstreamRemote:  "origin",
		UpstreamOid:     "def456",
		UpstreamSubject: "fix upstream",
		PushBranch:      "main",
		PushRemote:      "origin",
		PushOid:         "def456",
		PushSubject:     "fix push",
		Tag:             "v1.0.0",
		TagOid:          "aaa111",
		TagDistance:     3,
	}

	if h.Branch != "main" {
		t.Errorf("expected Branch=main, got %s", h.Branch)
	}
	if h.AbbrevOid != "abc123d" {
		t.Errorf("expected 7-char abbrev OID, got %s", h.AbbrevOid)
	}
}

func TestSection_HiddenAndFolded(t *testing.T) {
	// Every section must have both Folded and Hidden fields
	s := Section{
		Kind:   SectionUnstaged,
		Title:  "Unstaged changes",
		Folded: true,
		Hidden: false,
		Items:  nil,
	}

	if !s.Folded {
		t.Error("expected Folded=true")
	}
	if s.Hidden {
		t.Error("expected Hidden=false")
	}
}

func TestItem_AllFieldTypes(t *testing.T) {
	// Item must support file entries, stashes, commits, and sequencer items
	item := Item{
		Entry:         nil,
		Expanded:      true,
		Hunks:         nil,
		HunksLoading:  false,
		Stash:         nil,
		Commit:        nil,
		Action:        "pick",
		ActionHash:    "abc123",
		ActionSubject: "fix bug",
		ActionDone:    true,
		ActionStopped: false,
	}

	if !item.Expanded {
		t.Error("expected Expanded=true")
	}
	if item.Action != "pick" {
		t.Errorf("expected Action=pick, got %s", item.Action)
	}
}

func TestCursor_Fields(t *testing.T) {
	c := Cursor{
		Section: 2,
		Item:    5,
		Hunk:    1,
		Line:    3,
	}

	if c.Section != 2 {
		t.Errorf("expected Section=2, got %d", c.Section)
	}
	if c.Item != 5 {
		t.Errorf("expected Item=5, got %d", c.Item)
	}
	if c.Hunk != 1 {
		t.Errorf("expected Hunk=1, got %d", c.Hunk)
	}
	if c.Line != 3 {
		t.Errorf("expected Line=3, got %d", c.Line)
	}
}

func TestNew_InitializesLineField(t *testing.T) {
	m := New(nil, nil, Tokens{}, KeyMap{})

	// Line should be -1 (on hunk header, not within lines)
	if m.cursor.Line != -1 {
		t.Errorf("expected cursor.Line=-1, got %d", m.cursor.Line)
	}
}

// === KeyMap Tests ===

func TestDefaultKeyMap_AllBindingsSet(t *testing.T) {
	km := DefaultKeyMap()

	// Test that keys are set by checking their Keys() method
	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"MoveDown", km.MoveDown, []string{"j"}},
		{"MoveUp", km.MoveUp, []string{"k"}},
		{"Toggle", km.Toggle, []string{"tab", "za"}},
		{"Close", km.Close, []string{"q"}},
		{"HelpPopup", km.HelpPopup, []string{"?"}},
		{"CommitPopup", km.CommitPopup, []string{"c"}},
		{"BranchPopup", km.BranchPopup, []string{"b"}},
		{"BisectPopup", km.BisectPopup, []string{"B"}},
		{"DiffPopup", km.DiffPopup, []string{"d"}},
		{"LogPopup", km.LogPopup, []string{"l"}},
		{"MarginPopup", km.MarginPopup, []string{"L"}},
		{"IgnorePopup", km.IgnorePopup, []string{"i"}},
		{"TagPopup", km.TagPopup, []string{"t"}},
		{"CherryPickPopup", km.CherryPickPopup, []string{"A"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := tt.binding.Keys()
			if len(keys) == 0 {
				t.Errorf("%s: expected keys to be set", tt.name)
				return
			}
			// Check that the first expected key is present
			found := false
			for _, k := range keys {
				if k == tt.keys[0] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: expected key %q, got %v", tt.name, tt.keys[0], keys)
			}
		})
	}
}

// === MoveCursor Tests ===

func TestMoveCursor_DownFromHeaderEntersItems(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}, {}}},
	}
	cursor := Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}

	result := moveCursor(sections, cursor, 1)

	if result.Item != 0 {
		t.Errorf("expected Item=0 (first item), got %d", result.Item)
	}
}

func TestMoveCursor_DownFromLastItemGoesToNextSection(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}, {}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 0, Item: 1, Hunk: -1, Line: -1} // Last item of first section

	result := moveCursor(sections, cursor, 1)

	if result.Section != 1 || result.Item != -1 {
		t.Errorf("expected Section=1, Item=-1 (header), got Section=%d, Item=%d", result.Section, result.Item)
	}
}

func TestMoveCursor_UpFromFirstItemGoesToHeader(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}, {}}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}

	result := moveCursor(sections, cursor, -1)

	if result.Item != -1 {
		t.Errorf("expected Item=-1 (header), got %d", result.Item)
	}
}

func TestMoveCursor_UpFromHeaderGoesToPreviousSectionLastItem(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}, {}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 1, Item: -1, Hunk: -1, Line: -1}

	result := moveCursor(sections, cursor, -1)

	if result.Section != 0 || result.Item != 1 {
		t.Errorf("expected Section=0, Item=1 (last item), got Section=%d, Item=%d", result.Section, result.Item)
	}
}

func TestMoveCursor_WrapFromBottomToTop(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 1, Item: 0, Hunk: -1, Line: -1} // Last item of last section

	result := moveCursor(sections, cursor, 1)

	if result.Section != 0 || result.Item != -1 {
		t.Errorf("expected wrap to Section=0, Item=-1, got Section=%d, Item=%d", result.Section, result.Item)
	}
}

func TestMoveCursor_WrapFromTopToBottom(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}

	result := moveCursor(sections, cursor, -1)

	// Should wrap to last item of last section
	if result.Section != 1 || result.Item != 0 {
		t.Errorf("expected wrap to Section=1, Item=0, got Section=%d, Item=%d", result.Section, result.Item)
	}
}

func TestMoveCursor_SkipsFoldedSections(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: true, Items: []Item{{}}}, // Folded
		{Kind: SectionStaged, Title: "Staged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}

	// Move down - should skip folded section's items
	result := moveCursor(sections, cursor, 1)

	if result.Section != 1 || result.Item != -1 {
		t.Errorf("expected Section=1 (folded, so header), Item=-1, got Section=%d, Item=%d", result.Section, result.Item)
	}

	// Move down again - should go to staged header
	result = moveCursor(sections, result, 1)
	if result.Section != 2 || result.Item != -1 {
		t.Errorf("expected Section=2, Item=-1, got Section=%d, Item=%d", result.Section, result.Item)
	}
}

func TestMoveCursor_SkipsEmptySections(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{}}, // Empty
		{Kind: SectionStaged, Title: "Staged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}

	// Move down through headers
	cursor = moveCursor(sections, cursor, 1) // Section 1 header
	cursor = moveCursor(sections, cursor, 1) // Section 2 header (skips empty section items)

	if cursor.Section != 2 || cursor.Item != -1 {
		t.Errorf("expected Section=2, Item=-1, got Section=%d, Item=%d", cursor.Section, cursor.Item)
	}
}

func TestMoveCursor_HunkNavigation(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{{}, {}}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}

	// Move down into hunks
	result := moveCursor(sections, cursor, 1)
	if result.Hunk != 0 {
		t.Errorf("expected Hunk=0, got %d", result.Hunk)
	}

	// Move down - since no lines, go to next hunk
	result = moveCursor(sections, result, 1)
	if result.Hunk != 1 {
		t.Errorf("expected Hunk=1, got %d", result.Hunk)
	}

	// Move up back to first hunk header
	result = moveCursor(sections, result, -1)
	if result.Hunk != 0 || result.Line != -1 {
		t.Errorf("expected Hunk=0, Line=-1, got Hunk=%d, Line=%d", result.Hunk, result.Line)
	}

	// Move up to exit hunks
	result = moveCursor(sections, result, -1)
	if result.Hunk != -1 || result.Item != 0 {
		t.Errorf("expected Item=0, Hunk=-1, got Item=%d, Hunk=%d", result.Item, result.Hunk)
	}
}

func TestMoveCursor_SkipsHiddenSections(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Hidden: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Hidden: true, Items: []Item{{}}}, // Hidden
		{Kind: SectionStaged, Title: "Staged", Folded: false, Hidden: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}

	// Move down - should skip hidden section
	result := moveCursor(sections, cursor, 1)

	if result.Section != 2 || result.Item != -1 {
		t.Errorf("expected Section=2, Item=-1 (skipping hidden), got Section=%d, Item=%d", result.Section, result.Item)
	}
}

// === Line-Level Navigation Tests ===

func TestMoveCursor_DownFromHunkHeaderEntersLines(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,3 +1,4 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
				}},
			}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: -1} // On hunk header

	result := moveCursor(sections, cursor, 1)

	if result.Line != 0 {
		t.Errorf("expected Line=0 (first diff line), got %d", result.Line)
	}
}

func TestMoveCursor_DownThroughDiffLines(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,3 +1,4 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
					{Op: git.DiffOpDelete, Content: "line3"},
				}},
			}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 0}

	// Move down through lines
	result := moveCursor(sections, cursor, 1)
	if result.Line != 1 {
		t.Errorf("expected Line=1, got %d", result.Line)
	}

	result = moveCursor(sections, result, 1)
	if result.Line != 2 {
		t.Errorf("expected Line=2, got %d", result.Line)
	}
}

func TestMoveCursor_DownFromLastLineGoesToNextHunk(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,1 +1,1 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "line1"}}},
				{Header: "@@ -5,1 +5,1 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "line5"}}},
			}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 0} // Last line of first hunk

	result := moveCursor(sections, cursor, 1)

	if result.Hunk != 1 || result.Line != -1 {
		t.Errorf("expected Hunk=1, Line=-1 (next hunk header), got Hunk=%d, Line=%d", result.Hunk, result.Line)
	}
}

func TestMoveCursor_UpFromLineGoesToPreviousLine(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,3 +1,4 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
				}},
			}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 1}

	result := moveCursor(sections, cursor, -1)

	if result.Line != 0 {
		t.Errorf("expected Line=0, got %d", result.Line)
	}
}

func TestMoveCursor_UpFromFirstLineGoesToHunkHeader(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,3 +1,4 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
				}},
			}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 0, Line: 0}

	result := moveCursor(sections, cursor, -1)

	if result.Hunk != 0 || result.Line != -1 {
		t.Errorf("expected Hunk=0, Line=-1 (hunk header), got Hunk=%d, Line=%d", result.Hunk, result.Line)
	}
}

func TestMoveCursor_UpFromHunkHeaderGoesToPreviousHunkLastLine(t *testing.T) {
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,2 +1,2 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
				}},
				{Header: "@@ -5,1 +5,1 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "line5"}}},
			}},
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 1, Line: -1} // On second hunk header

	result := moveCursor(sections, cursor, -1)

	if result.Hunk != 0 || result.Line != 1 {
		t.Errorf("expected Hunk=0, Line=1 (last line of prev hunk), got Hunk=%d, Line=%d", result.Hunk, result.Line)
	}
}

// === Toggle Tests ===

// === Folded Hunk Navigation Tests ===

func TestMoveCursor_UpFromHunkHeaderSkipsFoldedPreviousHunk(t *testing.T) {
	// When moving up from hunk header to a folded previous hunk,
	// should go to previous hunk header, NOT its last line
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,2 +1,2 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
				}},
				{Header: "@@ -5,1 +5,1 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "line5"}}},
			}, HunksFolded: []bool{true, false}}, // First hunk is folded
		}},
	}
	cursor := Cursor{Section: 0, Item: 0, Hunk: 1, Line: -1} // On second hunk header

	result := moveCursor(sections, cursor, -1)

	// Should go to previous hunk HEADER (not its last line, since it's folded)
	if result.Hunk != 0 || result.Line != -1 {
		t.Errorf("expected Hunk=0, Line=-1 (prev hunk header, folded), got Hunk=%d, Line=%d", result.Hunk, result.Line)
	}
}

func TestMoveCursor_UpFromItemSkipsFoldedLastHunkOfPreviousItem(t *testing.T) {
	// When moving up from item to previous item with a folded last hunk,
	// should go to that hunk's header, NOT its last line
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,2 +1,2 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
				}},
			}, HunksFolded: []bool{true}}, // Hunk is folded
			{Expanded: false}, // Second item (cursor starts here)
		}},
	}
	cursor := Cursor{Section: 0, Item: 1, Hunk: -1, Line: -1} // On second item

	result := moveCursor(sections, cursor, -1)

	// Should go to first item's hunk HEADER (not its last line, since it's folded)
	if result.Item != 0 || result.Hunk != 0 || result.Line != -1 {
		t.Errorf("expected Item=0, Hunk=0, Line=-1 (hunk header, folded), got Item=%d, Hunk=%d, Line=%d",
			result.Item, result.Hunk, result.Line)
	}
}

func TestMoveCursor_UpFromSectionSkipsFoldedLastHunkOfPreviousSection(t *testing.T) {
	// When moving up from section header to previous section with a folded last hunk,
	// should go to that hunk's header, NOT its last line
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,2 +1,2 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "line1"},
					{Op: git.DiffOpAdd, Content: "line2"},
				}},
			}, HunksFolded: []bool{true}}, // Hunk is folded
		}},
		{Kind: SectionStaged, Title: "Staged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 1, Item: -1, Hunk: -1, Line: -1} // On second section header

	result := moveCursor(sections, cursor, -1)

	// Should go to last item's hunk HEADER (not its last line, since it's folded)
	if result.Section != 0 || result.Item != 0 || result.Hunk != 0 || result.Line != -1 {
		t.Errorf("expected Section=0, Item=0, Hunk=0, Line=-1 (hunk header, folded), got Section=%d, Item=%d, Hunk=%d, Line=%d",
			result.Section, result.Item, result.Hunk, result.Line)
	}
}

func TestHandleToggle_OnHunkTogglesHunkFold(t *testing.T) {
	// When on a hunk header, Tab should toggle hunk fold state
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{{Header: "@@ -1,1 +1,1 @@"}}},
		}},
	}
	m := Model{
		sections: sections,
		cursor:   Cursor{Section: 0, Item: 0, Hunk: 0, Line: -1},
	}

	result, _ := handleToggle(m)
	resultModel := result.(Model)

	// Hunk should now be folded
	if len(resultModel.sections[0].Items[0].HunksFolded) == 0 ||
		!resultModel.sections[0].Items[0].HunksFolded[0] {
		t.Error("expected hunk to be folded after toggle")
	}

	// Toggle again to unfold
	result, _ = handleToggle(resultModel)
	resultModel = result.(Model)

	if resultModel.sections[0].Items[0].HunksFolded[0] {
		t.Error("expected hunk to be unfolded after second toggle")
	}
}

func TestHandleToggle_OnLineTogglesHunkFoldAndMovesToHeader(t *testing.T) {
	// When on a diff line, Tab should toggle hunk fold and move cursor to hunk header
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,1 +1,1 @@", Lines: []git.DiffLine{{Op: git.DiffOpContext, Content: "test"}}},
			}},
		}},
	}
	m := Model{
		sections: sections,
		cursor:   Cursor{Section: 0, Item: 0, Hunk: 0, Line: 0},
	}

	result, _ := handleToggle(m)
	resultModel := result.(Model)

	// Cursor should move to hunk header
	if resultModel.cursor.Line != -1 {
		t.Errorf("expected Line=-1 (moved to hunk header), got %d", resultModel.cursor.Line)
	}
	// Hunk should be folded
	if len(resultModel.sections[0].Items[0].HunksFolded) == 0 ||
		!resultModel.sections[0].Items[0].HunksFolded[0] {
		t.Error("expected hunk to be folded after toggle")
	}
}

func TestHandleToggle_OnItemTogglesExpansion(t *testing.T) {
	// When on item (not hunk/line), Tab should toggle expansion
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{
			{Expanded: true, Hunks: []git.Hunk{{Header: "@@ -1,1 +1,1 @@"}}},
		}},
	}
	m := Model{
		sections: sections,
		cursor:   Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}

	result, _ := handleToggle(m)
	resultModel := result.(Model)

	if resultModel.sections[0].Items[0].Expanded {
		t.Error("expected item to be collapsed after toggle")
	}
}

// === View Tests ===

func TestView_HeadBar_ShowsBranchAndSubject(t *testing.T) {
	m := Model{
		head: HeadState{
			Branch:    "main",
			AbbrevOid: "abc1234",
			Subject:   "add config loader",
		},
		sections: []Section{},
	}

	output := view(m)

	if !contains(output, "main") {
		t.Error("expected branch name in output")
	}
	if !contains(output, "abc1234") {
		t.Error("expected abbreviated OID in output")
	}
	if !contains(output, "add config loader") {
		t.Error("expected subject in output")
	}
}

func TestView_HeadBar_DetachedHEAD(t *testing.T) {
	m := Model{
		head: HeadState{
			Branch:    "HEAD",
			AbbrevOid: "abc1234",
			Subject:   "detached commit",
			Detached:  true,
		},
		sections: []Section{},
	}

	output := view(m)

	if !contains(output, "(detached)") {
		t.Error("expected (detached) in output")
	}
}

func TestView_SectionHeader_ShowsTitle(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Folded: false, Items: []Item{{}, {}, {}}},
		},
	}

	output := view(m)

	if !contains(output, "Untracked files") {
		t.Error("expected section title in output")
	}
}

func TestView_SectionHeader_ShowsCount(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Folded: false, Items: []Item{{}, {}, {}}},
		},
	}

	output := view(m)

	if !contains(output, "(3)") {
		t.Error("expected item count (3) in output")
	}
}

func TestView_SectionHeader_SignChars(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Folded: false, Items: []Item{{}}},
			{Kind: SectionUnstaged, Title: "Unstaged changes", Folded: true, Items: []Item{{}}},
		},
	}

	output := view(m)

	// Open section should have 'v'
	if !contains(output, "v Untracked") {
		t.Errorf("expected 'v' sign for open section, got: %s", output)
	}
	// Closed section should have '>'
	if !contains(output, "> Unstaged") {
		t.Errorf("expected '>' sign for closed section, got: %s", output)
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// === Phase 4 Integration Tests ===

func TestModel_ConfirmMode_Values(t *testing.T) {
	// ConfirmMode must have all expected values
	modes := []ConfirmMode{
		ConfirmNone,
		ConfirmDiscard,
		ConfirmDiscardHunk,
		ConfirmUntrack,
	}

	if len(modes) != 4 {
		t.Errorf("expected 4 confirm modes, got %d", len(modes))
	}

	// Verify they are distinct
	seen := make(map[ConfirmMode]bool)
	for _, m := range modes {
		if seen[m] {
			t.Errorf("duplicate confirm mode value: %d", m)
		}
		seen[m] = true
	}
}

func TestModel_StashesSection_Rendered(t *testing.T) {
	stash := &git.StashEntry{
		Index:   0,
		Name:    "stash@{0}",
		Message: "WIP on main: abc123 fix bug",
	}

	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{
				Kind:  SectionStashes,
				Title: "Stashes",
				Items: []Item{{Stash: stash}},
			},
		},
	}

	output := view(m)

	if !contains(output, "Stashes") {
		t.Error("expected Stashes section title")
	}
	if !contains(output, "stash@{0}") {
		t.Error("expected stash name in output")
	}
}

func TestModel_RebaseSection_ShowsEntries(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "feature"},
		sections: []Section{
			{
				Kind:  SectionRebase,
				Title: "Rebasing feature onto main (2/4)",
				Items: []Item{
					{Action: "pick", ActionHash: "abc1234", ActionSubject: "First commit", ActionDone: true},
					{Action: "pick", ActionHash: "def5678", ActionSubject: "Second commit", ActionStopped: true},
					{Action: "squash", ActionHash: "ghi9012", ActionSubject: "Third commit"},
				},
			},
		},
	}

	output := view(m)

	if !contains(output, "Rebasing") {
		t.Error("expected Rebasing section title")
	}
	if !contains(output, "pick") {
		t.Error("expected pick action in output")
	}
	if !contains(output, "squash") {
		t.Error("expected squash action in output")
	}
}

func TestModel_BisectSection_ShowsEntries(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{
				Kind:  SectionBisect,
				Title: "Bisecting Log",
				Items: []Item{
					{Action: "good", ActionHash: "abc1234", ActionSubject: "Known good commit"},
					{Action: "bad", ActionHash: "def5678", ActionSubject: "Known bad commit"},
				},
			},
		},
	}

	output := view(m)

	if !contains(output, "Bisecting") {
		t.Error("expected Bisecting section title")
	}
	if !contains(output, "good") {
		t.Error("expected good action in output")
	}
	if !contains(output, "bad") {
		t.Error("expected bad action in output")
	}
}

func TestModel_SequencerSection_ShowsCherryPick(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{
				Kind:  SectionSequencer,
				Title: "Cherry Picking (2)",
				Items: []Item{
					{Action: "pick", ActionHash: "abc1234", ActionSubject: "Feature commit"},
					{Action: "pick", ActionHash: "def5678", ActionSubject: "Another commit"},
				},
			},
		},
	}

	output := view(m)

	if !contains(output, "Cherry Picking") {
		t.Error("expected Cherry Picking section title")
	}
}

func TestModel_CommitSection_ShowsRefs(t *testing.T) {
	commit := &git.LogEntry{
		Hash:            "abc123def456abc123def456abc123def456abc1",
		AbbreviatedHash: "abc123d",
		Subject:         "add config loader",
		Refs: []git.Ref{
			{Name: "main", Kind: git.RefKindLocal},
			{Name: "main", Kind: git.RefKindRemote, Remote: "origin"},
		},
	}

	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{
				Kind:  SectionRecentCommits,
				Title: "Recent Commits",
				Items: []Item{{Commit: commit}},
			},
		},
	}

	output := view(m)

	if !contains(output, "Recent Commits") {
		t.Error("expected Recent Commits section title")
	}
	if !contains(output, "abc123d") {
		t.Error("expected abbreviated hash in output")
	}
}

func TestModel_EmptySection_NotRenderedInCount(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Items: []Item{}},
		},
	}

	output := view(m)

	// Empty section should still show in output (possibly with count)
	// This test just verifies we don't crash on empty sections
	_ = output
}

func TestGetCurrentItem_ReturnsNilForHeaderPosition(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Items: []Item{{}}},
		},
		cursor: Cursor{Section: 0, Item: -1},
	}

	item, _ := getCurrentItem(m)
	if item != nil {
		t.Error("expected nil item when on section header")
	}
}

func TestGetCurrentItem_ReturnsItemForValidPosition(t *testing.T) {
	entry := &git.StatusEntry{}
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Items: []Item{{Entry: entry}}},
		},
		cursor: Cursor{Section: 0, Item: 0},
	}

	item, kind := getCurrentItem(m)
	if item == nil {
		t.Error("expected non-nil item")
	}
	if kind != SectionUnstaged {
		t.Errorf("expected SectionUnstaged, got %d", kind)
	}
}

func TestHandleConfirmKey_CancelsOnN(t *testing.T) {
	m := Model{
		confirmMode: ConfirmDiscard,
		confirmPath: "test.txt",
	}

	result, _ := handleConfirmKey(m, keyMsg("n"))
	resultModel := result.(Model)

	if resultModel.confirmMode != ConfirmNone {
		t.Error("expected confirm mode to be cancelled")
	}
}

func TestHandleConfirmKey_CancelsOnEsc(t *testing.T) {
	m := Model{
		confirmMode: ConfirmDiscard,
		confirmPath: "test.txt",
	}

	result, _ := handleConfirmKey(m, keyMsg("esc"))
	resultModel := result.(Model)

	if resultModel.confirmMode != ConfirmNone {
		t.Error("expected confirm mode to be cancelled on esc")
	}
}

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// === Viewport and Scroll Tests ===

func TestRenderContent_ReturnsCursorLine(t *testing.T) {
	// When rendering content, we should be able to determine
	// which visual line the cursor is on
	m := Model{
		head: HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Folded: false, Items: []Item{{}, {}, {}}},
			{Kind: SectionUnstaged, Title: "Unstaged changes", Folded: false, Items: []Item{{}, {}}},
		},
		cursor: Cursor{Section: 1, Item: 0, Hunk: -1, Line: -1}, // First item of second section
	}

	_, cursorLine := renderContent(m)

	// Cursor should be on a line > 0 (after HEAD bar and first section)
	if cursorLine <= 0 {
		t.Errorf("expected cursorLine > 0, got %d", cursorLine)
	}
}

func TestRenderContent_CursorLineOnSectionHeader(t *testing.T) {
	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Folded: false, Items: []Item{{}}},
		},
		cursor: Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}, // On section header
	}

	content, cursorLine := renderContent(m)

	// Cursor line should be where the section header is
	lines := strings.Split(content, "\n")
	if cursorLine >= len(lines) {
		t.Errorf("cursorLine %d out of range (content has %d lines)", cursorLine, len(lines))
		return
	}

	// The line at cursorLine should contain the section title
	if !strings.Contains(lines[cursorLine], "Untracked") {
		t.Errorf("expected cursor line to contain section title, got: %q", lines[cursorLine])
	}
}

func TestModel_ViewportInitialized(t *testing.T) {
	// Viewport should be initialized with dimensions after WindowSizeMsg
	m := New(nil, nil, Tokens{}, DefaultKeyMap())
	m.loading = false
	m.sections = []Section{{Kind: SectionUntracked, Title: "Test"}}

	// Send WindowSizeMsg
	result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	resultModel := result.(Model)

	if resultModel.viewport.Width != 80 {
		t.Errorf("expected viewport.Width=80, got %d", resultModel.viewport.Width)
	}
	if resultModel.viewport.Height != 24 {
		t.Errorf("expected viewport.Height=24, got %d", resultModel.viewport.Height)
	}
}

func TestEnsureCursorVisible_ScrollsDownWhenNeeded(t *testing.T) {
	// When cursor is below visible area, viewport should scroll down
	m := Model{
		width:  80,
		height: 10,
		head:   HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Test", Items: []Item{}},
		},
		cursor: Cursor{Section: 0, Item: -1},
	}
	// Initialize viewport
	m.viewport.Width = 80
	m.viewport.Height = 10
	m.viewport.YOffset = 0

	// Simulate cursor being at line 15 (below visible area of 10 lines)
	cursorLine := 15
	ensureCursorVisible(&m, cursorLine)

	// Viewport should have scrolled so cursorLine is visible
	// cursorLine should be within [YOffset, YOffset+Height)
	if cursorLine < m.viewport.YOffset || cursorLine >= m.viewport.YOffset+m.viewport.Height {
		t.Errorf("cursor at line %d should be visible with YOffset=%d, Height=%d",
			cursorLine, m.viewport.YOffset, m.viewport.Height)
	}
}

func TestEnsureCursorVisible_ScrollsUpWhenNeeded(t *testing.T) {
	// When cursor is above visible area, viewport should scroll up
	m := Model{
		width:  80,
		height: 10,
	}
	m.viewport.Width = 80
	m.viewport.Height = 10
	m.viewport.YOffset = 20 // Scrolled down

	// Simulate cursor being at line 5 (above visible area starting at 20)
	cursorLine := 5
	ensureCursorVisible(&m, cursorLine)

	// Viewport should have scrolled up
	if m.viewport.YOffset != 5 {
		t.Errorf("expected YOffset=5, got %d", m.viewport.YOffset)
	}
}

func TestPreserveScreenPosition_MaintainsCursorRow(t *testing.T) {
	// When expanding content, cursor should stay at same screen position
	m := Model{
		width:  80,
		height: 20,
	}
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.viewport.YOffset = 10

	// Cursor was at visual line 15, which is screen row 5 (15 - 10)
	oldCursorLine := 15
	screenRow := oldCursorLine - m.viewport.YOffset // 5

	// After expansion, cursor moved to visual line 25
	newCursorLine := 25

	preserveScreenPosition(&m, newCursorLine, screenRow)

	// New YOffset should keep cursor at same screen row
	// newCursorLine - YOffset should equal screenRow
	expectedYOffset := newCursorLine - screenRow // 25 - 5 = 20
	if m.viewport.YOffset != expectedYOffset {
		t.Errorf("expected YOffset=%d, got %d", expectedYOffset, m.viewport.YOffset)
	}
}

func TestDefaultKeyMap_ScrollBindings(t *testing.T) {
	km := DefaultKeyMap()

	// Test scroll navigation keys
	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"PageUp", km.PageUp, []string{"ctrl+b"}},
		{"PageDown", km.PageDown, []string{"ctrl+f"}},
		{"HalfPageUp", km.HalfPageUp, []string{"ctrl+u"}},
		{"HalfPageDown", km.HalfPageDown, []string{"ctrl+d"}},
		{"GoToTop", km.GoToTop, []string{"g"}},
		{"GoToBottom", km.GoToBottom, []string{"G"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := tt.binding.Keys()
			if len(keys) == 0 {
				t.Errorf("%s: expected keys to be set", tt.name)
				return
			}
			found := false
			for _, k := range keys {
				if k == tt.keys[0] {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: expected key %q, got %v", tt.name, tt.keys[0], keys)
			}
		})
	}
}

func TestHandlePageDown_ScrollsViewport(t *testing.T) {
	// Create a model with enough items to scroll through
	m := Model{
		width:  80,
		height: 20,
		keys:   DefaultKeyMap(),
		head:   HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Test", Folded: false, Items: make([]Item, 50)},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.viewport.YOffset = 0

	// Initialize viewport content
	content, _ := renderContent(m)
	m.viewport.SetContent(content)

	result, _ := handlePageDown(m)
	resultModel := result.(Model)

	// Cursor should have moved down
	if resultModel.cursor.Item <= 0 {
		t.Errorf("expected cursor to move down, got Item=%d", resultModel.cursor.Item)
	}
}

func TestHandlePageUp_ScrollsViewport(t *testing.T) {
	// Create a model with enough items and cursor in middle
	m := Model{
		width:  80,
		height: 20,
		keys:   DefaultKeyMap(),
		head:   HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Test", Folded: false, Items: make([]Item, 50)},
		},
		cursor: Cursor{Section: 0, Item: 30, Hunk: -1, Line: -1}, // Start in middle
	}
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.viewport.YOffset = 20 // Scrolled down

	// Initialize viewport content
	content, _ := renderContent(m)
	m.viewport.SetContent(content)

	result, _ := handlePageUp(m)
	resultModel := result.(Model)

	// Cursor should have moved up (wrapping is ok, just shouldn't be at 30)
	if resultModel.cursor.Item == 30 {
		t.Errorf("expected cursor to move up from Item=30, but it stayed at %d", resultModel.cursor.Item)
	}
}

func TestHandleGoToTop_MovesToFirstSection(t *testing.T) {
	m := Model{
		keys: DefaultKeyMap(),
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked", Items: []Item{{}}},
			{Kind: SectionUnstaged, Title: "Unstaged", Items: []Item{{}}},
		},
		cursor: Cursor{Section: 1, Item: 0, Hunk: -1, Line: -1}, // On second section item
	}
	m.viewport.Width = 80
	m.viewport.Height = 20
	m.viewport.YOffset = 10

	result, _ := handleGoToTop(m)
	resultModel := result.(Model)

	// Cursor should be on first section header
	if resultModel.cursor.Section != 0 || resultModel.cursor.Item != -1 {
		t.Errorf("expected cursor on first section header, got Section=%d, Item=%d",
			resultModel.cursor.Section, resultModel.cursor.Item)
	}
	// Viewport should scroll to top
	if resultModel.viewport.YOffset != 0 {
		t.Errorf("expected YOffset=0, got %d", resultModel.viewport.YOffset)
	}
}

func TestHandleGoToBottom_MovesToLastItem(t *testing.T) {
	m := Model{
		keys: DefaultKeyMap(),
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked", Items: []Item{{}}},
			{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}, {}, {}}},
		},
		cursor: Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}, // On first section header
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	result, _ := handleGoToBottom(m)
	resultModel := result.(Model)

	// Cursor should be on last item of last section
	if resultModel.cursor.Section != 1 || resultModel.cursor.Item != 2 {
		t.Errorf("expected cursor on last item (Section=1, Item=2), got Section=%d, Item=%d",
			resultModel.cursor.Section, resultModel.cursor.Item)
	}
}

func TestView_UsesViewport(t *testing.T) {
	// After initialization, View() should return viewport content
	m := Model{
		width:  80,
		height: 20,
		head:   HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Items: []Item{}},
		},
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// Render content and set it in viewport
	content, _ := renderContent(m)
	m.viewport.SetContent(content)

	output := m.View()

	// Output should contain the HEAD bar content
	if !strings.Contains(output, "main") {
		t.Error("expected viewport output to contain branch name")
	}
}

func TestGGSequence_GoesToTop(t *testing.T) {
	// Test that pressing "g" twice triggers GoToTop
	m := Model{
		keys: DefaultKeyMap(),
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked", Items: []Item{{}}},
			{Kind: SectionUnstaged, Title: "Unstaged", Items: []Item{{}}},
		},
		cursor: Cursor{Section: 1, Item: 0, Hunk: -1, Line: -1}, // Start on second section
	}
	m.viewport.Width = 80
	m.viewport.Height = 20

	// First "g" press - should set pending key
	result, _ := handleKeyMsg(m, keyMsg("g"))
	resultModel := result.(Model)

	if resultModel.pendingKey != "g" {
		t.Errorf("expected pendingKey='g' after first g press, got %q", resultModel.pendingKey)
	}
	// Cursor should not have moved yet
	if resultModel.cursor.Section != 1 {
		t.Errorf("expected cursor to stay at Section=1 after first g, got Section=%d", resultModel.cursor.Section)
	}

	// Second "g" press - should trigger GoToTop
	result, _ = handleKeyMsg(resultModel, keyMsg("g"))
	resultModel = result.(Model)

	// Pending key should be cleared
	if resultModel.pendingKey != "" {
		t.Errorf("expected pendingKey='' after gg, got %q", resultModel.pendingKey)
	}
	// Cursor should be on first section header
	if resultModel.cursor.Section != 0 || resultModel.cursor.Item != -1 {
		t.Errorf("expected cursor at Section=0, Item=-1 after gg, got Section=%d, Item=%d",
			resultModel.cursor.Section, resultModel.cursor.Item)
	}
}

// === Block Cursor Tests ===

func TestRenderWithBlockCursor_EmptyLine(t *testing.T) {
	tokens := Tokens{
		CursorBlock: lipgloss.NewStyle().Reverse(true),
		Cursor:      lipgloss.NewStyle().Background(lipgloss.Color("#333333")),
	}

	result := renderWithBlockCursor(tokens, "")

	// Empty line should render a single space with the CursorBlock style
	// Note: In test environment without TTY, ANSI codes may not be emitted
	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("expected result to end with newline")
	}
	// Should contain at least a space (the block cursor)
	if len(result) < 2 { // minimum: " " + "\n"
		t.Errorf("expected result to contain at least space+newline, got len=%d", len(result))
	}
}

func TestRenderWithBlockCursor_SingleChar(t *testing.T) {
	tokens := Tokens{
		CursorBlock: lipgloss.NewStyle().Reverse(true),
		Cursor:      lipgloss.NewStyle().Background(lipgloss.Color("#333333")),
	}

	result := renderWithBlockCursor(tokens, "X")

	// Should contain the character
	if !strings.Contains(result, "X") {
		t.Error("expected character X in output")
	}
	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("expected result to end with newline")
	}
}

func TestRenderWithBlockCursor_MultiCharLine(t *testing.T) {
	tokens := Tokens{
		CursorBlock: lipgloss.NewStyle().Reverse(true),
		Cursor:      lipgloss.NewStyle().Background(lipgloss.Color("#333333")),
	}

	result := renderWithBlockCursor(tokens, "Hello")

	// Should contain 'H' (first char)
	if !strings.Contains(result, "H") {
		t.Error("expected first character H in output")
	}
	// Should contain 'ello' (rest of line)
	if !strings.Contains(result, "ello") {
		t.Error("expected rest of line in output")
	}
	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("expected result to end with newline")
	}
}

func TestRenderWithBlockCursor_UTF8Rune(t *testing.T) {
	tokens := Tokens{
		CursorBlock: lipgloss.NewStyle().Reverse(true),
		Cursor:      lipgloss.NewStyle().Background(lipgloss.Color("#333333")),
	}

	result := renderWithBlockCursor(tokens, "世界")

	// Should handle multi-byte UTF-8 runes correctly
	// First rune is "世" which is 3 bytes
	if !strings.Contains(result, "世") {
		t.Error("expected first UTF-8 rune in output")
	}
	if !strings.Contains(result, "界") {
		t.Error("expected second UTF-8 rune in output")
	}
}
