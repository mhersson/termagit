package status

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/ui/branchselect"
	"github.com/mhersson/conjit/internal/ui/commit"
	"github.com/mhersson/conjit/internal/ui/commitselect"
	"github.com/mhersson/conjit/internal/ui/commitview"
	"github.com/mhersson/conjit/internal/ui/notification"
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

			// If restoring after a hunk operation, expand the file and reload hunks.
			// Only do this when the cursor landed on the SAME file — if the file
			// moved (e.g. last hunk staged), we must not expand the next file.
			if restore.hunk >= 0 && m.cursor.Item >= 0 {
				s := &m.sections[m.cursor.Section]
				if m.cursor.Item < len(s.Items) {
					item := &s.Items[m.cursor.Item]
					if item.Entry != nil && item.Entry.Path() == restore.path {
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
			m.viewport.YOffset = 0
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
		wasRestore := false
		if m.pendingHunkRestore.active &&
			msg.sectionIdx == m.pendingHunkRestore.sectionIdx &&
			msg.itemIdx == m.pendingHunkRestore.itemIdx {
			wasRestore = true
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

		// Update viewport content
		if m.viewport.Width > 0 {
			content, newCursorLine := renderContent(m)
			m.viewport.SetContent(content)
			if wasRestore {
				// After a stage/unstage hunk restore, reset scroll to top
				// so the hint bar stays visible.
				m.viewport.YOffset = 0
				ensureCursorVisible(&m, newCursorLine)
			} else {
				preserveScreenPosition(&m, newCursorLine, screenRow)
			}
		}
		return m, nil

	case operationDoneMsg:
		cmds := []tea.Cmd{loadStatusCmd(m.repo, m.cfg)}
		if msg.err != nil {
			errMsg := msg.err.Error()
			if msg.op != "" {
				errMsg = msg.op + " failed: " + errMsg
			}
			cmds = append(cmds, notifyAppCmd(errMsg, notification.Error))
		} else if msg.op != "" {
			cmds = append(cmds, notifyAppCmd(msg.op+" complete", notification.Success))
		}
		return m, tea.Batch(cmds...)

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

	case commitview.CommitDataLoadedMsg:
		// Forward to commit view if active
		if m.commitView != nil {
			cv := *m.commitView
			newCV, cmd := cv.Update(msg)
			cvModel := newCV.(commitview.Model)
			m.commitView = &cvModel
			return m, cmd
		}
		return m, nil

	case commitview.CloseCommitViewMsg:
		// Handle close from the overlay commit view - don't bubble up to app
		if m.commitView != nil {
			m.commitView = nil
		}
		return m, nil

	case commitview.OpenPopupMsg:
		return openPopupByName(m, msg.Type)

	case commitselect.AbortedMsg:
		m.commitSpecialKind = commitSpecialNone
		m.commitSpecialOpts = git.CommitOpts{}
		return m, nil

	case branchselect.SelectedMsg:
		return handleBranchSelected(m, msg)

	case branchselect.AbortedMsg:
		m.branchActionKind = branchActionNone
		return m, nil

	case peekFileMsg:
		if msg.err != nil {
			return m, notifyAppCmd("Failed to load file: "+msg.err.Error(), notification.Error)
		}
		m.peekActive = true
		m.peekPath = msg.path
		m.peekContent = msg.content
		m.peekViewport = viewport.New(m.width, m.height*60/100)
		m.peekViewport.SetContent(msg.content)
		return m, nil

	case closePeekMsg:
		m.peekActive = false
		m.peekPath = ""
		m.peekContent = ""
		return m, nil

	case branchesLoadedMsg:
		if msg.err != nil {
			m.branchActionKind = branchActionNone
			return m, notifyAppCmd("Failed to load branches: "+msg.err.Error(), notification.Error)
		}
		if len(msg.branches) == 0 {
			m.branchActionKind = branchActionNone
			return m, notifyAppCmd("No branches found", notification.Warning)
		}
		return m, func() tea.Msg {
			return branchselect.OpenBranchSelectMsg{Branches: msg.branches}
		}

	case commitsLoadedMsg:
		if msg.err != nil {
			m.commitSpecialKind = commitSpecialNone
			m.commitSpecialOpts = git.CommitOpts{}
			return m, notifyAppCmd("Failed to load commits: "+msg.err.Error(), notification.Error)
		}
		if len(msg.commits) == 0 {
			m.commitSpecialKind = commitSpecialNone
			m.commitSpecialOpts = git.CommitOpts{}
			return m, notifyAppCmd("No commits found", notification.Warning)
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
	// If commit view overlay is active, delegate to it
	if m.commitView != nil {
		return handleCommitViewKey(m, msg)
	}

	// If popup is active, delegate to it
	if m.popup != nil {
		return handlePopupKey(m, msg)
	}

	// Handle confirmation mode first
	if m.confirmMode != ConfirmNone {
		return handleConfirmKey(m, msg)
	}

	// Handle input prompt mode
	if m.inputPromptKind != inputPromptNone {
		return handleInputPromptKey(m, msg)
	}

	// Handle pending key sequences (e.g., "gg", "gp")
	if m.pendingKey == "g" {
		m.pendingKey = "" // Clear pending key
		switch msg.String() {
		case "g":
			return handleGoToTop(m)
		case "p":
			return handleGoToParentRepo(m)
		}
		// "g" followed by something else - ignore the g prefix
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

	case key.Matches(msg, m.keys.CommandHistory):
		return m, func() tea.Msg { return OpenCmdHistoryMsg{} }

	case key.Matches(msg, m.keys.GoToFile):
		return handleGoToFile(m)

	case key.Matches(msg, m.keys.ShowRefs):
		return handleShowRefs(m)

	case key.Matches(msg, m.keys.Command):
		// Q = Console = same as $ (CommandHistory) in Neogit
		return m, func() tea.Msg { return OpenCmdHistoryMsg{} }

	case key.Matches(msg, m.keys.InitRepo):
		return m, notifyAppCmd("Already in a git repository", notification.Info)

	case key.Matches(msg, m.keys.Rename):
		return handleRenameFile(m)

	case key.Matches(msg, m.keys.PeekFile):
		return handlePeekFile(m)

	case key.Matches(msg, m.keys.VSplitOpen),
		key.Matches(msg, m.keys.SplitOpen),
		key.Matches(msg, m.keys.TabOpen):
		// In TUI these are all aliases for GoToFile
		return handleGoToFile(m)

	case key.Matches(msg, m.keys.OpenOrScrollDown):
		return handleOpenOrScrollDown(m)

	case key.Matches(msg, m.keys.OpenOrScrollUp):
		return handleOpenOrScrollUp(m)

	case key.Matches(msg, m.keys.PeekDown):
		return handlePeekDown(m)

	case key.Matches(msg, m.keys.PeekUp):
		return handlePeekUp(m)
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
		return m, nil
	default:
		// Any other key cancels
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.confirmHunk = -1
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
		return m, discardFileCmd(m.repo, path)

	case ConfirmDiscardHunk:
		path := m.confirmPath
		hunkIdx := m.confirmHunk
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
		m.confirmHunk = -1
		// Look up the hunk from the current item
		if hunk := findHunkByPathAndIndex(m, path, hunkIdx); hunk != nil {
			return m, discardHunkCmd(m.repo, path, *hunk)
		}
		return m, nil

	case ConfirmUntrack:
		path := m.confirmPath
		m.confirmMode = ConfirmNone
		m.confirmPath = ""
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
		return m, notifyAppCmd("Can only discard unstaged changes", notification.Warning)
	}

	path := item.Entry.Path()

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
		content, cursorLine := renderContent(m)
		m.viewport.SetContent(content)
		ensureCursorVisible(&m, cursorLine)
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

	path := item.Entry.Path()
	m.confirmMode = ConfirmUntrack
	m.confirmPath = path

	// Refresh viewport so confirmation prompt is visible
	if m.viewport.Width > 0 {
		content, cursorLine := renderContent(m)
		m.viewport.SetContent(content)
		ensureCursorVisible(&m, cursorLine)
	}

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

	m.notification = ""
	return m, tea.Batch(
		yankToClipboardCmd(text),
		notifyAppCmd("Yanked: "+text, notification.Info),
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
		return m, openInEditorCmd(repoPath, item.Entry.Path())
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

// handleCommitViewKey delegates key handling to the commit view overlay.
func handleCommitViewKey(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cv := *m.commitView
	newCV, cmd := cv.Update(msg)
	cvModel := newCV.(commitview.Model)
	m.commitView = &cvModel

	if cvModel.Done() {
		m.commitView = nil
		// Don't return the CloseCommitViewMsg command - we handle the close internally
		return m, nil
	}
	return m, cmd
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
	case PopupBranch:
		return handleBranchPopupAction(m, result)
	case PopupRebase:
		return handleRebasePopupAction(m, result)
	case PopupPush:
		return handlePushPopupAction(m, result)
	case PopupPull:
		return handlePullPopupAction(m, result)
	case PopupFetch:
		return handleFetchPopupAction(m, result)
	case PopupLog:
		return handleLogPopupAction(m, result)
	case PopupMerge:
		return handleMergePopupAction(m, result)
	case PopupCherryPick:
		return handleCherryPickPopupAction(m, result)
	case PopupRevert:
		return handleRevertPopupAction(m, result)
	case PopupStash:
		return handleStashPopupAction(m, result)
	case PopupReset:
		return handleResetPopupAction(m, result)
	case PopupTag:
		return handleTagPopupAction(m, result)
	case PopupRemote:
		return handleRemotePopupAction(m, result)
	case PopupWorktree:
		return handleWorktreePopupAction(m, result)
	case PopupBisect:
		return handleBisectPopupAction(m, result)
	case PopupIgnore:
		return handleIgnorePopupAction(m, result)
	case PopupDiff:
		return handleDiffPopupAction(m, result)
	case PopupMargin:
		return handleMarginPopupAction(m, result)
	case PopupHelp:
		return handleHelpPopupAction(m, result)
	default:
		return m, notifyAppCmd("Action: "+result.Action, notification.Info)
	}
}

// handleCommitPopupAction handles actions from the commit popup.
func handleCommitPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildCommitOpts(result)

	switch result.Action {
	case "c": // Commit
		if !opts.AllowEmpty && !opts.All && !hasStagedChanges(m) {
			return m, notifyAppCmd("No changes to commit.", notification.Warning)
		}
		return m, openCommitEditorCmd(opts, "commit")
	case "e": // Extend (amend without editing)
		if !opts.AllowEmpty && !opts.All && !hasStagedChanges(m) {
			return m, notifyAppCmd("No changes to commit.", notification.Warning)
		}
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
	case "x": // Absorb — requires external git-absorb
		return m, notifyAppCmd("Absorb requires git-absorb to be installed", notification.Warning)
	default:
		return m, notifyAppCmd("Unknown commit action: "+result.Action, notification.Warning)
	}
}

// hasStagedChanges returns true if the model has a non-empty staged section.
func hasStagedChanges(m Model) bool {
	for i := range m.sections {
		if m.sections[i].Kind == SectionStaged && len(m.sections[i].Items) > 0 {
			return true
		}
	}
	return false
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

// handleRebasePopupAction handles actions from the rebase popup.
func handleRebasePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inRebase := isInRebase(m.sections)

	if inRebase {
		return handleRebaseInProgressAction(m, result)
	}

	opts := buildRebaseOpts(result)

	switch result.Action {
	// Rebase onto group
	case "p": // pushRemote
		remote := m.head.PushRemote
		if remote == "" {
			remote = m.head.UpstreamRemote
		}
		if remote == "" {
			return m, notifyAppCmd("No push remote configured", notification.Warning)
		}
		target := remote + "/" + m.head.Branch
		opts.Onto = target
		return m, rebaseCmd(m.repo, opts)
	case "u": // @{upstream}
		remote := m.head.UpstreamRemote
		if remote == "" {
			return m, notifyAppCmd("No upstream configured", notification.Warning)
		}
		target := remote + "/" + m.head.UpstreamBranch
		opts.Onto = target
		return m, rebaseCmd(m.repo, opts)
	case "e": // elsewhere — select branch to rebase onto
		m.branchActionKind = branchActionRebaseElsewhere
		m.rebaseSpecialOpts = opts
		return m, loadAllBranchesCmd(m.repo)
	case "b": // base branch — rebase onto base (main/master detection)
		return m, notifyAppCmd("Base branch detection not configured", notification.Warning)

	// Rebase group
	case "i": // interactively — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialInteractive)
	case "s": // a subset — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialSubset)

	// Modify commits group
	case "m": // to modify a commit — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialModify)
	case "w": // to reword a commit — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialReword)
	case "d": // to remove a commit — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialDrop)
	case "f": // to autosquash
		target := m.head.UpstreamRemote + "/" + m.head.UpstreamBranch
		if m.head.UpstreamRemote == "" {
			return m, notifyAppCmd("No upstream configured for autosquash", notification.Warning)
		}
		return m, autosquashCmd(m.repo, opts, target)

	default:
		return m, notifyAppCmd("Unknown rebase action: "+result.Action, notification.Warning)
	}
}

// handleRebaseInProgressAction handles rebase actions when a rebase is already in progress.
func handleRebaseInProgressAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "r": // Continue
		return m, rebaseContinueCmd(m.repo)
	case "s": // Skip
		return m, rebaseSkipCmd(m.repo)
	case "e": // Edit todo
		return m, func() tea.Msg {
			return popup.OpenRebaseEditorMsg{}
		}
	case "a": // Abort
		return m, rebaseAbortCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown rebase action: "+result.Action, notification.Warning)
	}
}

// buildRebaseOpts builds RebaseOpts from popup result switches and options.
func buildRebaseOpts(result popup.Result) git.RebaseOpts {
	return git.RebaseOpts{
		Interactive:               result.Switches["interactive"],
		Autosquash:                result.Switches["autosquash"],
		Autostash:                 result.Switches["autostash"],
		KeepEmpty:                 result.Switches["keep-empty"],
		UpdateRefs:                result.Switches["update-refs"],
		NoVerify:                  result.Switches["no-verify"],
		CommitterDateIsAuthorDate: result.Switches["committer-date-is-author-date"],
		IgnoreDate:                result.Switches["ignore-date"],
		RebaseMerges:              result.Options["rebase-merges"],
		GpgSign:                   result.Options["gpg-sign"],
	}
}

// openRebaseCommitSelect opens the commit select view for rebase actions that need a target commit.
func openRebaseCommitSelect(m Model, opts git.RebaseOpts, kind rebaseSpecialKind) (tea.Model, tea.Cmd) {
	m.rebaseSpecialOpts = opts
	m.rebaseSpecialKind = kind
	return m, loadCommitsForSelectCmd(m.repo)
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
	// Check if this is a rebase commit selection
	if m.rebaseSpecialKind != rebaseSpecialNone {
		return handleRebaseCommitSelected(m, msg)
	}

	// Check if this is a cherry-pick commit selection
	if m.cherryPickActionKind != cherryPickActionNone {
		return handleCherryPickCommitSelected(m, msg)
	}

	// Check if this is a revert commit selection
	if m.revertActionKind != revertActionNone {
		return handleRevertCommitSelected(m, msg)
	}

	// Check if this is a reset commit selection
	if m.resetActionKind != resetActionNone {
		return handleResetCommitSelected(m, msg)
	}

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

// handleRebaseCommitSelected handles the user selecting a commit for a rebase action.
func handleRebaseCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.rebaseSpecialOpts
	kind := m.rebaseSpecialKind

	// Clear rebase select state
	m.rebaseSpecialKind = rebaseSpecialNone
	m.rebaseSpecialOpts = git.RebaseOpts{}

	switch kind {
	case rebaseSpecialInteractive:
		opts.Interactive = true
		opts.Onto = msg.FullHash
		return m, interactiveRebaseCmd(m.repo, opts)
	case rebaseSpecialSubset:
		opts.Onto = msg.FullHash
		return m, rebaseCmd(m.repo, opts)
	case rebaseSpecialModify:
		return m, modifyCommitCmd(m.repo, msg.FullHash)
	case rebaseSpecialReword:
		return m, rewordCommitCmd(m.repo, msg.FullHash)
	case rebaseSpecialDrop:
		return m, dropCommitCmd(m.repo, msg.FullHash)
	default:
		return m, nil
	}
}

// handleOpenBranchPopup opens the branch popup.
func handleOpenBranchPopup(m Model) (tea.Model, tea.Cmd) {
	branch := m.head.Branch
	showConfig := branch != "" && !m.head.Detached
	hasUpstream := m.head.UpstreamRemote != ""
	p := popup.NewBranchPopup(m.tokens, nil, branch, showConfig, hasUpstream)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupBranch
	return m, nil
}

// handleOpenPushPopup opens the push popup.
func handleOpenPushPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewPushPopup(m.tokens, nil, popup.PushPopupParams{
		Branch:          m.head.Branch,
		IsDetached:      m.head.Detached,
		PushRemoteLabel: resolveRemoteLabel(m.head.PushRemote, m.head.PushBranch),
		UpstreamLabel:   resolveRemoteLabel(m.head.UpstreamRemote, m.head.UpstreamBranch),
	})
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupPush
	return m, nil
}

// handleOpenPullPopup opens the pull popup.
func handleOpenPullPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewPullPopup(m.tokens, nil, popup.PullPopupParams{
		Branch:          m.head.Branch,
		IsDetached:      m.head.Detached,
		PushRemoteLabel: resolveRemoteLabel(m.head.PushRemote, m.head.PushBranch),
		UpstreamLabel:   resolveRemoteLabel(m.head.UpstreamRemote, m.head.UpstreamBranch),
	})
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupPull
	return m, nil
}

