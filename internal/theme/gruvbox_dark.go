package theme

func init() {
	Register(&gruvboxDark{})
}

type gruvboxDark struct{}

func (t *gruvboxDark) Name() string { return "gruvbox-dark" }

func (t *gruvboxDark) Raw() RawTokens {
	return FromPalette(Palette{
		Bg:            "#282828", // bg0
		Bg1:           "#3c3836", // bg1
		Bg2:           "#504945", // bg2
		Bg3:           "#665c54", // bg3
		DiffAddBg:     "#32361a",
		DiffDelBg:     "#3c1f1e",
		DiffContextBg: "#3c3836",
		Fg:            "#ebdbb2", // fg1
		Fg1:           "#ebdbb2", // fg1
		Fg2:           "#d5c4a1", // fg2
		Dim:           "#928374", // gray
		Dim1:          "#a89984", // fg4
		Blue:          "#83a598", // bright_blue
		Green:         "#b8bb26", // bright_green
		Red:           "#fb4934", // bright_red
		Yellow:        "#fabd2f", // bright_yellow
		Purple:        "#d3869b", // bright_purple
		Teal:          "#8ec07c", // bright_aqua
		Cyan:          "#8ec07c", // bright_aqua (no separate cyan)
		Orange:        "#fe8019", // bright_orange
		Pink:          "#d3869b", // same as purple
		Lavender:      "#83a598", // same as blue
	})
}
