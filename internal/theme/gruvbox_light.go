package theme

func init() {
	Register(NewTheme("gruvbox-light", FromPalette(Palette{
		Bg:            "#fbf1c7", // bg0
		Bg1:           "#ebdbb2", // bg1
		Bg2:           "#d5c4a1", // bg2
		Bg3:           "#bdae93", // bg3
		DiffAddBg:     "#d5e5a3",
		DiffDelBg:     "#f2c6b6",
		DiffContextBg: "#ebdbb2",
		DiffHunkBg:    "#f3e6bc",
		Fg:            "#3c3836", // fg1
		Fg1:           "#3c3836", // fg1
		Fg2:           "#504945", // fg2
		Dim:           "#928374", // gray
		Dim1:          "#7c6f64", // fg4
		Blue:          "#076678",
		Green:         "#79740e",
		Red:           "#9d0006",
		Yellow:        "#b57614",
		Purple:        "#8f3f71",
		Teal:          "#427b58", // aqua
		Cyan:          "#427b58", // aqua (no separate cyan)
		Orange:        "#af3a03",
		Pink:          "#8f3f71", // same as purple
		Lavender:      "#076678", // same as blue
	})))
}
