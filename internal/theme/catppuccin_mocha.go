package theme

func init() {
	t := &catppuccinMocha{}
	Register(t)
	setFallback(t)
}

type catppuccinMocha struct{}

func (t *catppuccinMocha) Name() string { return "catppuccin-mocha" }

func (t *catppuccinMocha) Raw() RawTokens {
	return RawTokens{
		// Text colors
		Normal:  "#cdd6f4", // text
		Bold:    "#cdd6f4",
		Dim:     "#6c7086", // overlay0
		Comment: "#6c7086",

		// Git object colors
		Branch:       "#89b4fa", // blue
		BranchHead:   "#89b4fa",
		Remote:       "#a6e3a1", // green
		Tag:          "#f9e2af", // yellow
		Hash:         "#7f849c", // overlay1
		HashCurrent:  "#b4befe", // lavender
		CommitAuthor: "#f5c2e7", // pink
		CommitDate:   "#bac2de", // subtext1

		// Section headers
		SectionHeader: "#cba6f7", // mauve

		// Diff colors
		DiffAdd:        "#a6e3a1", // green
		DiffAddBg:      "#1e3a2f",
		DiffDelete:     "#f38ba8", // red
		DiffDeleteBg:   "#3b1f29",
		DiffContext:    "#a6adc8", // subtext0
		DiffHunkHeader: "#94e2d5", // teal

		// Change indicators (match Neogit: Modified=blue, Added=green, Deleted=red, Renamed=purple, Copied=cyan)
		ChangeModified:  "#89b4fa", // blue
		ChangeAdded:     "#a6e3a1", // green
		ChangeDeleted:   "#f38ba8", // red
		ChangeRenamed:   "#cba6f7", // mauve (purple)
		ChangeCopied:    "#94e2d5", // teal (cyan)
		ChangeUntracked: "#9399b2", // overlay2

		// Status
		Staged:   "#a6e3a1", // green
		Unstaged: "#f9e2af", // yellow
		Conflict: "#f38ba8", // red

		// Popup
		PopupBorder:  "#585b70", // surface2
		PopupTitle:   "#cdd6f4",
		PopupKey:     "#89b4fa", // blue
		PopupKeyBg:   "#89b4fa", // blue (unused now)
		PopupSwitch:  "#89dceb", // sky
		PopupOption:  "#f9e2af", // yellow
		PopupAction:  "#cdd6f4",
		PopupSection: "#cba6f7", // mauve

		// Notification
		NotificationInfo:  "#89b4fa", // blue
		NotificationWarn:  "#fab387", // peach
		NotificationError: "#f38ba8", // red

		// Cursor and selection
		Cursor:     "#cdd6f4", // text (for use elsewhere)
		CursorBg:   "#313244", // surface0 - subtle highlight
		Selection:  "#cdd6f4",
		SelectBg:   "#45475a", // surface0
		Background: "#1e1e2e", // base

		// Graph/sequencer colors
		GraphOrange: "#fab387", // peach
		GraphGreen:  "#a6e3a1", // green
		GraphRed:    "#f38ba8", // red
		GraphBlue:   "#89b4fa", // blue

		// Sequencer section headers
		Merging:   "#f5c2e7", // pink
		Rebasing:  "#94e2d5", // teal
		Picking:   "#a6e3a1", // green
		Reverting: "#f38ba8", // red
		Bisecting: "#f9e2af", // yellow

		// Misc
		RebaseDone: "#6c7086", // overlay0
		SubtleText: "#6c7086", // overlay0
		Stashes:    "#cba6f7", // mauve
	}
}
