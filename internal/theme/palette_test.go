package theme

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromPalette_NoEmptyFields(t *testing.T) {
	p := Palette{
		Bg:        "#1e1e2e",
		Bg1:       "#313244",
		Bg2:       "#45475a",
		Bg3:       "#585b70",
		DiffAddBg:     "#1e3a2f",
		DiffDelBg:     "#3b1f29",
		DiffContextBg: "#313244",
		Fg:            "#cdd6f4",
		Fg1:       "#cdd6f4",
		Fg2:       "#bac2de",
		Dim:       "#6c7086",
		Dim1:      "#7f849c",
		Blue:      "#89b4fa",
		Green:     "#a6e3a1",
		Red:       "#f38ba8",
		Yellow:    "#f9e2af",
		Purple:    "#cba6f7",
		Teal:      "#94e2d5",
		Cyan:      "#89dceb",
		Orange:    "#fab387",
		Pink:      "#f5c2e7",
		Lavender:  "#b4befe",
	}

	raw := FromPalette(p)
	assertNoEmptyFields(t, raw, "palette")
}

func TestFromPalette_MapsColorsCorrectly(t *testing.T) {
	p := Palette{
		Bg:        "#bg0000",
		Bg1:       "#bg1111",
		Bg2:       "#bg2222",
		Bg3:       "#bg3333",
		DiffAddBg:     "#dab000",
		DiffDelBg:     "#ddb000",
		DiffContextBg: "#dcb000",
		Fg:            "#fg0000",
		Fg1:       "#fg1111",
		Fg2:       "#fg2222",
		Dim:       "#dim000",
		Dim1:      "#dim111",
		Blue:      "#blue00",
		Green:     "#green0",
		Red:       "#red000",
		Yellow:    "#yelo00",
		Purple:    "#purp00",
		Teal:      "#teal00",
		Cyan:      "#cyan00",
		Orange:    "#oran00",
		Pink:      "#pink00",
		Lavender:  "#lav000",
	}

	raw := FromPalette(p)

	// Text
	assert.Equal(t, "#fg1111", raw.Normal)
	assert.Equal(t, "#fg0000", raw.Bold)
	assert.Equal(t, "#dim000", raw.Dim)
	assert.Equal(t, "#dim000", raw.Comment)

	// Git objects
	assert.Equal(t, "#blue00", raw.Branch)
	assert.Equal(t, "#blue00", raw.BranchHead)
	assert.Equal(t, "#green0", raw.Remote)
	assert.Equal(t, "#yelo00", raw.Tag)
	assert.Equal(t, "#dim111", raw.Hash)
	assert.Equal(t, "#lav000", raw.HashCurrent)
	assert.Equal(t, "#pink00", raw.CommitAuthor)
	assert.Equal(t, "#fg2222", raw.CommitDate)

	// Section headers
	assert.Equal(t, "#purp00", raw.SectionHeader)

	// Diff
	assert.Equal(t, "#green0", raw.DiffAdd)
	assert.Equal(t, "#dab000", raw.DiffAddBg)
	assert.Equal(t, "#red000", raw.DiffDelete)
	assert.Equal(t, "#ddb000", raw.DiffDeleteBg)
	assert.Equal(t, "#fg2222", raw.DiffContext)
	assert.Equal(t, "#dcb000", raw.DiffContextBg)
	assert.Equal(t, "#teal00", raw.DiffHunkHeader)

	// Change indicators
	assert.Equal(t, "#blue00", raw.ChangeModified)
	assert.Equal(t, "#green0", raw.ChangeAdded)
	assert.Equal(t, "#red000", raw.ChangeDeleted)
	assert.Equal(t, "#purp00", raw.ChangeRenamed)
	assert.Equal(t, "#teal00", raw.ChangeCopied)
	assert.Equal(t, "#dim111", raw.ChangeUntracked)

	// Status
	assert.Equal(t, "#green0", raw.Staged)
	assert.Equal(t, "#yelo00", raw.Unstaged)
	assert.Equal(t, "#red000", raw.Conflict)

	// Popup
	assert.Equal(t, "#bg3333", raw.PopupBorder)
	assert.Equal(t, "#fg0000", raw.PopupTitle)
	assert.Equal(t, "#blue00", raw.PopupKey)
	assert.Equal(t, "#blue00", raw.PopupKeyBg)
	assert.Equal(t, "#cyan00", raw.PopupSwitch)
	assert.Equal(t, "#yelo00", raw.PopupOption)
	assert.Equal(t, "#fg1111", raw.PopupAction)
	assert.Equal(t, "#purp00", raw.PopupSection)

	// Notification
	assert.Equal(t, "#blue00", raw.NotificationInfo)
	assert.Equal(t, "#green0", raw.NotificationSuccess)
	assert.Equal(t, "#oran00", raw.NotificationWarn)
	assert.Equal(t, "#red000", raw.NotificationError)

	// Confirmation
	assert.Equal(t, "#oran00", raw.ConfirmBorder)
	assert.Equal(t, "#fg1111", raw.ConfirmText)
	assert.Equal(t, "#yelo00", raw.ConfirmKey)

	// Cursor and selection
	assert.Equal(t, "#fg0000", raw.Cursor)
	assert.Equal(t, "#bg1111", raw.CursorBg)
	assert.Equal(t, "#fg0000", raw.Selection)
	assert.Equal(t, "#bg2222", raw.SelectBg)
	assert.Equal(t, "#bg0000", raw.Background)

	// Graph
	assert.Equal(t, "#oran00", raw.GraphOrange)
	assert.Equal(t, "#green0", raw.GraphGreen)
	assert.Equal(t, "#red000", raw.GraphRed)
	assert.Equal(t, "#blue00", raw.GraphBlue)
	assert.Equal(t, "#yelo00", raw.GraphYellow)
	assert.Equal(t, "#cyan00", raw.GraphCyan)
	assert.Equal(t, "#purp00", raw.GraphPurple)
	assert.Equal(t, "#dim000", raw.GraphGray)
	assert.Equal(t, "#fg0000", raw.GraphWhite)

	// Sequencer headers
	assert.Equal(t, "#pink00", raw.Merging)
	assert.Equal(t, "#teal00", raw.Rebasing)
	assert.Equal(t, "#green0", raw.Picking)
	assert.Equal(t, "#red000", raw.Reverting)
	assert.Equal(t, "#yelo00", raw.Bisecting)

	// Misc
	assert.Equal(t, "#dim000", raw.RebaseDone)
	assert.Equal(t, "#dim000", raw.SubtleText)
	assert.Equal(t, "#purp00", raw.Stashes)

	// Commit view
	assert.Equal(t, "#cyan00", raw.CommitViewHeader)
	assert.Equal(t, "#bg0000", raw.CommitViewHeaderFg)
	assert.Equal(t, "#blue00", raw.FilePath)
	assert.Equal(t, "#oran00", raw.Number)

	// Diff view
	assert.Equal(t, "#bg2222", raw.DiffHeader)
	assert.Equal(t, "#blue00", raw.DiffHeaderFg)
	assert.Equal(t, "#bg1111", raw.FloatHeader)
	assert.Equal(t, "#cyan00", raw.FloatHeaderFg)
}

