package status

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/commitview"
	"github.com/mhersson/conjit/internal/ui/notification"
	"github.com/mhersson/conjit/internal/ui/popup"
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

func TestMoveCursor_StayAtBottomBoundary(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 1, Item: 0, Hunk: -1, Line: -1} // Last item of last section

	result := moveCursor(sections, cursor, 1)

	// Should stay at boundary, not wrap
	if result.Section != 1 || result.Item != 0 {
		t.Errorf("expected to stay at Section=1, Item=0, got Section=%d, Item=%d", result.Section, result.Item)
	}
}

func TestMoveCursor_StayAtTopBoundary(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked", Folded: false, Items: []Item{{}}},
		{Kind: SectionUnstaged, Title: "Unstaged", Folded: false, Items: []Item{{}}},
	}
	cursor := Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}

	result := moveCursor(sections, cursor, -1)

	// Should stay at boundary, not wrap
	if result.Section != 0 || result.Item != -1 {
		t.Errorf("expected to stay at Section=0, Item=-1, got Section=%d, Item=%d", result.Section, result.Item)
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

func TestView_HeadBar_SingleSpaceBeforeSubject(t *testing.T) {
	m := Model{
		head: HeadState{
			Branch:    "main",
			AbbrevOid: "abc1234",
			Subject:   "add config loader",
		},
		sections: []Section{},
	}

	output := view(m)

	// Neogit uses single space between branch and subject
	if !contains(output, "main add config loader") {
		t.Errorf("expected single space between branch and subject, got:\n%s", output)
	}
	if contains(output, "main  add config loader") {
		t.Error("found double space between branch and subject, should be single space")
	}
}

func TestView_HeadBar_MergeLine_SingleSpaceBeforeSubject(t *testing.T) {
	m := Model{
		head: HeadState{
			Branch:          "main",
			AbbrevOid:       "abc1234",
			Subject:         "head commit",
			UpstreamRemote:  "origin",
			UpstreamBranch:  "main",
			UpstreamOid:     "def4567890",
			UpstreamSubject: "upstream commit",
		},
		sections: []Section{},
	}

	output := view(m)

	// Single space between remote ref and subject
	if !contains(output, "origin/main upstream commit") {
		t.Errorf("expected single space between remote and subject in Merge line, got:\n%s", output)
	}
	if contains(output, "origin/main  upstream commit") {
		t.Error("found double space in Merge line")
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

// === Cursor Restore Tests ===

func makeEntry(path string) *git.StatusEntry {
	e := git.NewStatusEntry(path, git.FileStatusNone, git.FileStatusNone)
	return &e
}

func TestRestoreCursor_FileFoundInExpectedSection(t *testing.T) {
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked files", Items: []Item{
			{Entry: makeEntry("a.txt")},
			{Entry: makeEntry("b.txt")},
		}},
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("c.txt")},
			{Entry: makeEntry("d.txt")},
		}},
	}

	restore := cursorRestore{
		active:      true,
		path:        "d.txt",
		sectionKind: SectionUnstaged,
		hunk:        -1,
	}

	cur := restoreCursor(sections, restore)

	if cur.Section != 1 {
		t.Errorf("expected Section=1, got %d", cur.Section)
	}
	if cur.Item != 1 {
		t.Errorf("expected Item=1, got %d", cur.Item)
	}
	if cur.Hunk != -1 {
		t.Errorf("expected Hunk=-1, got %d", cur.Hunk)
	}
}

func TestRestoreCursor_FileMovedToOtherSection(t *testing.T) {
	// After staging "c.txt", it moves from unstaged to staged.
	// The cursor should stay at the same item index in the original section,
	// or clamp to the last item if the section shrunk.
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("d.txt")},
		}},
		{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
			{Entry: makeEntry("c.txt")},
		}},
	}

	restore := cursorRestore{
		active:      true,
		path:        "c.txt",
		sectionKind: SectionUnstaged, // it was in unstaged before staging
		hunk:        -1,
	}

	cur := restoreCursor(sections, restore)

	// File is no longer in unstaged. Cursor should land on item 0 of unstaged
	// (the item now at the position, or clamped).
	if cur.Section != 0 {
		t.Errorf("expected Section=0 (original section), got %d", cur.Section)
	}
	if cur.Item != 0 {
		t.Errorf("expected Item=0 (clamped to remaining item), got %d", cur.Item)
	}
}

