package status

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/platform"
	"github.com/mhersson/termagit/internal/ui/commitview"
	"github.com/mhersson/termagit/internal/ui/notification"
)

// In visual mode it stages only the selected line range within the current hunk.
func handleStage(m Model) (tea.Model, tea.Cmd) {
	// Visual mode: stage only the selected line range.
	if m.visualMode {
		return handleStageLineRange(m)
	}

	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}

	// Only stage from untracked or unstaged sections
	if sectionKind != SectionUntracked && sectionKind != SectionUnstaged {
		return m, nil
	}

	// Save cursor context for restore after reload
	m.pendingRestore = saveCursorContext(m)

	// If on a hunk, stage just the hunk
	if m.cursor.Hunk >= 0 && len(item.Hunks) > m.cursor.Hunk {
		return m, stageHunkCmd(m.repo, item.Entry.Path, item.Hunks[m.cursor.Hunk])
	}

	// Stage the whole file
	return m, stageFileCmd(m.repo, item.Entry.Path)
}

// handleStageUnstaged stages all unstaged files.
func handleStageUnstaged(m Model) (tea.Model, tea.Cmd) {
	return m, stageAllUnstagedCmd(m.repo)
}

// handleStageAll stages everything including untracked.
func handleStageAll(m Model) (tea.Model, tea.Cmd) {
	return m, stageAllUnstagedCmd(m.repo)
}

// handleUnstage unstages the current file or hunk.
// In visual mode it unstages only the selected line range within the current hunk.
func handleUnstage(m Model) (tea.Model, tea.Cmd) {
	// Visual mode: unstage only the selected line range.
	if m.visualMode {
		return handleUnstageLineRange(m)
	}

	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}

	// Only unstage from staged section
	if sectionKind != SectionStaged {
		return m, nil
	}

	// Save cursor context for restore after reload
	m.pendingRestore = saveCursorContext(m)

	// If on a hunk, unstage just the hunk
	if m.cursor.Hunk >= 0 && len(item.Hunks) > m.cursor.Hunk {
		return m, unstageHunkCmd(m.repo, item.Entry.Path, item.Hunks[m.cursor.Hunk])
	}

	// Unstage the whole file
	return m, unstageFileCmd(m.repo, item.Entry.Path)
}

// handleUnstageStaged unstages all staged files.
func handleUnstageStaged(m Model) (tea.Model, tea.Cmd) {
	return m, unstageAllStagedCmd(m.repo)
}

// handleDiscardStart initiates discard with confirmation.
func handleDiscardStart(m Model) (tea.Model, tea.Cmd) {
	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}

	// Only discard from unstaged section
	if sectionKind != SectionUnstaged {
		return m, notifyAppCmd("Can only discard unstaged changes", notification.Warning)
	}

	path := item.Entry.Path

	// Check if on a hunk
	if m.cursor.Hunk >= 0 && len(item.Hunks) > m.cursor.Hunk {
		m.confirmMode = ConfirmDiscardHunk
		m.confirmPath = path
		m.confirmHunk = m.cursor.Hunk
	} else {
		m.confirmMode = ConfirmDiscard
		m.confirmPath = path
	}

	// Refresh viewport so confirmation prompt is visible
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handleUntrackStart initiates untrack with confirmation.
func handleUntrackStart(m Model) (tea.Model, tea.Cmd) {
	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}

	// Only untrack from staged section (remove from index)
	if sectionKind != SectionStaged {
		return m, notifyAppCmd("Can only untrack staged files", notification.Warning)
	}

	path := item.Entry.Path
	m.confirmMode = ConfirmUntrack
	m.confirmPath = path

	// Refresh viewport so confirmation prompt is visible
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handleYank copies the current selection to clipboard.
func handleYank(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil {
		return m, nil
	}

	var text string
	if item.Entry != nil {
		text = item.Entry.Path
	} else if item.Commit != nil {
		text = item.Commit.AbbreviatedHash
	} else if item.Stash != nil {
		text = item.Stash.Name
	} else if item.ActionHash != "" {
		text = item.ActionHash
	}

	if text == "" {
		return m, nil
	}

	return m, func() tea.Msg {
		if err := platform.CopyToClipboard(text); err != nil {
			return notification.NotifyMsg{
				Message: "Failed to copy to clipboard: " + err.Error(),
				Kind:    notification.Error,
			}
		}
		return notification.NotifyMsg{
			Message: "Yanked: " + text,
			Kind:    notification.Info,
		}
	}
}

