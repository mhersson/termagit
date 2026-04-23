package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/cmdlog"
	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/platform"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/branchselect"
	"github.com/mhersson/termagit/internal/ui/cmdhistory"
	"github.com/mhersson/termagit/internal/ui/commit"
	"github.com/mhersson/termagit/internal/ui/commitselect"
	"github.com/mhersson/termagit/internal/ui/commitview"
	"github.com/mhersson/termagit/internal/ui/diffview"
	"github.com/mhersson/termagit/internal/ui/logview"
	"github.com/mhersson/termagit/internal/ui/notification"
	"github.com/mhersson/termagit/internal/ui/popup"
	"github.com/mhersson/termagit/internal/ui/rebaseeditor"
	"github.com/mhersson/termagit/internal/ui/reflogview"
	"github.com/mhersson/termagit/internal/ui/refsview"
	"github.com/mhersson/termagit/internal/ui/shared"
	"github.com/mhersson/termagit/internal/ui/stashlist"
	"github.com/mhersson/termagit/internal/ui/status"
	"github.com/mhersson/termagit/internal/watcher"
)

// Screen represents the active screen.
type Screen int

const (
	ScreenStatus Screen = iota
	ScreenLog
	ScreenReflog
	ScreenCommitView
	ScreenRefsView
	ScreenStashList
	ScreenDiffView
	ScreenRebaseEditor
	ScreenCmdHistory
	ScreenCommitEditor
	ScreenCommitSelect
	ScreenBranchSelect
)

// Model is the main application model.
type Model struct {
	repo    *git.Repository
	cfg     *config.Config
	tokens  theme.Tokens
	logger  *cmdlog.Logger
	watcher *watcher.Watcher

	active         Screen
	previousScreen Screen // for returning from commit view
	status         status.Model
	commitEditor   commit.Model
	commitSelect   commitselect.Model
	branchSelect   branchselect.Model
	commitView     *commitview.Model
	diffView       *diffview.Model
	rebaseEditor   rebaseeditor.Model
	cmdHistory     *cmdhistory.Model
	logView        *logview.Model
	reflogView     *reflogview.Model
	refsView       *refsview.Model
	stashList      *stashlist.Model

	notifications notification.Stack

	width  int
	height int
}

// New creates a new application model.
func New(repo *git.Repository, cfg *config.Config, tokens theme.Tokens, logger *cmdlog.Logger) Model {
	keys := status.DefaultKeyMap()
	m := Model{
		repo:   repo,
		cfg:    cfg,
		tokens: tokens,
		logger: logger,
		active: ScreenStatus,
		status: status.New(repo, cfg, tokens, keys),
	}

	// Create watcher if enabled (started later via StartWatcher)
	if cfg.Filewatcher.Enabled {
		w, err := watcher.New(repo.GitDir())
		if err == nil {
			m.watcher = w
		}
	}

	return m
}

