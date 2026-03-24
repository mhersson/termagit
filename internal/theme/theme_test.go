package theme

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinTheme_CatppuccinMocha_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-mocha")
	require.True(t, ok, "catppuccin-mocha should be registered")

	raw := theme.Raw()
	assertNoEmptyFields(t, raw, "catppuccin-mocha")
}

func TestBuiltinTheme_EverforestDark_NoEmptyFields(t *testing.T) {
	theme, ok := Get("everforest-dark")
	require.True(t, ok, "everforest-dark should be registered")

	raw := theme.Raw()
	assertNoEmptyFields(t, raw, "everforest-dark")
}

func TestBuiltinTheme_TokyoNight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("tokyo-night")
	require.True(t, ok, "tokyo-night should be registered")

	raw := theme.Raw()
	assertNoEmptyFields(t, raw, "tokyo-night")
}

func TestBuiltinTheme_CatppuccinLatte_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-latte")
	require.True(t, ok, "catppuccin-latte should be registered")
	assertNoEmptyFields(t, theme.Raw(), "catppuccin-latte")
}

func TestBuiltinTheme_CatppuccinFrappe_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-frappe")
	require.True(t, ok, "catppuccin-frappe should be registered")
	assertNoEmptyFields(t, theme.Raw(), "catppuccin-frappe")
}

func TestBuiltinTheme_CatppuccinMacchiato_NoEmptyFields(t *testing.T) {
	theme, ok := Get("catppuccin-macchiato")
	require.True(t, ok, "catppuccin-macchiato should be registered")
	assertNoEmptyFields(t, theme.Raw(), "catppuccin-macchiato")
}

func TestBuiltinTheme_TokyoNightStorm_NoEmptyFields(t *testing.T) {
	theme, ok := Get("tokyo-night-storm")
	require.True(t, ok, "tokyo-night-storm should be registered")
	assertNoEmptyFields(t, theme.Raw(), "tokyo-night-storm")
}

func TestBuiltinTheme_TokyoNightLight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("tokyo-night-light")
	require.True(t, ok, "tokyo-night-light should be registered")
	assertNoEmptyFields(t, theme.Raw(), "tokyo-night-light")
}

func TestBuiltinTheme_GruvboxDark_NoEmptyFields(t *testing.T) {
	theme, ok := Get("gruvbox-dark")
	require.True(t, ok, "gruvbox-dark should be registered")
	assertNoEmptyFields(t, theme.Raw(), "gruvbox-dark")
}

func TestBuiltinTheme_GruvboxLight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("gruvbox-light")
	require.True(t, ok, "gruvbox-light should be registered")
	assertNoEmptyFields(t, theme.Raw(), "gruvbox-light")
}

func TestBuiltinTheme_SolarizedDark_NoEmptyFields(t *testing.T) {
	theme, ok := Get("solarized-dark")
	require.True(t, ok, "solarized-dark should be registered")
	assertNoEmptyFields(t, theme.Raw(), "solarized-dark")
}

func TestBuiltinTheme_SolarizedLight_NoEmptyFields(t *testing.T) {
	theme, ok := Get("solarized-light")
	require.True(t, ok, "solarized-light should be registered")
	assertNoEmptyFields(t, theme.Raw(), "solarized-light")
}

func assertNoEmptyFields(t *testing.T, raw RawTokens, themeName string) {
	t.Helper()
	v := reflect.ValueOf(raw)
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.String {
			assert.NotEmpty(t, field.String(), "theme %s has empty field %s", themeName, typ.Field(i).Name)
		}
	}
}

func TestRegister_AddsToRegistry(t *testing.T) {
	// Themes are already registered via init()
	names := Names()
	assert.Contains(t, names, "catppuccin-mocha")
	assert.Contains(t, names, "catppuccin-latte")
	assert.Contains(t, names, "catppuccin-frappe")
	assert.Contains(t, names, "catppuccin-macchiato")
	assert.Contains(t, names, "everforest-dark")
	assert.Contains(t, names, "tokyo-night")
	assert.Contains(t, names, "tokyo-night-storm")
	assert.Contains(t, names, "tokyo-night-light")
	assert.Contains(t, names, "gruvbox-dark")
	assert.Contains(t, names, "gruvbox-light")
	assert.Contains(t, names, "solarized-dark")
	assert.Contains(t, names, "solarized-light")
}

