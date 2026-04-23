package theme

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

// themeFile is the TOML structure for external theme files.
// It supports both palette-based and token-based definitions.
type themeFile struct {
	Pal       Palette `toml:"palette"`
	RawTokens         // anonymous embed — token fields decode at top level
}

// LoadDir loads all *.toml theme files from the given directory.
// Missing fields are filled from the fallback theme.
func LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	fb := Fallback()
	if fb.Name() == "" {
		return nil
	}
	fallbackRaw := fb.Raw()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		name := strings.TrimSuffix(entry.Name(), ".toml")

		var f themeFile
		if _, err := toml.DecodeFile(path, &f); err != nil {
			// Skip malformed files
			continue
		}

		var raw RawTokens
		if f.Pal.hasPalette() {
			// Generate base tokens from palette
			raw = FromPalette(f.Pal)
			// Overlay any explicit token-level overrides
			mergeRawTokens(&raw, &f.RawTokens, true)
		} else {
			raw = f.RawTokens
		}

		// Fill remaining empty fields from fallback
		mergeRawTokens(&raw, &fallbackRaw, false)
		Register(NewTheme(name, raw))
	}

	return nil
}

// mergeRawTokens copies string fields from src into dst.
// When overwrite is true, non-empty src fields replace dst fields (overlay).
// When overwrite is false, src fields only fill empty dst fields (fallback).
func mergeRawTokens(dst, src *RawTokens, overwrite bool) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src).Elem()

	for i := range dv.NumField() {
		df := dv.Field(i)
		sf := sv.Field(i)
		if sf.Kind() != reflect.String || sf.String() == "" {
			continue
		}
		if overwrite || df.String() == "" {
			df.SetString(sf.String())
		}
	}
}
