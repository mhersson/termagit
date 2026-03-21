package rebaseeditor

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/conjit/internal/git"
)

// Update handles messages for the rebase editor.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case todoLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.entries = msg.Entries
		return m, nil

	case rebaseSubmitResultMsg:
		m.done = true
		return m, func() tea.Msg {
			return RebaseEditorDoneMsg(msg)
		}

	case rebaseAbortResultMsg:
		m.done = true
		m.aborted = true
		return m, func() tea.Msg {
			return RebaseEditorAbortMsg{}
		}

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg handles keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle pending two-key sequences first
	if m.pendingKey == "ctrl+c" {
		return m.handlePendingCtrlC(msg)
	}
	if m.pendingG {
		return m.handlePendingG(msg)
	}
	if m.pendingZ {
		return m.handlePendingZ(msg)
	}
	if m.pendingOSU {
		return m.handlePendingOpenScrollUp(msg)
	}
	if m.pendingOSD {
		return m.handlePendingOpenScrollDown(msg)
	}

	// Don't handle keys while loading or if no entries
	if m.loading || m.done {
		return m, nil
	}

	switch {
	case msg.Type == tea.KeyCtrlC:
		m.pendingKey = "ctrl+c"
		return m, nil

	case key.Matches(msg, m.keys.MoveUp):
		return m.moveUp(), nil

	case key.Matches(msg, m.keys.MoveDown):
		return m.moveDown(), nil

	case msg.String() == "g":
		m.pendingG = true
		return m, nil

	case msg.String() == "Z":
		m.pendingZ = true
		return m, nil

	case msg.String() == "[":
		m.pendingOSU = true
		return m, nil

	case msg.String() == "]":
		m.pendingOSD = true
		return m, nil

	case msg.String() == "j" || msg.Type == tea.KeyDown:
		m.cursorDown()
		return m, nil

	case msg.String() == "k" || msg.Type == tea.KeyUp:
		m.cursorUp()
		return m, nil
	}

	// Action keys only apply to commit entries
	if len(m.entries) == 0 {
		return m, nil
	}

	entry := &m.entries[m.cursor]

	switch {
	case key.Matches(msg, m.keys.Pick):
		if isCommitEntry(entry) {
			entry.Action = git.TodoPick
		}
		return m, nil

	case key.Matches(msg, m.keys.Reword):
		if isCommitEntry(entry) {
			entry.Action = git.TodoReword
		}
		return m, nil

	case key.Matches(msg, m.keys.Edit):
		if isCommitEntry(entry) {
			entry.Action = git.TodoEdit
		}
		return m, nil

	case key.Matches(msg, m.keys.Squash):
		if isCommitEntry(entry) {
			entry.Action = git.TodoSquash
		}
		return m, nil

	case key.Matches(msg, m.keys.Fixup):
		if isCommitEntry(entry) {
			entry.Action = git.TodoFixup
		}
		return m, nil

	case key.Matches(msg, m.keys.Execute):
		return m.insertExec(), nil

	case key.Matches(msg, m.keys.Drop):
		if isCommitEntry(entry) {
			entry.Action = git.TodoDrop
		}
		return m, nil

	case key.Matches(msg, m.keys.Break):
		return m.insertBreak(), nil

	case key.Matches(msg, m.keys.Close):
		m.done = true
		m.aborted = true
		return m, func() tea.Msg {
			return RebaseEditorAbortMsg{}
		}
	}

	return m, nil
}

// handlePendingCtrlC handles the second key after ctrl+c.
func (m Model) handlePendingCtrlC(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.pendingKey = ""
	switch {
	case msg.Type == tea.KeyCtrlC:
		// <c-c><c-c> = Submit
		m.done = true
		return m, submitRebaseCmd(m.repo, m.entries, m.base, m.rebaseOpts)
	case msg.String() == "k":
		// <c-c><c-k> = Abort
		m.done = true
		m.aborted = true
		return m, abortRebaseCmd(m.repo)
	}
	// Any other key cancels the pending sequence
	return m, nil
}

