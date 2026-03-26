package config

// SectionConfig controls visibility and folding of a status buffer section.
type SectionConfig struct {
	Folded bool `toml:"folded"`
	Hidden bool `toml:"hidden"`
}

// SectionsConfig holds config for all 12 status buffer sections.
type SectionsConfig struct {
	Sequencer          SectionConfig `toml:"sequencer"`
	Untracked          SectionConfig `toml:"untracked"`
	Unstaged           SectionConfig `toml:"unstaged"`
	Staged             SectionConfig `toml:"staged"`
	Stashes            SectionConfig `toml:"stashes"`
	UnpulledUpstream   SectionConfig `toml:"unpulled_upstream"`
	UnmergedUpstream   SectionConfig `toml:"unmerged_upstream"`
	UnpulledPushRemote SectionConfig `toml:"unpulled_pushremote"`
	UnmergedPushRemote SectionConfig `toml:"unmerged_pushremote"`
	Recent             SectionConfig `toml:"recent"`
	Rebase             SectionConfig `toml:"rebase"`
	Bisect             SectionConfig `toml:"bisect"`
}

// GitConfig holds git-related settings.
type GitConfig struct {
	Executable   string `toml:"executable"`
	SortBranches string `toml:"sort_branches"`
	CommitOrder  string `toml:"commit_order"`
	GraphStyle   string `toml:"graph_style"`
}

// UIConfig holds UI-related settings.
type UIConfig struct {
	DisableHint        bool `toml:"disable_hint"`
	DisableLineNumbers bool `toml:"disable_line_numbers"`
	RecentCommitCount  int  `toml:"recent_commit_count"`
	HEADPadding        int  `toml:"HEAD_padding"`
}

// CommitEditorConfig holds commit editor settings.
type CommitEditorConfig struct {
	ShowStagedDiff               bool   `toml:"show_staged_diff"`
	DisableInsertOnCommit        bool   `toml:"disable_insert_on_commit"`
	GenerateCommitMessageCommand string `toml:"generate_commit_message_command"`
}

// CommitViewConfig holds commit view settings.
type CommitViewConfig struct{}

// FilewatcherConfig holds file watcher settings.
type FilewatcherConfig struct {
	Enabled bool `toml:"enabled"`
}

// Config holds the application configuration.
type Config struct {
	Theme        string             `toml:"theme"`
	Git          GitConfig          `toml:"git"`
	UI           UIConfig           `toml:"ui"`
	CommitEditor CommitEditorConfig `toml:"commit_editor"`
	CommitView   CommitViewConfig   `toml:"commit_view"`
	Filewatcher  FilewatcherConfig  `toml:"filewatcher"`
	Sections     SectionsConfig     `toml:"sections"`
	Log          LogConfig          `toml:"log"`
}

// LogConfig holds command log settings.
type LogConfig struct {
	MaxSize string `toml:"max_size"` // e.g., "10MB", "1GB"
	Keep    int    `toml:"keep"`     // number of rotated files to keep
}

// defaults returns a fully-populated Config with default values.
func defaults() *Config {
	return &Config{
		Theme: "catppuccin-mocha",
		Git: GitConfig{
			Executable:   "git",
			SortBranches: "-committerdate",
			CommitOrder:  "topo",
			GraphStyle:   "unicode",
		},
		UI: UIConfig{
			DisableHint:        false,
			DisableLineNumbers: false,
			RecentCommitCount:  10,
			HEADPadding:        10,
		},
		CommitEditor: CommitEditorConfig{
			ShowStagedDiff:        true,
			DisableInsertOnCommit: false,
		},
		CommitView: CommitViewConfig{},
		Filewatcher: FilewatcherConfig{
			Enabled: true,
		},
		Sections: SectionsConfig{
			Sequencer:          SectionConfig{Folded: false, Hidden: false},
			Untracked:          SectionConfig{Folded: false, Hidden: false},
			Unstaged:           SectionConfig{Folded: false, Hidden: false},
			Staged:             SectionConfig{Folded: false, Hidden: false},
			Stashes:            SectionConfig{Folded: true, Hidden: false},
			UnpulledUpstream:   SectionConfig{Folded: true, Hidden: false},
			UnmergedUpstream:   SectionConfig{Folded: false, Hidden: false},
			UnpulledPushRemote: SectionConfig{Folded: true, Hidden: false},
			UnmergedPushRemote: SectionConfig{Folded: false, Hidden: false},
			Recent:             SectionConfig{Folded: false, Hidden: false},
			Rebase:             SectionConfig{Folded: false, Hidden: false},
			Bisect:             SectionConfig{Folded: false, Hidden: false},
		},
		Log: LogConfig{
			MaxSize: "10MB",
			Keep:    3,
		},
	}
}
