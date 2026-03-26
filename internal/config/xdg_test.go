package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDir_UsesXDGConfigHome_WhenSet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	t.Setenv("HOME", "/home/user")

	dir, err := ConfigDir()

	require.NoError(t, err)
	assert.Equal(t, "/custom/config/termagit", dir)
}

func TestConfigDir_FallsBackToHomeConfig_WhenUnset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/user")

	dir, err := ConfigDir()

	require.NoError(t, err)
	assert.Equal(t, "/home/user/.config/termagit", dir)
}

func TestStateDir_UsesXDGStateHome_WhenSet(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/custom/state")
	t.Setenv("HOME", "/home/user")

	dir, err := StateDir()

	require.NoError(t, err)
	assert.Equal(t, "/custom/state/termagit", dir)
}

func TestStateDir_FallsBackToLocalState_WhenUnset(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/user")

	dir, err := StateDir()

	require.NoError(t, err)
	assert.Equal(t, "/home/user/.local/state/termagit", dir)
}

func TestThemesDir_IsSubdirOfConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	t.Setenv("HOME", "/home/user")

	themesDir, err := ThemesDir()
	require.NoError(t, err)

	configDir, err := ConfigDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(configDir, "themes"), themesDir)
}

func TestConfigFile_IsTomlInConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	t.Setenv("HOME", "/home/user")

	configFile, err := ConfigFile()
	require.NoError(t, err)

	configDir, err := ConfigDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(configDir, "config.toml"), configFile)
}

func TestLogFile_IsInStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/custom/state")
	t.Setenv("HOME", "/home/user")

	logFile, err := LogFile()
	require.NoError(t, err)

	stateDir, err := StateDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(stateDir, "commands.log"), logFile)
}

func TestConfigDir_EmptyHome_ReturnsError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")

	_, err := ConfigDir()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "neither XDG_CONFIG_HOME nor HOME is set")
}

func TestStateDir_EmptyHome_ReturnsError(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")

	_, err := StateDir()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "neither XDG_STATE_HOME nor HOME is set")
}
