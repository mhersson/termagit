package config

import (
	"os"
	"path/filepath"
)

const appName = "conjit"

// ConfigDir returns the conjit configuration directory.
// Uses $XDG_CONFIG_HOME/conjit or falls back to $HOME/.config/conjit.
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", appName), nil
}

// StateDir returns the conjit state directory.
// Uses $XDG_STATE_HOME/conjit or falls back to $HOME/.local/state/conjit.
func StateDir() (string, error) {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, appName), nil
	}
	home := os.Getenv("HOME")
	return filepath.Join(home, ".local", "state", appName), nil
}

// ThemesDir returns the themes subdirectory of ConfigDir.
func ThemesDir() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "themes"), nil
}

// ConfigFile returns the path to config.toml.
func ConfigFile() (string, error) {
	cfg, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "config.toml"), nil
}

// LogFile returns the path to the command log file.
func LogFile() (string, error) {
	state, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(state, "commands.log"), nil
}
