package theme

func init() {
	r := FromPalette(Palette{
		Bg:            "#eff1f5", // base
		Bg1:           "#ccd0da", // surface0
		Bg2:           "#bcc0cc", // surface1
		Bg3:           "#acb0be", // surface2
		DiffAddBg:     "#d4edda",
		DiffDelBg:     "#f0d4d9",
		DiffContextBg: "#ccd0da",
		Fg:            "#4c4f69", // text
		Fg1:           "#4c4f69", // text
		Fg2:           "#5c5f77", // subtext1
		Dim:           "#7c7f93", // overlay0
		Dim1:          "#8c8fa1", // overlay1
		Blue:          "#1e66f5",
		Green:         "#40a02b",
		Red:           "#d20f39",
		Yellow:        "#df8e1d",
		Purple:        "#8839ef", // mauve
		Teal:          "#179299",
		Cyan:          "#04a5e5", // sky
		Orange:        "#fe640b", // peach
		Pink:          "#ea76cb",
		Lavender:      "#7287fd",
	})

	r.DiffContext = "#6c6f85"     // subtext0
	r.ChangeUntracked = "#9ca0b0" // overlay2
	r.GraphGray = "#acb0be"       // surface2

	Register(NewTheme("catppuccin-latte", r))
}