// handleOpenFetchPopup opens the fetch popup.
func handleOpenFetchPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewFetchPopup(m.tokens, nil, popup.FetchPopupParams{
		PushRemoteLabel: resolveRemoteLabel(m.head.PushRemote, m.head.PushBranch),
		UpstreamLabel:   resolveRemoteLabel(m.head.UpstreamRemote, m.head.UpstreamBranch),
	})
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
	p := popup.NewRebasePopup(m.tokens, nil, inRebase, m.head.Branch, "")
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRebase
	return m, nil
}

// handleOpenRevertPopup opens the revert popup.
func handleOpenRevertPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "revert")
	hasHunk := cursorOnHunk(m)
	p := popup.NewRevertPopup(m.tokens, nil, inProgress, hasHunk)
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
	item, _ := getCurrentItem(m)
	hasItem := item != nil
	commitSelected := item != nil && item.Commit != nil
	p := popup.NewDiffPopup(m.tokens, nil, hasItem, commitSelected)
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

// openPopupByName opens a popup by its string name (used by commit view).
func openPopupByName(m Model, name string) (tea.Model, tea.Cmd) {
	switch name {
	case "commit":
		return handleOpenCommitPopup(m)
	case "branch":
		return handleOpenBranchPopup(m)
	case "push":
		return handleOpenPushPopup(m)
	case "pull":
		return handleOpenPullPopup(m)
	case "fetch":
		return handleOpenFetchPopup(m)
	case "merge":
		return handleOpenMergePopup(m)
	case "rebase":
		return handleOpenRebasePopup(m)
	case "revert":
		return handleOpenRevertPopup(m)
	case "cherry-pick":
		return handleOpenCherryPickPopup(m)
	case "reset":
		return handleOpenResetPopup(m)
	case "stash":
		return handleOpenStashPopup(m)
	case "tag":
		return handleOpenTagPopup(m)
	case "remote":
		return handleOpenRemotePopup(m)
	case "worktree":
		return handleOpenWorktreePopup(m)
	case "bisect":
		return handleOpenBisectPopup(m)
	case "ignore":
		return handleOpenIgnorePopup(m)
	case "diff":
		return handleOpenDiffPopup(m)
	case "log":
		return handleOpenLogPopup(m)
	}
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
// resolveRemoteLabel builds "remote/branch" from the two parts.
// Returns empty string if either is empty, so callers fall back to the default label.
func resolveRemoteLabel(remote, branch string) string {
	if remote == "" || branch == "" {
		return ""
	}
	return remote + "/" + branch
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
	case "C": // Configure — open branch select to pick branch to configure
		m.branchActionKind = branchActionBranchConfigure
		return m, loadLocalBranchesCmd(m.repo)
	case "e": // Elsewhere — select remote/branch to push to
		m.branchActionKind = branchActionPushElsewhere
		return m, loadAllBranchesCmd(m.repo)
	case "o": // Another branch — select source branch
		m.branchActionKind = branchActionPushOther
		return m, loadLocalBranchesCmd(m.repo)
	case "r": // Explicit refspec — text input
		return openBranchInput(m, inputPromptPushRefspec, "Push refspec: ")
	case "T": // A tag — text input for tag name
		return openBranchInput(m, inputPromptPushTag, "Push tag: ")
	}

	opts := buildPushOpts(result)
	remote, branch, setUpstream := resolvePushTarget(result.Action, m.head, m.repo)
	opts.Remote = remote
	opts.Branch = branch
	if setUpstream {
		opts.SetUpstream = true
	}

	if remote == "" {
		return m, notifyAppCmd("No remote configured for push", notification.Warning)
	}

	return m, tea.Batch(
		pushCmd(m.repo, opts),
		notifyAppCmd("Pushing to "+remote+"/"+branch+"...", notification.Info),
	)
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
// When the action is "p" or "u" and no remote is configured, it attempts to
// resolve a sensible default remote and signals that --set-upstream should be
// used so the upstream tracking branch is created automatically.
func resolvePushTarget(action string, head HeadState, repo *git.Repository) (remote, branch string, setUpstream bool) {
	switch action {
	case "p": // pushRemote
		remote = head.PushRemote
		if remote == "" {
			remote, _ = repo.SmartDefaultRemote(context.Background())
			setUpstream = true
		}
		return remote, head.Branch, setUpstream
	case "u": // @{upstream}
		remote = head.UpstreamRemote
		if remote == "" {
			remote, _ = repo.SmartDefaultRemote(context.Background())
			setUpstream = true
		}
		return remote, head.Branch, setUpstream
	case "t": // all tags
		return defaultRemote(head), "", false
	case "m": // matching branches
		return defaultRemote(head), "", false
	default:
		return defaultRemote(head), head.Branch, false
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
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Push"}
		}
		var err error
		if opts.Tags && opts.Branch == "" {
			err = repo.PushTags(context.Background(), opts.Remote)
		} else {
			err = repo.Push(context.Background(), opts)
		}
		return operationDoneMsg{err: err, op: "Push"}
	}
}