func TestRestoreCursor_SectionGone(t *testing.T) {
	// After staging the only unstaged file, the unstaged section becomes empty.
	// Cursor should fall back to the next available section.
	sections := []Section{
		{Kind: SectionUntracked, Title: "Untracked files", Hidden: true},
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{}},
		{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
			{Entry: makeEntry("c.txt")},
		}},
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Entry: makeEntry("dummy")},
		}},
	}

	restore := cursorRestore{
		active:      true,
		path:        "c.txt",
		sectionKind: SectionUnstaged, // was the only file in unstaged
		hunk:        -1,
	}

	cur := restoreCursor(sections, restore)

	// Original section is empty, cursor should move to next visible section
	// (SectionStaged at index 2)
	if cur.Section == 1 && cur.Item == -1 {
		// Acceptable: on the empty section header
	} else if cur.Section == 2 {
		// Also acceptable: moved to staged section
	} else {
		t.Errorf("expected cursor on staged section (2) or on empty unstaged header, got Section=%d Item=%d", cur.Section, cur.Item)
	}
}

func TestRestoreCursor_HunkRestore(t *testing.T) {
	// After staging a hunk, cursor should land on the file item (not hunk)
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("a.txt")},
			{Entry: makeEntry("big.txt"), Hunks: []git.Hunk{{}, {}}},
		}},
	}

	restore := cursorRestore{
		active:      true,
		path:        "big.txt",
		sectionKind: SectionUnstaged,
		hunk:        1, // was on hunk 1
	}

	cur := restoreCursor(sections, restore)

	if cur.Section != 0 {
		t.Errorf("expected Section=0, got %d", cur.Section)
	}
	if cur.Item != 1 {
		t.Errorf("expected Item=1, got %d", cur.Item)
	}
	// After reload, cursor should be on the file, not on a specific hunk
	if cur.Hunk != -1 {
		t.Errorf("expected Hunk=-1 (on file, not hunk), got %d", cur.Hunk)
	}
}

func TestRestoreCursor_FallbackToFindFirst(t *testing.T) {
	// When nothing matches at all, fall back to findFirstValidCursor
	sections := []Section{
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Entry: makeEntry("whatever")},
		}},
	}

	restore := cursorRestore{
		active:      true,
		path:        "gone.txt",
		sectionKind: SectionUnstaged,
		hunk:        -1,
	}

	cur := restoreCursor(sections, restore)

	// Should fall back to first valid cursor
	if cur.Section != 0 {
		t.Errorf("expected fallback Section=0, got %d", cur.Section)
	}
	if cur.Item != -1 {
		t.Errorf("expected fallback Item=-1 (header), got %d", cur.Item)
	}
}

func TestRestoreCursor_ClampsItemIndex(t *testing.T) {
	// Original section had 3 items, cursor was on item 2.
	// After staging item 2, section now has 2 items. Cursor should clamp.
	sections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("a.txt")},
			{Entry: makeEntry("b.txt")},
		}},
	}

	restore := cursorRestore{
		active:      true,
		path:        "c.txt",            // no longer in unstaged
		sectionKind: SectionUnstaged,
		itemIndex:   2,                  // was at index 2
		hunk:        -1,
	}

	cur := restoreCursor(sections, restore)

	if cur.Section != 0 {
		t.Errorf("expected Section=0, got %d", cur.Section)
	}
	// Item index 2 is out of bounds (only 2 items: 0,1), clamp to last item
	if cur.Item != 1 {
		t.Errorf("expected Item=1 (clamped), got %d", cur.Item)
	}
}

func TestHandleStage_SetsPendingRestore(t *testing.T) {
	entry := makeEntry("test.go")
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: entry},
				{Entry: makeEntry("other.go")},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}

	result, cmd := handleStage(m)
	resultModel := result.(Model)

	if cmd == nil {
		t.Fatal("expected a command from handleStage")
	}
	if !resultModel.pendingRestore.active {
		t.Error("expected pendingRestore.active=true after stage")
	}
	if resultModel.pendingRestore.path != "test.go" {
		t.Errorf("expected pendingRestore.path=test.go, got %s", resultModel.pendingRestore.path)
	}
	if resultModel.pendingRestore.sectionKind != SectionUnstaged {
		t.Errorf("expected pendingRestore.sectionKind=SectionUnstaged, got %d", resultModel.pendingRestore.sectionKind)
	}
	if resultModel.pendingRestore.itemIndex != 0 {
		t.Errorf("expected pendingRestore.itemIndex=0, got %d", resultModel.pendingRestore.itemIndex)
	}
}

func TestHandleUnstage_SetsPendingRestore(t *testing.T) {
	entry := makeEntry("staged.go")
	m := Model{
		sections: []Section{
			{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
				{Entry: makeEntry("first.go")},
				{Entry: entry},
			}},
		},
		cursor: Cursor{Section: 0, Item: 1, Hunk: -1, Line: -1},
	}

	result, cmd := handleUnstage(m)
	resultModel := result.(Model)

	if cmd == nil {
		t.Fatal("expected a command from handleUnstage")
	}
	if !resultModel.pendingRestore.active {
		t.Error("expected pendingRestore.active=true after unstage")
	}
	if resultModel.pendingRestore.path != "staged.go" {
		t.Errorf("expected pendingRestore.path=staged.go, got %s", resultModel.pendingRestore.path)
	}
	if resultModel.pendingRestore.sectionKind != SectionStaged {
		t.Errorf("expected pendingRestore.sectionKind=SectionStaged, got %d", resultModel.pendingRestore.sectionKind)
	}
}

