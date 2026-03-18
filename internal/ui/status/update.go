package status

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/ui/commit"
	"github.com/mhersson/conjit/internal/ui/commitselect"
	"github.com/mhersson/conjit/internal/ui/popup"
)

// update handles messages for the status model.
func update(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height

		// Re-render content and update viewport
		if !m.loading {
			content, cursorLine := renderContent(m)
			m.viewport.SetContent(content)
			ensureCursorVisible(&m, cursorLine)
		}
		return m, nil

	case tea.KeyMsg:
		return handleKeyMsg(m, msg)

	case statusLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.head = msg.head
		m.sections = msg.sections

		var cmd tea.Cmd

		if m.pendingRestore.active {
			restore := m.pendingRestore
			m.cursor = restoreCursor(m.sections, restore)
			m.pendingRestore = cursorRestore{}

			// If restoring after a hunk operation, expand the file and reload hunks
			if restore.hunk >= 0 && m.cursor.Item >= 0 {
				s := &m.sections[m.cursor.Section]
				if m.cursor.Item < len(s.Items) {
					item := &s.Items[m.cursor.Item]
					if item.Entry != nil {
						item.Expanded = true
						item.HunksLoading = true
						m.pendingHunkRestore = hunkRestore{
							active:     true,
							sectionIdx: m.cursor.Section,
							itemIdx:    m.cursor.Item,
							hunkIdx:    restore.hunk,
						}
						kind := diffKindForSection(s.Kind)
						cmd = loadHunksCmd(m.repo, m.cursor.Section, m.cursor.Item, item.Entry, kind)
					}
				}
			}
		} else {
			// Position cursor on first non-empty, non-hidden section
			m.cursor = findFirstValidCursor(m.sections)
		}

		// Update viewport content
		if m.viewport.Width > 0 {
			content, cursorLine := renderContent(m)
			m.viewport.SetContent(content)
			ensureCursorVisible(&m, cursorLine)
		}
		return m, cmd

	case hunksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Save current screen position before updating
		_, oldCursorLine := renderContent(m)
		screenRow := oldCursorLine - m.viewport.YOffset

		if msg.sectionIdx < len(m.sections) {
			s := &m.sections[msg.sectionIdx]
			if msg.itemIdx < len(s.Items) {
				s.Items[msg.itemIdx].Hunks = msg.hunks
				s.Items[msg.itemIdx].HunksLoading = false
			}
		}

		// Apply pending hunk cursor restore if this is the load we're waiting for
		if m.pendingHunkRestore.active &&
			msg.sectionIdx == m.pendingHunkRestore.sectionIdx &&
			msg.itemIdx == m.pendingHunkRestore.itemIdx {
			hunkIdx := m.pendingHunkRestore.hunkIdx
			m.pendingHunkRestore = hunkRestore{} // clear

			if len(msg.hunks) > 0 {
				// Clamp to available hunks
				if hunkIdx >= len(msg.hunks) {
					hunkIdx = len(msg.hunks) - 1
				}
				if hunkIdx < 0 {
					hunkIdx = 0
				}
				m.cursor.Hunk = hunkIdx
				m.cursor.Line = -1 // on hunk header
			}
			// If no hunks, cursor stays on file item (Hunk=-1)
		}

		// Update viewport content and preserve screen position
		if m.viewport.Width > 0 {
			content, newCursorLine := renderContent(m)
			m.viewport.SetContent(content)
			preserveScreenPosition(&m, newCursorLine, screenRow)
		}
		return m, nil

	case operationDoneMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		// Reload status after operation
		return m, loadStatusCmd(m.repo, m.cfg)

	case notificationExpiredMsg:
		m.notification = ""
		return m, nil

	case repoChangedMsg:
		return m, loadStatusCmd(m.repo, m.cfg)

	case openCommitEditorMsg:
		// Convert to commit.OpenCommitEditorMsg for the app to handle
		return m, func() tea.Msg {
			return commit.OpenCommitEditorMsg{
				Opts:   msg.opts,
				Action: msg.action,
			}
		}

	case commitselect.SelectedMsg:
		return handleCommitSelected(m, msg)

	case commitselect.AbortedMsg:
		m.commitSpecialKind = commitSpecialNone
		m.commitSpecialOpts = git.CommitOpts{}
		return m, nil

	case commitsLoadedMsg:
		if msg.err != nil {
			m.notification = "Failed to load commits: " + msg.err.Error()
			m.commitSpecialKind = commitSpecialNone
			m.commitSpecialOpts = git.CommitOpts{}
			return m, notifyCmd(2 * time.Second)
		}
		if len(msg.commits) == 0 {
			m.notification = "No commits found"
			m.commitSpecialKind = commitSpecialNone
			m.commitSpecialOpts = git.CommitOpts{}
			return m, notifyCmd(2 * time.Second)
		}
		// Send up to the app to take over the screen
		return m, func() tea.Msg {
			return commitselect.OpenCommitSelectMsg{Commits: msg.commits}
		}
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func handleKeyMsg(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If popup is active, delegate to it
	if m.popup != nil {
		return handlePopupKey(m, msg)
	}

	// Handle confirmation mode first
	if m.confirmMode != ConfirmNone {
		return handleConfirmKey(m, msg)
	}

	// Handle pending key sequences (e.g., "gg")
	if m.pendingKey == "g" {
		m.pendingKey = "" // Clear pending key
		if msg.String() == "g" {
			// "gg" - go to top
			return handleGoToTop(m)
		}
		// "g" followed by something else - ignore the g prefix for now
		// (could handle "gp" for GoToParentRepo here if needed)
	}

	switch {
	case key.Matches(msg, m.keys.Close):
		return m, tea.Quit

	case key.Matches(msg, m.keys.MoveDown):
		m.cursor = moveCursor(m.sections, m.cursor, 1)
		// Update viewport to keep cursor visible
		if m.viewport.Width > 0 {
			content, cursorLine := renderContent(m)
			m.viewport.SetContent(content)
			ensureCursorVisible(&m, cursorLine)
		}
		return m, nil

	case key.Matches(msg, m.keys.MoveUp):
		m.cursor = moveCursor(m.sections, m.cursor, -1)
		// Update viewport to keep cursor visible
		if m.viewport.Width > 0 {
			content, cursorLine := renderContent(m)
			m.viewport.SetContent(content)
			ensureCursorVisible(&m, cursorLine)
		}
		return m, nil

	case key.Matches(msg, m.keys.Toggle):
		return handleToggle(m)

	case key.Matches(msg, m.keys.OpenFold):
		return handleOpenFold(m)

	case key.Matches(msg, m.keys.CloseFold):
		return handleCloseFold(m)

	case key.Matches(msg, m.keys.Depth1):
		return handleDepth(m, 1)

	case key.Matches(msg, m.keys.Depth2):
		return handleDepth(m, 2)

	case key.Matches(msg, m.keys.Depth3):
		return handleDepth(m, 3)

	case key.Matches(msg, m.keys.Depth4):
		return handleDepth(m, 4)

	case key.Matches(msg, m.keys.RefreshBuffer):
		m.loading = true
		return m, loadStatusCmd(m.repo, m.cfg)

	// Stage/Unstage actions
	case key.Matches(msg, m.keys.Stage):
		return handleStage(m)

	case key.Matches(msg, m.keys.StageUnstaged):
		return handleStageUnstaged(m)

	case key.Matches(msg, m.keys.StageAll):
		return handleStageAll(m)

	case key.Matches(msg, m.keys.Unstage):
		return handleUnstage(m)

	case key.Matches(msg, m.keys.UnstageStaged):
		return handleUnstageStaged(m)

	// Discard action (requires confirmation)
	case key.Matches(msg, m.keys.Discard):
		return handleDiscardStart(m)

	// Untrack action (requires confirmation)
	case key.Matches(msg, m.keys.Untrack):
		return handleUntrackStart(m)

	// Navigation keys
	case key.Matches(msg, m.keys.NextSection):
		return handleNextSection(m)

	case key.Matches(msg, m.keys.PreviousSection):
		return handlePreviousSection(m)

	case key.Matches(msg, m.keys.NextHunkHeader):
		return handleNextHunk(m)

	case key.Matches(msg, m.keys.PrevHunkHeader):
		return handlePrevHunk(m)

	// Scroll navigation
	case key.Matches(msg, m.keys.PageUp):
		return handlePageUp(m)

	case key.Matches(msg, m.keys.PageDown):
		return handlePageDown(m)

	case key.Matches(msg, m.keys.HalfPageUp):
		return handleHalfPageUp(m)

	case key.Matches(msg, m.keys.HalfPageDown):
		return handleHalfPageDown(m)

	case key.Matches(msg, m.keys.GoToTop):
		// First "g" press - set pending key and wait for second key
		m.pendingKey = "g"
		return m, nil

	case key.Matches(msg, m.keys.GoToBottom):
		return handleGoToBottom(m)

	// Yank to clipboard
	case key.Matches(msg, m.keys.YankSelected):
		return handleYank(m)

	// Open directory in file manager
	case key.Matches(msg, m.keys.OpenTree):
		return handleOpenTree(m)

	// Popup keys - create real popups
	case key.Matches(msg, m.keys.CommitPopup):
		return handleOpenCommitPopup(m)

	case key.Matches(msg, m.keys.BranchPopup):
		return handleOpenBranchPopup(m)

	case key.Matches(msg, m.keys.PushPopup):
		return handleOpenPushPopup(m)

	case key.Matches(msg, m.keys.PullPopup):
		return handleOpenPullPopup(m)

	case key.Matches(msg, m.keys.FetchPopup):
		return handleOpenFetchPopup(m)

	case key.Matches(msg, m.keys.MergePopup):
		return handleOpenMergePopup(m)

	case key.Matches(msg, m.keys.RebasePopup):
		return handleOpenRebasePopup(m)

	case key.Matches(msg, m.keys.RevertPopup):
		return handleOpenRevertPopup(m)

	case key.Matches(msg, m.keys.CherryPickPopup):
		return handleOpenCherryPickPopup(m)

	case key.Matches(msg, m.keys.ResetPopup):
		return handleOpenResetPopup(m)

	case key.Matches(msg, m.keys.StashPopup):
		return handleOpenStashPopup(m)

	case key.Matches(msg, m.keys.TagPopup):
		return handleOpenTagPopup(m)

	case key.Matches(msg, m.keys.RemotePopup):
		return handleOpenRemotePopup(m)

	case key.Matches(msg, m.keys.WorktreePopup):
		return handleOpenWorktreePopup(m)

	case key.Matches(msg, m.keys.BisectPopup):
		return handleOpenBisectPopup(m)

	case key.Matches(msg, m.keys.IgnorePopup):
		return handleOpenIgnorePopup(m)

	case key.Matches(msg, m.keys.DiffPopup):
		return handleOpenDiffPopup(m)

	case key.Matches(msg, m.keys.LogPopup):
		return handleOpenLogPopup(m)

	case key.Matches(msg, m.keys.MarginPopup):
		return handleOpenMarginPopup(m)

	case key.Matches(msg, m.keys.HelpPopup):
		return handleOpenHelpPopup(m)

	// Other stub keys
	case key.Matches(msg, m.keys.ShowRefs),
		key.Matches(msg, m.keys.CommandHistory),
		key.Matches(msg, m.keys.Command),
		key.Matches(msg, m.keys.InitRepo),
		key.Matches(msg, m.keys.GoToParentRepo),
		key.Matches(msg, m.keys.Rename),
		key.Matches(msg, m.keys.PeekFile),
		key.Matches(msg, m.keys.GoToFile),
		key.Matches(msg, m.keys.VSplitOpen),
		key.Matches(msg, m.keys.SplitOpen),
		key.Matches(msg, m.keys.TabOpen),
		key.Matches(msg, m.keys.OpenOrScrollDown),
		key.Matches(msg, m.keys.OpenOrScrollUp),
		key.Matches(msg, m.keys.PeekDown),
		key.Matches(msg, m.keys.PeekUp):
		m.notification = "Not yet implemented"
		return m, notifyCmd(2 * time.Second)
	}

	return m, nil
}

// handleConfirmKey handles keypresses during confirmation mode.
func handleConfirmKey(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Execute the confirmed action
		return executeConfirmedAction(m)
	case "n", "N", "esc":
		// Cancel
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.confirmHunk = -1
		m.notification = ""
		return m, nil
	default:
		// Any other key cancels
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.confirmHunk = -1
		m.notification = ""
		return m, nil
	}
}

