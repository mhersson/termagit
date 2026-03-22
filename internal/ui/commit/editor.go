package commit

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/mhersson/termagit/internal/ui/commit/vim"
	"github.com/mhersson/termagit/internal/ui/notification"
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
	branch      string // current branch name
	status      *git.StatusResult

	// Loading state tracking
	commentCharLoaded  bool
	statusLoaded       bool
	diffLoaded         bool
	contentInitialized bool

	headMessage string // pre-populated message for reword/amend

	width, height int
	done          bool
	aborted       bool
	generating    bool   // true while an external generate command is running
	repoPath      string // working directory for external commands
}

// New creates a new commit editor model.
func New(repo *git.Repository, opts git.CommitOpts, cfg *config.Config, tokens theme.Tokens, action string) Model {
	// Create vim tokens from theme tokens
	vimTokens := vim.Tokens{
		Normal:      tokens.Normal,
		CursorBlock: tokens.CursorBlock,
		Selection:   tokens.Selection,
		Comment:     tokens.Comment,

		// Diff syntax highlighting
		DiffAdd:        tokens.DiffAdd,
		DiffDelete:     tokens.DiffDelete,
		DiffContext:    tokens.DiffContext,
		DiffHunkHeader: tokens.DiffHunkHeader,
		DiffHeader:     tokens.Dim, // Use dim for diff headers (diff --git, ---, +++)
	}

	initialMode := vim.ModeInsert
	if cfg.CommitEditor.DisableInsertOnCommit {
		initialMode = vim.ModeNormal
	}

	editor := vim.NewEditor(vimTokens, initialMode)

	var repoPath string
	var headMessage string
	if repo != nil {
		repoPath = repo.Path()
		// Load HEAD commit message synchronously for reword/amend
		if action == "reword" || action == "amend" {
			msg, err := repo.HeadCommitMessage(context.Background())
			if err == nil {
				headMessage = msg
			}
		}
	}

	return Model{
		repo:        repo,
		opts:        opts,
		cfg:         cfg,
		tokens:      tokens,
		keys:        DefaultKeyMap(),
		action:      action,
		vimEditor:   editor,
		showDiff:    cfg.CommitEditor.ShowStagedDiff,
		repoPath:    repoPath,
		commentChar: "#", // Default, will be overridden from git config
		headMessage: headMessage,
	}
}

