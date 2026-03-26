package theme

func init() {
	Register(NewTheme("solarized-dark", FromPalette(Palette{
		Bg:            "#002b36", // base03
		Bg1:           "#073642", // base02
		Bg2:           "#586e75", // base01
		Bg3:           "#657b83", // base00
		DiffAddBg:     "#0a3a2a",
		DiffDelBg:     "#372020",
		DiffContextBg: "#073642",
		DiffHunkBg:    "#03303c",
		Fg:            "#93a1a1", // base1
		Fg1:           "#839496", // base0
		Fg2:           "#657b83", // base00
		Dim:           "#586e75", // base01
		Dim1:          "#657b83", // base00
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
	})))
}