// handlePullPopupAction handles actions from the pull popup.
func handlePullPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildPullOpts(result)
	remote, branch := resolvePullTarget(result.Action, m.head)
	opts.Remote = remote
	opts.Branch = branch

	if remote == "" {
		return m, notifyAppCmd("No remote configured for pull", notification.Warning)
	}

	return m, tea.Batch(
		pullCmd(m.repo, opts),
		notifyAppCmd("Pulling from "+remote+"/"+branch+"...", notification.Info),
	)
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
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Pull"}
		}
		err := repo.Pull(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Pull"}
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
		return m, notifyAppCmd("No remote configured for fetch", notification.Warning)
	}

	return m, tea.Batch(
		fetchCmd(m.repo, opts),
		notifyAppCmd("Fetching from "+opts.Remote+"...", notification.Info),
	)
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
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Fetch"}
		}
		err := repo.Fetch(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Fetch"}
	}
}

// handleLogPopupAction handles actions from the log popup.
func handleLogPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildLogOpts(result)

	// Log actions
	switch result.Action {
	case "l": // current branch
		branch := m.head.Branch
		if branch == "" {
			branch = "HEAD"
		}
		return m, loadLogCmd(m.repo, opts, branch)

	case "h": // HEAD
		return m, loadLogCmd(m.repo, opts, "HEAD")

	case "u": // related (upstream)
		if m.head.UpstreamRemote == "" {
			return m, notifyAppCmd("No upstream configured", notification.Warning)
		}
		branch := m.head.UpstreamRemote + "/" + m.head.UpstreamBranch
		return m, loadLogCmd(m.repo, opts, branch)

	case "L": // local branches
		opts.All = false
		opts.Branch = ""
		return m, loadLogCmd(m.repo, opts, m.head.Branch)

	case "b": // all branches
		opts.All = true
		return m, loadLogCmd(m.repo, opts, "")

	case "a": // all references
		opts.All = true
		opts.Decorate = true
		return m, loadLogCmd(m.repo, opts, "")

	// Reflog actions
	case "r": // current branch reflog
		branch := m.head.Branch
		if branch == "" {
			branch = "HEAD"
		}
		return m, loadReflogCmd(m.repo, branch)

	case "H": // HEAD reflog
		return m, loadReflogCmd(m.repo, "HEAD")

	case "O": // other reflog — prompt for ref
		return openBranchInput(m, inputPromptReflogRef, "Reflog for ref: ")

	case "o": // other branch log — open branch select
		m.branchActionKind = branchActionLogOtherBranch
		return m, loadAllBranchesCmd(m.repo)

	default:
		return m, notifyAppCmd("Unknown log action: "+result.Action, notification.Warning)
	}
}

