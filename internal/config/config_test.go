package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "catppuccin-mocha", cfg.Theme)
}

func TestLoad_OnlyTheme_AllOtherFieldsAreDefaults(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "conjit")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", dir)

	content := `theme = "everforest-dark"`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644))

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Theme should be overwritten
	assert.Equal(t, "everforest-dark", cfg.Theme)

	// All sections should still have defaults
	assert.False(t, cfg.Sections.Untracked.Folded)
	assert.False(t, cfg.Sections.Untracked.Hidden)
	assert.True(t, cfg.Sections.Stashes.Folded) // default is folded
	assert.Equal(t, "10MB", cfg.Log.MaxSize)
	assert.Equal(t, 3, cfg.Log.Keep)
}

func TestLoad_PartialSections_UnsetSectionsAreDefaults(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "conjit")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", dir)

	content := `
[sections.untracked]
folded = true
hidden = true
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644))

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Untracked should be overwritten
	assert.True(t, cfg.Sections.Untracked.Folded)
	assert.True(t, cfg.Sections.Untracked.Hidden)

	// Other sections should have defaults
	assert.False(t, cfg.Sections.Unstaged.Folded)
	assert.True(t, cfg.Sections.Stashes.Folded)
}

func TestLoad_FullFile_OverridesAllDefaults(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "conjit")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", dir)

	content := `
theme = "tokyo-night"

[log]
max_size = "50MB"
keep = 5

[sections.sequencer]
folded = true
hidden = true

[sections.untracked]
folded = true
hidden = false
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644))

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "tokyo-night", cfg.Theme)
	assert.Equal(t, "50MB", cfg.Log.MaxSize)
	assert.Equal(t, 5, cfg.Log.Keep)
	assert.True(t, cfg.Sections.Sequencer.Folded)
	assert.True(t, cfg.Sections.Sequencer.Hidden)
	assert.True(t, cfg.Sections.Untracked.Folded)
	assert.False(t, cfg.Sections.Untracked.Hidden)
}

func TestDefaults_AllTwelveSectionsPresent(t *testing.T) {
	cfg := defaults()

	// All 12 sections must be accessible (compile-time check via field access)
	_ = cfg.Sections.Sequencer
	_ = cfg.Sections.Untracked
	_ = cfg.Sections.Unstaged
	_ = cfg.Sections.Staged
	_ = cfg.Sections.Stashes
	_ = cfg.Sections.UnpulledUpstream
	_ = cfg.Sections.UnmergedUpstream
	_ = cfg.Sections.UnpulledPushRemote
	_ = cfg.Sections.UnmergedPushRemote
	_ = cfg.Sections.Recent
	_ = cfg.Sections.Rebase
	_ = cfg.Sections.Bisect
}

func TestDefaults_EachSectionHasFoldedAndHidden(t *testing.T) {
	cfg := defaults()

	// Verify fields exist and have expected default values
	sections := []struct {
		name   string
		config SectionConfig
	}{
		{"Sequencer", cfg.Sections.Sequencer},
		{"Untracked", cfg.Sections.Untracked},
		{"Unstaged", cfg.Sections.Unstaged},
		{"Staged", cfg.Sections.Staged},
		{"Stashes", cfg.Sections.Stashes},
		{"UnpulledUpstream", cfg.Sections.UnpulledUpstream},
		{"UnmergedUpstream", cfg.Sections.UnmergedUpstream},
		{"UnpulledPushRemote", cfg.Sections.UnpulledPushRemote},
		{"UnmergedPushRemote", cfg.Sections.UnmergedPushRemote},
		{"Recent", cfg.Sections.Recent},
		{"Rebase", cfg.Sections.Rebase},
		{"Bisect", cfg.Sections.Bisect},
	}

	for _, s := range sections {
		// All sections should have Hidden = false by default
		assert.False(t, s.config.Hidden, "%s should not be hidden by default", s.name)
	}
}

func TestDefaults_ThemeIsCatppuccinMocha(t *testing.T) {
	cfg := defaults()
	assert.Equal(t, "catppuccin-mocha", cfg.Theme)
}

func TestParseMaxSize_MB(t *testing.T) {
	size, err := ParseMaxSize("10MB")
	require.NoError(t, err)
	assert.Equal(t, int64(10*1024*1024), size)
}