// handlePendingG handles the second key after 'g'.
func (m Model) handlePendingG(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.pendingG = false
	switch msg.String() {
	case "k":
		return m.moveUp(), nil
	case "j":
		return m.moveDown(), nil
	}
	return m, nil
}

// handlePendingZ handles the second key after 'Z'.
func (m Model) handlePendingZ(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.pendingZ = false
	switch msg.String() {
	case "Z":
		// ZZ = Submit
		m.done = true
		return m, submitRebaseCmd(m.repo, m.entries, m.base, m.rebaseOpts)
	case "Q":
		// ZQ = Abort
		m.done = true
		m.aborted = true
		return m, abortRebaseCmd(m.repo)
	}
	return m, nil
}

// handlePendingOpenScrollUp handles [c sequence.
func (m Model) handlePendingOpenScrollUp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.pendingOSU = false
	if msg.String() == "c" {
		if m.cursor < len(m.entries) && m.entries[m.cursor].Hash != "" {
			hash := m.entries[m.cursor].Hash
			return m, func() tea.Msg { return OpenCommitViewMsg{Hash: hash} }
		}
		return m, nil
	}
	return m, nil
}

// handlePendingOpenScrollDown handles ]c sequence.
func (m Model) handlePendingOpenScrollDown(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.pendingOSD = false
	if msg.String() == "c" {
		if m.cursor < len(m.entries) && m.entries[m.cursor].Hash != "" {
			hash := m.entries[m.cursor].Hash
			return m, func() tea.Msg { return OpenCommitViewMsg{Hash: hash} }
		}
		return m, nil
	}
	return m, nil
}

// cursorDown moves the cursor down one row.
func (m *Model) cursorDown() {
	if m.cursor < len(m.entries)-1 {
		m.cursor++
	}
}

// cursorUp moves the cursor up one row.
func (m *Model) cursorUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// moveUp swaps the current entry with the one above and moves cursor up.
func (m Model) moveUp() Model {
	if m.cursor <= 0 || len(m.entries) < 2 {
		return m
	}
	m.entries[m.cursor], m.entries[m.cursor-1] = m.entries[m.cursor-1], m.entries[m.cursor]
	m.cursor--
	return m
}

// moveDown swaps the current entry with the one below and moves cursor down.
func (m Model) moveDown() Model {
	if m.cursor >= len(m.entries)-1 || len(m.entries) < 2 {
		return m
	}
	m.entries[m.cursor], m.entries[m.cursor+1] = m.entries[m.cursor+1], m.entries[m.cursor]
	m.cursor++
	return m
}

// insertBreak inserts a "break" entry after the cursor position.
func (m Model) insertBreak() Model {
	entry := git.TodoEntry{Action: git.TodoBreak}
	return m.insertAfterCursor(entry)
}

// insertExec inserts an "exec" entry after the cursor position.
// In a full implementation this would prompt for the command; for now inserts a placeholder.
func (m Model) insertExec() Model {
	entry := git.TodoEntry{Action: git.TodoExec, Subject: ""}
	return m.insertAfterCursor(entry)
}

// insertAfterCursor inserts an entry after the current cursor position.
func (m Model) insertAfterCursor(entry git.TodoEntry) Model {
	pos := m.cursor + 1
	if pos > len(m.entries) {
		pos = len(m.entries)
	}
	m.entries = append(m.entries[:pos], append([]git.TodoEntry{entry}, m.entries[pos:]...)...)
	m.cursor = pos
	return m
}

// isCommitEntry returns true if the entry has a commit hash (not break/exec/label/reset).
func isCommitEntry(e *git.TodoEntry) bool {
	switch e.Action {
	case git.TodoBreak, git.TodoExec, git.TodoLabel, git.TodoReset:
		return false
	}
	return true
}