// buildLogOpts builds LogOpts from popup result switches and options.
func buildLogOpts(result popup.Result) git.LogOpts {
	maxCount := 256
	if maxStr, ok := result.Options["max-count"]; ok && maxStr != "" {
		if n, err := parseMaxCount(maxStr); err == nil {
			maxCount = n
		}
	}

	return git.LogOpts{
		MaxCount:    maxCount,
		Author:      result.Options["author"],
		Grep:        result.Options["grep"],
		Since:       result.Options["since"],
		Until:       result.Options["until"],
		NoMerges:    result.Switches["no-merges"],
		FirstParent: result.Switches["first-parent"],
		Reverse:     result.Switches["reverse"],
		Graph:       result.Switches["graph"],
		Decorate:    result.Switches["decorate"],
	}
}

// parseMaxCount parses the max-count string to an int.
func parseMaxCount(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// loadLogCmd loads commits and opens the log view.
func loadLogCmd(repo *git.Repository, opts git.LogOpts, branch string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return notification.NotifyMsg{Message: "No repository", Kind: notification.Error}
		}

		opts.Branch = branch
		opts.Decorate = true // Always show decorations in log view

		commits, hasMore, err := repo.Log(context.Background(), opts)
		if err != nil {
			return notification.NotifyMsg{Message: "Failed to load log: " + err.Error(), Kind: notification.Error}
		}

		return OpenLogViewMsg{
			Commits: commits,
			HasMore: hasMore,
			Branch:  branch,
		}
	}
}

// loadReflogCmd loads reflog entries and opens the reflog view.
func loadReflogCmd(repo *git.Repository, ref string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return notification.NotifyMsg{Message: "No repository", Kind: notification.Error}
		}

		entries, err := repo.Reflog(context.Background(), ref, 256)
		if err != nil {
			return notification.NotifyMsg{Message: "Failed to load reflog: " + err.Error(), Kind: notification.Error}
		}

		return OpenReflogViewMsg{
			Entries: entries,
			Ref:     ref,
		}
	}
}

// --- Branch popup action handling ---

