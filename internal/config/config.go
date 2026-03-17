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

// Config holds the application configuration.
type Config struct {
	Theme    string         `toml:"theme"`
	Sections SectionsConfig `toml:"sections"`
	Log      LogConfig      `toml:"log"`
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
