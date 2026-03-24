package theme

func init() {
	Register(&tokyoNightLight{})
}

type tokyoNightLight struct{}

func (t *tokyoNightLight) Name() string { return "tokyo-night-light" }

func (t *tokyoNightLight) Raw() RawTokens {
	return FromPalette(Palette{
		Bg:            "#e1e2e7",
		Bg1:           "#c4c8da", // bg_highlight
		Bg2:           "#a8aecb", // fg_gutter
		Bg3:           "#8990b3", // dark3
		DiffAddBg:     "#c3ddb8",
		DiffDelBg:     "#f0c5c6",
		DiffContextBg: "#d5d6db",
		Fg:            "#3760bf",
		Fg1:           "#3760bf",
		Fg2:           "#6172b0", // fg_dark
		Dim:           "#848cb5", // comment
		Dim1:          "#8990b3", // dark3
		Blue:          "#2e7de9",
		Green:         "#587539",
		Red:           "#f52a65",
		Yellow:        "#8c6c3e",
		Purple:        "#7847bd",
		Teal:          "#118c74",
		Cyan:          "#007197",
		Orange:        "#b15c00",
		Pink:          "#9854f1", // magenta
		Lavender:      "#2e7de9",
	})
}