// handleBranchPopupAction handles actions from the branch popup.
func handleBranchPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	// Branch selection actions
	case "b": // checkout branch/revision
		m.branchActionKind = branchActionCheckout
		return m, loadAllBranchesCmd(m.repo)
	case "l": // checkout local branch
		m.branchActionKind = branchActionCheckoutLocal
		return m, loadLocalBranchesCmd(m.repo)
	case "r": // checkout recent branch
		m.branchActionKind = branchActionCheckoutRecent
		return m, loadRecentBranchesCmd(m.repo)
	case "D": // delete
		m.branchActionKind = branchActionDelete
		return m, loadLocalBranchesCmd(m.repo)

	// Text input actions
	case "c": // new branch + checkout
		return openBranchInput(m, inputPromptNewBranchCheckout, "Create and checkout branch: ")
	case "n": // new branch no checkout
		return openBranchInput(m, inputPromptNewBranch, "Create branch: ")
	case "s": // spin-off
		return openBranchInput(m, inputPromptSpinOff, "Spin-off branch name: ")
	case "S": // spin-out
		return openBranchInput(m, inputPromptSpinOut, "Spin-out branch name: ")
	case "m": // rename
		return openBranchInput(m, inputPromptRename, "Rename "+m.head.Branch+" to: ")

	// Immediate actions
	case "X": // reset to upstream
		if m.head.UpstreamRemote == "" {
			return m, notifyAppCmd("No upstream configured for "+m.head.Branch, notification.Warning)
		}
		return m, resetBranchToUpstreamCmd(m.repo)

	case "w", "W": // Worktree — prompt for path
		return openBranchInput(m, inputPromptWorktreePath, "Worktree path: ")
	case "C": // Configure — select branch to configure
		m.branchActionKind = branchActionBranchConfigure
		return m, loadLocalBranchesCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown branch action: "+result.Action, notification.Warning)
	}
}

// openBranchInput sets up the inline text input prompt for branch name entry.
func openBranchInput(m Model, kind inputPromptKind, label string) (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.CharLimit = 200
	ti.Focus()
	m.inputPromptKind = kind
	m.inputPromptLabel = label
	m.inputPrompt = ti
	return m, textinput.Blink
}

// handleInputPromptKey handles key presses while the input prompt is active.
func handleInputPromptKey(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := m.inputPrompt.Value()
		kind := m.inputPromptKind
		m.inputPromptKind = inputPromptNone
		m.inputPromptLabel = ""

		if name == "" {
			return m, nil
		}

		switch kind {
		case inputPromptNewBranchCheckout:
			return m, createAndCheckoutBranchCmd(m.repo, name, "HEAD")
		case inputPromptNewBranch:
			return m, createBranchCmd(m.repo, name, "HEAD")
		case inputPromptSpinOff:
			return m, spinOffBranchCmd(m.repo, name)
		case inputPromptSpinOut:
			return m, spinOutBranchCmd(m.repo, name)
		case inputPromptRename:
			return m, renameBranchCmd(m.repo, m.head.Branch, name)
		case inputPromptRenameFile:
			oldPath := m.confirmPath
			m.confirmPath = ""
			m.pendingRestore = saveCursorContext(m)
			return m, renameFileCmd(m.repo, oldPath, name)
		case inputPromptPushRefspec:
			return m, pushRefspecCmd(m.repo, name)
		case inputPromptPushTag:
			return m, pushTagCmd(m.repo, name)
		case inputPromptReflogRef:
			return m, loadReflogCmd(m.repo, name)
		case inputPromptWorktreePath:
			return m, worktreeAddCmd(m.repo, name, m.head.Branch)
		case inputPromptTagName:
			return m, tagCreateCmd(m.repo, name, "HEAD", git.TagOpts{})
		case inputPromptTagRelease:
			return m, tagCreateCmd(m.repo, name, "HEAD", git.TagOpts{Annotate: true})
		case inputPromptTagDelete:
			return m, tagDeleteCmd(m.repo, name)
		case inputPromptRemoteName:
			// Store name, now prompt for URL
			m.confirmPath = name // reuse confirmPath to carry the remote name
			return openBranchInput(m, inputPromptRemoteURL, "Remote URL: ")
		case inputPromptRemoteURL:
			remoteName := m.confirmPath
			m.confirmPath = ""
			return m, remoteAddCmd(m.repo, remoteName, name)
		case inputPromptRemoteRename:
			// name is the new name; confirmPath holds the old name
			oldName := m.confirmPath
			m.confirmPath = ""
			return m, remoteRenameCmd(m.repo, oldName, name)
		case inputPromptRemoteRemove:
			return m, remoteRemoveCmd(m.repo, name)
		case inputPromptRemotePrune:
			return m, remotePruneCmd(m.repo, name)
		case inputPromptWorktreeCreate:
			return m, worktreeAddCmd(m.repo, name, m.head.Branch)
		case inputPromptWorktreeMove:
			oldPath := m.confirmPath
			m.confirmPath = ""
			return m, worktreeMoveCmd(m.repo, oldPath, name)
		case inputPromptWorktreeDelete:
			return m, worktreeRemoveCmd(m.repo, name)
		case inputPromptBisectScript:
			return m, bisectRunCmd(m.repo, name)
		case inputPromptStashMessage:
			opts := git.StashOpts{Message: name}
			return m, stashPushCmd(m.repo, opts)
		case inputPromptStashRename:
			idx, ok := getStashIndex(m)
			if !ok {
				return m, notifyAppCmd("No stash selected", notification.Warning)
			}
			return m, stashRenameCmd(m.repo, idx, name)
		case inputPromptStashBranch:
			idx, ok := getStashIndex(m)
			if !ok {
				return m, notifyAppCmd("No stash selected", notification.Warning)
			}
			return m, stashBranchCmd(m.repo, name, idx)
		default:
			return m, nil
		}

	case tea.KeyEscape:
		m.inputPromptKind = inputPromptNone
		m.inputPromptLabel = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.inputPrompt, cmd = m.inputPrompt.Update(msg)
	return m, cmd
}

// --- Newly wired key handlers ---

// handleShowRefs opens the YankPopup for the current item.
func handleShowRefs(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil {
		return m, nil
	}

	hasURL := m.head.UpstreamRemote != ""
	hasTags := m.head.Tag != ""
	p := popup.NewYankPopup(m.tokens, nil, hasURL, hasTags)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupHelp // reuse a popup kind since yank has no dedicated kind
	return m, nil
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

	m.confirmPath = item.Entry.Path()
	return openBranchInput(m, inputPromptRenameFile, "Rename "+item.Entry.Path()+" to: ")
}

// handlePeekFile loads file content for a read-only overlay.
func handlePeekFile(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return m, nil
	}
	return m, loadPeekFileCmd(m.repo.Path(), item.Entry.Path())
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