func TestHandleStage_HunkSetsPendingRestore(t *testing.T) {
	entry := makeEntry("patched.go")
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: entry, Hunks: []git.Hunk{{}, {}, {}}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: 1, Line: -1},
	}

	result, cmd := handleStage(m)
	resultModel := result.(Model)

	if cmd == nil {
		t.Fatal("expected a command from handleStage (hunk)")
	}
	if !resultModel.pendingRestore.active {
		t.Error("expected pendingRestore.active=true after hunk stage")
	}
	if resultModel.pendingRestore.hunk != 1 {
		t.Errorf("expected pendingRestore.hunk=1, got %d", resultModel.pendingRestore.hunk)
	}
}

func TestExecuteConfirmedAction_SetsPendingRestore(t *testing.T) {
	entry := makeEntry("discard.go")
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: entry},
			}},
		},
		cursor:      Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		confirmMode: ConfirmDiscard,
		confirmPath: "discard.go",
	}

	result, _ := executeConfirmedAction(m)
	resultModel := result.(Model)

	if !resultModel.pendingRestore.active {
		t.Error("expected pendingRestore.active=true after confirmed discard")
	}
	if resultModel.pendingRestore.path != "discard.go" {
		t.Errorf("expected pendingRestore.path=discard.go, got %s", resultModel.pendingRestore.path)
	}
}

func TestStatusLoadedMsg_UsesPendingRestore(t *testing.T) {
	m := Model{
		loading: true,
		pendingRestore: cursorRestore{
			active:      true,
			path:        "target.go",
			sectionKind: SectionUnstaged,
			itemIndex:   0,
			hunk:        -1,
		},
	}

	newSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("target.go")},
			{Entry: makeEntry("other.go")},
		}},
	}

	result, _ := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main"},
		sections: newSections,
	})
	resultModel := result.(Model)

	// Should restore cursor to "target.go" position
	if resultModel.cursor.Section != 0 {
		t.Errorf("expected cursor Section=0, got %d", resultModel.cursor.Section)
	}
	if resultModel.cursor.Item != 0 {
		t.Errorf("expected cursor Item=0, got %d", resultModel.cursor.Item)
	}
	// pendingRestore should be cleared
	if resultModel.pendingRestore.active {
		t.Error("expected pendingRestore to be cleared after restore")
	}
}

func TestStatusLoadedMsg_WithoutRestore_UsesDefault(t *testing.T) {
	m := Model{loading: true}

	newSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("a.go")},
		}},
	}

	result, _ := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main"},
		sections: newSections,
	})
	resultModel := result.(Model)

	// Without pending restore, should use findFirstValidCursor (section header)
	if resultModel.cursor.Item != -1 {
		t.Errorf("expected cursor on section header (Item=-1), got %d", resultModel.cursor.Item)
	}
}

// --- Hunk-level cursor restore tests ---

func TestStatusLoadedMsg_HunkRestore_ExpandsFileAndTriggersDiffLoad(t *testing.T) {
	// When restoring after a hunk operation, the file should be expanded
	// and a hunk load command should be triggered.
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: makeEntry("file.go")},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: 1, Line: -1},
		pendingRestore: cursorRestore{
			active:      true,
			path:        "file.go",
			sectionKind: SectionUnstaged,
			itemIndex:   0,
			hunk:        1,
		},
	}

	newSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("file.go")},
		}},
	}

	result, cmd := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main"},
		sections: newSections,
	})
	resultModel := result.(Model)

	// File should be expanded (to show hunks once loaded)
	if !resultModel.sections[0].Items[0].Expanded {
		t.Error("expected file to be expanded for hunk restore")
	}

	// Should have a pending hunk restore
	if !resultModel.pendingHunkRestore.active {
		t.Error("expected pendingHunkRestore.active=true")
	}
	if resultModel.pendingHunkRestore.hunkIdx != 1 {
		t.Errorf("expected pendingHunkRestore.hunkIdx=1, got %d", resultModel.pendingHunkRestore.hunkIdx)
	}

	// Should return a command (to load hunks)
	if cmd == nil {
		t.Error("expected a command to load hunks, got nil")
	}

	// Cursor should be on the file item while hunks are loading
	if resultModel.cursor.Item != 0 {
		t.Errorf("expected cursor on item 0, got %d", resultModel.cursor.Item)
	}
}

