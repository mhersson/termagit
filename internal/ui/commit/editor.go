package commit

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
	"github.com/mhersson/conjit/internal/ui/commit/vim"
)

// Model is the commit editor model.
type Model struct {
	repo   *git.Repository
	opts   git.CommitOpts
	cfg    *config.Config
	tokens theme.Tokens
	keys   KeyMap
	action string // "commit", "amend", "reword", "extend", etc.

	vimEditor *vim.Editor
	cycler    *git.CommitHistoryCycler

	diff     []git.FileDiff
	showDiff bool

	pendingKey  string // for ctrl+c ctrl+c / ctrl+c ctrl+k sequences
	commentChar string // from git core.commentChar, default "#"

	width, height int
	done          bool
	aborted       bool
	hash          string //nolint:unused // Stores commit hash after successful commit
	err           error  //nolint:unused // Stores error from commit operation
}

// New creates a new commit editor model.
func New(repo *git.Repository, opts git.CommitOpts, cfg *config.Config, tokens theme.Tokens, action string) Model {
	// Create vim tokens from theme tokens
	vimTokens := vim.Tokens{
		Normal:      tokens.Normal,
		CursorBlock: tokens.CursorBlock,
		Selection:   tokens.Selection,
	}

	editor := vim.NewEditor(vimTokens)

	return Model{
		repo:        repo,
		opts:        opts,
		cfg:         cfg,
		tokens:      tokens,
		keys:        DefaultKeyMap(),
		action:      action,
		vimEditor:   editor,
		showDiff:    cfg.CommitEditor.ShowStagedDiff,
		commentChar: "#", // Default, will be overridden from git config
	}
}

// Init initializes the model and starts loading data.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Load commit history for cycling
	if m.repo != nil {
		cmds = append(cmds, loadCommitHistoryCmd(m.repo))
	}

	// Load staged diff if enabled
	if m.showDiff && m.repo != nil {
		cmds = append(cmds, loadStagedDiffCmd(m.repo))
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vimEditor.SetSize(msg.Width-4, msg.Height-12)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case commitHistoryLoadedMsg:
		if msg.Err == nil && len(msg.Messages) > 0 {
			m.cycler = git.NewCycler(msg.Messages)
		}
		return m, nil

	case stagedDiffLoadedMsg:
		if msg.Err == nil {
			m.diff = msg.Diff
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle pending two-key sequences
	if m.pendingKey == "ctrl+c" {
		m.pendingKey = ""
		switch msg.Type {
		case tea.KeyCtrlC:
			// ctrl+c ctrl+c = Submit
			return m.submit()
		}
		// Check for 'k' key
		if msg.String() == "k" {
			// ctrl+c k = Abort
			return m.abort()
		}
		// Any other key cancels the sequence, fall through to normal handling
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		// First ctrl+c, wait for second key
		m.pendingKey = "ctrl+c"
		return m, nil
	}

	// History cycling works in all modes
	switch {
	case key.Matches(msg, m.keys.PrevMessage):
		if m.cycler != nil {
			text := m.cycler.Prev(m.vimEditor.Content())
			m.vimEditor.SetContent(text)
		}
		return m, nil

	case key.Matches(msg, m.keys.NextMessage):
		if m.cycler != nil {
			text := m.cycler.Next()
			m.vimEditor.SetContent(text)
		}
		return m, nil

	case key.Matches(msg, m.keys.ResetMessage):
		if m.cycler != nil {
			text := m.cycler.Reset()
			m.vimEditor.SetContent(text)
		}
		return m, nil
	}

	// 'q' only closes in normal mode
	if key.Matches(msg, m.keys.Close) && m.vimEditor.Mode() == vim.ModeNormal {
		return m.abort()
	}

	// Forward all other keys to VimEditor
	m.vimEditor.HandleKey(msg)
	return m, nil
}

// submit completes the commit.
func (m Model) submit() (tea.Model, tea.Cmd) {
	m.done = true
	m.aborted = false

	message := strings.TrimSpace(m.vimEditor.Content())
	m.opts.Message = message

	// Return command to perform the commit
	return m, func() tea.Msg {
		if m.repo == nil {
			return CommitEditorDoneMsg{Err: fmt.Errorf("no repository")}
		}
		hash, err := m.repo.Commit(context.Background(), m.opts)
		return CommitEditorDoneMsg{Hash: hash, Err: err}
	}
}

// abort cancels the commit.
func (m Model) abort() (tea.Model, tea.Cmd) {
	m.done = true
	m.aborted = true
	return m, func() tea.Msg {
		return CommitEditorAbortMsg{}
	}
}

// View renders the editor.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var b strings.Builder

	// Title with mode indicator
	title := m.titleForAction()
	modeStr := m.modeString()
	b.WriteString(m.tokens.Bold.Render(title))
	b.WriteString(" ")
	b.WriteString(m.tokens.Dim.Render(modeStr))
	b.WriteString("\n\n")

	// VimEditor content
	b.WriteString(m.vimEditor.View())
	b.WriteString("\n\n")

	// Help lines (matching Neogit style)
	b.WriteString(m.renderHelpLines())

	return b.String()
}