// handleOpenTree opens the directory containing the current file.
func handleOpenTree(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}

	return m, openTreeCmd(m.repo.Path(), item.Entry.Path)
}

// handleGoToFile opens the commit view for commits, or the file for files.
func handleGoToFile(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil {
		return m, nil
	}

	// If it's a commit, open commit view as overlay
	if item.Commit != nil {
		cv := commitview.New(m.repo, item.Commit.Hash, m.tokens, nil)
		cv.SetSize(m.width, m.height*60/100)
		cv.SetOverlayMode(true)
		m.commitView = &cv
		return m, cv.Init()
	}

	// If it's a file, open in $EDITOR
	if item.Entry != nil {
		repoPath := ""
		if m.repo != nil {
			repoPath = m.repo.Path()
		}
		return m, openInEditorCmd(repoPath, item.Entry.Path)
	}

	return m, nil
}

// getCurrentItem returns the current item and its section kind.
func getCurrentItem(m Model) (*Item, SectionKind) {
	if m.cursor.Section >= len(m.sections) {
		return nil, 0
	}

	s := &m.sections[m.cursor.Section]
	if m.cursor.Item < 0 || m.cursor.Item >= len(s.Items) {
		return nil, s.Kind
	}

	return &s.Items[m.cursor.Item], s.Kind
}

// findHunkByPathAndIndex searches all sections for a file matching path
// and returns the hunk at the given index, or nil if not found.
func findHunkByPathAndIndex(m Model, path string, hunkIdx int) *git.Hunk {
	for _, s := range m.sections {
		for _, item := range s.Items {
			if item.Entry != nil && item.Entry.Path == path {
				if hunkIdx >= 0 && hunkIdx < len(item.Hunks) {
					return &item.Hunks[hunkIdx]
				}
			}
		}
	}
	return nil
}

// cursorOnHunk returns true if the cursor is positioned on an expanded hunk.
func cursorOnHunk(m Model) bool {
	sec := m.cursor.Section
	item := m.cursor.Item
	if sec < 0 || sec >= len(m.sections) {
		return false
	}
	s := m.sections[sec]
	if item < 0 || item >= len(s.Items) {
		return false
	}
	return s.Items[item].Expanded && len(s.Items[item].Hunks) > 0
}

// handleShowRefs opens the refs view (matching Neogit y = ShowRefs).
func handleShowRefs(m Model) (tea.Model, tea.Cmd) {
	return m, loadRefsCmd(m.repo)
}

// handleGoToParentRepo navigates to the parent repository if in a submodule.
func handleGoToParentRepo(m Model) (tea.Model, tea.Cmd) {
	return m, goToParentRepoCmd(m.repo)
}

// goToParentRepoCmd attempts to find a parent repository.
func goToParentRepoCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		parent, err := repo.ParentRepo(context.Background())
		if err != nil || parent == "" {
			return notification.NotifyMsg{Message: "Not in a submodule", Kind: notification.Info}
		}
		return notification.NotifyMsg{Message: "Parent repo: " + parent, Kind: notification.Info}
	}
}

// handleRenameFile opens a text input prompt to rename the current file item.
func handleRenameFile(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, notifyAppCmd("No file selected", notification.Warning)
	}

	m.confirmPath = item.Entry.Path
	return openBranchInput(m, inputPromptRenameFile, "Rename "+item.Entry.Path+" to: ")
}

// handlePeekFile loads file content for a read-only overlay.
func handlePeekFile(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}
	return m, loadPeekFileCmd(m.repo.Path(), item.Entry.Path)
}

// handleOpenOrScrollDown scrolls the commit view overlay down, or opens one for the current commit.
func handleOpenOrScrollDown(m Model) (tea.Model, tea.Cmd) {
	// When commit view overlay is NOT open, open it for the current commit
	return openCommitViewForCurrentItem(m)
}

// handleOpenOrScrollUp scrolls the commit view overlay up, or opens one for the current commit.
func handleOpenOrScrollUp(m Model) (tea.Model, tea.Cmd) {
	// When commit view overlay is NOT open, open it for the current commit
	return openCommitViewForCurrentItem(m)
}

// handlePeekDown moves the cursor down and opens/updates the commit view overlay.
func handlePeekDown(m Model) (tea.Model, tea.Cmd) {
	m.cursor = moveCursor(m.sections, m.cursor, 1)
	return openCommitViewForCurrentItem(m)
}

// handlePeekUp moves the cursor up and opens/updates the commit view overlay.
func handlePeekUp(m Model) (tea.Model, tea.Cmd) {
	m.cursor = moveCursor(m.sections, m.cursor, -1)
	return openCommitViewForCurrentItem(m)
}

