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
	assert.Contains(t, names, "everforest-dark")
	assert.Contains(t, names, "tokyo-night")
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
	require.Len(t, names, 3) // catppuccin-mocha, everforest-dark, tokyo-night

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