// modeString returns a string representation of the current vim mode.
func (m Model) modeString() string {
	switch m.vimEditor.Mode() {
	case vim.ModeNormal:
		return "[NORMAL]"
	case vim.ModeInsert:
		return "[INSERT]"
	case vim.ModeVisualLine:
		return "[V-LINE]"
	default:
		return ""
	}
}

// titleForAction returns the title based on the action.
func (m Model) titleForAction() string {
	switch m.action {
	case "amend":
		return "Amend Commit"
	case "reword":
		return "Reword Commit"
	case "extend":
		return "Extend Commit"
	case "fixup":
		return "Fixup Commit"
	case "squash":
		return "Squash Commit"
	default:
		return "Create Commit"
	}
}

// renderHelpLines renders the help section matching Neogit style.
func (m Model) renderHelpLines() string {
	c := m.commentChar
	style := m.tokens.Comment

	lines := []string{
		c,
		fmt.Sprintf("%s Commands:", c),
		fmt.Sprintf("%s   %-16s Close (normal mode)", c, m.keys.Close.Help().Key),
		fmt.Sprintf("%s   %-16s Submit", c, "<c-c><c-c>"),
		fmt.Sprintf("%s   %-16s Abort", c, "<c-c><c-k>"),
		fmt.Sprintf("%s   %-16s Previous Message", c, m.keys.PrevMessage.Help().Key),
		fmt.Sprintf("%s   %-16s Next Message", c, m.keys.NextMessage.Help().Key),
		fmt.Sprintf("%s   %-16s Reset Message", c, m.keys.ResetMessage.Help().Key),
	}

	var styled []string
	for _, line := range lines {
		styled = append(styled, style.Render(line))
	}

	return strings.Join(styled, "\n")
}

// Done returns whether the editor is done.
func (m Model) Done() bool {
	return m.done
}

// Aborted returns whether the editor was aborted.
func (m Model) Aborted() bool {
	return m.aborted
}

// SetSize sets the editor dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.vimEditor.SetSize(width-4, height-12)
}

// loadCommitHistoryCmd loads commit messages for history cycling.
func loadCommitHistoryCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		messages, err := repo.CommitMessagesForCycling(context.Background(), 50)
		return commitHistoryLoadedMsg{Messages: messages, Err: err}
	}
}

// loadStagedDiffCmd loads the staged diff for preview.
func loadStagedDiffCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		diff, err := repo.StagedDiff(context.Background(), "")
		return stagedDiffLoadedMsg{Diff: diff, Err: err}
	}
}

// OpenCommitEditorCmd returns a command to open the commit editor.
func OpenCommitEditorCmd(opts git.CommitOpts, action string) tea.Cmd {
	return func() tea.Msg {
		return OpenCommitEditorMsg{Opts: opts, Action: action}
	}
}
