package theme

func init() {
	Register(&tokyoNight{})
}

type tokyoNight struct{}

func (t *tokyoNight) Name() string { return "tokyo-night" }

func (t *tokyoNight) Raw() RawTokens {
	return RawTokens{
		// Text colors
		Normal:  "#a9b1d6",
		Bold:    "#c0caf5",
		Dim:     "#565f89",
		Comment: "#565f89",

		// Git object colors
		Branch:       "#7aa2f7", // blue
		BranchHead:   "#7aa2f7",
		Remote:       "#9ece6a", // green
		Tag:          "#e0af68", // yellow
		Hash:         "#737aa2",
		HashCurrent:  "#7dcfff", // cyan
		CommitAuthor: "#bb9af7", // magenta
		CommitDate:   "#9aa5ce",

		// Section headers
		SectionHeader: "#bb9af7", // magenta

		// Diff colors
		DiffAdd:        "#9ece6a", // green
		DiffAddBg:      "#20303b",
		DiffDelete:     "#f7768e", // red
		DiffDeleteBg:   "#37222c",
		DiffContext:    "#9aa5ce",
		DiffHunkHeader: "#7dcfff", // cyan

		// Change indicators (match Neogit: Modified=blue, Added=green, Deleted=red, Renamed=purple, Copied=cyan)
		ChangeModified:  "#7aa2f7", // blue
		ChangeAdded:     "#9ece6a", // green
		ChangeDeleted:   "#f7768e", // red
		ChangeRenamed:   "#bb9af7", // magenta (purple)
		ChangeCopied:    "#7dcfff", // cyan
		ChangeUntracked: "#737aa2",

		// Status
		Staged:   "#9ece6a", // green
		Unstaged: "#e0af68", // yellow
		Conflict: "#f7768e", // red

		// Popup
		PopupBorder:  "#3b4261",
		PopupTitle:   "#c0caf5",
		PopupKey:     "#7aa2f7", // blue
		PopupKeyBg:   "#7aa2f7", // blue (unused now)
		PopupSwitch:  "#7dcfff",
		PopupOption:  "#e0af68",
		PopupAction:  "#a9b1d6",
		PopupSection: "#bb9af7",

		// Notification
		NotificationInfo:  "#7aa2f7",
		NotificationWarn:  "#ff9e64", // orange
		NotificationError: "#f7768e",

		// Cursor and selection
		Cursor:     "#c0caf5", // text (for use elsewhere)
		CursorBg:   "#292e42", // bg_highlight - subtle highlight
		Selection:  "#c0caf5",
		SelectBg:   "#33467c",
		Background: "#1a1b26",

		// Graph/sequencer colors
		GraphOrange: "#ff9e64", // orange
		GraphGreen:  "#9ece6a", // green
		GraphRed:    "#f7768e", // red
		GraphBlue:   "#7aa2f7", // blue

		// Sequencer section headers
		Merging:   "#bb9af7", // magenta
		Rebasing:  "#7dcfff", // cyan
		Picking:   "#9ece6a", // green
		Reverting: "#f7768e", // red
		Bisecting: "#e0af68", // yellow

		// Misc
		RebaseDone: "#565f89", // dim
		SubtleText: "#565f89", // dim
		Stashes:    "#bb9af7", // magenta
	}
}