func TestHunksLoadedMsg_WithPendingHunkRestore_PlacesCursorOnHunk(t *testing.T) {
	// After hunks load, cursor should be placed on the target hunk.
	testHunks := []git.Hunk{
		{Header: "@@ -1,3 +1,3 @@"},
		{Header: "@@ -10,5 +10,5 @@"},
		{Header: "@@ -20,3 +20,3 @@"},
	}

	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: makeEntry("file.go"), Expanded: true, HunksLoading: true},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		pendingHunkRestore: hunkRestore{
			active:     true,
			sectionIdx: 0,
			itemIdx:    0,
			hunkIdx:    1,
		},
	}

	result, _ := update(m, hunksLoadedMsg{
		sectionIdx: 0,
		itemIdx:    0,
		hunks:      testHunks,
	})
	resultModel := result.(Model)

	// Cursor should be on hunk 1
	if resultModel.cursor.Hunk != 1 {
		t.Errorf("expected cursor.Hunk=1, got %d", resultModel.cursor.Hunk)
	}
	if resultModel.cursor.Line != -1 {
		t.Errorf("expected cursor.Line=-1 (hunk header), got %d", resultModel.cursor.Line)
	}

	// pendingHunkRestore should be cleared
	if resultModel.pendingHunkRestore.active {
		t.Error("expected pendingHunkRestore to be cleared")
	}
}

func TestHunksLoadedMsg_WithPendingHunkRestore_ClampsToLastHunk(t *testing.T) {
	// If original hunk index exceeds available hunks, clamp to last.
	testHunks := []git.Hunk{
		{Header: "@@ -1,3 +1,3 @@"},
	}

	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: makeEntry("file.go"), Expanded: true, HunksLoading: true},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		pendingHunkRestore: hunkRestore{
			active:     true,
			sectionIdx: 0,
			itemIdx:    0,
			hunkIdx:    3, // was on hunk 3, but only 1 remains
		},
	}

	result, _ := update(m, hunksLoadedMsg{
		sectionIdx: 0,
		itemIdx:    0,
		hunks:      testHunks,
	})
	resultModel := result.(Model)

	// Should clamp to last available hunk (0)
	if resultModel.cursor.Hunk != 0 {
		t.Errorf("expected cursor.Hunk=0 (clamped), got %d", resultModel.cursor.Hunk)
	}
}

func TestHunksLoadedMsg_WithPendingHunkRestore_NoHunksLeft(t *testing.T) {
	// If no hunks remain after the operation, cursor stays on file item.
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: makeEntry("file.go"), Expanded: true, HunksLoading: true},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		pendingHunkRestore: hunkRestore{
			active:     true,
			sectionIdx: 0,
			itemIdx:    0,
			hunkIdx:    0,
		},
	}

	result, _ := update(m, hunksLoadedMsg{
		sectionIdx: 0,
		itemIdx:    0,
		hunks:      nil, // no hunks
	})
	resultModel := result.(Model)

	// Cursor should be on file item (Hunk=-1)
	if resultModel.cursor.Hunk != -1 {
		t.Errorf("expected cursor.Hunk=-1 (on file), got %d", resultModel.cursor.Hunk)
	}

	// pendingHunkRestore should be cleared
	if resultModel.pendingHunkRestore.active {
		t.Error("expected pendingHunkRestore to be cleared")
	}
}

func TestStatusLoadedMsg_HintBarVisibleAfterStageFromScrolledDiff(t *testing.T) {
	// Regression: staging a file while scrolled down (e.g. viewing expanded
	// inline diff) left viewport.YOffset stale, hiding the hint bar.
	cfg, _ := config.Load()
	tokens := theme.Compile(theme.Fallback().Raw())

	m := Model{
		cfg:     cfg,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		loading: true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 30
	m.width = 80
	m.height = 30

	// Initial load with unstaged files.
	initialSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("file1.go")},
			{Entry: makeEntry("file2.go")},
		}},
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Commit: &git.LogEntry{AbbreviatedHash: "abc1234", Subject: "test"}},
		}},
	}

	result, _ := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: initialSections,
	})
	m = result.(Model)

	// Expand the file's inline diff (many lines to force scrolling).
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}
	m.sections[0].Items[0].Expanded = true
	m.sections[0].Items[0].Hunks = []git.Hunk{
		{Header: "@@ -1,30 +1,30 @@", Lines: makeDiffLines(40)},
	}

	// Navigate to a diff line deep in the expanded hunk.
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: 30}
	content, cursorLine := renderContent(m)
	m.viewport.SetContent(content)
	ensureCursorVisible(&m, cursorLine)

	// Sanity: viewport should now be scrolled down.
	if m.viewport.YOffset == 0 {
		t.Fatal("precondition: expected viewport to be scrolled down")
	}

	// Stage the file → save pendingRestore (same as handleStage).
	m.pendingRestore = cursorRestore{
		active:      true,
		path:        "file1.go",
		sectionKind: SectionUnstaged,
		itemIndex:   0,
		hunk:        0,
	}

	// Simulate statusLoadedMsg after staging: expanded state gone, file moved.
	postStageSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("file2.go")},
		}},
		{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
			{Entry: makeEntry("file1.go")},
		}},
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Commit: &git.LogEntry{AbbreviatedHash: "abc1234", Subject: "test"}},
		}},
	}

	result2, _ := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: postStageSections,
	})
	m = result2.(Model)

	// The hint bar must be visible in the viewport.
	viewContent := m.viewport.View()
	if !strings.Contains(viewContent, "Hint:") {
		t.Errorf("hint bar missing from viewport after stage+reload (YOffset=%d, cursor=%+v)",
			m.viewport.YOffset, m.cursor)
	}
}

