package theme

func init() {
	r := FromPalette(Palette{
		Bg:            "#303446", // base
		Bg1:           "#414559", // surface0
		Bg2:           "#51576d", // surface1
		Bg3:           "#626880", // surface2
		DiffAddBg:     "#29393a",
		DiffDelBg:     "#3e2a33",
		DiffContextBg: "#414559",
		Fg:            "#c6d0f5", // text
		Fg1:           "#c6d0f5", // text
		Fg2:           "#b5bfe2", // subtext1
		Dim:           "#737994", // overlay0
		Dim1:          "#838ba7", // overlay1
		Blue:          "#8caaee",
		Green:         "#a6d189",
		Red:           "#e78284",
		Yellow:        "#e5c890",
		Purple:        "#ca9ee6", // mauve
		Teal:          "#81c8be",
		Cyan:          "#99d1db", // sky
		Orange:        "#ef9f76", // peach
		Pink:          "#f4b8e4",
		Lavender:      "#babbf1",
	})

	r.DiffContext = "#a5adce"     // subtext0
	r.ChangeUntracked = "#949cbb" // overlay2
	r.GraphGray = "#626880"       // surface2

	Register(NewTheme("catppuccin-frappe", r))
}
