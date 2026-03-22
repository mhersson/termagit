package popup

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/mhersson/termagit/internal/config"
)

// stateFile is the name of the popup state file.
const stateFile = "popup_state.toml"

// State holds persisted popup switch/option values.
type State struct {
	// Switches maps popup name -> label -> enabled
	Switches map[string]map[string]bool `toml:"switches"`

	// Options maps popup name -> label -> value
	Options map[string]map[string]string `toml:"options"`

	// nonPersisted tracks switches that should not be saved
	nonPersisted map[string]map[string]bool
}

// NewState creates a new empty state.
func NewState() *State {
	return &State{
		Switches:     make(map[string]map[string]bool),
		Options:      make(map[string]map[string]string),
		nonPersisted: make(map[string]map[string]bool),
	}
}

// statePath returns the path to the state file.
func statePath() (string, error) {
	dir, err := config.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFile), nil
}

// Load reads state from disk.
func (s *State) Load() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, that's fine
		return nil
	}

	_, err = toml.DecodeFile(path, s)
	return err
}

// Save writes state to disk.
func (s *State) Save() error {
	path, err := statePath()
	if err != nil {
		return err
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Create a copy without non-persisted switches
	toSave := &State{
		Switches: make(map[string]map[string]bool),
		Options:  make(map[string]map[string]string),
	}

	for popup, switches := range s.Switches {
		toSave.Switches[popup] = make(map[string]bool)
		for label, enabled := range switches {
			// Skip non-persisted
			if s.isNonPersisted(popup, label) {
				continue
			}
			// Only save enabled switches
			if enabled {
				toSave.Switches[popup][label] = enabled
			}
		}
		// Remove empty maps
		if len(toSave.Switches[popup]) == 0 {
			delete(toSave.Switches, popup)
		}
	}

	for popup, options := range s.Options {
		toSave.Options[popup] = make(map[string]string)
		for label, value := range options {
			if value != "" {
				toSave.Options[popup][label] = value
			}
		}
		// Remove empty maps
		if len(toSave.Options[popup]) == 0 {
			delete(toSave.Options, popup)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	return toml.NewEncoder(f).Encode(toSave)
}

// SetSwitch sets a switch value.
func (s *State) SetSwitch(popup, label string, enabled bool) {
	if s.Switches[popup] == nil {
		s.Switches[popup] = make(map[string]bool)
	}
	s.Switches[popup][label] = enabled
}

// GetSwitch returns a switch value, defaulting to false if not set.
func (s *State) GetSwitch(popup, label string) bool {
	if s.Switches[popup] == nil {
		return false
	}
	return s.Switches[popup][label]
}

// SetOption sets an option value.
func (s *State) SetOption(popup, label, value string) {
	if s.Options[popup] == nil {
		s.Options[popup] = make(map[string]string)
	}
	s.Options[popup][label] = value
}

// GetOption returns an option value, defaulting to empty string if not set.
func (s *State) GetOption(popup, label string) string {
	if s.Options[popup] == nil {
		return ""
	}
	return s.Options[popup][label]
}

// MarkNonPersisted marks a switch as non-persisted.
func (s *State) MarkNonPersisted(popup, label string) {
	if s.nonPersisted[popup] == nil {
		s.nonPersisted[popup] = make(map[string]bool)
	}
	s.nonPersisted[popup][label] = true
}

func (s *State) isNonPersisted(popup, label string) bool {
	if s.nonPersisted[popup] == nil {
		return false
	}
	return s.nonPersisted[popup][label]
}

// ApplyToPopup applies saved state to a popup.
func (s *State) ApplyToPopup(popupName string, p *Popup) {
	// Apply switches
	for i := range p.switches {
		sw := &p.switches[i]
		if s.GetSwitch(popupName, sw.Label) {
			sw.Enabled = true
		}
	}

	// Apply options
	for i := range p.options {
		opt := &p.options[i]
		if v := s.GetOption(popupName, opt.Label); v != "" {
			opt.Value = v
		}
	}
}

// SaveFromPopup saves popup state to this State.
func (s *State) SaveFromPopup(popupName string, p *Popup) {
	// Save switches
	for _, sw := range p.switches {
		if !sw.Persisted {
			s.MarkNonPersisted(popupName, sw.Label)
		}
		s.SetSwitch(popupName, sw.Label, sw.Enabled)
	}

	// Save options
	for _, opt := range p.options {
		s.SetOption(popupName, opt.Label, opt.Value)
	}
}