func makeDiffLines(n int) []git.DiffLine {
	lines := make([]git.DiffLine, n)
	for i := range lines {
		switch i % 3 {
		case 0:
			lines[i] = git.DiffLine{Op: git.DiffOpAdd, Content: "added line"}
		case 1:
			lines[i] = git.DiffLine{Op: git.DiffOpDelete, Content: "deleted line"}
		default:
			lines[i] = git.DiffLine{Op: git.DiffOpContext, Content: "context line"}
		}
	}
	return lines
}

func TestHunksLoadedMsg_HintBarVisibleAfterHunkStageRestore(t *testing.T) {
	// Regression: after staging a hunk, the two-phase restore (statusLoadedMsg
	// expands the file, hunksLoadedMsg places cursor on hunk) used
	// preserveScreenPosition which pushed YOffset > 0, hiding the hint bar.
	cfg, _ := config.Load()
	tokens := theme.Compile(theme.Fallback().Raw())

	m := Model{
		cfg:     cfg,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		loading: true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 30
	m.width = 80
	m.height = 30

	// Initial load with an unstaged file.
	initialSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("file1.go")},
		}},
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Commit: &git.LogEntry{AbbreviatedHash: "abc1234", Subject: "test"}},
		}},
	}

	result, _ := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: initialSections,
	})
	m = result.(Model)

	// Simulate the state after statusLoadedMsg with hunk restore:
	// The file is expanded, hunks are loading, pendingHunkRestore is set.
	m.sections[0].Items[0].Expanded = true
	m.sections[0].Items[0].HunksLoading = true
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1}
	m.pendingHunkRestore = hunkRestore{
		active:     true,
		sectionIdx: 0,
		itemIdx:    0,
		hunkIdx:    0,
	}

	// Re-render viewport (YOffset should be 0 from statusLoadedMsg fix).
	content, cursorLine := renderContent(m)
	m.viewport.SetContent(content)
	m.viewport.YOffset = 0
	ensureCursorVisible(&m, cursorLine)

	// Now hunksLoadedMsg arrives with the loaded hunks.
	hunks := []git.Hunk{
		{Header: "@@ -1,10 +1,10 @@", Lines: makeDiffLines(10)},
		{Header: "@@ -20,5 +20,8 @@", Lines: makeDiffLines(5)},
	}

	result2, _ := update(m, hunksLoadedMsg{
		sectionIdx: 0,
		itemIdx:    0,
		hunks:      hunks,
	})
	m = result2.(Model)

	// The hint bar must still be visible.
	viewContent := m.viewport.View()
	if !strings.Contains(viewContent, "Hint:") {
		t.Errorf("hint bar missing from viewport after hunk restore (YOffset=%d, cursor=%+v)",
			m.viewport.YOffset, m.cursor)
	}
}

func TestStatusLoadedMsg_LastHunkStaged_NextFileNotExpanded(t *testing.T) {
	// Regression: when staging the last hunk of a file causes the file to move
	// from unstaged to staged, the next file in unstaged should NOT be
	// auto-expanded. The cursor should land on the next file's filename line
	// with fold state unchanged.
	cfg, _ := config.Load()
	tokens := theme.Compile(theme.Fallback().Raw())

	m := Model{
		cfg:     cfg,
		tokens:  tokens,
		keys:    DefaultKeyMap(),
		loading: true,
	}
	m.viewport.Width = 80
	m.viewport.Height = 30
	m.width = 80
	m.height = 30

	// Initial load: two unstaged files, file1 expanded with one hunk.
	initialSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("file1.go"), Expanded: true, Hunks: []git.Hunk{
				{Header: "@@ -1,5 +1,6 @@", Lines: []git.DiffLine{
					{Op: git.DiffOpContext, Content: "context"},
					{Op: git.DiffOpAdd, Content: "added"},
				}},
			}},
			{Entry: makeEntry("file2.go")}, // collapsed (Expanded=false)
		}},
		{Kind: SectionStaged, Title: "Staged changes", Items: []Item{}},
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Commit: &git.LogEntry{AbbreviatedHash: "abc1234", Subject: "test"}},
		}},
	}

	result, _ := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: initialSections,
	})
	m = result.(Model)

	// Place cursor on hunk 0 of file1.go (the only hunk).
	m.cursor = Cursor{Section: 0, Item: 0, Hunk: 0, Line: -1}

	// Simulate staging the hunk: save pendingRestore as handleStage would.
	m.pendingRestore = cursorRestore{
		active:      true,
		path:        "file1.go",
		sectionKind: SectionUnstaged,
		itemIndex:   0,
		hunk:        0, // was on hunk 0
	}

	// After staging the last hunk, file1.go moves entirely to staged.
	postStageSections := []Section{
		{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
			{Entry: makeEntry("file2.go")}, // collapsed
		}},
		{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
			{Entry: makeEntry("file1.go")},
		}},
		{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
			{Commit: &git.LogEntry{AbbreviatedHash: "abc1234", Subject: "test"}},
		}},
	}

	result2, cmd := update(m, statusLoadedMsg{
		head:     HeadState{Branch: "main", AbbrevOid: "abc1234"},
		sections: postStageSections,
	})
	m = result2.(Model)

	// Cursor should be on file2.go's filename, NOT on a hunk.
	if m.cursor.Section != 0 {
		t.Errorf("expected cursor Section=0 (unstaged), got %d", m.cursor.Section)
	}
	if m.cursor.Item != 0 {
		t.Errorf("expected cursor Item=0 (file2.go), got %d", m.cursor.Item)
	}
	if m.cursor.Hunk != -1 {
		t.Errorf("expected cursor Hunk=-1 (on filename, not hunk), got %d", m.cursor.Hunk)
	}
	if m.cursor.Line != -1 {
		t.Errorf("expected cursor Line=-1, got %d", m.cursor.Line)
	}

	// file2.go must NOT be expanded.
	if m.sections[0].Items[0].Expanded {
		t.Error("file2.go should NOT be expanded after staging last hunk of file1.go")
	}

	// No hunk-loading command should have been issued for file2.go.
	if cmd != nil {
		t.Error("expected no command (no hunk loading for the next file), but got one")
	}
}