// StartWatcher begins file watching. Call after the tea.Program is created
// so that program.Send can be passed as the callback.
func (m *Model) StartWatcher(send func(tea.Msg)) {
	if m.watcher != nil {
		m.watcher.Start(send)
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.status.Init()
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate to active screen
		switch m.active {
		case ScreenStatus:
			var cmd tea.Cmd
			newStatus, cmd := m.status.Update(msg)
			m.status = newStatus.(status.Model)
			return m, cmd
		case ScreenCommitEditor:
			var cmd tea.Cmd
			newEditor, cmd := m.commitEditor.Update(msg)
			m.commitEditor = newEditor.(commit.Model)
			return m, cmd
		case ScreenCommitSelect:
			var cmd tea.Cmd
			newSelect, cmd := m.commitSelect.Update(msg)
			m.commitSelect = newSelect.(commitselect.Model)
			return m, cmd
		case ScreenBranchSelect:
			var cmd tea.Cmd
			newSelect, cmd := m.branchSelect.Update(msg)
			m.branchSelect = newSelect.(branchselect.Model)
			return m, cmd
		case ScreenRebaseEditor:
			var cmd tea.Cmd
			newEditor, cmd := m.rebaseEditor.Update(msg)
			m.rebaseEditor = newEditor.(rebaseeditor.Model)
			return m, cmd
		case ScreenCmdHistory:
			if m.cmdHistory != nil {
				m.cmdHistory.SetSize(msg.Width, msg.Height)
			}
		case ScreenLog:
			if m.logView != nil {
				m.logView.SetSize(msg.Width, msg.Height)
			}
		case ScreenReflog:
			if m.reflogView != nil {
				m.reflogView.SetSize(msg.Width, msg.Height)
			}
		case ScreenCommitView:
			if m.commitView != nil {
				m.commitView.SetSize(msg.Width, msg.Height)
			}
		case ScreenRefsView:
			if m.refsView != nil {
				m.refsView.SetSize(msg.Width, msg.Height)
			}
		case ScreenStashList:
			if m.stashList != nil {
				m.stashList.SetSize(msg.Width, msg.Height)
			}
		case ScreenDiffView:
			if m.diffView != nil {
				m.diffView.SetSize(msg.Width, msg.Height)
			}
		}
		return m, nil

	case tea.QuitMsg:
		if m.watcher != nil {
			m.watcher.Stop()
		}
		return m, tea.Quit

	case watcher.RepoChangedMsg:
		// Refresh the status buffer on repo changes, unless a multi-step
		// git operation is running (e.g. instant fixup commit + autosquash
		// rebase). In that case suppress the reload to avoid racing with
		// the in-flight operation for the git index lock.
		return m, m.status.MaybeInit()

	// Notification system
	case notification.NotifyMsg:
		dur := notification.DefaultDuration(msg.Kind)
		n := notification.New(msg.Message, msg.Kind, dur)
		m.notifications.Add(n)
		return m, n.ExpireCmd()

	case notification.ExpiredMsg:
		m.notifications.RemoveByID(msg.ID)
		return m, nil

	// Command history
	case status.OpenCmdHistoryMsg:
		return m.openCmdHistory()

	case cmdhistory.CloseMsg:
		m.active = ScreenStatus
		return m, nil

	case commit.OpenCommitEditorMsg:
		m.active = ScreenCommitEditor
		m.commitEditor = commit.New(m.repo, msg.Opts, m.cfg, m.tokens, msg.Action)
		m.commitEditor.SetSize(m.width, m.height)
		return m, m.commitEditor.Init()

	case commit.CommitEditorDoneMsg:
		m.active = ScreenStatus
		cmds := []tea.Cmd{m.status.Init()}
		if msg.Err != nil {
			n := notification.New("Commit failed: "+msg.Err.Error(), notification.Error, notification.DefaultDuration(notification.Error))
			m.notifications.Add(n)
			cmds = append(cmds, n.ExpireCmd())
		}
		return m, tea.Batch(cmds...)

	case commit.CommitEditorAbortMsg:
		// Return to status without any changes
		m.active = ScreenStatus
		return m, nil

	case commitselect.OpenCommitSelectMsg:
		m.active = ScreenCommitSelect
		m.commitSelect = commitselect.New(msg.Commits, m.tokens, m.width, m.height)
		return m, nil

	case commitselect.SelectedMsg:
		// Return to status and forward the selection
		m.active = ScreenStatus
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		return m, cmd

	case commitselect.AbortedMsg:
		// Return to status and forward the abort
		m.active = ScreenStatus
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		return m, cmd

	case branchselect.OpenBranchSelectMsg:
		m.active = ScreenBranchSelect
		m.branchSelect = branchselect.New(msg.Branches, m.tokens, m.width, m.height)
		return m, nil

	case branchselect.SelectedMsg:
		m.active = ScreenStatus
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		return m, cmd

	case branchselect.AbortedMsg:
		m.active = ScreenStatus
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		return m, cmd

	case popup.OpenRebaseEditorMsg:
		// Open rebase editor for an in-progress rebase (editing existing todo)
		m.active = ScreenRebaseEditor
		m.rebaseEditor = rebaseeditor.New(m.repo, m.tokens)
		m.rebaseEditor.SetSize(m.width, m.height)
		return m, m.rebaseEditor.Init()

	case rebaseeditor.OpenRebaseEditorMsg:
		// Open rebase editor with pre-generated entries (new interactive rebase)
		m.active = ScreenRebaseEditor
		m.rebaseEditor = rebaseeditor.NewWithEntries(m.repo, m.tokens, msg.Entries, msg.Base, msg.RebaseOpts)
		m.rebaseEditor.SetSize(m.width, m.height)
		return m, nil

	case rebaseeditor.RebaseEditorDoneMsg:
		m.active = ScreenStatus
		cmds := []tea.Cmd{m.status.Init()}
		if msg.Err != nil {
			n := notification.New("Rebase failed: "+msg.Err.Error(), notification.Error, notification.DefaultDuration(notification.Error))
			m.notifications.Add(n)
			cmds = append(cmds, n.ExpireCmd())
		}
		return m, tea.Batch(cmds...)

	case rebaseeditor.RebaseEditorAbortMsg:
		m.active = ScreenStatus
		return m, m.status.Init()

	case rebaseeditor.OpenCommitViewMsg:
		return m.openCommitView(msg.Hash, nil)

	// Log view
	case status.OpenLogViewMsg:
		return m.openLogView(msg.Commits, msg.HasMore, msg.Branch, msg.Opts)

	// Shared messages from any sub-view
	case shared.YankMsg:
		return m, yankToClipboardCmd(msg.Text)

	case shared.OpenPopupMsg:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		m.active = ScreenStatus
		return m, cmd

	case shared.OpenCommitViewMsg:
		return m.openCommitView(msg.Hash, nil)

	case shared.OpenCommitLinkMsg:
		return m, openCommitURLCmd(m.repo, msg.Hash)

	case logview.CloseLogViewMsg:
		m.active = ScreenStatus
		return m, nil

	// Reflog view
	case status.OpenReflogViewMsg:
		return m.openReflogView(msg.Entries, msg.Ref)

	case reflogview.CloseReflogViewMsg:
		m.active = ScreenStatus
		return m, nil

	// Refs view
	case status.OpenRefsViewMsg:
		return m.openRefsView(msg.Refs, msg.Remotes)

	case refsview.CloseRefsViewMsg:
		m.active = ScreenStatus
		return m, nil

	// Stash list view
	case status.OpenStashListMsg:
		return m.openStashList(msg.Stashes)

	case stashlist.CloseStashListMsg:
		m.active = ScreenStatus
		return m, nil

	// Commit view (has its own OpenCommitViewMsg with Filter field)
	case commitview.OpenCommitViewMsg:
		return m.openCommitView(msg.CommitID, msg.Filter)

	case commitview.CloseCommitViewMsg:
		m.active = m.previousScreen
		return m, nil

	case commitview.OpenFileMsg:
		return m, openFileCmd(m.repo.Path(), msg.Path)

	case commitview.OpenURLMsg:
		return m, openURLCmd(msg.URL)

	// Diff view
	case diffview.OpenDiffViewMsg:
		return m.openDiffView(msg.Source)

	case diffview.CloseDiffViewMsg:
		m.active = m.previousScreen
		return m, nil
	}

	// Delegate to active screen
	switch m.active {
	case ScreenStatus:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		return m, cmd
	case ScreenCommitEditor:
		newEditor, cmd := m.commitEditor.Update(msg)
		m.commitEditor = newEditor.(commit.Model)
		return m, cmd
	case ScreenCommitSelect:
		newSelect, cmd := m.commitSelect.Update(msg)
		m.commitSelect = newSelect.(commitselect.Model)
		return m, cmd
	case ScreenBranchSelect:
		newSelect, cmd := m.branchSelect.Update(msg)
		m.branchSelect = newSelect.(branchselect.Model)
		return m, cmd
	case ScreenRebaseEditor:
		newEditor, cmd := m.rebaseEditor.Update(msg)
		m.rebaseEditor = newEditor.(rebaseeditor.Model)
		return m, cmd
	case ScreenCmdHistory:
		if m.cmdHistory != nil {
			newCmdHistory, cmd := m.cmdHistory.Update(msg)
			ch := newCmdHistory.(cmdhistory.Model)
			m.cmdHistory = &ch
			return m, cmd
		}
		m.active = ScreenStatus
	case ScreenLog:
		if m.logView != nil {
			newLogView, cmd := m.logView.Update(msg)
			lv := newLogView.(logview.Model)
			m.logView = &lv
			return m, cmd
		}
		m.active = ScreenStatus
	case ScreenReflog:
		if m.reflogView != nil {
			newReflogView, cmd := m.reflogView.Update(msg)
			rv := newReflogView.(reflogview.Model)
			m.reflogView = &rv
			return m, cmd
		}
		m.active = ScreenStatus
	case ScreenCommitView:
		if m.commitView != nil {
			newCommitView, cmd := m.commitView.Update(msg)
			cv := newCommitView.(commitview.Model)
			m.commitView = &cv
			return m, cmd
		}
		m.active = ScreenStatus
	case ScreenRefsView:
		if m.refsView != nil {
			newRefsView, cmd := m.refsView.Update(msg)
			rv := newRefsView.(refsview.Model)
			m.refsView = &rv
			return m, cmd
		}
		m.active = ScreenStatus
	case ScreenStashList:
		if m.stashList != nil {
			newStashList, cmd := m.stashList.Update(msg)
			sl := newStashList.(stashlist.Model)
			m.stashList = &sl
			return m, cmd
		}
		m.active = ScreenStatus
	case ScreenDiffView:
		if m.diffView != nil {
			newDiffView, cmd := m.diffView.Update(msg)
			dv := newDiffView.(diffview.Model)
			m.diffView = &dv
			return m, cmd
		}
		m.active = ScreenStatus
	}

	return m, nil
}

