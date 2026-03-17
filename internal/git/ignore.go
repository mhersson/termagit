package git

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreScope represents which ignore file to write to.
type IgnoreScope int

const (
	IgnoreScopeTopLevel IgnoreScope = iota // .gitignore in repo root
	IgnoreScopeSubdir                      // .gitignore in file's subdirectory
	IgnoreScopePrivate                     // .git/info/exclude
	IgnoreScopeGlobal                      // user's global gitignore
)

// AddIgnoreRule appends pattern to the appropriate ignore file.
// Creates the file if it doesn't exist. Does not add duplicates.
func (r *Repository) AddIgnoreRule(ctx context.Context, pattern string, scope IgnoreScope) error {
	path, err := r.ignoreFilePath(ctx, scope)
	if err != nil {
		return fmt.Errorf("resolve ignore file path: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create ignore file directory: %w", err)
	}

	// Check if pattern already exists
	if containsPattern(path, pattern) {
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open ignore file: %w", err)
	}
	defer func() { _ = f.Close() }()

	_, err = fmt.Fprintf(f, "%s\n", pattern)
	if err != nil {
		return fmt.Errorf("write ignore pattern: %w", err)
	}

	return nil
}

// GlobalIgnoreFile reads core.excludesFile from git config.
func (r *Repository) GlobalIgnoreFile(ctx context.Context) (string, error) {
	out, err := r.runGit(ctx, "config", "--global", "core.excludesFile")
	if err != nil {
		// Not configured is not an error
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// IgnorePatternForPath generates a root-relative pattern for a path.
// Prefixes with / to anchor the pattern to the repo root.
func IgnorePatternForPath(path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

// ignoreFilePath resolves the path to the ignore file for the given scope.
func (r *Repository) ignoreFilePath(ctx context.Context, scope IgnoreScope) (string, error) {
	switch scope {
	case IgnoreScopeTopLevel, IgnoreScopeSubdir:
		return filepath.Join(r.path, ".gitignore"), nil
	case IgnoreScopePrivate:
		return filepath.Join(r.gitDir, "info", "exclude"), nil
	case IgnoreScopeGlobal:
		path, err := r.GlobalIgnoreFile(ctx)
		if err != nil {
			return "", err
		}
		if path == "" {
			// Fall back to XDG default
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("get home directory: %w", err)
			}
			xdg := os.Getenv("XDG_CONFIG_HOME")
			if xdg == "" {
				xdg = filepath.Join(home, ".config")
			}
			return filepath.Join(xdg, "git", "ignore"), nil
		}
		return path, nil
	default:
		return "", fmt.Errorf("unknown ignore scope: %d", scope)
	}
}

// containsPattern checks if the ignore file already contains the pattern.
func containsPattern(path, pattern string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == pattern {
			return true
		}
	}
	return false
}
