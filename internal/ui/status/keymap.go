package status

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the status buffer.
// All keys match Neogit's config.lua exactly.
type KeyMap struct {
	// Navigation
	MoveDown        key.Binding
	MoveUp          key.Binding
	Toggle          key.Binding // <tab>, za
	OpenFold        key.Binding // zo
	CloseFold       key.Binding // zc
	Depth1          key.Binding // 1, zC
	Depth2          key.Binding // 2
	Depth3          key.Binding // 3
	Depth4          key.Binding // 4, zO
	NextSection     key.Binding // <c-n>
	PreviousSection key.Binding // <c-p>
	NextHunkHeader  key.Binding // }
	PrevHunkHeader  key.Binding // {
	PeekDown        key.Binding // <c-j>
	PeekUp          key.Binding // <c-k>

	// Scroll navigation
	PageUp       key.Binding // <c-b>
	PageDown     key.Binding // <c-f>
	HalfPageUp   key.Binding // <c-u>
	HalfPageDown key.Binding // <c-d>
	GoToTop      key.Binding // gg (via g prefix)
	GoToBottom   key.Binding // G

	// Actions
	Stage         key.Binding // s
	StageUnstaged key.Binding // S
	StageAll      key.Binding // <c-s>
	Unstage       key.Binding // u
	UnstageStaged key.Binding // U
	Discard       key.Binding // x
	Untrack       key.Binding // K
	Rename        key.Binding // R
	GoToFile      key.Binding // <cr>
	PeekFile      key.Binding // <s-cr>
	VSplitOpen    key.Binding // <c-v>
	SplitOpen     key.Binding // <c-x>
	TabOpen       key.Binding // <c-t>
	OpenTree      key.Binding // o
	GoToParentRepo key.Binding // gp
	YankSelected  key.Binding // Y
	ShowRefs      key.Binding // y
	CommandHistory key.Binding // $
	RefreshBuffer key.Binding // <c-r>
	InitRepo      key.Binding // I
	Close         key.Binding // q
	Command       key.Binding // Q

	// Popup keys (from config.lua popup mappings)
	HelpPopup       key.Binding // ?
	CherryPickPopup key.Binding // A
	DiffPopup       key.Binding // d
	RemotePopup     key.Binding // M
	PushPopup       key.Binding // P
	ResetPopup      key.Binding // X
	StashPopup      key.Binding // Z
	IgnorePopup     key.Binding // i
	TagPopup        key.Binding // t
	BranchPopup     key.Binding // b
	BisectPopup     key.Binding // B
	WorktreePopup   key.Binding // w
	CommitPopup     key.Binding // c
	FetchPopup      key.Binding // f
	LogPopup        key.Binding // l
	MarginPopup     key.Binding // L
	MergePopup      key.Binding // m
	PullPopup       key.Binding // p
	RebasePopup     key.Binding // r
	RevertPopup     key.Binding // v (when not on a diff line)

	// Visual mode
	VisualMode     key.Binding // v (when on a diff line — enter visual selection)
	ExitVisualMode key.Binding // Esc — exit visual selection mode
}

// DefaultKeyMap returns the default key bindings matching Neogit exactly.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		// Navigation
		MoveDown: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "move down"),
		),
		MoveUp: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "move up"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("tab", "za"),
			key.WithHelp("tab", "toggle"),
		),
		OpenFold: key.NewBinding(
			key.WithKeys("zo"),
			key.WithHelp("zo", "open fold"),
		),
		CloseFold: key.NewBinding(
			key.WithKeys("zc"),
			key.WithHelp("zc", "close fold"),
		),
		Depth1: key.NewBinding(
			key.WithKeys("1", "zC"),
			key.WithHelp("1", "depth 1"),
		),
		Depth2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "depth 2"),
		),
		Depth3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "depth 3"),
		),
		Depth4: key.NewBinding(
			key.WithKeys("4", "zO"),
			key.WithHelp("4", "depth 4"),
		),
		NextSection: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("C-n", "next section"),
		),
		PreviousSection: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("C-p", "prev section"),
		),
		NextHunkHeader: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next hunk"),
		),
		PrevHunkHeader: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "prev hunk"),
		),
		PeekDown: key.NewBinding(
			key.WithKeys("ctrl+j"),
			key.WithHelp("C-j", "peek down"),
		),
		PeekUp: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("C-k", "peek up"),
		),

		// Scroll navigation
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("C-b", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f"),
			key.WithHelp("C-f", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("C-u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("C-d", "half page down"),
		),
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GoToBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),

		// Actions
		Stage: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stage"),
		),
		StageUnstaged: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "stage unstaged"),
		),
		StageAll: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("C-s", "stage all"),
		),
		Unstage: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unstage"),
		),
		UnstageStaged: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "unstage staged"),
		),
		Discard: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "discard"),
		),
		Untrack: key.NewBinding(
			key.WithKeys("K"),
			key.WithHelp("K", "untrack"),
		),
		Rename: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "rename"),
		),
		GoToFile: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("cr", "go to file"),
		),
		PeekFile: key.NewBinding(
			key.WithKeys("shift+enter"),
			key.WithHelp("S-cr", "peek file"),
		),
		VSplitOpen: key.NewBinding(
			key.WithKeys("ctrl+v"),
			key.WithHelp("C-v", "vsplit"),
		),
		SplitOpen: key.NewBinding(
			key.WithKeys("ctrl+x"),
			key.WithHelp("C-x", "split"),
		),
		TabOpen: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("C-t", "tab"),
		),
		OpenTree: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open tree"),
		),
		GoToParentRepo: key.NewBinding(
			key.WithKeys("g", "p"),
			key.WithHelp("gp", "parent repo"),
		),
		YankSelected: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank"),
		),
		ShowRefs: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "show refs"),
		),
		CommandHistory: key.NewBinding(
			key.WithKeys("$"),
			key.WithHelp("$", "cmd history"),
		),
		RefreshBuffer: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("C-r", "refresh"),
		),
		InitRepo: key.NewBinding(
			key.WithKeys("I"),
			key.WithHelp("I", "init repo"),
		),
		Close: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "close"),
		),
		Command: key.NewBinding(
			key.WithKeys("Q"),
			key.WithHelp("Q", "command"),
		),

		// Popup keys
		HelpPopup: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		CherryPickPopup: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "cherry-pick"),
		),
		DiffPopup: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "diff"),
		),
		RemotePopup: key.NewBinding(
			key.WithKeys("M"),
			key.WithHelp("M", "remote"),
		),
		PushPopup: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "push"),
		),
		ResetPopup: key.NewBinding(
			key.WithKeys("X"),
			key.WithHelp("X", "reset"),
		),
		StashPopup: key.NewBinding(
			key.WithKeys("Z"),
			key.WithHelp("Z", "stash"),
		),
		IgnorePopup: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "ignore"),
		),
		TagPopup: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "tag"),
		),
		BranchPopup: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "branch"),
		),
		BisectPopup: key.NewBinding(
			key.WithKeys("B"),
			key.WithHelp("B", "bisect"),
		),
		WorktreePopup: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "worktree"),
		),
		CommitPopup: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "commit"),
		),
		FetchPopup: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "fetch"),
		),
		LogPopup: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "log"),
		),
		MarginPopup: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "margin"),
		),
		MergePopup: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "merge"),
		),
		PullPopup: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pull"),
		),
		RebasePopup: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rebase"),
		),
		RevertPopup: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "revert"),
		),

		// Visual mode
		VisualMode: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "visual select"),
		),
		ExitVisualMode: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit visual mode"),
		),
	}
}