// executeConfirmedAction executes the action after confirmation.
func executeConfirmedAction(m Model) (tea.Model, tea.Cmd) {
	// Save cursor context for restore after reload
	m.pendingRestore = saveCursorContext(m)

	switch m.confirmMode {
	case ConfirmDiscard:
		path := m.confirmPath
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.notification = ""
		return m, discardFileCmd(m.repo, path)

	case ConfirmDiscardHunk:
		path := m.confirmPath
		hunkIdx := m.confirmHunk
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.confirmHunk = -1
		m.notification = ""
		// Look up the hunk from the current item
		if hunk := findHunkByPathAndIndex(m, path, hunkIdx); hunk != nil {
			return m, discardHunkCmd(m.repo, path, *hunk)
		}
		return m, nil

	case ConfirmUntrack:
		path := m.confirmPath
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.notification = ""
		return m, untrackFileCmd(m.repo, path)
	}

	m.confirmMode = ConfirmNone
	return m, nil
}

// handleStage stages the current file or hunk.
func handleStage(m Model) (tea.Model, tea.Cmd) {
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
		return m, stageHunkCmd(m.repo, item.Entry.Path(), item.Hunks[m.cursor.Hunk])
	}

	// Stage the whole file
	return m, stageFileCmd(m.repo, item.Entry.Path())
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
func handleUnstage(m Model) (tea.Model, tea.Cmd) {
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
		return m, unstageHunkCmd(m.repo, item.Entry.Path(), item.Hunks[m.cursor.Hunk])
	}

	// Unstage the whole file
	return m, unstageFileCmd(m.repo, item.Entry.Path())
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
		m.notification = "Can only discard unstaged changes"
		return m, notifyCmd(2 * time.Second)
	}

	path := item.Entry.Path()

	// Check if on a hunk
	if m.cursor.Hunk >= 0 && len(item.Hunks) > m.cursor.Hunk {
		m.confirmMode = ConfirmDiscardHunk
		m.confirmPath = path
		m.confirmHunk = m.cursor.Hunk
		m.notification = "Discard hunk in " + path + "? (y/N)"
	} else {
		m.confirmMode = ConfirmDiscard
		m.confirmPath = path
		m.notification = "Discard changes to " + path + "? (y/N)"
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
		m.notification = "Can only untrack staged files"
		return m, notifyCmd(2 * time.Second)
	}

	path := item.Entry.Path()
	m.confirmMode = ConfirmUntrack
	m.confirmPath = path
	m.notification = "Untrack " + path + "? (y/N)"

	return m, nil
}