// --- Commit action staged-changes guard tests ---

// executeCmd runs a tea.Cmd and returns the resulting message.
func executeCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	return cmd()
}

func TestCommitAction_NoStagedChanges_ShowsWarning(t *testing.T) {
	// Model with sections but no staged section (or empty staged)
	m := Model{
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Items: []Item{{}}},
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{{}}},
		},
	}

	result := popup.Result{
		Action:   "c",
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}

	_, cmd := handleCommitPopupAction(m, result)
	msg := executeCmd(t, cmd)

	notifyMsg, ok := msg.(notification.NotifyMsg)
	if !ok {
		t.Fatalf("expected notification.NotifyMsg, got %T", msg)
	}
	if notifyMsg.Kind != notification.Warning {
		t.Errorf("expected Warning kind, got %s", notifyMsg.Kind)
	}
	if notifyMsg.Message != "No changes to commit." {
		t.Errorf("expected 'No changes to commit.', got %q", notifyMsg.Message)
	}
}

func TestCommitAction_WithStagedChanges_OpensEditor(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{{}}},
			{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
				{Entry: &git.StatusEntry{}},
			}},
		},
	}

	result := popup.Result{
		Action:   "c",
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}

	_, cmd := handleCommitPopupAction(m, result)
	msg := executeCmd(t, cmd)

	if _, ok := msg.(openCommitEditorMsg); !ok {
		t.Fatalf("expected openCommitEditorMsg, got %T", msg)
	}
}

func TestCommitAction_AllowEmpty_BypassesGuard(t *testing.T) {
	// No staged changes, but --allow-empty is set
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{{}}},
		},
	}

	result := popup.Result{
		Action:   "c",
		Switches: map[string]bool{"allow-empty": true},
		Options:  map[string]string{},
	}

	_, cmd := handleCommitPopupAction(m, result)
	msg := executeCmd(t, cmd)

	if _, ok := msg.(openCommitEditorMsg); !ok {
		t.Fatalf("expected openCommitEditorMsg (allow-empty bypass), got %T", msg)
	}
}

func TestCommitAction_All_BypassesGuard(t *testing.T) {
	// No staged changes, but --all is set (stages everything)
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{{}}},
		},
	}

	result := popup.Result{
		Action:   "c",
		Switches: map[string]bool{"all": true},
		Options:  map[string]string{},
	}

	_, cmd := handleCommitPopupAction(m, result)
	msg := executeCmd(t, cmd)

	if _, ok := msg.(openCommitEditorMsg); !ok {
		t.Fatalf("expected openCommitEditorMsg (--all bypass), got %T", msg)
	}
}

func TestExtendAction_NoStagedChanges_ShowsWarning(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Items: []Item{{}}},
		},
	}

	result := popup.Result{
		Action:   "e",
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}

	_, cmd := handleCommitPopupAction(m, result)
	msg := executeCmd(t, cmd)

	notifyMsg, ok := msg.(notification.NotifyMsg)
	if !ok {
		t.Fatalf("expected notification.NotifyMsg, got %T", msg)
	}
	if notifyMsg.Kind != notification.Warning {
		t.Errorf("expected Warning kind, got %s", notifyMsg.Kind)
	}
}