// handleBranchSelected processes a branch selection result.
func handleBranchSelected(m Model, msg branchselect.SelectedMsg) (tea.Model, tea.Cmd) {
	kind := m.branchActionKind
	m.branchActionKind = branchActionNone

	switch kind {
	case branchActionCheckout, branchActionCheckoutLocal, branchActionCheckoutRecent:
		return m, checkoutBranchCmd(m.repo, msg.Name)
	case branchActionDelete:
		return m, deleteBranchCmd(m.repo, msg.Name)
	case branchActionPushElsewhere:
		opts := buildPushOpts(popup.Result{Switches: map[string]bool{}, Options: map[string]string{}})
		opts.Remote = msg.Name
		opts.Branch = m.head.Branch
		return m, pushCmd(m.repo, opts)
	case branchActionPushOther:
		opts := buildPushOpts(popup.Result{Switches: map[string]bool{}, Options: map[string]string{}})
		remote, _ := m.repo.SmartDefaultRemote(context.Background())
		opts.Remote = remote
		opts.Branch = msg.Name
		return m, pushCmd(m.repo, opts)
	case branchActionRebaseElsewhere:
		opts := m.rebaseSpecialOpts
		opts.Onto = msg.Name
		m.rebaseSpecialOpts = git.RebaseOpts{}
		return m, rebaseCmd(m.repo, opts)
	case branchActionLogOtherBranch:
		logOpts := git.LogOpts{MaxCount: 256, Decorate: true}
		return m, loadLogCmd(m.repo, logOpts, msg.Name)
	case branchActionBranchConfigure:
		return m, notifyAppCmd("Branch config for "+msg.Name+" (popup not yet available)", notification.Info)
	case branchActionMergeBranch:
		opts := m.mergeOpts
		opts.Branch = msg.Name
		kind := m.mergeActionKind
		m.mergeActionKind = mergeActionNone
		m.mergeOpts = git.MergeOpts{}
		switch kind {
		case mergeActionEdit:
			// merge + edit: just pass through, the editor will open
		case mergeActionNoCommit:
			opts.NoCommit = true
		case mergeActionSquash:
			opts.Squash = true
		case mergeActionDissolve:
			opts.Squash = true
			opts.NoCommit = true
		default:
			// mergeActionMerge, mergeActionAbsorb — default merge
		}
		return m, tea.Batch(
			mergeCmd(m.repo, opts),
			notifyAppCmd("Merging "+msg.Name+"...", notification.Info),
		)
	case branchActionWorktreeCheckout:
		return m, worktreeAddCmd(m.repo, "", msg.Name)
	default:
		return m, nil
	}
}

// --- Merge popup action handling ---

// handleMergePopupAction handles actions from the merge popup.
func handleMergePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inMerge := isInMerge(m.sections)

	if inMerge {
		switch result.Action {
		case "m": // Commit merge
			return m, mergeCommitCmd(m.repo)
		case "a": // Abort merge
			return m, mergeAbortCmd(m.repo)
		default:
			return m, notifyAppCmd("Unknown merge action: "+result.Action, notification.Warning)
		}
	}

	opts := buildMergeOpts(result)

	var kind mergeActionKind
	switch result.Action {
	case "m":
		kind = mergeActionMerge
	case "e":
		kind = mergeActionEdit
	case "n":
		kind = mergeActionNoCommit
	case "a":
		kind = mergeActionAbsorb
	case "p": // Preview merge
		return m, notifyAppCmd("Merge preview not yet implemented", notification.Info)
	case "s":
		kind = mergeActionSquash
	case "i":
		kind = mergeActionDissolve
	default:
		return m, notifyAppCmd("Unknown merge action: "+result.Action, notification.Warning)
	}

	m.mergeActionKind = kind
	m.mergeOpts = opts
	m.branchActionKind = branchActionMergeBranch
	return m, loadAllBranchesCmd(m.repo)
}

// buildMergeOpts builds MergeOpts from popup result switches and options.
func buildMergeOpts(result popup.Result) git.MergeOpts {
	return git.MergeOpts{
		FFOnly:         result.Switches["ff-only"],
		NoFF:           result.Switches["no-ff"],
		Strategy:       result.Options["strategy"],
		StrategyOption: result.Options["strategy-option"],
		DiffAlgorithm:  result.Options["Xdiff-algorithm"],
		GpgSign:        result.Options["gpg-sign"],
	}
}

// --- Cherry-pick popup action handling ---

// handleCherryPickPopupAction handles actions from the cherry-pick popup.
func handleCherryPickPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "pick")

	if inProgress {
		switch result.Action {
		case "A": // Continue
			return m, cherryPickContinueCmd(m.repo)
		case "s": // Skip
			return m, cherryPickSkipCmd(m.repo)
		case "a": // Abort
			return m, cherryPickAbortCmd(m.repo)
		default:
			return m, notifyAppCmd("Unknown cherry-pick action: "+result.Action, notification.Warning)
		}
	}

	opts := buildCherryPickOpts(result)

	var kind cherryPickActionKind
	switch result.Action {
	case "A":
		kind = cherryPickActionPick
	case "a":
		kind = cherryPickActionApply
	case "h":
		kind = cherryPickActionHarvest
	case "m":
		kind = cherryPickActionSquash
	case "d":
		kind = cherryPickActionDonate
	case "n":
		kind = cherryPickActionSpinout
	case "s":
		kind = cherryPickActionSpinoff
	default:
		return m, notifyAppCmd("Unknown cherry-pick action: "+result.Action, notification.Warning)
	}

	m.cherryPickActionKind = kind
	m.cherryPickOpts = opts
	return m, loadCommitsForSelectCmd(m.repo)
}

// buildCherryPickOpts builds CherryPickOpts from popup result switches and options.
func buildCherryPickOpts(result popup.Result) git.CherryPickOpts {
	mainline := 0
	if ml, ok := result.Options["mainline"]; ok && ml != "" {
		if n, err := parseMaxCount(ml); err == nil {
			mainline = n
		}
	}

	return git.CherryPickOpts{
		Mainline:           mainline,
		Strategy:           result.Options["strategy"],
		GpgSign:            result.Options["gpg-sign"],
		FF:                 result.Switches["ff"],
		ReferenceInMessage: result.Switches["x"],
		Edit:               result.Switches["edit"],
		Signoff:            result.Switches["signoff"],
	}
}

// handleCherryPickCommitSelected handles the user selecting a commit for a cherry-pick action.
func handleCherryPickCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.cherryPickOpts
	kind := m.cherryPickActionKind

	// Clear cherry-pick select state
	m.cherryPickActionKind = cherryPickActionNone
	m.cherryPickOpts = git.CherryPickOpts{}

	hashes := []string{msg.FullHash}

	switch kind {
	case cherryPickActionPick:
		return m, cherryPickCmd(m.repo, hashes, opts)
	case cherryPickActionApply:
		return m, cherryPickApplyCmd(m.repo, hashes, opts)
	case cherryPickActionHarvest:
		// Harvest: cherry-pick but don't remove from source
		return m, cherryPickCmd(m.repo, hashes, opts)
	case cherryPickActionSquash:
		// Squash: cherry-pick --no-commit (apply changes without commit)
		return m, cherryPickApplyCmd(m.repo, hashes, opts)
	case cherryPickActionDonate, cherryPickActionSpinout, cherryPickActionSpinoff:
		return m, notifyAppCmd("Cherry-pick action not yet implemented", notification.Info)
	default:
		return m, nil
	}
}

// --- Revert popup action handling ---

