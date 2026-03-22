package theme

func init() {
	Register(&everforestDark{})
}

type everforestDark struct{}

func (t *everforestDark) Name() string { return "everforest-dark" }

func (t *everforestDark) Raw() RawTokens {
	return RawTokens{
		// Text colors
		Normal:  "#d3c6aa",
		Bold:    "#d3c6aa",
		Dim:     "#7a8478",
		Comment: "#7a8478",

		// Git object colors
		Branch:       "#7fbbb3", // blue
		BranchHead:   "#7fbbb3",
		Remote:       "#a7c080", // green
		Tag:          "#dbbc7f", // yellow
		Hash:         "#859289", // grey1
		HashCurrent:  "#83c092", // aqua
		CommitAuthor: "#d699b6", // purple
		CommitDate:   "#9da9a0", // grey2

		// Section headers
		SectionHeader: "#d699b6", // purple

		// Diff colors
		DiffAdd:        "#a7c080", // green
		DiffAddBg:      "#2d4a3e",
		DiffDelete:     "#e67e80", // red
		DiffDeleteBg:   "#4c3743",
		DiffContext:    "#9da9a0",
		DiffHunkHeader: "#83c092", // aqua

		// Change indicators (match Neogit: Modified=blue, Added=green, Deleted=red, Renamed=purple, Copied=cyan)
		ChangeModified:  "#7fbbb3", // blue
		ChangeAdded:     "#a7c080", // green
		ChangeDeleted:   "#e67e80", // red
		ChangeRenamed:   "#d699b6", // purple
		ChangeCopied:    "#83c092", // aqua (cyan)
		ChangeUntracked: "#859289",

		// Status
		Staged:   "#a7c080", // green
		Unstaged: "#dbbc7f", // yellow
		Conflict: "#e67e80", // red

		// Popup
		PopupBorder:  "#4f585e",
		PopupTitle:   "#d3c6aa",
		PopupKey:     "#7fbbb3", // blue
		PopupKeyBg:   "#7fbbb3", // blue (unused now)
		PopupSwitch:  "#7fbbb3",
		PopupOption:  "#dbbc7f",
		PopupAction:  "#d3c6aa",
		PopupSection: "#d699b6",

		// Notification
		NotificationInfo:    "#7fbbb3",
		NotificationSuccess: "#a7c080", // green
		NotificationWarn:    "#e69875", // orange
		NotificationError:   "#e67e80",

		// Confirmation dialog
		ConfirmBorder: "#e69875", // orange — warm "action required" color
		ConfirmText:   "#d3c6aa", // normal fg
		ConfirmKey:    "#dbbc7f", // yellow — highlighted keys

		// Cursor and selection
		Cursor:     "#d3c6aa", // text (for use elsewhere)
		CursorBg:   "#374145", // bg_visual - subtle highlight
		Selection:  "#d3c6aa",
		SelectBg:   "#475258",
		Background: "#2d353b",

		// Graph/sequencer colors
		GraphOrange: "#e69875", // orange
		GraphGreen:  "#a7c080", // green
		GraphRed:    "#e67e80", // red
		GraphBlue:   "#7fbbb3", // blue
		GraphYellow: "#dbbc7f", // yellow
		GraphCyan:   "#7fbbb3", // aqua
		GraphPurple: "#d699b6", // purple
		GraphGray:   "#7a8478", // grey0
		GraphWhite:  "#d3c6aa", // fg

		// Sequencer section headers
		Merging:   "#d699b6", // purple
		Rebasing:  "#83c092", // aqua
		Picking:   "#a7c080", // green
		Reverting: "#e67e80", // red
		Bisecting: "#dbbc7f", // yellow

		// Misc
		RebaseDone: "#7a8478", // grey
		SubtleText: "#7a8478", // grey
		Stashes:    "#d699b6", // purple

		// Commit view
		CommitViewHeader:   "#83c092", // aqua (cyan background like Neogit)
		CommitViewHeaderFg: "#2d353b", // bg (dark text on cyan)
		FilePath:           "#7fbbb3", // blue (italic for paths)
		Number:             "#e69875", // orange (numbers)

		// Diff view
		DiffHeader:   "#475258", // bg3
		DiffHeaderFg: "#7fbbb3", // blue
		FloatHeader:  "#374145", // bg2
		FloatHeaderFg: "#83c092", // aqua (cyan)
	}
}
