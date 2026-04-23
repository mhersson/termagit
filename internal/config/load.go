package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Load reads the config file and returns the configuration.
// Missing file returns defaults, no error.
// Partial config merges with defaults (only present keys are overwritten).
func Load() (*Config, error) {
	cfg := defaults()

	path, err := ConfigFile()
	if err != nil {
		return cfg, nil //nolint:nilerr // config dir unavailable → use defaults, not an error for the caller
	}

	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}

// ParseMaxSize parses a size string like "10MB" or "1GB" into bytes.
func ParseMaxSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	var multiplier int64 = 1
	var numStr string

	switch {
	case strings.HasSuffix(s, "GB"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		numStr = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		numStr = strings.TrimSuffix(s, "B")
	default:
		numStr = s
	}

	numStr = strings.TrimSpace(numStr)
	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size: %s", s)
	}

	return n * multiplier, nil
}