// Init initializes the model and starts loading data.
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	if m.repo != nil {
		// Load commit history for cycling
		cmds = append(cmds, loadCommitHistoryCmd(m.repo))

		// Load comment character from git config
		cmds = append(cmds, loadCommentCharCmd(m.repo))

		// Load branch name
		cmds = append(cmds, loadBranchCmd(m.repo))

		// Load status for template
		cmds = append(cmds, loadStatusCmd(m.repo))
	}

	// Load staged diff if enabled
	if m.showDiff && m.repo != nil {
		cmds = append(cmds, loadStagedDiffCmd(m.repo))
	}
	// Note: diffLoaded is set in Update when stagedDiffLoadedMsg arrives,
	// or defaults to true when showDiff is false (handled in maybeInitializeContent)

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
		// Reserve 2 lines: top bar + 1 blank line before editor
		m.vimEditor.SetSize(msg.Width-4, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case commitHistoryLoadedMsg:
		if msg.Err == nil && len(msg.Messages) > 0 {
			m.cycler = git.NewCycler(msg.Messages)
		}
		return m, nil

	case stagedDiffLoadedMsg:
		m.diffLoaded = true
		if msg.Err == nil {
			m.diff = msg.Diff
		}
		return m.maybeInitializeContent()

	case commentCharLoadedMsg:
		m.commentCharLoaded = true
		if msg.Err == nil && msg.Char != "" {
			m.commentChar = msg.Char
		}
		return m.maybeInitializeContent()

	case branchLoadedMsg:
		if msg.Err == nil {
			m.branch = msg.Branch
		}
		return m, nil

	case statusLoadedMsg:
		m.statusLoaded = true
		if msg.Err == nil {
			m.status = msg.Status
		}
		return m.maybeInitializeContent()

	case generateCommitMessageMsg:
		m.generating = false
		if msg.Err != nil {
			return m, func() tea.Msg {
				return notification.NotifyMsg{
					Message: "Generate failed: " + msg.Err.Error(),
					Kind:    notification.Error,
				}
			}
		}
		generated := strings.TrimSpace(ansi.Strip(msg.Message))
		if generated != "" {
			m.vimEditor.SetContent(generated + "\n" + m.buildCommentContent())
			m.vimEditor.SetCursor(0, 0)
		}
		return m, func() tea.Msg {
			return notification.NotifyMsg{
				Message: "Commit message generated",
				Kind:    notification.Success,
			}
		}
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

	// Ctrl+G generates a commit message (normal mode only, when configured)
	if key.Matches(msg, m.keys.GenerateMessage) && m.vimEditor.Mode() == vim.ModeNormal {
		cmd := m.cfg.CommitEditor.GenerateCommitMessageCommand
		if cmd == "" || m.generating || m.repoPath == "" {
			return m, nil
		}
		m.generating = true
		return m, tea.Batch(
			generateCommitMessageCmd(cmd, m.repoPath),
			func() tea.Msg {
				return notification.NotifyMsg{
					Message: "Generating commit message…",
					Kind:    notification.Info,
				}
			},
		)
	}

	// Forward all other keys to VimEditor
	m.vimEditor.HandleKey(msg)
	return m, nil
}

// submit completes the commit.
func (m Model) submit() (tea.Model, tea.Cmd) {
	m.done = true
	m.aborted = false

	message := m.extractMessage()
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

// extractMessage extracts the commit message from the buffer content.
// It filters out comment lines and stops at the scissors line.
func (m Model) extractMessage() string {
	lines := strings.Split(m.vimEditor.Content(), "\n")
	var msgLines []string

	for _, line := range lines {
		// Stop at scissors line
		if strings.Contains(line, "> 8 ") || strings.Contains(line, ">8") {
			break
		}
		// Skip comment lines
		if strings.HasPrefix(line, m.commentChar) {
			continue
		}
		msgLines = append(msgLines, line)
	}

	return strings.TrimSpace(strings.Join(msgLines, "\n"))
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

	// Top bar: mode badge (left) + centered title, full-width CursorBg background
	b.WriteString(m.renderTopBar())
	b.WriteString("\n")

	// VimEditor content (includes the git-style template)
	b.WriteString(m.vimEditor.View())

	return b.String()
}

// renderTopBar renders the header bar with mode badge and centered title.
// The CursorBg background covers the entire terminal width.
// The mode badge sits on the left with its own colored background.
func (m Model) renderTopBar() string {
	mode := m.modeString()
	title := m.titleForAction()
	if m.generating {
		title += " — Generating…"
	}

	// Render mode badge with padding: " [NORMAL] "
	badge := m.modeStyle().Render(" " + mode + " ")
	badgeWidth := lipgloss.Width(badge)

	// Calculate center position for the title within the full width
	titleWidth := len(title)
	centerPos := (m.width - titleWidth) / 2
	gapAfterBadge := centerPos - badgeWidth
	if gapAfterBadge < 1 {
		gapAfterBadge = 1
	}

	// Title styled with bold + CursorBg background
	titleStyle := m.tokens.Bold.Background(m.tokens.EditorBar.GetBackground())
	styledTitle := titleStyle.Render(title)

	// Gap and right fill use CursorBg background explicitly
	gap := m.tokens.EditorBar.Render(strings.Repeat(" ", gapAfterBadge))
	rightFill := m.width - badgeWidth - gapAfterBadge - titleWidth
	if rightFill < 0 {
		rightFill = 0
	}
	fill := m.tokens.EditorBar.Render(strings.Repeat(" ", rightFill))

	return badge + gap + styledTitle + fill
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

// modeStyle returns the lipgloss style for the current vim mode badge.
func (m Model) modeStyle() lipgloss.Style {
	switch m.vimEditor.Mode() {
	case vim.ModeNormal:
		return m.tokens.EditorModeNormal
	case vim.ModeInsert:
		return m.tokens.EditorModeInsert
	case vim.ModeVisualLine:
		return m.tokens.EditorModeVisual
	default:
		return m.tokens.EditorModeNormal
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
	// Reserve 2 lines: top bar + 1 blank line before editor
	m.vimEditor.SetSize(width-4, height-2)
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

// loadCommentCharCmd loads the git comment character from config.
func loadCommentCharCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		char, err := repo.GetConfigValue(context.Background(), "core.commentChar")
		if char == "" {
			char = "#" // Default
		}
		return commentCharLoadedMsg{Char: char, Err: err}
	}
}

// loadBranchCmd loads the current branch name.
func loadBranchCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		branch, _, err := repo.HeadInfo(context.Background())
		return branchLoadedMsg{Branch: branch, Err: err}
	}
}