func TestView_UntrackedFiles_NoModePadding(t *testing.T) {
	// Untracked file entries must render as "  > filename" without the
	// 12-char mode column that staged/unstaged entries have.
	// Neogit skips mode padding entirely when mode text is empty.
	untrackedEntry := git.NewStatusEntry("newfile.txt", git.FileStatusUntracked, git.FileStatusUntracked)
	stagedEntry := git.NewStatusEntry("changed.go", git.FileStatusModified, git.FileStatusNone)

	m := Model{
		head: HeadState{Branch: "main"},
		sections: []Section{
			{Kind: SectionUntracked, Title: "Untracked files", Folded: false, Items: []Item{
				{Entry: &untrackedEntry},
			}},
			{Kind: SectionStaged, Title: "Staged changes", Folded: false, Items: []Item{
				{Entry: &stagedEntry},
			}},
		},
	}

	output := view(m)
	lines := strings.Split(output, "\n")

	// Find the untracked file line
	var untrackedLine string
	for _, line := range lines {
		if strings.Contains(line, "newfile.txt") {
			untrackedLine = line
			break
		}
	}
	if untrackedLine == "" {
		t.Fatal("expected to find 'newfile.txt' in output")
	}

	// Untracked: should be "  > newfile.txt" — no mode column padding
	if untrackedLine != "  > newfile.txt" {
		t.Errorf("untracked file has wrong indentation:\ngot:  %q\nwant: %q", untrackedLine, "  > newfile.txt")
	}

	// Find the staged file line
	var stagedLine string
	for _, line := range lines {
		if strings.Contains(line, "changed.go") {
			stagedLine = line
			break
		}
	}
	if stagedLine == "" {
		t.Fatal("expected to find 'changed.go' in output")
	}

	// Staged: should still have the padded mode column "  > modified    changed.go"
	if !strings.Contains(stagedLine, "modified") {
		t.Error("staged file should contain 'modified' mode text")
	}
}

func TestHandleDiscardStart_SetsConfirmModeAndNotification(t *testing.T) {
	entry := makeEntry("dirty.go")
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: entry},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		keys:   DefaultKeyMap(),
	}

	result, _ := handleDiscardStart(m)
	rm := result.(Model)

	if rm.confirmMode != ConfirmDiscard {
		t.Errorf("expected confirmMode=ConfirmDiscard, got %d", rm.confirmMode)
	}
	if rm.confirmPath != "dirty.go" {
		t.Errorf("expected confirmPath=dirty.go, got %s", rm.confirmPath)
	}
	// Confirmation is now rendered via ConfirmView overlay, not m.notification
	v := rm.ConfirmView(60)
	if v == "" {
		t.Error("expected ConfirmView to return non-empty for active confirmation")
	}
	if !strings.Contains(v, "dirty.go") {
		t.Errorf("ConfirmView should mention file name, got: %s", v)
	}
}

func TestConfirmView_ReturnsEmpty_WhenNoConfirm(t *testing.T) {
	tokens := theme.Compile(theme.Fallback().Raw())
	m := Model{tokens: tokens, confirmMode: ConfirmNone}
	v := m.ConfirmView(60)
	if v != "" {
		t.Error("expected empty ConfirmView when confirmMode is ConfirmNone")
	}
}

func TestConfirmView_ReturnsNonEmpty_WhenConfirmDiscard(t *testing.T) {
	tokens := theme.Compile(theme.Fallback().Raw())
	m := Model{tokens: tokens, confirmMode: ConfirmDiscard, confirmPath: "main.go"}
	v := m.ConfirmView(60)
	if v == "" {
		t.Error("expected non-empty ConfirmView for ConfirmDiscard")
	}
	if !strings.Contains(v, "main.go") {
		t.Error("expected ConfirmView to mention file name")
	}
}

func TestConfirmView_ReturnsNonEmpty_WhenConfirmDiscardHunk(t *testing.T) {
	tokens := theme.Compile(theme.Fallback().Raw())
	m := Model{tokens: tokens, confirmMode: ConfirmDiscardHunk, confirmPath: "test.go", confirmHunk: 2}
	v := m.ConfirmView(60)
	if v == "" {
		t.Error("expected non-empty ConfirmView for ConfirmDiscardHunk")
	}
	if !strings.Contains(v, "test.go") {
		t.Error("expected ConfirmView to mention file name")
	}
}

func TestConfirmView_ReturnsNonEmpty_WhenConfirmUntrack(t *testing.T) {
	tokens := theme.Compile(theme.Fallback().Raw())
	m := Model{tokens: tokens, confirmMode: ConfirmUntrack, confirmPath: "tracked.go"}
	v := m.ConfirmView(60)
	if v == "" {
		t.Error("expected non-empty ConfirmView for ConfirmUntrack")
	}
	if !strings.Contains(v, "tracked.go") {
		t.Error("expected ConfirmView to mention file name")
	}
}

