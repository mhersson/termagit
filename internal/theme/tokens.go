package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// RawTokens holds hex color strings decoded from TOML.
// Field names map to Neogit highlight groups.
type RawTokens struct {
	// Text colors
	Normal  string `toml:"normal"`
	Bold    string `toml:"bold"`
	Dim     string `toml:"dim"`
	Comment string `toml:"comment"`

	// Git object colors
	Branch       string `toml:"branch"`
	BranchHead   string `toml:"branch_head"`
	Remote       string `toml:"remote"`
	Tag          string `toml:"tag"`
	Hash         string `toml:"hash"`
	HashCurrent  string `toml:"hash_current"`
	CommitAuthor string `toml:"commit_author"`
	CommitDate   string `toml:"commit_date"`

	// Section headers
	SectionHeader   string `toml:"section_header"`
	SectionHeaderBg string `toml:"section_header_bg"`

	// Diff colors
	DiffAdd        string `toml:"diff_add"`
	DiffAddBg      string `toml:"diff_add_bg"`
	DiffDelete     string `toml:"diff_delete"`
	DiffDeleteBg   string `toml:"diff_delete_bg"`
	DiffContext    string `toml:"diff_context"`
	DiffHunkHeader string `toml:"diff_hunk_header"`

	// Change indicators
	ChangeModified  string `toml:"change_modified"`
	ChangeAdded     string `toml:"change_added"`
	ChangeDeleted   string `toml:"change_deleted"`
	ChangeRenamed   string `toml:"change_renamed"`
	ChangeCopied    string `toml:"change_copied"`
	ChangeUntracked string `toml:"change_untracked"`

	// Status
	Staged   string `toml:"staged"`
	Unstaged string `toml:"unstaged"`
	Conflict string `toml:"conflict"`

	// Popup
	PopupBorder  string `toml:"popup_border"`
	PopupTitle   string `toml:"popup_title"`
	PopupKey     string `toml:"popup_key"`
	PopupKeyBg   string `toml:"popup_key_bg"`
	PopupSwitch  string `toml:"popup_switch"`
	PopupOption  string `toml:"popup_option"`
	PopupAction  string `toml:"popup_action"`
	PopupSection string `toml:"popup_section"`

	// Notification
	NotificationInfo  string `toml:"notification_info"`
	NotificationWarn  string `toml:"notification_warn"`
	NotificationError string `toml:"notification_error"`

	// Cursor and selection
	Cursor     string `toml:"cursor"`
	CursorBg   string `toml:"cursor_bg"`
	Selection  string `toml:"selection"`
	SelectBg   string `toml:"select_bg"`
	Background string `toml:"background"`

	// Graph/sequencer colors (from Neogit hl.lua)
	GraphOrange string `toml:"graph_orange"`
	GraphGreen  string `toml:"graph_green"`
	GraphRed    string `toml:"graph_red"`
	GraphBlue   string `toml:"graph_blue"`

	// Sequencer section headers
	Merging   string `toml:"merging"`
	Rebasing  string `toml:"rebasing"`
	Picking   string `toml:"picking"`
	Reverting string `toml:"reverting"`
	Bisecting string `toml:"bisecting"`

	// Misc
	RebaseDone string `toml:"rebase_done"`
	SubtleText string `toml:"subtle_text"`
	Stashes    string `toml:"stashes"`
}

// Tokens holds compiled lipgloss.Style values for rendering.
type Tokens struct {
	// Text styles
	Normal  lipgloss.Style
	Bold    lipgloss.Style
	Dim     lipgloss.Style
	Comment lipgloss.Style

	// Git object styles
	Branch       lipgloss.Style
	BranchHead   lipgloss.Style
	Remote       lipgloss.Style
	Tag          lipgloss.Style
	Hash         lipgloss.Style
	HashCurrent  lipgloss.Style
	CommitAuthor lipgloss.Style
	CommitDate   lipgloss.Style

	// Section headers
	SectionHeader lipgloss.Style

	// Diff styles
	DiffAdd        lipgloss.Style
	DiffDelete     lipgloss.Style
	DiffContext    lipgloss.Style
	DiffHunkHeader lipgloss.Style

	// Change indicator styles
	ChangeModified  lipgloss.Style
	ChangeAdded     lipgloss.Style
	ChangeDeleted   lipgloss.Style
	ChangeRenamed   lipgloss.Style
	ChangeCopied    lipgloss.Style
	ChangeUntracked lipgloss.Style

	// Status styles
	Staged   lipgloss.Style
	Unstaged lipgloss.Style
	Conflict lipgloss.Style

	// Popup styles
	PopupBorder  lipgloss.Style
	PopupTitle   lipgloss.Style
	PopupKey     lipgloss.Style
	PopupSwitch  lipgloss.Style
	PopupOption  lipgloss.Style
	PopupAction  lipgloss.Style
	PopupSection lipgloss.Style

	// Notification styles
	NotificationInfo  lipgloss.Style
	NotificationWarn  lipgloss.Style
	NotificationError lipgloss.Style

	// Cursor and selection styles
	Cursor    lipgloss.Style
	Selection lipgloss.Style

	// Graph/sequencer styles
	GraphOrange lipgloss.Style
	GraphGreen  lipgloss.Style
	GraphRed    lipgloss.Style
	GraphBlue   lipgloss.Style

	// Sequencer section header styles
	Merging   lipgloss.Style
	Rebasing  lipgloss.Style
	Picking   lipgloss.Style
	Reverting lipgloss.Style
	Bisecting lipgloss.Style

	// Misc styles
	RebaseDone lipgloss.Style
	SubtleText lipgloss.Style
	Stashes    lipgloss.Style
}