// loadStatusCmd loads the git status for the template.
func loadStatusCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		status, err := repo.Status(context.Background())
		return statusLoadedMsg{Status: status, Err: err}
	}
}

// maybeInitializeContent checks if all required data is loaded and initializes
// the buffer content with the git-style template.
func (m Model) maybeInitializeContent() (tea.Model, tea.Cmd) {
	// Wait until all required data is loaded
	// diffLoaded is true when stagedDiffLoadedMsg arrives, or when showDiff is false
	diffReady := m.diffLoaded || !m.showDiff
	if !m.commentCharLoaded || !m.statusLoaded || !diffReady {
		return m, nil
	}

	// Only initialize once
	if m.contentInitialized {
		return m, nil
	}

	m.contentInitialized = true
	content := m.buildInitialContent()
	m.vimEditor.SetContent(content)
	// Ensure cursor is at the top of the buffer (first line where user types message)
	m.vimEditor.SetCursor(0, 0)

	return m, nil
}

// buildInitialContent generates the git-style commit message template.
func (m Model) buildInitialContent() string {
	c := m.commentChar
	var b strings.Builder

	// Initial message area: pre-populate for reword/amend, empty for new commits
	if m.headMessage != "" {
		b.WriteString(m.headMessage)
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// Git header comment
	fmt.Fprintf(&b, "%s Please enter the commit message for your changes. Lines starting\n", c)
	fmt.Fprintf(&b, "%s with '%s' will be ignored, and an empty message aborts the commit.\n", c, c)
	fmt.Fprintf(&b, "%s\n", c)

	// Commands section (like Neogit)
	fmt.Fprintf(&b, "%s Commands:\n", c)
	fmt.Fprintf(&b, "%s   %-16s Close\n", c, m.keys.Close.Help().Key)
	fmt.Fprintf(&b, "%s   %-16s Submit\n", c, "<c-c><c-c>")
	fmt.Fprintf(&b, "%s   %-16s Abort\n", c, "<c-c><c-k>")
	fmt.Fprintf(&b, "%s   %-16s Previous Message\n", c, m.keys.PrevMessage.Help().Key)
	fmt.Fprintf(&b, "%s   %-16s Next Message\n", c, m.keys.NextMessage.Help().Key)
	fmt.Fprintf(&b, "%s   %-16s Reset Message\n", c, m.keys.ResetMessage.Help().Key)
	if m.cfg.CommitEditor.GenerateCommitMessageCommand != "" {
		fmt.Fprintf(&b, "%s   %-16s Generate Message\n", c, m.keys.GenerateMessage.Help().Key)
	}
	fmt.Fprintf(&b, "%s\n", c)

	// Branch info
	if m.branch != "" {
		fmt.Fprintf(&b, "%s On branch %s\n", c, m.branch)
	}

	// Status sections
	if m.status != nil {
		// Changes to be committed (staged)
		if len(m.status.Staged) > 0 {
			fmt.Fprintf(&b, "%s Changes to be committed:\n", c)
			for _, entry := range m.status.Staged {
				modeText := git.ModeText[string(entry.Staged)]
				if modeText == "" {
					modeText = "modified"
				}
				fmt.Fprintf(&b, "%s    %s:   %s\n", c, modeText, entry.Path())
			}
			fmt.Fprintf(&b, "%s\n", c)
		}

		// Changes not staged for commit (unstaged)
		if len(m.status.Unstaged) > 0 {
			fmt.Fprintf(&b, "%s Changes not staged for commit:\n", c)
			for _, entry := range m.status.Unstaged {
				modeText := git.ModeText[string(entry.Unstaged)]
				if modeText == "" {
					modeText = "modified"
				}
				fmt.Fprintf(&b, "%s    %s:   %s\n", c, modeText, entry.Path())
			}
			fmt.Fprintf(&b, "%s\n", c)
		}

		// Untracked files
		if len(m.status.Untracked) > 0 {
			fmt.Fprintf(&b, "%s Untracked files:\n", c)
			for _, entry := range m.status.Untracked {
				fmt.Fprintf(&b, "%s    %s\n", c, entry.Path())
			}
			fmt.Fprintf(&b, "%s\n", c)
		}
	}

	// Scissors and diff (if enabled)
	if m.showDiff && len(m.diff) > 0 {
		fmt.Fprintf(&b, "%s ------------------------ >8 ------------------------\n", c)
		fmt.Fprintf(&b, "%s Do not modify or remove the line above.\n", c)
		fmt.Fprintf(&b, "%s Everything below it will be ignored.\n", c)

		for _, fd := range m.diff {
			b.WriteString(m.formatFileDiff(fd))
		}
	}

	return b.String()
}

// buildCommentContent generates just the comment/template portion of the buffer.
// Used to preserve comments when replacing the user-editable message content.
func (m Model) buildCommentContent() string {
	c := m.commentChar
	var b strings.Builder

	// Git header comment
	fmt.Fprintf(&b, "%s Please enter the commit message for your changes. Lines starting\n", c)
	fmt.Fprintf(&b, "%s with '%s' will be ignored, and an empty message aborts the commit.\n", c, c)
	fmt.Fprintf(&b, "%s\n", c)

	// Commands section
	fmt.Fprintf(&b, "%s Commands:\n", c)
	fmt.Fprintf(&b, "%s   %-16s Close\n", c, m.keys.Close.Help().Key)
	fmt.Fprintf(&b, "%s   %-16s Submit\n", c, "<c-c><c-c>")
	fmt.Fprintf(&b, "%s   %-16s Abort\n", c, "<c-c><c-k>")
	fmt.Fprintf(&b, "%s   %-16s Previous Message\n", c, m.keys.PrevMessage.Help().Key)
	fmt.Fprintf(&b, "%s   %-16s Next Message\n", c, m.keys.NextMessage.Help().Key)
	fmt.Fprintf(&b, "%s   %-16s Reset Message\n", c, m.keys.ResetMessage.Help().Key)
	if m.cfg.CommitEditor.GenerateCommitMessageCommand != "" {
		fmt.Fprintf(&b, "%s   %-16s Generate Message\n", c, m.keys.GenerateMessage.Help().Key)
	}
	fmt.Fprintf(&b, "%s\n", c)

	// Branch info
	if m.branch != "" {
		fmt.Fprintf(&b, "%s On branch %s\n", c, m.branch)
	}

	// Status sections
	if m.status != nil {
		if len(m.status.Staged) > 0 {
			fmt.Fprintf(&b, "%s Changes to be committed:\n", c)
			for _, entry := range m.status.Staged {
				modeText := git.ModeText[string(entry.Staged)]
				if modeText == "" {
					modeText = "modified"
				}
				fmt.Fprintf(&b, "%s    %s:   %s\n", c, modeText, entry.Path())
			}
			fmt.Fprintf(&b, "%s\n", c)
		}

		if len(m.status.Unstaged) > 0 {
			fmt.Fprintf(&b, "%s Changes not staged for commit:\n", c)
			for _, entry := range m.status.Unstaged {
				modeText := git.ModeText[string(entry.Unstaged)]
				if modeText == "" {
					modeText = "modified"
				}
				fmt.Fprintf(&b, "%s    %s:   %s\n", c, modeText, entry.Path())
			}
			fmt.Fprintf(&b, "%s\n", c)
		}

		if len(m.status.Untracked) > 0 {
			fmt.Fprintf(&b, "%s Untracked files:\n", c)
			for _, entry := range m.status.Untracked {
				fmt.Fprintf(&b, "%s    %s\n", c, entry.Path())
			}
			fmt.Fprintf(&b, "%s\n", c)
		}
	}

	// Scissors and diff
	if m.showDiff && len(m.diff) > 0 {
		fmt.Fprintf(&b, "%s ------------------------ >8 ------------------------\n", c)
		fmt.Fprintf(&b, "%s Do not modify or remove the line above.\n", c)
		fmt.Fprintf(&b, "%s Everything below it will be ignored.\n", c)

		for _, fd := range m.diff {
			b.WriteString(m.formatFileDiff(fd))
		}
	}

	return b.String()
}

// formatFileDiff formats a FileDiff for display in the template.
func (m Model) formatFileDiff(fd git.FileDiff) string {
	var b strings.Builder

	// Diff header
	if fd.OldPath != "" && fd.OldPath != fd.Path {
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", fd.OldPath, fd.Path)
	} else {
		fmt.Fprintf(&b, "diff --git a/%s b/%s\n", fd.Path, fd.Path)
	}

	if fd.IsNew {
		b.WriteString("new file mode 100644\n")
	} else if fd.IsDelete {
		b.WriteString("deleted file mode 100644\n")
	}

	// File headers
	if fd.OldPath != "" && fd.OldPath != fd.Path {
		fmt.Fprintf(&b, "--- a/%s\n", fd.OldPath)
	} else if fd.IsNew {
		b.WriteString("--- /dev/null\n")
	} else {
		fmt.Fprintf(&b, "--- a/%s\n", fd.Path)
	}

	if fd.IsDelete {
		b.WriteString("+++ /dev/null\n")
	} else {
		fmt.Fprintf(&b, "+++ b/%s\n", fd.Path)
	}

	// Hunks
	for _, hunk := range fd.Hunks {
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)
		for _, line := range hunk.Lines {
			// Format line with operation prefix (like git diff output)
			switch line.Op {
			case git.DiffOpAdd:
				b.WriteString("+")
			case git.DiffOpDelete:
				b.WriteString("-")
			case git.DiffOpContext:
				b.WriteString(" ")
			}
			b.WriteString(line.Content)
			if !strings.HasSuffix(line.Content, "\n") {
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

// generateCommitMessageCmd runs an external command to generate a commit message.
func generateCommitMessageCmd(command string, repoPath string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = repoPath

		out, err := cmd.Output()
		if err != nil {
			return generateCommitMessageMsg{Err: fmt.Errorf("generate command: %w", err)}
		}

		return generateCommitMessageMsg{Message: string(out)}
	}
}

// OpenCommitEditorCmd returns a command to open the commit editor.
func OpenCommitEditorCmd(opts git.CommitOpts, action string) tea.Cmd {
	return func() tea.Msg {
		return OpenCommitEditorMsg{Opts: opts, Action: action}
	}
}
