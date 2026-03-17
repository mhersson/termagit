package theme

import (
	"sort"
	"sync"
)

// Theme represents a color theme.
type Theme interface {
	Name() string
	Raw() RawTokens
}

var (
	mu       sync.RWMutex
	registry = make(map[string]Theme)
	fallback Theme
)

// Register adds a theme to the registry.
func Register(t Theme) {
	mu.Lock()
	defer mu.Unlock()
	registry[t.Name()] = t
}

// Get returns a theme by name.
func Get(name string) (Theme, bool) {
	mu.RLock()
	defer mu.RUnlock()
	t, ok := registry[name]
	return t, ok
}

// Names returns all registered theme names, sorted.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Fallback returns the default theme (catppuccin-mocha).
func Fallback() Theme {
	mu.RLock()
	defer mu.RUnlock()
	return fallback
}

// setFallback is called by the catppuccin_mocha init() function.
func setFallback(t Theme) {
	mu.Lock()
	defer mu.Unlock()
	fallback = t
}