// handleNextSection moves to the next section header.
func handleNextSection(m Model) (tea.Model, tea.Cmd) {
	visible := visibleSections(m.sections)
	if len(visible) == 0 {
		return m, nil
	}

	// Find current section in visible list
	currentIdx := -1
	for i, v := range visible {
		if v == m.cursor.Section {
			currentIdx = i
			break
		}
	}

	// Move to next visible section
	nextIdx := (currentIdx + 1) % len(visible)
	m.cursor = Cursor{Section: visible[nextIdx], Item: -1, Hunk: -1, Line: -1}

	return m, nil
}

// handlePreviousSection moves to the previous section header.
func handlePreviousSection(m Model) (tea.Model, tea.Cmd) {
	visible := visibleSections(m.sections)
	if len(visible) == 0 {
		return m, nil
	}

	// Find current section in visible list
	currentIdx := 0
	for i, v := range visible {
		if v == m.cursor.Section {
			currentIdx = i
			break
		}
	}

	// Move to previous visible section
	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(visible) - 1
	}
	m.cursor = Cursor{Section: visible[prevIdx], Item: -1, Hunk: -1, Line: -1}

	return m, nil
}

// handleNextHunk moves to the next hunk header.
func handleNextHunk(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil || len(item.Hunks) == 0 {
		// No hunks in current item, try next item
		return m, nil
	}

	// If not in hunks yet, go to first hunk
	if m.cursor.Hunk < 0 {
		m.cursor.Hunk = 0
		m.cursor.Line = -1
		return m, nil
	}

	// Move to next hunk
	if m.cursor.Hunk < len(item.Hunks)-1 {
		m.cursor.Hunk++
		m.cursor.Line = -1
	}

	return m, nil
}

// handlePrevHunk moves to the previous hunk header.
func handlePrevHunk(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil || len(item.Hunks) == 0 {
		return m, nil
	}

	// If on first hunk or before, go to item
	if m.cursor.Hunk <= 0 {
		m.cursor.Hunk = -1
		m.cursor.Line = -1
		return m, nil
	}

	// Move to previous hunk
	m.cursor.Hunk--
	m.cursor.Line = -1

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
		text = item.Entry.Path()
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

	m.notification = "Yanked: " + text
	return m, tea.Batch(
		yankToClipboardCmd(text),
		notifyCmd(2*time.Second),
	)
}

// handleOpenTree opens the directory containing the current file.
func handleOpenTree(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}

	return m, openTreeCmd(m.repo.Path(), item.Entry.Path())
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
			if item.Entry != nil && item.Entry.Path() == path {
				if hunkIdx >= 0 && hunkIdx < len(item.Hunks) {
					return &item.Hunks[hunkIdx]
				}
			}
		}
	}
	return nil
}

// handleToggle toggles fold state of current section or item.
func handleToggle(m Model) (tea.Model, tea.Cmd) {
	if m.cursor.Section >= len(m.sections) {
		return m, nil
	}

	// Save current screen position before toggling
	_, oldCursorLine := renderContent(m)
	screenRow := oldCursorLine - m.viewport.YOffset

	s := &m.sections[m.cursor.Section]

	if m.cursor.Item == -1 {
		// Toggle section fold
		s.Folded = !s.Folded
	} else if m.cursor.Item < len(s.Items) {
		item := &s.Items[m.cursor.Item]

		// If on a hunk (header or line), toggle hunk fold
		if m.cursor.Hunk >= 0 && m.cursor.Hunk < len(item.Hunks) {
			// Ensure HunksFolded slice is initialized
			if item.HunksFolded == nil {
				item.HunksFolded = make([]bool, len(item.Hunks))
			}
			// Toggle hunk fold
			item.HunksFolded[m.cursor.Hunk] = !item.HunksFolded[m.cursor.Hunk]
			// If we were on a line, move to hunk header
			if m.cursor.Line >= 0 {
				m.cursor.Line = -1
			}

			// Update viewport with preserved screen position
			if m.viewport.Width > 0 {
				content, newCursorLine := renderContent(m)
				m.viewport.SetContent(content)
				preserveScreenPosition(&m, newCursorLine, screenRow)
			}
			return m, nil
		}

		// Toggle item expansion
		item.Expanded = !item.Expanded

		// Load hunks if expanding and not loaded
		if item.Expanded && item.Hunks == nil && item.Entry != nil && !item.HunksLoading {
			item.HunksLoading = true
			kind := diffKindForSection(s.Kind)

			// Update viewport with preserved screen position
			if m.viewport.Width > 0 {
				content, newCursorLine := renderContent(m)
				m.viewport.SetContent(content)
				preserveScreenPosition(&m, newCursorLine, screenRow)
			}
			return m, loadHunksCmd(m.repo, m.cursor.Section, m.cursor.Item, item.Entry, kind)
		}
	}

	// Update viewport with preserved screen position
	if m.viewport.Width > 0 {
		content, newCursorLine := renderContent(m)
		m.viewport.SetContent(content)
		preserveScreenPosition(&m, newCursorLine, screenRow)
	}

	return m, nil
}