// openCmdHistory switches to the command history screen.
func (m Model) openCmdHistory() (Model, tea.Cmd) { //nolint:unparam // tea.Cmd reserved for future async init
	entries := m.logger.Entries()
	ch := cmdhistory.New(entries, m.tokens, m.width, m.height)
	m.cmdHistory = &ch
	m.active = ScreenCmdHistory
	return m, nil
}

// openLogView switches to the log view screen.
func (m Model) openLogView(commits []git.LogEntry, hasMore bool, branch string, opts *git.LogOpts) (Model, tea.Cmd) { //nolint:unparam // tea.Cmd reserved for future async init
	lv := logview.New(commits, m.repo, m.tokens, opts, hasMore, branch)
	lv.SetSize(m.width, m.height)
	m.logView = &lv
	m.active = ScreenLog
	return m, nil
}

// openReflogView switches to the reflog view screen.
func (m Model) openReflogView(entries []git.ReflogEntry, ref string) (Model, tea.Cmd) { //nolint:unparam // tea.Cmd reserved for future async init
	rv := reflogview.New(entries, m.tokens, ref)
	rv.SetSize(m.width, m.height)
	m.reflogView = &rv
	m.active = ScreenReflog
	return m, nil
}

// openCommitView switches to the commit view screen.
func (m Model) openCommitView(commitID string, filter []string) (Model, tea.Cmd) {
	// Singleton pattern: if already viewing this commit, no-op
	if m.active == ScreenCommitView && m.commitView != nil && m.commitView.CommitID() == commitID {
		return m, nil
	}

	// Save current screen to return to on close
	m.previousScreen = m.active

	cv := commitview.New(m.repo, commitID, m.tokens, filter)
	cv.SetSize(m.width, m.height)
	m.commitView = &cv
	m.active = ScreenCommitView
	return m, cv.Init()
}

