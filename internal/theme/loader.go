package theme

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

// LoadDir loads all *.toml theme files from the given directory.
// Missing fields are filled from the fallback theme.
func LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	fb := Fallback()
	if fb == nil {
		return nil
	}
	fallbackRaw := fb.Raw()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		name := strings.TrimSuffix(entry.Name(), ".toml")

		var raw RawTokens
		if _, err := toml.DecodeFile(path, &raw); err != nil {
			// Skip malformed files
			continue
		}

		mergeTokens(&raw, &fallbackRaw)
		Register(&externalTheme{name: name, raw: raw})
	}

	return nil
}

// mergeTokens fills empty fields in dst from src.
func mergeTokens(dst, src *RawTokens) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src).Elem()

	for i := 0; i < dv.NumField(); i++ {
		df := dv.Field(i)
		sf := sv.Field(i)
		if df.Kind() == reflect.String && df.String() == "" {
			df.SetString(sf.String())
		}
	}
}

// externalTheme wraps a loaded theme file.
type externalTheme struct {
	name string
	raw  RawTokens
}

func (t *externalTheme) Name() string   { return t.name }
func (t *externalTheme) Raw() RawTokens { return t.raw }
