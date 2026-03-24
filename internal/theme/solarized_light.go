package theme

func init() {
	Register(&solarizedLight{})
}

type solarizedLight struct{}

func (t *solarizedLight) Name() string { return "solarized-light" }

func (t *solarizedLight) Raw() RawTokens {
	return FromPalette(Palette{
		Bg:            "#fdf6e3", // base3
		Bg1:           "#eee8d5", // base2
		Bg2:           "#93a1a1", // base1
		Bg3:           "#839496", // base0
		DiffAddBg:     "#d9ead3",
		DiffDelBg:     "#f5d6d0",
		DiffContextBg: "#eee8d5",
		Fg:            "#586e75", // base01
		Fg1:           "#657b83", // base00
		Fg2:           "#839496", // base0
		Dim:           "#93a1a1", // base1
		Dim1:          "#839496", // base0
		Blue:          "#268bd2",
		Green:         "#859900",
		Red:           "#dc322f",
		Yellow:        "#b58900",
		Purple:        "#6c71c4", // violet
		Teal:          "#2aa198", // cyan
		Cyan:          "#2aa198",
		Orange:        "#cb4b16",
		Pink:          "#d33682", // magenta
		Lavender:      "#6c71c4", // violet
	})
}