// handleOpenFold opens the current fold.
func handleOpenFold(m Model) (tea.Model, tea.Cmd) {
	if m.cursor.Section >= len(m.sections) {
		return m, nil
	}

	s := &m.sections[m.cursor.Section]

	if m.cursor.Item == -1 {
		s.Folded = false
	} else if m.cursor.Item < len(s.Items) {
		item := &s.Items[m.cursor.Item]
		if !item.Expanded {
			item.Expanded = true
			if item.Hunks == nil && item.Entry != nil && !item.HunksLoading {
				item.HunksLoading = true
				kind := diffKindForSection(s.Kind)
				return m, loadHunksCmd(m.repo, m.cursor.Section, m.cursor.Item, item.Entry, kind)
			}
		}
	}

	return m, nil
}

// handleCloseFold closes the current fold.
func handleCloseFold(m Model) (tea.Model, tea.Cmd) {
	if m.cursor.Section >= len(m.sections) {
		return m, nil
	}

	s := &m.sections[m.cursor.Section]

	if m.cursor.Item == -1 {
		s.Folded = true
	} else if m.cursor.Item < len(s.Items) {
		s.Items[m.cursor.Item].Expanded = false
	}

	return m, nil
}

// handleDepth sets fold depth: 1=headers only, 2=items, 3=hunks, 4=all.
func handleDepth(m Model, depth int) (tea.Model, tea.Cmd) {
	for i := range m.sections {
		s := &m.sections[i]
		switch depth {
		case 1:
			s.Folded = true
			for j := range s.Items {
				s.Items[j].Expanded = false
			}
		case 2:
			s.Folded = false
			for j := range s.Items {
				s.Items[j].Expanded = false
			}
		case 3, 4:
			s.Folded = false
			for j := range s.Items {
				s.Items[j].Expanded = true
			}
		}
	}
	return m, nil
}

// moveCursor moves the cursor by dir (+1 = down, -1 = up).
func moveCursor(sections []Section, cursor Cursor, dir int) Cursor {
	if len(sections) == 0 {
		return cursor
	}

	// Get list of visible sections
	visible := visibleSections(sections)
	if len(visible) == 0 {
		return cursor
	}

	// Find current position in visible sections
	visIdx := -1
	for i, v := range visible {
		if v == cursor.Section {
			visIdx = i
			break
		}
	}

	// If current section not visible, go to first visible
	if visIdx == -1 {
		return Cursor{Section: visible[0], Item: -1, Hunk: -1, Line: -1}
	}

	s := &sections[cursor.Section]

	if dir > 0 {
		// Moving down
		if cursor.Item == -1 {
			// On section header
			if !s.Folded && len(s.Items) > 0 {
				// Enter section
				return Cursor{Section: cursor.Section, Item: 0, Hunk: -1, Line: -1}
			}
			// Go to next section
			nextVisIdx := visIdx + 1
			if nextVisIdx >= len(visible) {
				return cursor // Stay at boundary
			}
			return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
		}

		// On an item
		item := &s.Items[cursor.Item]

		// If on item line (not in hunks) and item is expanded with hunks, enter hunks
		if cursor.Hunk == -1 && item.Expanded && len(item.Hunks) > 0 {
			return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: 0, Line: -1}
		}

		// If on hunk header, enter diff lines (unless hunk is folded)
		if cursor.Hunk >= 0 && cursor.Line == -1 {
			hunk := &item.Hunks[cursor.Hunk]
			isFolded := len(item.HunksFolded) > cursor.Hunk && item.HunksFolded[cursor.Hunk]
			if !isFolded && len(hunk.Lines) > 0 {
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: 0}
			}
			// Hunk is folded or has no lines, go to next hunk
			if cursor.Hunk < len(item.Hunks)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk + 1, Line: -1}
			}
			// Last hunk, go to next item
			if cursor.Item < len(s.Items)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item + 1, Hunk: -1, Line: -1}
			}
			// Go to next section
			nextVisIdx := visIdx + 1
			if nextVisIdx >= len(visible) {
				return cursor // Stay at boundary
			}
			return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
		}

		// If on a diff line, move to next line or next hunk
		if cursor.Line >= 0 {
			hunk := &item.Hunks[cursor.Hunk]
			if cursor.Line < len(hunk.Lines)-1 {
				// Next line in same hunk
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: cursor.Line + 1}
			}
			// Last line, go to next hunk header
			if cursor.Hunk < len(item.Hunks)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk + 1, Line: -1}
			}
			// Last hunk, go to next item
			if cursor.Item < len(s.Items)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item + 1, Hunk: -1, Line: -1}
			}
			// Go to next section
			nextVisIdx := visIdx + 1
			if nextVisIdx >= len(visible) {
				return cursor // Stay at boundary
			}
			return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
		}

		// Exit hunks / go to next item (shouldn't reach here with line-level navigation)
		if cursor.Item < len(s.Items)-1 {
			return Cursor{Section: cursor.Section, Item: cursor.Item + 1, Hunk: -1, Line: -1}
		}

		// Go to next section
		nextVisIdx := visIdx + 1
		if nextVisIdx >= len(visible) {
			return cursor // Stay at boundary
		}
		return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
	}

	// Moving up (dir < 0)

	// If on a diff line, go to previous line or hunk header
	if cursor.Line > 0 {
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: cursor.Line - 1}
	}

	if cursor.Line == 0 {
		// First line, go to hunk header
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: -1}
	}

	// On hunk header (Line == -1)
	if cursor.Hunk > 0 {
		// Go to last line of previous hunk (unless folded)
		item := &s.Items[cursor.Item]
		prevHunk := &item.Hunks[cursor.Hunk-1]
		isFolded := len(item.HunksFolded) > cursor.Hunk-1 && item.HunksFolded[cursor.Hunk-1]
		if !isFolded && len(prevHunk.Lines) > 0 {
			return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk - 1, Line: len(prevHunk.Lines) - 1}
		}
		// Previous hunk is folded or has no lines, go to its header
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk - 1, Line: -1}
	}

	if cursor.Hunk == 0 {
		// First hunk, go to item
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: -1, Line: -1}
	}

	// On item (Hunk == -1)
	if cursor.Item > 0 {
		// Previous item
		prevItem := &s.Items[cursor.Item-1]
		if prevItem.Expanded && len(prevItem.Hunks) > 0 {
			// Go to last line of last hunk of previous item (unless folded)
			lastHunkIdx := len(prevItem.Hunks) - 1
			lastHunk := &prevItem.Hunks[lastHunkIdx]
			isFolded := len(prevItem.HunksFolded) > lastHunkIdx && prevItem.HunksFolded[lastHunkIdx]
			if !isFolded && len(lastHunk.Lines) > 0 {
				return Cursor{Section: cursor.Section, Item: cursor.Item - 1, Hunk: lastHunkIdx, Line: len(lastHunk.Lines) - 1}
			}
			return Cursor{Section: cursor.Section, Item: cursor.Item - 1, Hunk: lastHunkIdx, Line: -1}
		}
		return Cursor{Section: cursor.Section, Item: cursor.Item - 1, Hunk: -1, Line: -1}
	}

	if cursor.Item == 0 {
		// Go to section header
		return Cursor{Section: cursor.Section, Item: -1, Hunk: -1, Line: -1}
	}

	// On section header, go to previous section
	prevVisIdx := visIdx - 1
	if prevVisIdx < 0 {
		return cursor // Stay at boundary
	}
	prevSection := &sections[visible[prevVisIdx]]
	if !prevSection.Folded && len(prevSection.Items) > 0 {
		// Go to last item of previous section
		lastIdx := len(prevSection.Items) - 1
		lastItem := &prevSection.Items[lastIdx]
		if lastItem.Expanded && len(lastItem.Hunks) > 0 {
			// Go to last line of last hunk (unless folded)
			lastHunkIdx := len(lastItem.Hunks) - 1
			lastHunk := &lastItem.Hunks[lastHunkIdx]
			isFolded := len(lastItem.HunksFolded) > lastHunkIdx && lastItem.HunksFolded[lastHunkIdx]
			if !isFolded && len(lastHunk.Lines) > 0 {
				return Cursor{Section: visible[prevVisIdx], Item: lastIdx, Hunk: lastHunkIdx, Line: len(lastHunk.Lines) - 1}
			}
			return Cursor{Section: visible[prevVisIdx], Item: lastIdx, Hunk: lastHunkIdx, Line: -1}
		}
		return Cursor{Section: visible[prevVisIdx], Item: lastIdx, Hunk: -1, Line: -1}
	}
	return Cursor{Section: visible[prevVisIdx], Item: -1, Hunk: -1, Line: -1}
}