// openRefsView switches to the refs view screen.
func (m Model) openRefsView(refs *git.RefsResult, remotes []git.Remote) (Model, tea.Cmd) { //nolint:unparam // tea.Cmd reserved for future async init
	rv := refsview.New(refs, remotes, m.repo, m.tokens)
	rv.SetSize(m.width, m.height)
	m.refsView = &rv
	m.active = ScreenRefsView
	return m, nil
}

// openStashList switches to the stash list view screen.
func (m Model) openStashList(stashes []git.StashEntry) (Model, tea.Cmd) { //nolint:unparam // tea.Cmd reserved for future async init
	sl := stashlist.New(stashes, m.repo, m.tokens)
	sl.SetSize(m.width, m.height)
	m.stashList = &sl
	m.active = ScreenStashList
	return m, nil
}

// openDiffView switches to the diff view screen.
func (m Model) openDiffView(source diffview.DiffSource) (Model, tea.Cmd) {
	m.previousScreen = m.active
	dv := diffview.New(m.repo, source, m.cfg, m.tokens)
	dv.SetSize(m.width, m.height)
	m.diffView = &dv
	m.active = ScreenDiffView
	return m, dv.Init()
}

// View renders the model.
func (m Model) View() string {
	var base string
	switch m.active {
	case ScreenStatus:
		base = m.status.View()
	case ScreenCommitEditor:
		base = m.commitEditor.View()
	case ScreenCommitSelect:
		base = m.commitSelect.View()
	case ScreenBranchSelect:
		base = m.branchSelect.View()
	case ScreenCmdHistory:
		if m.cmdHistory != nil {
			base = m.cmdHistory.View()
		} else {
			base = "Command history not available"
		}
	case ScreenRebaseEditor:
		base = m.rebaseEditor.View()
	case ScreenLog:
		if m.logView != nil {
			base = m.logView.View()
		} else {
			base = "Log view not available"
		}
	case ScreenReflog:
		if m.reflogView != nil {
			base = m.reflogView.View()
		} else {
			base = "Reflog view not available"
		}
	case ScreenCommitView:
		if m.commitView != nil {
			base = m.commitView.View()
		} else {
			base = "Commit view not available"
		}
	case ScreenRefsView:
		if m.refsView != nil {
			base = m.refsView.View()
		} else {
			base = "Refs view not available"
		}
	case ScreenStashList:
		if m.stashList != nil {
			base = m.stashList.View()
		} else {
			base = "Stash list not available"
		}
	case ScreenDiffView:
		if m.diffView != nil {
			base = m.diffView.View()
		} else {
			base = "Diff view not available"
		}
	default:
		base = "Unknown screen"
	}

	// Overlay notifications on top-right
	notifView := m.notifications.View(m.tokens, m.width-2)
	if notifView != "" {
		base = notification.Overlay(base, notifView, m.width)
	}

	// Overlay confirmation dialog (centered)
	if m.active == ScreenStatus {
		confirmView := m.status.ConfirmView(m.width - 4)
		if confirmView != "" {
			base = notification.CenterOverlay(base, confirmView, m.width, m.height)
		}

		inputView := m.status.InputPromptView(m.width - 4)
		if inputView != "" {
			base = notification.CenterOverlay(base, inputView, m.width, m.height)
		}
	}

	return base
}