func TestHandleDiscardStart_ConfirmViewShowsPrompt(t *testing.T) {
	entry := makeEntry("dirty.go")
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Title: "Unstaged changes", Items: []Item{
				{Entry: entry},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		keys:   DefaultKeyMap(),
		cfg:    &config.Config{UI: config.UIConfig{DisableHint: true}},
		tokens: theme.Compile(theme.Fallback().Raw()),
	}
	m.viewport.Width = 80
	m.viewport.Height = 24

	result, _ := handleDiscardStart(m)
	rm := result.(Model)

	// Confirmation is now rendered as a centered overlay via ConfirmView
	v := rm.ConfirmView(60)
	if !strings.Contains(v, "Discard") {
		t.Error("ConfirmView should include the discard confirmation prompt")
	}
}

func TestHandleUntrackStart_ConfirmViewShowsPrompt(t *testing.T) {
	entry := makeEntry("tracked.go")
	m := Model{
		sections: []Section{
			{Kind: SectionStaged, Title: "Staged changes", Items: []Item{
				{Entry: entry},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		keys:   DefaultKeyMap(),
		cfg:    &config.Config{UI: config.UIConfig{DisableHint: true}},
		tokens: theme.Compile(theme.Fallback().Raw()),
	}
	m.viewport.Width = 80
	m.viewport.Height = 24

	result, _ := handleUntrackStart(m)
	rm := result.(Model)

	v := rm.ConfirmView(60)
	if !strings.Contains(v, "Untrack") {
		t.Error("ConfirmView should include the untrack confirmation prompt")
	}
}

// === Commit View Overlay Tests ===

func TestModel_CommitViewOverlay_InitiallyNil(t *testing.T) {
	m := New(nil, nil, Tokens{}, KeyMap{})
	if m.commitView != nil {
		t.Error("expected commitView to be nil initially")
	}
}

func TestHandleGoToFile_OpensCommitViewOverlay(t *testing.T) {
	commit := &git.LogEntry{
		Hash:            "abc123def456",
		AbbreviatedHash: "abc123d",
		Subject:         "test commit",
	}
	m := Model{
		sections: []Section{
			{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{
				{Commit: commit},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
		keys:   DefaultKeyMap(),
		tokens: theme.Compile(theme.Fallback().Raw()),
		width:  80,
		height: 40,
	}
	m.viewport.Width = 80
	m.viewport.Height = 40

	result, cmd := handleGoToFile(m)
	rm := result.(Model)

	// commitView should be set
	if rm.commitView == nil {
		t.Fatal("expected commitView to be set after GoToFile on commit")
	}

	// commitView should have the correct commit ID
	if rm.commitView.CommitID() != "abc123def456" {
		t.Errorf("expected commitID=abc123def456, got %s", rm.commitView.CommitID())
	}

	// Should return an init command
	if cmd == nil {
		t.Error("expected Init command to be returned")
	}
}

func TestHandleKeyMsg_DelegatesToCommitView(t *testing.T) {
	tokens := theme.Compile(theme.Fallback().Raw())
	cv := createTestCommitView(tokens)
	cv.SetSize(80, 20)

	m := Model{
		sections: []Section{
			{Kind: SectionRecentCommits, Title: "Recent Commits", Items: []Item{}},
		},
		cursor:     Cursor{Section: 0, Item: -1},
		keys:       DefaultKeyMap(),
		tokens:     tokens,
		width:      80,
		height:     40,
		commitView: &cv,
	}

	// Press 'q' to close commit view
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	result, _ := handleKeyMsg(m, keyMsg)
	rm := result.(Model)

	// commitView should be cleared after close
	if rm.commitView != nil {
		t.Error("expected commitView to be nil after pressing q")
	}
}

func TestHandleKeyMsg_CommitViewCursorMovement(t *testing.T) {
	tokens := theme.Compile(theme.Fallback().Raw())
	cv := createTestCommitView(tokens)
	cv.SetSize(80, 20)

	// Load data to enable cursor movement
	info := &git.LogEntry{
		Hash:        "abc123",
		Subject:     "test",
		AuthorName:  "Author",
		AuthorEmail: "a@b.com",
		AuthorDate:  "2024-01-01",
	}
	dataMsg := commitview.CommitDataLoadedMsg{Info: info, Overview: &git.CommitOverview{}}
	newCV, _ := cv.Update(dataMsg)
	cv = newCV.(commitview.Model)

	m := Model{
		sections:   []Section{},
		keys:       DefaultKeyMap(),
		tokens:     tokens,
		width:      80,
		height:     40,
		commitView: &cv,
	}

	// Press 'j' to move cursor down in commit view
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	_, _ = handleKeyMsg(m, keyMsg)

	// Just verify it doesn't crash - cursor movement is tested in commitview tests
}

func createTestCommitView(tokens theme.Tokens) commitview.Model {
	return commitview.New(nil, "abc123", tokens, nil)
}
