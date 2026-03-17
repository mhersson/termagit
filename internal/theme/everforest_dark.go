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
		SectionHeader:   "#d3c6aa",
		SectionHeaderBg: "#d699b6", // purple

		// Diff colors
		DiffAdd:        "#a7c080", // green
		DiffAddBg:      "#2d4a3e",
		DiffDelete:     "#e67e80", // red
		DiffDeleteBg:   "#4c3743",
		DiffContext:    "#9da9a0",
		DiffHunkHeader: "#83c092", // aqua

		// Change indicators
		ChangeModified:  "#dbbc7f", // yellow
		ChangeAdded:     "#a7c080", // green
		ChangeDeleted:   "#e67e80", // red
		ChangeRenamed:   "#7fbbb3", // blue
		ChangeCopied:    "#7fbbb3",
		ChangeUntracked: "#859289",

		// Status
		Staged:   "#a7c080", // green
		Unstaged: "#dbbc7f", // yellow
		Conflict: "#e67e80", // red

		// Popup
		PopupBorder:  "#4f585e",
		PopupTitle:   "#d3c6aa",
		PopupKey:     "#2d353b",
		PopupKeyBg:   "#7fbbb3",
		PopupSwitch:  "#7fbbb3",
		PopupOption:  "#dbbc7f",
		PopupAction:  "#d3c6aa",
		PopupSection: "#d699b6",

		// Notification
		NotificationInfo:  "#7fbbb3",
		NotificationWarn:  "#e69875", // orange
		NotificationError: "#e67e80",

		// Cursor and selection
		Cursor:     "#2d353b",
		CursorBg:   "#d3c6aa",
		Selection:  "#d3c6aa",
		SelectBg:   "#475258",
		Background: "#2d353b",

		// Graph/sequencer colors
		GraphOrange: "#e69875", // orange
		GraphGreen:  "#a7c080", // green
		GraphRed:    "#e67e80", // red
		GraphBlue:   "#7fbbb3", // blue

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
	}
}
