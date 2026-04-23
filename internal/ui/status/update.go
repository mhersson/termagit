package status

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/ui/branchselect"
	"github.com/mhersson/termagit/internal/ui/commit"
	"github.com/mhersson/termagit/internal/ui/commitselect"
	"github.com/mhersson/termagit/internal/ui/commitview"
	"github.com/mhersson/termagit/internal/ui/notification"
	"github.com/mhersson/termagit/internal/ui/popup"
	"github.com/mhersson/termagit/internal/ui/shared"
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
			m.invalidateContent()
			if m.viewport.Width > 0 {
				m.applyViewportWithCursor()
			}
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

		// Save cursor context before replacing sections so we can preserve
		// position across watcher-triggered reloads.
		prevCursor := m.cursor
		prevSections := m.sections

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
			if restore.hunk >= 0 && m.cursor.Item >= 0 && m.cursor.Section < len(m.sections) {
				s := &m.sections[m.cursor.Section]
				if m.cursor.Item < len(s.Items) {
					item := &s.Items[m.cursor.Item]
					if item.Entry != nil && item.Entry.Path == restore.path {
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
		} else if len(prevSections) > 0 {
			// Watcher-triggered reload: preserve UI state and cursor position.
			preserveUIState(prevSections, m.sections)
			m.cursor = preserveCursorAcrossReload(prevSections, prevCursor, m.sections)
		} else {
			// Initial load: position cursor on first non-empty, non-hidden section
			m.cursor = findFirstValidCursor(m.sections)
		}

		// Update viewport content
		m.invalidateContent()
		if m.viewport.Width > 0 {
			m.applyViewportWithCursor()
			m.viewport.YOffset = 0
			ensureCursorVisible(&m, computeCursorLine(m))
		}
		return m, cmd

	case hunksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Save current screen position before updating
		m.ensureContent()
		oldCursorLine := computeCursorLine(m)
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
		m.invalidateContent()
		if m.viewport.Width > 0 {
			m.applyViewportWithCursor()
			newCursorLine := computeCursorLine(m)
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
		m.opInProgress = false
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
			m.invalidateContent()
		}
		return m, nil

	case shared.OpenPopupMsg:
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

	case remoteConfigLoadedMsg:
		if msg.err != nil {
			return m, notifyAppCmd("Failed to load remote config: "+msg.err.Error(), notification.Error)
		}
		p := popup.NewRemoteConfigPopup(m.tokens, nil, msg.remote)
		// Set current config values on the popup
		cfgs := p.GetConfig()
		for i := range cfgs {
			if v, ok := msg.values[cfgs[i].Label]; ok {
				cfgs[i].Value = v
			}
		}
		p.SetSize(m.width, m.height)
		m.popup = &p
		m.popupKind = PopupRemoteConfig
		return m, nil

	case branchConfigLoadedMsg:
		if msg.err != nil {
			return m, notifyAppCmd("Failed to load branch config: "+msg.err.Error(), notification.Error)
		}
		p := popup.NewBranchConfigPopup(m.tokens, nil, msg.branch,
			msg.remotes, msg.pullRebase, msg.globalPullRebase)
		cfgs := p.GetConfig()
		for i := range cfgs {
			if v, ok := msg.values[cfgs[i].Label]; ok {
				cfgs[i].Value = v
			}
		}
		p.SetSize(m.width, m.height)
		m.popup = &p
		m.popupKind = PopupBranchConfig
		return m, nil

	case openStashInCommitViewMsg:
		cv := commitview.New(m.repo, msg.ref, m.tokens, nil)
		cv.SetSize(m.width, m.height*60/100)
		cv.SetOverlayMode(true)
		m.commitView = &cv
		return m, cv.Init()

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

	// Handle visual mode exit (Esc exits visual mode before any other handling)
	if m.visualMode && key.Matches(msg, m.keys.ExitVisualMode) {
		m.visualMode = false
		m.visualAnchor = Cursor{}
		if m.viewport.Width > 0 {
			m.applyViewportWithCursor()
		}
		return m, nil
	}

	// Handle pending key sequences (e.g., "gg", "gp", "[c", "]c")
	if m.pendingKey == "g" {
		m.pendingKey = ""
		switch msg.String() {
		case "g":
			return handleGoToTop(m)
		case "p":
			return handleGoToParentRepo(m)
		}
	}

	if m.pendingBracket != "" {
		bracket := m.pendingBracket
		m.pendingBracket = ""
		if msg.String() == "c" {
			if bracket == "]" {
				return handleOpenOrScrollDown(m)
			}
			return handleOpenOrScrollUp(m)
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Close):
		return m, tea.Quit

	case key.Matches(msg, m.keys.MoveDown):
		m.cursor = moveCursor(m.sections, m.cursor, 1)
		// Update viewport to keep cursor visible
		if m.viewport.Width > 0 {
			m.applyViewportWithCursor()
		}
		return m, nil

	case key.Matches(msg, m.keys.MoveUp):
		m.cursor = moveCursor(m.sections, m.cursor, -1)
		// Update viewport to keep cursor visible
		if m.viewport.Width > 0 {
			m.applyViewportWithCursor()
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

	case key.Matches(msg, m.keys.VisualMode):
		// 'V' on a diff line enters visual mode; elsewhere it's a no-op.
		if m.cursor.Hunk >= 0 && m.cursor.Line >= 0 {
			return handleEnterVisualMode(m)
		}
		return m, nil

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

	case msg.String() == "]":
		m.pendingBracket = "]"
		return m, nil

	case msg.String() == "[":
		m.pendingBracket = "["
		return m, nil

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

// handleCommitViewKey delegates key handling to the commit view overlay.
func handleCommitViewKey(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cv := *m.commitView
	newCV, cmd := cv.Update(msg)
	cvModel := newCV.(commitview.Model)
	m.commitView = &cvModel

	if cvModel.Done() {
		m.commitView = nil
		m.invalidateContent()
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
		m.invalidateContent()

		// Handle popup result
		if result.Action != "" {
			return handlePopupAction(m, kind, result)
		}
	}

	return m, cmd
}

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
			opts := m.tagOpts
			m.tagOpts = git.TagOpts{}
			return m, tagCreateCmd(m.repo, name, "HEAD", opts)
		case inputPromptTagRelease:
			opts := m.tagOpts
			m.tagOpts = git.TagOpts{}
			return m, tagCreateCmd(m.repo, name, "HEAD", opts)
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
		case inputPromptRemoteConfigure:
			return m, openRemoteConfigCmd(m.repo, name)
		case inputPromptRemoteSetHead:
			return m, remoteSetHeadCmd(m.repo, name)
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
		case inputPromptCherryPickSpinout:
			hashes := m.donateHashes
			opts := m.cherryPickOpts
			m.donateHashes = nil
			m.cherryPickOpts = git.CherryPickOpts{}
			return m, cherryPickSpinoutCmd(m.repo, hashes, name, opts)
		case inputPromptCherryPickSpinoff:
			hashes := m.donateHashes
			opts := m.cherryPickOpts
			m.donateHashes = nil
			m.cherryPickOpts = git.CherryPickOpts{}
			return m, cherryPickSpinoffCmd(m.repo, hashes, m.head.Branch, name, opts)
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
