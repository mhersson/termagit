package theme

// Palette defines a color scheme using a small set of named colors.
// FromPalette maps these to all RawTokens fields.
type Palette struct {
	Bg        string `toml:"bg"`         // base background
	Bg1       string `toml:"bg1"`        // surface 0 — cursor bg, float header bg
	Bg2       string `toml:"bg2"`        // surface 1 — select bg, diff header bg
	Bg3       string `toml:"bg3"`        // surface 2 — popup border
	DiffAddBg string `toml:"diff_add_bg"`  // diff added line background
	DiffDelBg string `toml:"diff_del_bg"`  // diff deleted line background
	Fg        string `toml:"fg"`         // bright foreground — bold text, popup title, cursor
	Fg1       string `toml:"fg1"`        // normal foreground — body text, popup actions
	Fg2       string `toml:"fg2"`        // secondary foreground — commit date, diff context
	Dim       string `toml:"dim"`        // dimmed — comments, subtle text, graph gray
	Dim1      string `toml:"dim1"`       // slightly brighter dim — hash, untracked label
	Blue      string `toml:"blue"`       // branches, modified, popup key, info
	Green     string `toml:"green"`      // remote, staged, added, diff add, success
	Red       string `toml:"red"`        // deleted, conflict, diff delete, error
	Yellow    string `toml:"yellow"`     // tag, unstaged, option, confirm key
	Purple    string `toml:"purple"`     // section header, renamed, popup section, stashes
	Teal      string `toml:"teal"`       // hunk header, copied, rebasing
	Cyan      string `toml:"cyan"`       // popup switch, commit view header, float header fg
	Orange    string `toml:"orange"`     // warn, confirm border, number
	Pink      string `toml:"pink"`       // commit author, merging
	Lavender  string `toml:"lavender"`   // hash current
}

// FromPalette generates a complete RawTokens from a Palette.
func FromPalette(p Palette) RawTokens {
	return RawTokens{
		// Text
		Normal:  p.Fg1,
		Bold:    p.Fg,
		Dim:     p.Dim,
		Comment: p.Dim,

		// Git objects
		Branch:       p.Blue,
		BranchHead:   p.Blue,
		Remote:       p.Green,
		Tag:          p.Yellow,
		Hash:         p.Dim1,
		HashCurrent:  p.Lavender,
		CommitAuthor: p.Pink,
		CommitDate:   p.Fg2,

		// Section headers
		SectionHeader: p.Purple,

		// Diff
		DiffAdd:        p.Green,
		DiffAddBg:      p.DiffAddBg,
		DiffDelete:     p.Red,
		DiffDeleteBg:   p.DiffDelBg,
		DiffContext:    p.Fg2,
		DiffHunkHeader: p.Teal,

		// Change indicators
		ChangeModified:  p.Blue,
		ChangeAdded:     p.Green,
		ChangeDeleted:   p.Red,
		ChangeRenamed:   p.Purple,
		ChangeCopied:    p.Teal,
		ChangeUntracked: p.Dim1,

		// Status
		Staged:   p.Green,
		Unstaged: p.Yellow,
		Conflict: p.Red,

		// Popup
		PopupBorder:  p.Bg3,
		PopupTitle:   p.Fg,
		PopupKey:     p.Blue,
		PopupKeyBg:   p.Blue,
		PopupSwitch:  p.Cyan,
		PopupOption:  p.Yellow,
		PopupAction:  p.Fg1,
		PopupSection: p.Purple,

		// Notification
		NotificationInfo:    p.Blue,
		NotificationSuccess: p.Green,
		NotificationWarn:    p.Orange,
		NotificationError:   p.Red,

		// Confirmation
		ConfirmBorder: p.Orange,
		ConfirmText:   p.Fg1,
		ConfirmKey:    p.Yellow,

		// Cursor and selection
		Cursor:     p.Fg,
		CursorBg:   p.Bg1,
		Selection:  p.Fg,
		SelectBg:   p.Bg2,
		Background: p.Bg,

		// Graph
		GraphOrange: p.Orange,
		GraphGreen:  p.Green,
		GraphRed:    p.Red,
		GraphBlue:   p.Blue,
		GraphYellow: p.Yellow,
		GraphCyan:   p.Cyan,
		GraphPurple: p.Purple,
		GraphGray:   p.Dim,
		GraphWhite:  p.Fg,

		// Sequencer headers
		Merging:   p.Pink,
		Rebasing:  p.Teal,
		Picking:   p.Green,
		Reverting: p.Red,
		Bisecting: p.Yellow,

		// Misc
		RebaseDone: p.Dim,
		SubtleText: p.Dim,
		Stashes:    p.Purple,

		// Commit view
		CommitViewHeader:   p.Cyan,
		CommitViewHeaderFg: p.Bg,
		FilePath:           p.Blue,
		Number:             p.Orange,

		// Diff view
		DiffHeader:    p.Bg2,
		DiffHeaderFg:  p.Blue,
		FloatHeader:   p.Bg1,
		FloatHeaderFg: p.Cyan,
	}
}

// hasPalette returns true if the palette has enough fields to generate tokens.
func (p Palette) hasPalette() bool {
	return p.Bg != "" && p.Fg != "" && p.Blue != "" && p.Green != "" && p.Red != ""
}