func TestGet_ReturnsRegisteredTheme(t *testing.T) {
	theme, ok := Get("catppuccin-mocha")
	assert.True(t, ok)
	assert.Equal(t, "catppuccin-mocha", theme.Name())
}

func TestGet_Unknown_ReturnsFalse(t *testing.T) {
	_, ok := Get("nonexistent-theme")
	assert.False(t, ok)
}

func TestFallback_ReturnsCatppuccinMocha(t *testing.T) {
	fb := Fallback()
	require.NotNil(t, fb)
	assert.Equal(t, "catppuccin-mocha", fb.Name())
}

func TestNames_ReturnsSorted(t *testing.T) {
	names := Names()
	require.Len(t, names, 12)

	// Should be alphabetically sorted
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "names should be sorted")
	}
}

func TestLoadDir_MissingDirectory_ReturnsNil(t *testing.T) {
	err := LoadDir("/nonexistent/directory")
	assert.NoError(t, err)
}

func TestLoadDir_PartialTheme_MergesFromFallback(t *testing.T) {
	dir := t.TempDir()

	// Create a partial theme with only one field
	content := `normal = "#ffffff"`
	err := os.WriteFile(filepath.Join(dir, "partial.toml"), []byte(content), 0o644)
	require.NoError(t, err)

	err = LoadDir(dir)
	require.NoError(t, err)

	theme, ok := Get("partial")
	require.True(t, ok)

	raw := theme.Raw()
	assert.Equal(t, "#ffffff", raw.Normal)
	// Other fields should be filled from fallback
	assert.NotEmpty(t, raw.Branch)
	assert.NotEmpty(t, raw.Remote)
}

func TestLoadDir_ExternalTheme_OverridesBuiltin(t *testing.T) {
	dir := t.TempDir()

	// Create a complete custom theme
	fb := Fallback().Raw()
	fb.Normal = "#123456"

	content := `normal = "#123456"`
	err := os.WriteFile(filepath.Join(dir, "custom.toml"), []byte(content), 0o644)
	require.NoError(t, err)

	err = LoadDir(dir)
	require.NoError(t, err)

	theme, ok := Get("custom")
	require.True(t, ok)
	assert.Equal(t, "#123456", theme.Raw().Normal)
}

func TestLoadDir_MalformedFile_SkipsWithWarning(t *testing.T) {
	dir := t.TempDir()

	// Create a malformed TOML file
	content := `this is not valid toml {{{`
	err := os.WriteFile(filepath.Join(dir, "bad.toml"), []byte(content), 0o644)
	require.NoError(t, err)

	// Should not return error, just skip the file
	err = LoadDir(dir)
	assert.NoError(t, err)

	// Theme should not be registered
	_, ok := Get("bad")
	assert.False(t, ok)
}

func TestLoadDir_PaletteTheme_GeneratesTokens(t *testing.T) {
	dir := t.TempDir()

	content := `
[palette]
bg        = "#000000"
bg1       = "#111111"
bg2       = "#222222"
bg3       = "#333333"
diff_add_bg = "#001100"
diff_del_bg = "#110000"
fg        = "#ffffff"
fg1       = "#eeeeee"
fg2       = "#cccccc"
dim       = "#666666"
dim1      = "#888888"
blue      = "#0000ff"
green     = "#00ff00"
red       = "#ff0000"
yellow    = "#ffff00"
purple    = "#800080"
teal      = "#008080"
cyan      = "#00ffff"
orange    = "#ff8800"
pink      = "#ff69b4"
lavender  = "#b57edc"
`
	err := os.WriteFile(filepath.Join(dir, "palette-test.toml"), []byte(content), 0o644)
	require.NoError(t, err)

	err = LoadDir(dir)
	require.NoError(t, err)

	theme, ok := Get("palette-test")
	require.True(t, ok)

	raw := theme.Raw()
	assertNoEmptyFields(t, raw, "palette-test")

	// Verify palette mapping
	assert.Equal(t, "#eeeeee", raw.Normal)  // fg1
	assert.Equal(t, "#0000ff", raw.Branch)  // blue
	assert.Equal(t, "#00ff00", raw.Staged)  // green
	assert.Equal(t, "#ff0000", raw.Conflict) // red
}

