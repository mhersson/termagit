package theme

func init() {
	Register(&catppuccinMacchiato{})
}

type catppuccinMacchiato struct{}

func (t *catppuccinMacchiato) Name() string { return "catppuccin-macchiato" }

func (t *catppuccinMacchiato) Raw() RawTokens {
	r := FromPalette(Palette{
		Bg:            "#24273a", // base
		Bg1:           "#363a4f", // surface0
		Bg2:           "#494d64", // surface1
		Bg3:           "#5b6078", // surface2
		DiffAddBg:     "#253b34",
		DiffDelBg:     "#3c2535",
		DiffContextBg: "#363a4f",
		Fg:            "#cad3f5", // text
		Fg1:           "#cad3f5", // text
		Fg2:           "#b8c0e0", // subtext1
		Dim:           "#6e738d", // overlay0
		Dim1:          "#8087a2", // overlay1
		Blue:          "#8aadf4",
		Green:         "#a6da95",
		Red:           "#ed8796",
		Yellow:        "#eed49f",
		Purple:        "#c6a0f6", // mauve
		Teal:          "#8bd5ca",
		Cyan:          "#91d7e3", // sky
		Orange:        "#f5a97f", // peach
		Pink:          "#f5bde6",
		Lavender:      "#b7bdf8",
	})

	r.DiffContext = "#a5adcb"     // subtext0
	r.ChangeUntracked = "#939ab7" // overlay2
	r.GraphGray = "#5b6078"       // surface2

	return r
}
