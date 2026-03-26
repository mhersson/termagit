package theme

func init() {
	r := FromPalette(Palette{
		Bg:            "#2d353b",
		Bg1:           "#374145", // bg_visual
		Bg2:           "#475258",
		Bg3:           "#4f585e",
		DiffAddBg:     "#2d4a3e",
		DiffDelBg:     "#4c3743",
		DiffContextBg: "#374145",
		DiffHunkBg:    "#323b40",
		Fg:            "#d3c6aa",
		Fg1:           "#d3c6aa",
		Fg2:           "#9da9a0", // grey2
		Dim:           "#7a8478", // grey0
		Dim1:          "#859289", // grey1
		Blue:          "#7fbbb3",
		Green:         "#a7c080",
		Red:           "#e67e80",
		Yellow:        "#dbbc7f",
		Purple:        "#d699b6",
		Teal:          "#83c092", // aqua
		Cyan:          "#83c092", // aqua (no separate cyan)
		Orange:        "#e69875",
		Pink:          "#d699b6", // same as purple
		Lavender:      "#83c092", // same as teal/aqua
	})

	// Everforest uses blue where the standard mapping uses cyan
	r.PopupSwitch = "#7fbbb3" // blue
	r.GraphCyan = "#7fbbb3"   // blue

	Register(NewTheme("everforest-dark", r))
}