// openCommitViewForCurrentItem opens a commit view overlay for the current commit item.
func openCommitViewForCurrentItem(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Commit == nil {
		return m, nil
	}
	cv := commitview.New(m.repo, item.Commit.Hash, m.tokens, nil)
	cv.SetSize(m.width, m.height*60/100)
	cv.SetOverlayMode(true)
	m.commitView = &cv
	return m, cv.Init()
}

// handleRevertHunk reverts the hunk under the cursor using a reverse patch.
func handleRevertHunk(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil || !item.Expanded || m.cursor.Hunk < 0 {
		return m, notifyAppCmd("No hunk selected", notification.Warning)
	}
	if m.cursor.Hunk >= len(item.Hunks) {
		return m, notifyAppCmd("No hunk selected", notification.Warning)
	}
	hunk := item.Hunks[m.cursor.Hunk]
	// Apply the hunk in reverse to the worktree
	return m, discardHunkCmd(m.repo, item.Entry.Path, hunk)
}

// getStashIndex returns the stash index for the item under the cursor, if any.
func getStashIndex(m Model) (int, bool) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Stash == nil {
		return 0, false
	}
	return item.Stash.Index, true
}

// getCursorFilePath returns the file path of the item under the cursor, if it's a file item.
func getCursorFilePath(m Model) (string, bool) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return "", false
	}
	return item.Entry.Path, true
}

// getCommitHashAtCursor returns the commit hash at the cursor, or "" if not on a commit.
func getCommitHashAtCursor(m Model) string {
	item, _ := getCurrentItem(m)
	if item == nil || item.Commit == nil {
		return ""
	}
	return item.Commit.Hash
}

// Called when 'v' is pressed while the cursor is on a diff line (Hunk >= 0 && Line >= 0).
func handleEnterVisualMode(m Model) (tea.Model, tea.Cmd) {
	m.visualMode = true
	m.visualAnchor = m.cursor
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}
	return m, nil
}

// visualLineRange returns the start and end line indices (0-based, inclusive) of
// the current visual selection within the active hunk.
// The anchor and cursor may be in either order; this normalises them.
func visualLineRange(m Model) (startLine, endLine int) {
	anchorLine := m.visualAnchor.Line
	cursorLine := m.cursor.Line
	if anchorLine <= cursorLine {
		return anchorLine, cursorLine
	}
	return cursorLine, anchorLine
}

// handleStageLineRange stages the visually selected line range within the current hunk.
// It exits visual mode after issuing the command.
func handleStageLineRange(m Model) (tea.Model, tea.Cmd) {
	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		return m, nil
	}

	// Only stage from untracked or unstaged sections
	if sectionKind != SectionUntracked && sectionKind != SectionUnstaged {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		return m, nil
	}

	// Must be on a valid hunk with lines
	if m.cursor.Hunk < 0 || m.cursor.Hunk >= len(item.Hunks) {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		return m, nil
	}

	hunk := item.Hunks[m.cursor.Hunk]
	startLine, endLine := visualLineRange(m)

	// Clamp to valid range
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(hunk.Lines) {
		endLine = len(hunk.Lines) - 1
	}

	// Exit visual mode
	m.visualMode = false
	m.visualAnchor = Cursor{}

	// Save cursor context for restore after reload
	m.pendingRestore = saveCursorContext(m)

	return m, stageLineRangeCmd(m.repo, item.Entry.Path, hunk, startLine, endLine)
}

// handleUnstageLineRange unstages the visually selected line range within the current staged hunk.
// It exits visual mode after issuing the command.
func handleUnstageLineRange(m Model) (tea.Model, tea.Cmd) {
	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		return m, nil
	}

	// Only unstage from staged section
	if sectionKind != SectionStaged {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		return m, nil
	}

	// Must be on a valid hunk with lines
	if m.cursor.Hunk < 0 || m.cursor.Hunk >= len(item.Hunks) {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		return m, nil
	}

	hunk := item.Hunks[m.cursor.Hunk]
	startLine, endLine := visualLineRange(m)

	// Clamp to valid range
	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(hunk.Lines) {
		endLine = len(hunk.Lines) - 1
	}

	// Exit visual mode
	m.visualMode = false
	m.visualAnchor = Cursor{}

	// Save cursor context for restore after reload
	m.pendingRestore = saveCursorContext(m)

	return m, unstageLineRangeCmd(m.repo, item.Entry.Path, hunk, startLine, endLine)
}