// Compile converts RawTokens to compiled Tokens.
func Compile(r RawTokens) Tokens {
	return Tokens{
		Normal:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.Normal)),
		Bold:    lipgloss.NewStyle().Foreground(lipgloss.Color(r.Bold)).Bold(true),
		Dim:     lipgloss.NewStyle().Foreground(lipgloss.Color(r.Dim)),
		Comment: lipgloss.NewStyle().Foreground(lipgloss.Color(r.Comment)).Italic(true),

		Branch:       lipgloss.NewStyle().Foreground(lipgloss.Color(r.Branch)).Bold(true),
		BranchHead:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.BranchHead)).Bold(true).Underline(true),
		Remote:       lipgloss.NewStyle().Foreground(lipgloss.Color(r.Remote)).Bold(true),
		Tag:          lipgloss.NewStyle().Foreground(lipgloss.Color(r.Tag)).Bold(true),
		Hash:         lipgloss.NewStyle().Foreground(lipgloss.Color(r.Hash)),
		HashCurrent:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.HashCurrent)).Bold(true),
		CommitAuthor: lipgloss.NewStyle().Foreground(lipgloss.Color(r.CommitAuthor)),
		CommitDate:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.CommitDate)),

		SectionHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color(r.SectionHeader)).
			Background(lipgloss.Color(r.SectionHeaderBg)).
			Bold(true),

		DiffAdd:        lipgloss.NewStyle().Foreground(lipgloss.Color(r.DiffAdd)).Background(lipgloss.Color(r.DiffAddBg)),
		DiffDelete:     lipgloss.NewStyle().Foreground(lipgloss.Color(r.DiffDelete)).Background(lipgloss.Color(r.DiffDeleteBg)),
		DiffContext:    lipgloss.NewStyle().Foreground(lipgloss.Color(r.DiffContext)),
		DiffHunkHeader: lipgloss.NewStyle().Foreground(lipgloss.Color(r.DiffHunkHeader)).Bold(true),

		ChangeModified:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.ChangeModified)).Bold(true),
		ChangeAdded:     lipgloss.NewStyle().Foreground(lipgloss.Color(r.ChangeAdded)).Bold(true),
		ChangeDeleted:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.ChangeDeleted)).Bold(true),
		ChangeRenamed:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.ChangeRenamed)).Bold(true).Italic(true),
		ChangeCopied:    lipgloss.NewStyle().Foreground(lipgloss.Color(r.ChangeCopied)).Bold(true).Italic(true),
		ChangeUntracked: lipgloss.NewStyle().Foreground(lipgloss.Color(r.ChangeUntracked)).Bold(true),

		Staged:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.Staged)),
		Unstaged: lipgloss.NewStyle().Foreground(lipgloss.Color(r.Unstaged)),
		Conflict: lipgloss.NewStyle().Foreground(lipgloss.Color(r.Conflict)).Bold(true),

		PopupBorder:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupBorder)),
		PopupTitle:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupTitle)).Bold(true),
		PopupKey:     lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupKey)).Background(lipgloss.Color(r.PopupKeyBg)).Bold(true),
		PopupSwitch:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupSwitch)),
		PopupOption:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupOption)),
		PopupAction:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupAction)),
		PopupSection: lipgloss.NewStyle().Foreground(lipgloss.Color(r.PopupSection)).Bold(true),

		NotificationInfo:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.NotificationInfo)),
		NotificationWarn:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.NotificationWarn)),
		NotificationError: lipgloss.NewStyle().Foreground(lipgloss.Color(r.NotificationError)).Bold(true),

		Cursor:    lipgloss.NewStyle().Foreground(lipgloss.Color(r.Cursor)).Background(lipgloss.Color(r.CursorBg)),
		Selection: lipgloss.NewStyle().Foreground(lipgloss.Color(r.Selection)).Background(lipgloss.Color(r.SelectBg)),

		GraphOrange: lipgloss.NewStyle().Foreground(lipgloss.Color(r.GraphOrange)),
		GraphGreen:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.GraphGreen)),
		GraphRed:    lipgloss.NewStyle().Foreground(lipgloss.Color(r.GraphRed)),
		GraphBlue:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.GraphBlue)),

		Merging:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.Merging)).Bold(true),
		Rebasing:  lipgloss.NewStyle().Foreground(lipgloss.Color(r.Rebasing)).Bold(true),
		Picking:   lipgloss.NewStyle().Foreground(lipgloss.Color(r.Picking)).Bold(true),
		Reverting: lipgloss.NewStyle().Foreground(lipgloss.Color(r.Reverting)).Bold(true),
		Bisecting: lipgloss.NewStyle().Foreground(lipgloss.Color(r.Bisecting)).Bold(true),

		RebaseDone: lipgloss.NewStyle().Foreground(lipgloss.Color(r.RebaseDone)),
		SubtleText: lipgloss.NewStyle().Foreground(lipgloss.Color(r.SubtleText)),
		Stashes:    lipgloss.NewStyle().Foreground(lipgloss.Color(r.Stashes)).Bold(true),
	}
}