// yankToClipboardCmd copies text to system clipboard.
func yankToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
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

// openFileCmd opens a file in the default editor using tea.ExecProcess.
// This properly suspends the TUI, runs the editor, then resumes.
func openFileCmd(repoPath, path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	fullPath := filepath.Join(repoPath, path)
	c := exec.Command(editor, "--", fullPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return notification.NotifyMsg{
				Message: "Failed to open file: " + err.Error(),
				Kind:    notification.Error,
			}
		}
		return nil
	})
}

// openCommitURLCmd resolves a commit hash to a web URL and opens it.
func openCommitURLCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		url, err := repo.CommitURL(context.Background(), hash)
		if err != nil || url == "" {
			return notification.NotifyMsg{
				Message: "Couldn't determine commit URL",
				Kind:    notification.Warning,
			}
		}
		if err := platform.Open(url); err != nil {
			return notification.NotifyMsg{
				Message: "Failed to open URL: " + err.Error(),
				Kind:    notification.Error,
			}
		}
		return notification.NotifyMsg{
			Message: "Opening " + url,
			Kind:    notification.Info,
		}
	}
}

// openURLCmd opens a URL in the default browser.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		if err := platform.Open(url); err != nil {
			return notification.NotifyMsg{
				Message: "Failed to open URL: " + err.Error(),
				Kind:    notification.Error,
			}
		}
		return nil
	}
}
