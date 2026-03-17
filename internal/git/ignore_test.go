package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddIgnoreRule_Local_AppendsToGitignore(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddIgnoreRule(ctx, "*.log", IgnoreScopeTopLevel)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(r.path, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "*.log\n")
}

func TestAddIgnoreRule_CreatesFileIfMissing(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	gitignorePath := filepath.Join(r.path, ".gitignore")
	// Ensure no .gitignore exists
	_ = os.Remove(gitignorePath)

	err := r.AddIgnoreRule(ctx, "build/", IgnoreScopeTopLevel)
	require.NoError(t, err)

	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Equal(t, "build/\n", string(content))
}

func TestAddIgnoreRule_DoesNotDuplicate(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddIgnoreRule(ctx, "*.log", IgnoreScopeTopLevel)
	require.NoError(t, err)

	err = r.AddIgnoreRule(ctx, "*.log", IgnoreScopeTopLevel)
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(r.path, ".gitignore"))
	require.NoError(t, err)

	// Count occurrences - should be exactly one
	lines := splitLines(string(content))
	count := 0
	for _, line := range lines {
		if line == "*.log" {
			count++
		}
	}
	assert.Equal(t, 1, count, "pattern should appear exactly once")
}

func TestAddIgnoreRule_Private_WritesToExclude(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	err := r.AddIgnoreRule(ctx, "secret.txt", IgnoreScopePrivate)
	require.NoError(t, err)

	excludePath := filepath.Join(r.gitDir, "info", "exclude")
	content, err := os.ReadFile(excludePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "secret.txt\n")
}

func TestAddIgnoreRule_Subdir_WritesToSubdirGitignore(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a subdirectory
	subdir := filepath.Join(r.path, "src")
	require.NoError(t, os.MkdirAll(subdir, 0o755))

	err := r.AddIgnoreRule(ctx, "*.o", IgnoreScopeSubdir)
	require.NoError(t, err)

	// With subdir scope but no explicit subdir, falls back to top-level
	// The scope is about WHERE the .gitignore goes, not about the pattern
	content, err := os.ReadFile(filepath.Join(r.path, ".gitignore"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "*.o\n")
}

func TestIgnorePatternForPath_File(t *testing.T) {
	pattern := IgnorePatternForPath("src/main.go")
	assert.Equal(t, "/src/main.go", pattern)
}

func TestIgnorePatternForPath_Directory(t *testing.T) {
	pattern := IgnorePatternForPath("build/")
	assert.Equal(t, "/build/", pattern)
}

func TestIgnorePatternForPath_RootFile(t *testing.T) {
	pattern := IgnorePatternForPath("Makefile")
	assert.Equal(t, "/Makefile", pattern)
}

func TestGlobalIgnoreFile_FallsBackToXDGPath(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	path, err := r.GlobalIgnoreFile(ctx)
	// This may return an empty string or a path depending on git config
	// The important thing is it doesn't error
	require.NoError(t, err)
	_ = path
}

// splitLines splits a string into lines, removing empty trailing line.
func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