func TestParseMaxSize_GB(t *testing.T) {
	size, err := ParseMaxSize("2GB")
	require.NoError(t, err)
	assert.Equal(t, int64(2*1024*1024*1024), size)
}

func TestParseMaxSize_Invalid_ReturnsError(t *testing.T) {
	_, err := ParseMaxSize("not a size")
	assert.Error(t, err)
}

func TestDefaults_GitConfigHasDefaults(t *testing.T) {
	cfg := defaults()
	assert.Equal(t, "git", cfg.Git.Executable)
	assert.Equal(t, "-committerdate", cfg.Git.SortBranches)
	assert.Equal(t, "topo", cfg.Git.CommitOrder)
	assert.Equal(t, "unicode", cfg.Git.GraphStyle)
}

func TestDefaults_UIConfigHasDefaults(t *testing.T) {
	cfg := defaults()
	assert.False(t, cfg.UI.DisableHint)
	assert.False(t, cfg.UI.DisableContextHighlighting)
	assert.False(t, cfg.UI.DisableSigns)
	assert.False(t, cfg.UI.DisableLineNumbers)
	assert.False(t, cfg.UI.ShowHeadCommitHash)
	assert.Equal(t, 10, cfg.UI.RecentCommitCount)
	assert.Equal(t, 10, cfg.UI.HEADPadding)
	assert.False(t, cfg.UI.HEADFolded)
	assert.Equal(t, 3, cfg.UI.ModePadding)
	assert.Equal(t, "󰐗", cfg.UI.NotificationIcon)
	assert.Equal(t, 5000, cfg.UI.ConsoleTimeout)
	assert.True(t, cfg.UI.AutoShowConsole)
	assert.True(t, cfg.UI.AutoCloseConsole)
}

func TestDefaults_CommitEditorConfigHasDefaults(t *testing.T) {
	cfg := defaults()
	assert.True(t, cfg.CommitEditor.ShowStagedDiff)
	assert.Equal(t, "split", cfg.CommitEditor.StagedDiffSplitKind)
	assert.False(t, cfg.CommitEditor.SpellCheck)
	assert.False(t, cfg.CommitEditor.DisableInsertOnCommit)
	assert.Empty(t, cfg.CommitEditor.GenerateCommitMessageCommand)
}

func TestLoad_GenerateCommitMessageCommand_Loaded(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "conjit")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", dir)

	content := `
[commit_editor]
generate_commit_message_command = "/usr/local/bin/ai-commit"
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644))

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "/usr/local/bin/ai-commit", cfg.CommitEditor.GenerateCommitMessageCommand)
	// Other defaults preserved
	assert.True(t, cfg.CommitEditor.ShowStagedDiff)
}

func TestDefaults_CommitViewConfigHasDefaults(t *testing.T) {
	cfg := defaults()
	assert.True(t, cfg.CommitView.VerifyCommit)
}

func TestDefaults_FilewatcherConfigHasDefaults(t *testing.T) {
	cfg := defaults()
	assert.True(t, cfg.Filewatcher.Enabled)
}

func TestLoad_PartialUI_UnsetFieldsAreDefaults(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "conjit")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", dir)

	content := `
[ui]
disable_hint = true
recent_commit_count = 10
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644))

	cfg, err := Load()
	require.NoError(t, err)

	// Explicitly set values
	assert.True(t, cfg.UI.DisableHint)
	assert.Equal(t, 10, cfg.UI.RecentCommitCount)

	// Unset values should be defaults
	assert.False(t, cfg.UI.DisableContextHighlighting)
	assert.True(t, cfg.UI.AutoShowConsole)
}

func TestLoad_PartialGit_UnsetFieldsAreDefaults(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "conjit")
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	t.Setenv("XDG_CONFIG_HOME", dir)

	content := `
[git]
executable = "/usr/local/bin/git"
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644))

	cfg, err := Load()
	require.NoError(t, err)

	// Explicitly set value
	assert.Equal(t, "/usr/local/bin/git", cfg.Git.Executable)

	// Unset values should be defaults
	assert.Equal(t, "-committerdate", cfg.Git.SortBranches)
	assert.Equal(t, "topo", cfg.Git.CommitOrder)
}