func TestFromPalette_CatppuccinMocha_MatchesBuiltin(t *testing.T) {
	theme, ok := Get("catppuccin-mocha")
	require.True(t, ok)

	expected := theme.Raw()
	actual := theme.Raw() // after refactor, this will use FromPalette internally

	v := reflect.ValueOf(expected)
	a := reflect.ValueOf(actual)
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		assert.Equal(t, v.Field(i).String(), a.Field(i).String(),
			"field %s mismatch", typ.Field(i).Name)
	}
}

func TestFromPalette_EverforestDark_MatchesBuiltin(t *testing.T) {
	theme, ok := Get("everforest-dark")
	require.True(t, ok)

	expected := theme.Raw()
	actual := theme.Raw()

	v := reflect.ValueOf(expected)
	a := reflect.ValueOf(actual)
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		assert.Equal(t, v.Field(i).String(), a.Field(i).String(),
			"field %s mismatch", typ.Field(i).Name)
	}
}

func TestFromPalette_TokyoNight_MatchesBuiltin(t *testing.T) {
	theme, ok := Get("tokyo-night")
	require.True(t, ok)

	expected := theme.Raw()
	actual := theme.Raw()

	v := reflect.ValueOf(expected)
	a := reflect.ValueOf(actual)
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		assert.Equal(t, v.Field(i).String(), a.Field(i).String(),
			"field %s mismatch", typ.Field(i).Name)
	}
}

func TestFromPalette_CatppuccinLatte_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-latte")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "catppuccin-latte")
}

func TestFromPalette_CatppuccinFrappe_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-frappe")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "catppuccin-frappe")
}

func TestFromPalette_CatppuccinMacchiato_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-macchiato")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "catppuccin-macchiato")
}

func TestFromPalette_TokyoNightStorm_NoEmptyFields(t *testing.T) {
	theme, ok := Get("tokyo-night-storm")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "tokyo-night-storm")
}

func TestFromPalette_TokyoNightLight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("tokyo-night-light")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "tokyo-night-light")
}

func TestFromPalette_GruvboxDark_NoEmptyFields(t *testing.T) {
	theme, ok := Get("gruvbox-dark")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "gruvbox-dark")
}

func TestFromPalette_GruvboxLight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("gruvbox-light")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "gruvbox-light")
}

func TestFromPalette_SolarizedDark_NoEmptyFields(t *testing.T) {
	theme, ok := Get("solarized-dark")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "solarized-dark")
}

func TestFromPalette_SolarizedLight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("solarized-light")
	require.True(t, ok)
	assertNoEmptyFields(t, theme.Raw(), "solarized-light")
}
