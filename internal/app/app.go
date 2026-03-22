package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"


	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/cmdlog"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/platform"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/branchselect"
	"github.com/mhersson/conjit/internal/ui/cmdhistory"
	"github.com/mhersson/conjit/internal/ui/commit"
	"github.com/mhersson/conjit/internal/ui/commitselect"
	"github.com/mhersson/conjit/internal/ui/commitview"
	"github.com/mhersson/conjit/internal/ui/logview"
	"github.com/mhersson/conjit/internal/ui/notification"
	"github.com/mhersson/conjit/internal/ui/popup"
	"github.com/mhersson/conjit/internal/ui/rebaseeditor"
	"github.com/mhersson/conjit/internal/ui/reflogview"
	"github.com/mhersson/conjit/internal/ui/status"
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

// SwitchScreenMsg is sent to switch to a different screen.
type SwitchScreenMsg struct {
	Screen     Screen
	CommitHash string // for ScreenCommitView
}

// Model is the main application model.
type Model struct {
	repo   *git.Repository
	cfg    *config.Config
	tokens theme.Tokens
	logger *cmdlog.Logger

	active         Screen
	previousScreen Screen // for returning from commit view
	status         status.Model
	commitEditor   commit.Model
	commitSelect   commitselect.Model
	branchSelect   branchselect.Model
	commitView     *commitview.Model
	rebaseEditor   rebaseeditor.Model
	cmdHistory     *cmdhistory.Model
	logView        *logview.Model
	reflogView     *reflogview.Model

	notifications notification.Stack

	width  int
	height int
}

// New creates a new application model.
func New(repo *git.Repository, cfg *config.Config, tokens theme.Tokens, logger *cmdlog.Logger) Model {
	keys := status.DefaultKeyMap()
	return Model{
		repo:   repo,
		cfg:    cfg,
		tokens: tokens,
		logger: logger,
		active: ScreenStatus,
		status: status.New(repo, cfg, tokens, keys),
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
		}
		return m, nil

	// Notification system
	case notification.NotifyMsg:
		dur := notification.DefaultDuration(msg.Kind)
		n := notification.New(msg.Message, msg.Kind, dur)
		m.notifications.Add(n)
		return m, n.ExpireCmd()

	case notification.ExpiredMsg:
		m.notifications.RemoveByID(msg.ID)
		return m, nil

	case SwitchScreenMsg:
		m.active = msg.Screen
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

	case logview.CloseLogViewMsg:
		m.active = ScreenStatus
		return m, nil

	case logview.OpenPopupMsg:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		m.active = ScreenStatus
		return m, cmd

	case logview.YankMsg:
		return m, yankToClipboardCmd(msg.Text)

	case logview.OpenCommitLinkMsg:
		return m, openCommitURLCmd(m.repo, msg.Hash)

	// Reflog view
	case status.OpenReflogViewMsg:
		return m.openReflogView(msg.Entries, msg.Ref)

	case reflogview.CloseReflogViewMsg:
		m.active = ScreenStatus
		return m, nil

	case reflogview.OpenCommitViewMsg:
		return m.openCommitView(msg.Hash, nil)

	case reflogview.OpenPopupMsg:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		m.active = ScreenStatus
		return m, cmd

	case reflogview.YankMsg:
		return m, yankToClipboardCmd(msg.Text)

	case reflogview.OpenCommitLinkMsg:
		return m, openCommitURLCmd(m.repo, msg.Hash)

	// Commit view
	case commitview.OpenCommitViewMsg:
		return m.openCommitView(msg.CommitID, msg.Filter)

	case commitview.CloseCommitViewMsg:
		// Return to the screen that opened the commit view
		m.active = m.previousScreen
		return m, nil

	case commitview.YankMsg:
		return m, yankToClipboardCmd(msg.Text)

	case commitview.OpenPopupMsg:
		newStatus, cmd := m.status.Update(msg)
		m.status = newStatus.(status.Model)
		m.active = ScreenStatus
		return m, cmd

	case commitview.OpenFileMsg:
		return m, openFileCmd(m.repo.Path(), msg.Path)

	case commitview.OpenURLMsg:
		return m, openURLCmd(msg.URL)
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
	case ScreenLog:
		if m.logView != nil {
			newLogView, cmd := m.logView.Update(msg)
			lv := newLogView.(logview.Model)
			m.logView = &lv
			return m, cmd
		}
	case ScreenReflog:
		if m.reflogView != nil {
			newReflogView, cmd := m.reflogView.Update(msg)
			rv := newReflogView.(reflogview.Model)
			m.reflogView = &rv
			return m, cmd
		}
	case ScreenCommitView:
		if m.commitView != nil {
			newCommitView, cmd := m.commitView.Update(msg)
			cv := newCommitView.(commitview.Model)
			m.commitView = &cv
			return m, cmd
		}
	}

	return m, nil
}

// openCmdHistory switches to the command history screen.
func (m Model) openCmdHistory() (Model, tea.Cmd) {
	entries := m.logger.Entries()
	ch := cmdhistory.New(entries, m.tokens, m.width, m.height)
	m.cmdHistory = &ch
	m.active = ScreenCmdHistory
	return m, nil
}

// openLogView switches to the log view screen.
func (m Model) openLogView(commits []git.LogEntry, hasMore bool, branch string, opts *git.LogOpts) (Model, tea.Cmd) {
	lv := logview.New(commits, m.repo, m.tokens, opts, hasMore, branch)
	lv.SetSize(m.width, m.height)
	m.logView = &lv
	m.active = ScreenLog
	return m, nil
}

// openReflogView switches to the reflog view screen.
func (m Model) openReflogView(entries []git.ReflogEntry, ref string) (Model, tea.Cmd) {
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
	default:
		base = "Unknown screen"
	}

	// Overlay notifications on top-right
	notifView := m.notifications.View(m.tokens, 50)
	if notifView != "" {
		base = notification.Overlay(base, notifView, m.width)
	}

	// Overlay confirmation dialog (centered)
	if m.active == ScreenStatus {
		confirmView := m.status.ConfirmView(50)
		if confirmView != "" {
			base = notification.CenterOverlay(base, confirmView, m.width, m.height)
		}

		inputView := m.status.InputPromptView(60)
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
	c := exec.Command(editor, fullPath)
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
