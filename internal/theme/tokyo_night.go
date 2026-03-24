package theme

func init() {
	Register(&tokyoNight{})
}

type tokyoNight struct{}

func (t *tokyoNight) Name() string { return "tokyo-night" }

func (t *tokyoNight) Raw() RawTokens {
	return FromPalette(Palette{
		Bg:        "#1a1b26",
		Bg1:       "#292e42", // bg_highlight
		Bg2:       "#33467c",
		Bg3:       "#3b4261",
		DiffAddBg:     "#20303b",
		DiffDelBg:     "#37222c",
		DiffContextBg: "#292e42",
		Fg:        "#c0caf5",
		Fg1:       "#a9b1d6",
		Fg2:       "#9aa5ce",
		Dim:       "#565f89", // comment
		Dim1:      "#737aa2",
		Blue:      "#7aa2f7",
		Green:     "#9ece6a",
		Red:       "#f7768e",
		Yellow:    "#e0af68",
		Purple:    "#bb9af7", // magenta
		Teal:      "#7dcfff", // cyan (no separate teal)
		Cyan:      "#7dcfff",
		Orange:    "#ff9e64",
		Pink:      "#bb9af7", // same as purple
		Lavender:  "#7dcfff", // same as cyan
	})
}