// visibleSections returns indices of non-hidden sections.
func visibleSections(sections []Section) []int {
	var visible []int
	for i, s := range sections {
		if !s.Hidden {
			visible = append(visible, i)
		}
	}
	return visible
}

// findFirstValidCursor finds the first non-hidden, non-empty section.
func findFirstValidCursor(sections []Section) Cursor {
	for i, s := range sections {
		if !s.Hidden {
			return Cursor{Section: i, Item: -1, Hunk: -1, Line: -1}
		}
	}
	return Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}
}

// restoreCursor tries to place the cursor near where the user was before a
// status reload. It searches for the file path in the original section first,
// then falls back to staying at the same index in that section, then falls
// back to findFirstValidCursor.
func restoreCursor(sections []Section, restore cursorRestore) Cursor {
	// Find the section matching the original kind.
	sectionIdx := -1
	for i, s := range sections {
		if s.Kind == restore.sectionKind {
			sectionIdx = i
			break
		}
	}

	// If original section found, look for the file in it.
	if sectionIdx >= 0 {
		s := sections[sectionIdx]
		for itemIdx, item := range s.Items {
			if item.Entry != nil && item.Entry.Path() == restore.path {
				return Cursor{Section: sectionIdx, Item: itemIdx, Hunk: -1, Line: -1}
			}
		}
		// File not in original section (it moved). Clamp to same item index.
		if len(s.Items) > 0 {
			idx := restore.itemIndex
			if idx >= len(s.Items) {
				idx = len(s.Items) - 1
			}
			if idx < 0 {
				idx = 0
			}
			return Cursor{Section: sectionIdx, Item: idx, Hunk: -1, Line: -1}
		}
		// Section is empty — try next visible section with items.
		for i := sectionIdx + 1; i < len(sections); i++ {
			if !sections[i].Hidden {
				return Cursor{Section: i, Item: -1, Hunk: -1, Line: -1}
			}
		}
		// Fall through to global fallback.
	}

	return findFirstValidCursor(sections)
}

// saveCursorContext builds a cursorRestore from the current model state.
func saveCursorContext(m Model) cursorRestore {
	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return cursorRestore{}
	}
	return cursorRestore{
		active:      true,
		path:        item.Entry.Path(),
		sectionKind: sectionKind,
		itemIndex:   m.cursor.Item,
		hunk:        m.cursor.Hunk,
	}
}

// diffKindForSection returns the diff kind for a section.
func diffKindForSection(kind SectionKind) git.DiffKind {
	switch kind {
	case SectionStaged:
		return git.DiffStaged
	default:
		return git.DiffUnstaged
	}
}

// handlePageUp scrolls the viewport up by a full page.
func handlePageUp(m Model) (tea.Model, tea.Cmd) {
	m.viewport.YOffset -= m.viewport.Height
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}

	// Move cursor up to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor up by approximately a page worth of items
		for i := 0; i < m.viewport.Height; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, -1)
		}
		content, cursorLine := renderContent(m)
		m.viewport.SetContent(content)
		ensureCursorVisible(&m, cursorLine)
	}

	return m, nil
}

