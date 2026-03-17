package status

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
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
		// Position cursor on first non-empty, non-hidden section
		m.cursor = findFirstValidCursor(m.sections)

		// Update viewport content
		if m.viewport.Width > 0 {
			content, cursorLine := renderContent(m)
			m.viewport.SetContent(content)
			ensureCursorVisible(&m, cursorLine)
		}
		return m, nil

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
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func handleKeyMsg(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	// All popup keys - stub with notification
	case key.Matches(msg, m.keys.HelpPopup),
		key.Matches(msg, m.keys.CherryPickPopup),
		key.Matches(msg, m.keys.DiffPopup),
		key.Matches(msg, m.keys.RemotePopup),
		key.Matches(msg, m.keys.PushPopup),
		key.Matches(msg, m.keys.ResetPopup),
		key.Matches(msg, m.keys.StashPopup),
		key.Matches(msg, m.keys.IgnorePopup),
		key.Matches(msg, m.keys.TagPopup),
		key.Matches(msg, m.keys.BranchPopup),
		key.Matches(msg, m.keys.BisectPopup),
		key.Matches(msg, m.keys.WorktreePopup),
		key.Matches(msg, m.keys.CommitPopup),
		key.Matches(msg, m.keys.FetchPopup),
		key.Matches(msg, m.keys.LogPopup),
		key.Matches(msg, m.keys.MarginPopup),
		key.Matches(msg, m.keys.MergePopup),
		key.Matches(msg, m.keys.PullPopup),
		key.Matches(msg, m.keys.RebasePopup),
		key.Matches(msg, m.keys.RevertPopup):
		m.notification = "Popups not yet implemented (Phase 6)"
		return m, notifyCmd(3 * time.Second)

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
		return m, discardHunkCmd(m.repo, path, hunkIdx)

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

	// If on a hunk, stage just the hunk
	if m.cursor.Hunk >= 0 && len(item.Hunks) > m.cursor.Hunk {
		return m, stageHunkCmd(m.repo, item.Entry.Path(), m.cursor.Hunk)
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

	// If on a hunk, unstage just the hunk
	if m.cursor.Hunk >= 0 && len(item.Hunks) > m.cursor.Hunk {
		return m, unstageHunkCmd(m.repo, item.Entry.Path(), m.cursor.Hunk)
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
