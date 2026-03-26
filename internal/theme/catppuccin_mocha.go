package theme

func init() {
	r := FromPalette(Palette{
		Bg:            "#1e1e2e", // base
		Bg1:           "#313244", // surface0
		Bg2:           "#45475a", // surface1
		Bg3:           "#585b70", // surface2
		DiffAddBg:     "#1e3a2f",
		DiffDelBg:     "#3b1f29",
		DiffContextBg: "#313244",
		Fg:            "#cdd6f4", // text
		Fg1:           "#cdd6f4", // text
		Fg2:           "#bac2de", // subtext1
		Dim:           "#6c7086", // overlay0
		Dim1:          "#7f849c", // overlay1
		Blue:          "#89b4fa",
		Green:         "#a6e3a1",
		Red:           "#f38ba8",
		Yellow:        "#f9e2af",
		Purple:        "#cba6f7", // mauve
		Teal:          "#94e2d5",
		Cyan:          "#89dceb", // sky
		Orange:        "#fab387", // peach
		Pink:          "#f5c2e7",
		Lavender:      "#b4befe",
	})

	// Tokens that differ from the standard palette mapping
	r.DiffContext = "#a6adc8"     // subtext0 (not subtext1)
	r.ChangeUntracked = "#9399b2" // overlay2 (not overlay1)
	r.GraphGray = "#585b70"       // surface2 (not overlay0)

	t := NewTheme("catppuccin-mocha", r)
	Register(t)
	setFallback(t)
}