// handlePageDown scrolls the viewport down by a full page.
func handlePageDown(m Model) (tea.Model, tea.Cmd) {
	maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	m.viewport.YOffset += m.viewport.Height
	if m.viewport.YOffset > maxOffset {
		m.viewport.YOffset = maxOffset
	}

	// Move cursor down to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor down by approximately a page worth of items
		for i := 0; i < m.viewport.Height; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, 1)
		}
		content, cursorLine := renderContent(m)
		m.viewport.SetContent(content)
		ensureCursorVisible(&m, cursorLine)
	}

	return m, nil
}

// handleHalfPageUp scrolls the viewport up by half a page.
func handleHalfPageUp(m Model) (tea.Model, tea.Cmd) {
	m.viewport.YOffset -= m.viewport.Height / 2
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}

	// Move cursor up to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor up by approximately half a page worth of items
		for i := 0; i < m.viewport.Height/2; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, -1)
		}
		content, cursorLine := renderContent(m)
		m.viewport.SetContent(content)
		ensureCursorVisible(&m, cursorLine)
	}

	return m, nil
}

// handleHalfPageDown scrolls the viewport down by half a page.
func handleHalfPageDown(m Model) (tea.Model, tea.Cmd) {
	maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	m.viewport.YOffset += m.viewport.Height / 2
	if m.viewport.YOffset > maxOffset {
		m.viewport.YOffset = maxOffset
	}

	// Move cursor down to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor down by approximately half a page worth of items
		for i := 0; i < m.viewport.Height/2; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, 1)
		}
		content, cursorLine := renderContent(m)
		m.viewport.SetContent(content)
		ensureCursorVisible(&m, cursorLine)
	}

	return m, nil
}

// handleGoToTop moves cursor to the first section header and scrolls to top.
func handleGoToTop(m Model) (tea.Model, tea.Cmd) {
	// Move cursor to first visible section header
	m.cursor = findFirstValidCursor(m.sections)

	// Scroll viewport to top
	m.viewport.YOffset = 0

	// Re-render content to update cursor highlighting
	if m.viewport.Width > 0 {
		content, _ := renderContent(m)
		m.viewport.SetContent(content)
	}

	return m, nil
}

// handleGoToBottom moves cursor to the last item and scrolls to show it.
func handleGoToBottom(m Model) (tea.Model, tea.Cmd) {
	// Find last visible section with items
	visible := visibleSections(m.sections)
	if len(visible) == 0 {
		return m, nil
	}

	// Start from last visible section and find last item
	for i := len(visible) - 1; i >= 0; i-- {
		s := &m.sections[visible[i]]
		if !s.Folded && len(s.Items) > 0 {
			// Go to last item in this section
			m.cursor = Cursor{
				Section: visible[i],
				Item:    len(s.Items) - 1,
				Hunk:    -1,
				Line:    -1,
			}
			break
		} else if i == 0 {
			// All sections are folded or empty, go to last section header
			m.cursor = Cursor{
				Section: visible[len(visible)-1],
				Item:    -1,
				Hunk:    -1,
				Line:    -1,
			}
		}
	}

	// Re-render to get cursor line, then scroll to show it
	content, cursorLine := renderContent(m)
	m.viewport.SetContent(content)
	ensureCursorVisible(&m, cursorLine)

	return m, nil
}

// handlePopupKey delegates key handling to the active popup.
func handlePopupKey(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := *m.popup
	newPopup, cmd := p.Update(msg)
	m.popup = &newPopup

	if m.popup.Done() {
		result := m.popup.Result()
		kind := m.popupKind
		m.popup = nil
		m.popupKind = PopupNone

		// Handle popup result
		if result.Action != "" {
			return handlePopupAction(m, kind, result)
		}
	}

	return m, cmd
}

// handlePopupAction processes the action from a closed popup.
func handlePopupAction(m Model, kind PopupKind, result popup.Result) (tea.Model, tea.Cmd) {
	switch kind {
	case PopupCommit:
		return handleCommitPopupAction(m, result)
	case PopupPush:
		return handlePushPopupAction(m, result)
	case PopupPull:
		return handlePullPopupAction(m, result)
	case PopupFetch:
		return handleFetchPopupAction(m, result)
	default:
		// For now, just show a notification with the action
		// The actual git operations will be wired up as needed
		m.notification = "Action: " + result.Action
		return m, notifyCmd(2 * time.Second)
	}
}

// handleCommitPopupAction handles actions from the commit popup.
func handleCommitPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildCommitOpts(result)

	switch result.Action {
	case "c": // Commit
		return m, openCommitEditorCmd(opts, "commit")
	case "e": // Extend (amend without editing)
		opts.Amend = true
		return m, openCommitEditorCmd(opts, "extend")
	case "a": // Amend
		opts.Amend = true
		return m, openCommitEditorCmd(opts, "amend")
	case "w": // Reword
		opts.Amend = true
		return m, openCommitEditorCmd(opts, "reword")
	case "f": // Fixup
		return openCommitSelect(m, opts, commitSpecialFixup)
	case "s": // Squash
		return openCommitSelect(m, opts, commitSpecialSquash)
	case "A": // Alter
		return openCommitSelect(m, opts, commitSpecialAlter)
	case "n": // Augment
		return openCommitSelect(m, opts, commitSpecialAugment)
	case "W": // Revise
		return openCommitSelect(m, opts, commitSpecialRevise)
	case "F": // Instant Fixup
		return openCommitSelect(m, opts, commitSpecialInstantFixup)
	case "S": // Instant Squash
		return openCommitSelect(m, opts, commitSpecialInstantSquash)
	case "x": // Absorb
		m.notification = "Absorb not yet implemented"
		return m, notifyCmd(2 * time.Second)
	default:
		m.notification = "Unknown commit action: " + result.Action
		return m, notifyCmd(2 * time.Second)
	}
}

// buildCommitOpts builds CommitOpts from popup result switches and options.
func buildCommitOpts(result popup.Result) git.CommitOpts {
	return git.CommitOpts{
		All:          result.Switches["all"],
		AllowEmpty:   result.Switches["allow-empty"],
		Verbose:      result.Switches["verbose"],
		NoVerify:     result.Switches["no-verify"],
		ResetAuthor:  result.Switches["reset-author"],
		Signoff:      result.Switches["signoff"],
		Author:       result.Options["author"],
		GpgSign:      result.Options["gpg-sign"],
		ReuseMessage: result.Options["reuse-message"],
	}
}

// handleOpenCommitPopup opens the commit popup.
func handleOpenCommitPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewCommitPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupCommit
	return m, nil
}