// handleRevertPopupAction handles actions from the revert popup.
func handleRevertPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "revert")

	if inProgress {
		switch result.Action {
		case "v": // Continue
			return m, revertContinueCmd(m.repo)
		case "s": // Skip
			return m, revertSkipCmd(m.repo)
		case "a": // Abort
			return m, revertAbortCmd(m.repo)
		default:
			return m, notifyAppCmd("Unknown revert action: "+result.Action, notification.Warning)
		}
	}

	opts := buildRevertOpts(result)

	switch result.Action {
	case "v": // Commit(s) — needs commit select
		m.revertActionKind = revertActionCommit
		m.revertOpts = opts
		return m, loadCommitsForSelectCmd(m.repo)
	case "V": // Changes (no commit) — needs commit select
		m.revertActionKind = revertActionChanges
		m.revertOpts = opts
		return m, loadCommitsForSelectCmd(m.repo)
	case "h": // Hunk — revert the current hunk
		return handleRevertHunk(m)
	default:
		return m, notifyAppCmd("Unknown revert action: "+result.Action, notification.Warning)
	}
}

// buildRevertOpts builds RevertOpts from popup result switches and options.
func buildRevertOpts(result popup.Result) git.RevertOpts {
	mainline := 0
	if ml, ok := result.Options["mainline"]; ok && ml != "" {
		if n, err := parseMaxCount(ml); err == nil {
			mainline = n
		}
	}

	return git.RevertOpts{
		Mainline: mainline,
		Strategy: result.Options["strategy"],
		GpgSign:  result.Options["gpg-sign"],
		Edit:     result.Switches["edit"],
		NoEdit:   result.Switches["no-edit"],
		Signoff:  result.Switches["signoff"],
	}
}

// handleRevertCommitSelected handles the user selecting a commit for a revert action.
func handleRevertCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.revertOpts
	kind := m.revertActionKind

	// Clear revert select state
	m.revertActionKind = revertActionNone
	m.revertOpts = git.RevertOpts{}

	hashes := []string{msg.FullHash}

	switch kind {
	case revertActionCommit:
		return m, revertCmd(m.repo, hashes, opts)
	case revertActionChanges:
		return m, revertChangesCmd(m.repo, hashes, opts)
	default:
		return m, nil
	}
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
	return m, discardHunkCmd(m.repo, item.Entry.Path(), hunk)
}

// --- Stash popup action handling ---

// handleStashPopupAction handles actions from the stash popup.
func handleStashPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildStashOpts(result)

	switch result.Action {
	// Stash group
	case "z": // both
		return m, stashPushCmd(m.repo, opts)
	case "i": // index
		opts.KeepIndex = true
		return m, stashPushCmd(m.repo, opts)
	case "w": // worktree
		// Stash only worktree changes (keep index)
		opts.KeepIndex = true
		return m, stashPushCmd(m.repo, opts)
	case "x": // keeping index
		opts.KeepIndex = true
		return m, stashPushCmd(m.repo, opts)
	case "P": // push (with message)
		return openBranchInput(m, inputPromptStashMessage, "Stash message: ")

	// Snapshot group
	case "Z": // snapshot both
		return m, stashSnapshotCmd(m.repo, opts, "snapshot")
	case "I": // snapshot index
		opts.KeepIndex = true
		return m, stashSnapshotCmd(m.repo, opts, "index snapshot")
	case "W": // snapshot worktree
		return m, stashSnapshotCmd(m.repo, opts, "worktree snapshot")
	case "r": // to wip ref
		return m, notifyAppCmd("Stash to WIP ref not yet implemented", notification.Info)

	// Use group
	case "p": // pop
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashPopCmd(m.repo, idx)
	case "a": // apply
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashApplyCmd(m.repo, idx)
	case "d": // drop
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashDropCmd(m.repo, idx)

	// Inspect group
	case "l": // list
		return m, notifyAppCmd("Stash list view not yet implemented", notification.Info)
	case "v": // show
		return m, notifyAppCmd("Stash show not yet implemented", notification.Info)

	// Transform group
	case "b": // branch
		return openBranchInput(m, inputPromptStashBranch, "Stash branch name: ")
	case "B": // branch here
		return openBranchInput(m, inputPromptStashBranch, "Stash branch name: ")
	case "m": // rename
		return openBranchInput(m, inputPromptStashRename, "New stash message: ")
	case "f": // format patch
		return m, notifyAppCmd("Stash format-patch not yet implemented", notification.Info)

	default:
		return m, notifyAppCmd("Unknown stash action: "+result.Action, notification.Warning)
	}
}

// buildStashOpts builds StashOpts from popup result switches.
func buildStashOpts(result popup.Result) git.StashOpts {
	return git.StashOpts{
		IncludeUntracked: result.Switches["include-untracked"],
		All:              result.Switches["all"],
	}
}

// --- Reset popup action handling ---

// handleResetPopupAction handles actions from the reset popup.
func handleResetPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "f": // file — reset the file under cursor
		path, ok := getCursorFilePath(m)
		if !ok {
			return m, notifyAppCmd("No file selected", notification.Warning)
		}
		return m, resetFileCmd(m.repo, path)
	case "b": // branch — select commit to reset branch to
		m.resetActionKind = resetActionBranch
		m.resetMode = git.ResetMixed
		return m, loadCommitsForSelectCmd(m.repo)
	default:
		// m/s/h/k/i/w — reset modes, need commit select for target
		mode, ok := resetModeForAction(result.Action)
		if !ok {
			return m, notifyAppCmd("Unknown reset action: "+result.Action, notification.Warning)
		}
		m.resetActionKind = resetActionBranch
		m.resetMode = mode
		return m, loadCommitsForSelectCmd(m.repo)
	}
}

// resetModeForAction maps a reset popup action key to a git.ResetMode.
func resetModeForAction(action string) (git.ResetMode, bool) {
	switch action {
	case "m":
		return git.ResetMixed, true
	case "s":
		return git.ResetSoft, true
	case "h":
		return git.ResetHard, true
	case "k":
		return git.ResetKeep, true
	case "i":
		return git.ResetIndex, true
	case "w":
		return git.ResetWorktree, true
	default:
		return "", false
	}
}

// handleResetCommitSelected handles the user selecting a commit for a reset action.
func handleResetCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	mode := m.resetMode

	// Clear reset select state
	m.resetActionKind = resetActionNone
	m.resetMode = ""

	return m, resetCmd(m.repo, msg.FullHash, mode)
}

// --- Tag popup action handling ---

// handleTagPopupAction handles actions from the tag popup.
func handleTagPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "t": // create tag
		return openBranchInput(m, inputPromptTagName, "Tag name: ")
	case "r": // release tag
		return openBranchInput(m, inputPromptTagRelease, "Release tag name: ")
	case "x": // delete tag
		return openBranchInput(m, inputPromptTagDelete, "Delete tag: ")
	case "p": // prune
		remote := defaultRemote(m.head)
		return m, tagPruneCmd(m.repo, remote)
	default:
		return m, notifyAppCmd("Unknown tag action: "+result.Action, notification.Warning)
	}
}