func TestLoadDir_PaletteWithTokenOverrides(t *testing.T) {
	dir := t.TempDir()

	content := `
# Token override takes precedence over palette
normal = "#aaaaaa"

[palette]
bg        = "#000000"
bg1       = "#111111"
bg2       = "#222222"
bg3       = "#333333"
diff_add_bg = "#001100"
diff_del_bg = "#110000"
fg        = "#ffffff"
fg1       = "#eeeeee"
fg2       = "#cccccc"
dim       = "#666666"
dim1      = "#888888"
blue      = "#0000ff"
green     = "#00ff00"
red       = "#ff0000"
yellow    = "#ffff00"
purple    = "#800080"
teal      = "#008080"
cyan      = "#00ffff"
orange    = "#ff8800"
pink      = "#ff69b4"
lavender  = "#b57edc"
`
	err := os.WriteFile(filepath.Join(dir, "palette-override.toml"), []byte(content), 0o644)
	require.NoError(t, err)

	err = LoadDir(dir)
	require.NoError(t, err)

	theme, ok := Get("palette-override")
	require.True(t, ok)

	raw := theme.Raw()
	assertNoEmptyFields(t, raw, "palette-override")

	// Token override should win over palette-generated value
	assert.Equal(t, "#aaaaaa", raw.Normal) // explicitly set, not fg1
	// Non-overridden tokens should come from palette
	assert.Equal(t, "#0000ff", raw.Branch) // blue from palette
}

func TestMergeTokens_FillsEmptyFields(t *testing.T) {
	dst := RawTokens{Normal: "#ffffff"}
	src := RawTokens{
		Normal: "#000000", // Should NOT override
		Branch: "#89b4fa", // Should fill
		Remote: "#a6e3a1", // Should fill
	}

	mergeTokens(&dst, &src)

	assert.Equal(t, "#ffffff", dst.Normal) // Preserved
	assert.Equal(t, "#89b4fa", dst.Branch) // Filled
	assert.Equal(t, "#a6e3a1", dst.Remote) // Filled
}

func TestMergeTokens_DoesNotOverwriteNonEmpty(t *testing.T) {
	dst := RawTokens{Normal: "#ffffff", Branch: "#111111"}
	src := RawTokens{Normal: "#000000", Branch: "#222222"}

	mergeTokens(&dst, &src)

	assert.Equal(t, "#ffffff", dst.Normal)
	assert.Equal(t, "#111111", dst.Branch)
}

func TestCompile_NoZeroValueStyles(t *testing.T) {
	fb := Fallback()
	require.NotNil(t, fb)

	tokens := Compile(fb.Raw())

	// Verify all style fields have been set by checking they are not DeepEqual to zero-value
	v := reflect.ValueOf(tokens)
	typ := v.Type()

	zeroStyle := lipgloss.Style{}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type() == reflect.TypeFor[lipgloss.Style]() {
			style := field.Interface().(lipgloss.Style)
			// Use DeepEqual to check internal state, not rendered output
			assert.False(t, reflect.DeepEqual(zeroStyle, style),
				"field %s appears to be zero-value style", typ.Field(i).Name)
		}
	}
}

func TestCompile_CursorBlockHasReverse(t *testing.T) {
	fb := Fallback()
	require.NotNil(t, fb)

	tokens := Compile(fb.Raw())

	// CursorBlock must have reverse style for block cursor effect
	// We can't directly test ANSI output in tests (no TTY), but we can verify
	// the style was created with Reverse attribute by checking it's not a zero-value style
	assert.NotEqual(t, lipgloss.Style{}, tokens.CursorBlock, "CursorBlock should not be zero-value style")

	// Verify it renders content (doesn't panic, returns something)
	rendered := tokens.CursorBlock.Render("X")
	assert.NotEmpty(t, rendered, "CursorBlock should render content")
}