// openCommitSelect initiates the commit select flow for special commit actions.
// It fetches recent commits; once loaded, the app switches to the commit select screen.
func openCommitSelect(m Model, opts git.CommitOpts, kind commitSpecialKind) (tea.Model, tea.Cmd) {
	m.commitSpecialOpts = opts
	m.commitSpecialKind = kind
	return m, loadCommitsForSelectCmd(m.repo)
}

// handleCommitSelected handles the user selecting a commit in the commit select view.
func handleCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.commitSpecialOpts
	kind := m.commitSpecialKind

	// Clear commit select state
	m.commitSpecialKind = commitSpecialNone
	m.commitSpecialOpts = git.CommitOpts{}

	switch kind {
	case commitSpecialFixup:
		opts.Fixup = msg.Hash
		opts.NoEdit = true
		return m, commitSpecialCmd(m.repo, opts)
	case commitSpecialSquash:
		opts.Squash = msg.Hash
		opts.NoEdit = true
		return m, commitSpecialCmd(m.repo, opts)
	case commitSpecialAugment:
		opts.Squash = msg.Hash
		return m, openCommitEditorCmd(opts, "augment")
	case commitSpecialAlter:
		opts.Fixup = "amend:" + msg.Hash
		return m, openCommitEditorCmd(opts, "alter")
	case commitSpecialRevise:
		opts.Fixup = "reword:" + msg.Hash
		return m, openCommitEditorCmd(opts, "revise")
	case commitSpecialInstantFixup:
		opts.Fixup = msg.Hash
		opts.NoEdit = true
		return m, commitAndAutosquashCmd(m.repo, opts, msg.FullHash)
	case commitSpecialInstantSquash:
		opts.Squash = msg.Hash
		opts.NoEdit = true
		return m, commitAndAutosquashCmd(m.repo, opts, msg.FullHash)
	default:
		return m, nil
	}
}

// handleOpenBranchPopup opens the branch popup.
func handleOpenBranchPopup(m Model) (tea.Model, tea.Cmd) {
	branch := m.head.Branch
	showConfig := branch != "" && !m.head.Detached
	p := popup.NewBranchPopup(m.tokens, nil, branch, showConfig)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupBranch
	return m, nil
}

// handleOpenPushPopup opens the push popup.
func handleOpenPushPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewPushPopup(m.tokens, nil, m.head.Branch, m.head.Detached)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupPush
	return m, nil
}

// handleOpenPullPopup opens the pull popup.
func handleOpenPullPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewPullPopup(m.tokens, nil, m.head.Branch)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupPull
	return m, nil
}

// handleOpenFetchPopup opens the fetch popup.
func handleOpenFetchPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewFetchPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupFetch
	return m, nil
}

// handleOpenMergePopup opens the merge popup.
func handleOpenMergePopup(m Model) (tea.Model, tea.Cmd) {
	inMerge := isInMerge(m.sections)
	p := popup.NewMergePopup(m.tokens, nil, inMerge)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupMerge
	return m, nil
}

// handleOpenRebasePopup opens the rebase popup.
func handleOpenRebasePopup(m Model) (tea.Model, tea.Cmd) {
	inRebase := isInRebase(m.sections)
	p := popup.NewRebasePopup(m.tokens, nil, inRebase)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRebase
	return m, nil
}

// handleOpenRevertPopup opens the revert popup.
func handleOpenRevertPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "revert")
	p := popup.NewRevertPopup(m.tokens, nil, inProgress)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRevert
	return m, nil
}

// handleOpenCherryPickPopup opens the cherry-pick popup.
func handleOpenCherryPickPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "pick")
	p := popup.NewCherryPickPopup(m.tokens, nil, inProgress)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupCherryPick
	return m, nil
}

// handleOpenResetPopup opens the reset popup.
func handleOpenResetPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewResetPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupReset
	return m, nil
}

// handleOpenStashPopup opens the stash popup.
func handleOpenStashPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewStashPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupStash
	return m, nil
}

// handleOpenTagPopup opens the tag popup.
func handleOpenTagPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewTagPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupTag
	return m, nil
}

// handleOpenRemotePopup opens the remote popup.
func handleOpenRemotePopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewRemotePopup(m.tokens, nil, "origin")
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRemote
	return m, nil
}

// handleOpenWorktreePopup opens the worktree popup.
func handleOpenWorktreePopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewWorktreePopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupWorktree
	return m, nil
}

// handleOpenBisectPopup opens the bisect popup.
func handleOpenBisectPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress, finished := getBisectState(m.sections)
	p := popup.NewBisectPopup(m.tokens, nil, inProgress, finished)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupBisect
	return m, nil
}

// handleOpenIgnorePopup opens the ignore popup.
func handleOpenIgnorePopup(m Model) (tea.Model, tea.Cmd) {
	// Check if global gitignore exists
	hasGlobalIgnore := false // Could check git config core.excludesfile
	p := popup.NewIgnorePopup(m.tokens, nil, hasGlobalIgnore)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupIgnore
	return m, nil
}

// handleOpenDiffPopup opens the diff popup.
func handleOpenDiffPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewDiffPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupDiff
	return m, nil
}

// handleOpenLogPopup opens the log popup.
func handleOpenLogPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewLogPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupLog
	return m, nil
}

// handleOpenMarginPopup opens the margin popup.
func handleOpenMarginPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewMarginPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupMargin
	return m, nil
}

// handleOpenHelpPopup opens the help popup.
func handleOpenHelpPopup(m Model) (tea.Model, tea.Cmd) {
	keys := popup.HelpKeys{
		CommitPopup:     "c",
		BranchPopup:     "b",
		PushPopup:       "P",
		PullPopup:       "p",
		FetchPopup:      "f",
		MergePopup:      "m",
		RebasePopup:     "r",
		RevertPopup:     "v",
		CherryPickPopup: "A",
		ResetPopup:      "X",
		StashPopup:      "Z",
		TagPopup:        "t",
		RemotePopup:     "M",
		WorktreePopup:   "w",
		BisectPopup:     "B",
		IgnorePopup:     "i",
		DiffPopup:       "d",
		LogPopup:        "l",
		MarginPopup:     "L",
		Stage:           "s",
		Unstage:         "u",
		Discard:         "x",
		MoveDown:        "j",
		MoveUp:          "k",
		Close:           "q",
		Refresh:         "C-r",
		NextSection:     "C-n",
		PrevSection:     "C-p",
		ToggleFold:      "tab",
	}
	p := popup.NewHelpPopup(m.tokens, keys)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupHelp
	return m, nil
}