// buildTagOpts builds TagOpts from popup result switches and options.
func buildTagOpts(result popup.Result) git.TagOpts {
	return git.TagOpts{
		Force:     result.Switches["force"],
		Annotate:  result.Switches["annotate"],
		Sign:      result.Switches["sign"],
		LocalUser: result.Options["local-user"],
	}
}

// --- Remote popup action handling ---

// handleRemotePopupAction handles actions from the remote popup.
func handleRemotePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "a": // Add
		return openBranchInput(m, inputPromptRemoteName, "Remote name: ")
	case "r": // Rename
		return openBranchInput(m, inputPromptRemoteRename, "Rename remote to: ")
	case "x": // Remove
		return openBranchInput(m, inputPromptRemoteRemove, "Remove remote: ")
	case "C": // Configure
		return m, notifyAppCmd("Remote configure not yet implemented", notification.Info)
	case "p": // Prune stale branches
		return openBranchInput(m, inputPromptRemotePrune, "Prune remote: ")
	case "P": // Prune stale refspecs
		return openBranchInput(m, inputPromptRemotePrune, "Prune refspecs for remote: ")
	case "b": // Update default branch
		return m, notifyAppCmd("Update default branch not yet implemented", notification.Info)
	case "z": // Unshallow
		return m, notifyAppCmd("Unshallow not yet implemented", notification.Info)
	default:
		return m, notifyAppCmd("Unknown remote action: "+result.Action, notification.Warning)
	}
}

// --- Worktree popup action handling ---

// handleWorktreePopupAction handles actions from the worktree popup.
func handleWorktreePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "w": // Checkout — select branch for worktree
		m.branchActionKind = branchActionWorktreeCheckout
		return m, loadAllBranchesCmd(m.repo)
	case "W": // Create — prompt for path
		return openBranchInput(m, inputPromptWorktreeCreate, "Worktree path: ")
	case "g": // Goto — prompt for path
		return m, notifyAppCmd("Goto worktree: switch not supported in terminal", notification.Info)
	case "m": // Move — prompt for destination
		return openBranchInput(m, inputPromptWorktreeMove, "Move worktree to: ")
	case "D": // Delete — prompt for path
		return openBranchInput(m, inputPromptWorktreeDelete, "Delete worktree path: ")
	default:
		return m, notifyAppCmd("Unknown worktree action: "+result.Action, notification.Warning)
	}
}

// --- Bisect popup action handling ---

// handleBisectPopupAction handles actions from the bisect popup.
func handleBisectPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildBisectOpts(result)

	switch result.Action {
	case "B": // Start
		return m, bisectStartCmd(m.repo, opts)
	case "S": // Scripted / Run script
		return openBranchInput(m, inputPromptBisectScript, "Bisect script: ")
	case "b": // Bad
		hash := getCommitHashAtCursor(m)
		return m, bisectBadCmd(m.repo, hash)
	case "g": // Good
		hash := getCommitHashAtCursor(m)
		return m, bisectGoodCmd(m.repo, hash)
	case "s": // Skip
		hash := getCommitHashAtCursor(m)
		return m, bisectSkipCmd(m.repo, hash)
	case "r": // Reset
		return m, bisectResetCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown bisect action: "+result.Action, notification.Warning)
	}
}

// buildBisectOpts builds BisectOpts from popup result switches.
func buildBisectOpts(result popup.Result) git.BisectOpts {
	return git.BisectOpts{
		NoCheckout:  result.Switches["no-checkout"],
		FirstParent: result.Switches["first-parent"],
	}
}

// --- Ignore popup action handling ---

// handleIgnorePopupAction handles actions from the ignore popup.
func handleIgnorePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	path, ok := getCursorFilePath(m)
	if !ok {
		return m, notifyAppCmd("No file selected to ignore", notification.Warning)
	}

	pattern := git.IgnorePatternForPath(path)

	var scope git.IgnoreScope
	switch result.Action {
	case "t": // shared at top-level
		scope = git.IgnoreScopeTopLevel
	case "s": // shared in sub-directory
		scope = git.IgnoreScopeSubdir
	case "p": // privately for this repository
		scope = git.IgnoreScopePrivate
	case "g": // globally for this user
		scope = git.IgnoreScopeGlobal
	default:
		return m, notifyAppCmd("Unknown ignore action: "+result.Action, notification.Warning)
	}

	return m, ignoreCmd(m.repo, pattern, scope)
}

// --- Diff popup action handling ---

// handleDiffPopupAction handles actions from the diff popup.
func handleDiffPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "d": // this — expand inline diff for current item
		return handleToggle(m)
	case "h": // this..HEAD — show diff for commit vs HEAD
		return m, notifyAppCmd("Diff commit..HEAD not yet implemented", notification.Info)
	case "r": // range
		return m, notifyAppCmd("Diff range not yet implemented", notification.Info)
	case "p": // paths
		return m, notifyAppCmd("Diff paths not yet implemented", notification.Info)
	case "u": // unstaged
		return m, notifyAppCmd("Diff unstaged view not yet implemented", notification.Info)
	case "s": // staged
		return m, notifyAppCmd("Diff staged view not yet implemented", notification.Info)
	case "w": // worktree
		return m, notifyAppCmd("Diff worktree view not yet implemented", notification.Info)
	case "c": // Commit
		return m, notifyAppCmd("Diff commit view not yet implemented", notification.Info)
	case "t": // Stash
		return m, notifyAppCmd("Diff stash view not yet implemented", notification.Info)
	default:
		return m, notifyAppCmd("Unknown diff action: "+result.Action, notification.Warning)
	}
}

// --- Margin popup action handling ---

// handleMarginPopupAction handles actions from the margin popup.
func handleMarginPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "g": // Refresh buffer
		return m, loadStatusCmd(m.repo, m.cfg)
	case "L": // Toggle visibility
		return m, notifyAppCmd("Margin visibility toggled", notification.Info)
	case "l": // Cycle style
		return m, notifyAppCmd("Margin style cycled", notification.Info)
	case "d": // Toggle details
		return m, notifyAppCmd("Margin details toggled", notification.Info)
	case "x": // Toggle shortstat
		return m, notifyAppCmd("Margin shortstat toggled", notification.Info)
	default:
		return m, notifyAppCmd("Unknown margin action: "+result.Action, notification.Warning)
	}
}

// --- Help popup action handling ---

// handleHelpPopupAction handles actions from the help popup.
// When the user presses a key in the help popup, the popup returns that key
// as the action. We show a notification with what was selected.
func handleHelpPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	return m, notifyAppCmd("Action: "+result.Action, notification.Info)
}

// --- Helper functions for popup handlers ---

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
	return item.Entry.Path(), true
}

// getCommitHashAtCursor returns the commit hash at the cursor, or "" if not on a commit.
func getCommitHashAtCursor(m Model) string {
	item, _ := getCurrentItem(m)
	if item == nil || item.Commit == nil {
		return ""
	}
	return item.Commit.Hash
}