// isInMerge checks if there's an active merge.
func isInMerge(sections []Section) bool {
	for _, s := range sections {
		if s.Kind == SectionSequencer && len(s.Items) > 0 {
			for _, item := range s.Items {
				if item.Action == "merge" {
					return true
				}
			}
		}
	}
	return false
}

// isInRebase checks if there's an active rebase.
func isInRebase(sections []Section) bool {
	for _, s := range sections {
		if s.Kind == SectionRebase && len(s.Items) > 0 {
			return true
		}
	}
	return false
}

// isInSequencer checks if there's an active sequencer operation of the given type.
func isInSequencer(sections []Section, action string) bool {
	for _, s := range sections {
		if s.Kind == SectionSequencer && len(s.Items) > 0 {
			for _, item := range s.Items {
				if item.Action == action {
					return true
				}
			}
		}
	}
	return false
}

// getBisectState returns whether bisect is in progress and if it's finished.
func getBisectState(sections []Section) (inProgress, finished bool) {
	for _, s := range sections {
		if s.Kind == SectionBisect && len(s.Items) > 0 {
			inProgress = true
			// Check if finished (implementation would check git bisect state)
			finished = false
			return
		}
	}
	return false, false
}

// handlePushPopupAction handles actions from the push popup.
func handlePushPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "C": // Configure
		m.notification = "Push configuration not yet implemented"
		return m, notifyCmd(2 * time.Second)
	case "e": // Elsewhere
		m.notification = "Push elsewhere not yet implemented"
		return m, notifyCmd(2 * time.Second)
	case "o": // Another branch
		m.notification = "Push another branch not yet implemented"
		return m, notifyCmd(2 * time.Second)
	case "r": // Explicit refspec
		m.notification = "Push explicit refspec not yet implemented"
		return m, notifyCmd(2 * time.Second)
	case "T": // A tag
		m.notification = "Push a tag not yet implemented"
		return m, notifyCmd(2 * time.Second)
	}

	opts := buildPushOpts(result)
	remote, branch := resolvePushTarget(result.Action, m.head)
	opts.Remote = remote
	opts.Branch = branch

	if remote == "" {
		m.notification = "No remote configured for push"
		return m, notifyCmd(2 * time.Second)
	}

	m.notification = "Pushing..."
	return m, pushCmd(m.repo, opts)
}

// buildPushOpts builds PushOpts from popup result switches.
func buildPushOpts(result popup.Result) git.PushOpts {
	return git.PushOpts{
		ForceWithLease: result.Switches["force-with-lease"],
		Force:          result.Switches["force"],
		NoVerify:       result.Switches["no-verify"],
		DryRun:         result.Switches["dry-run"],
		SetUpstream:    result.Switches["set-upstream"],
		Tags:           result.Switches["tags"],
		FollowTags:     result.Switches["follow-tags"],
	}
}

// resolvePushTarget returns the remote and branch for a push action key.
func resolvePushTarget(action string, head HeadState) (remote, branch string) {
	switch action {
	case "p": // pushRemote
		return head.PushRemote, head.Branch
	case "u": // @{upstream}
		return head.UpstreamRemote, head.Branch
	case "t": // all tags
		return defaultRemote(head), ""
	case "m": // matching branches
		return defaultRemote(head), ""
	default:
		return defaultRemote(head), head.Branch
	}
}

// defaultRemote returns the best remote to use for push operations.
func defaultRemote(head HeadState) string {
	if head.PushRemote != "" {
		return head.PushRemote
	}
	if head.UpstreamRemote != "" {
		return head.UpstreamRemote
	}
	return "origin"
}

// pushCmd creates a command that executes a git push.
func pushCmd(repo *git.Repository, opts git.PushOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository")}
		}
		var err error
		if opts.Tags && opts.Branch == "" {
			err = repo.PushTags(context.Background(), opts.Remote)
		} else {
			err = repo.Push(context.Background(), opts)
		}
		return operationDoneMsg{err: err}
	}
}

// handlePullPopupAction handles actions from the pull popup.
func handlePullPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildPullOpts(result)
	remote, branch := resolvePullTarget(result.Action, m.head)
	opts.Remote = remote
	opts.Branch = branch

	if remote == "" {
		m.notification = "No remote configured for pull"
		return m, notifyCmd(2 * time.Second)
	}

	m.notification = "Pulling..."
	return m, pullCmd(m.repo, opts)
}

// buildPullOpts builds PullOpts from popup result switches.
func buildPullOpts(result popup.Result) git.PullOpts {
	return git.PullOpts{
		Rebase:  result.Switches["rebase"],
		FFOnly:  result.Switches["ff-only"],
		Tags:    result.Switches["tags"],
		Autostash: result.Switches["autostash"],
	}
}

// resolvePullTarget returns the remote and branch for a pull action key.
func resolvePullTarget(action string, head HeadState) (remote, branch string) {
	switch action {
	case "p": // pushRemote
		return head.PushRemote, head.Branch
	case "u": // @{upstream}
		return head.UpstreamRemote, head.Branch
	default:
		return defaultRemote(head), head.Branch
	}
}

// pullCmd creates a command that executes a git pull.
func pullCmd(repo *git.Repository, opts git.PullOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository")}
		}
		err := repo.Pull(context.Background(), opts)
		return operationDoneMsg{err: err}
	}
}

// handleFetchPopupAction handles actions from the fetch popup.
func handleFetchPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildFetchOpts(result)

	switch result.Action {
	case "p": // pushRemote
		opts.Remote = m.head.PushRemote
	case "u": // upstream
		opts.Remote = m.head.UpstreamRemote
	default:
		opts.Remote = defaultRemote(m.head)
	}

	if opts.Remote == "" {
		m.notification = "No remote configured for fetch"
		return m, notifyCmd(2 * time.Second)
	}

	m.notification = "Fetching..."
	return m, fetchCmd(m.repo, opts)
}

// buildFetchOpts builds FetchOpts from popup result switches.
func buildFetchOpts(result popup.Result) git.FetchOpts {
	return git.FetchOpts{
		Prune: result.Switches["prune"],
		Tags:  result.Switches["tags"],
	}
}

// fetchCmd creates a command that executes a git fetch.
func fetchCmd(repo *git.Repository, opts git.FetchOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository")}
		}
		err := repo.Fetch(context.Background(), opts)
		return operationDoneMsg{err: err}
	}
}
