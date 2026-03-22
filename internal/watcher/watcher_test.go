package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// msgCollector collects messages sent by the watcher.
type msgCollector struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (c *msgCollector) send(msg tea.Msg) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
}

func (c *msgCollector) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.msgs)
}

func (c *msgCollector) waitFor(t *testing.T, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		if c.count() >= n {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d messages, got %d", n, c.count())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func setupFakeGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	// Create initial files the watcher expects
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "index"), []byte("initial"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644))
	return gitDir
}

func TestWatcher_DetectsIndexModification(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watcher test in short mode")
	}

	gitDir := setupFakeGitDir(t)
	collector := &msgCollector{}

	w, err := New(gitDir)
	require.NoError(t, err)

	w.Start(collector.send)
	defer w.Stop()

	// Give watcher time to set up
	time.Sleep(100 * time.Millisecond)

	// Modify the index file
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "index"), []byte("modified"), 0o644))

	collector.waitFor(t, 1, 2*time.Second)
	assert.GreaterOrEqual(t, collector.count(), 1)

	collector.mu.Lock()
	_, ok := collector.msgs[0].(RepoChangedMsg)
	collector.mu.Unlock()
	assert.True(t, ok, "expected RepoChangedMsg")
}

func TestWatcher_DetectsHeadChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watcher test in short mode")
	}

	gitDir := setupFakeGitDir(t)
	collector := &msgCollector{}

	w, err := New(gitDir)
	require.NoError(t, err)

	w.Start(collector.send)
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)

	// Modify HEAD
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature\n"), 0o644))

	collector.waitFor(t, 1, 2*time.Second)
	assert.GreaterOrEqual(t, collector.count(), 1)
}

func TestWatcher_DetectsNewFile_MergeHeadAppears(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watcher test in short mode")
	}

	gitDir := setupFakeGitDir(t)
	collector := &msgCollector{}

	w, err := New(gitDir)
	require.NoError(t, err)

	w.Start(collector.send)
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create MERGE_HEAD (simulates starting a merge)
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "MERGE_HEAD"), []byte("abc123\n"), 0o644))

	collector.waitFor(t, 1, 2*time.Second)
	assert.GreaterOrEqual(t, collector.count(), 1)
}

func TestWatcher_Stop_CancelsWatching(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watcher test in short mode")
	}

	gitDir := setupFakeGitDir(t)
	collector := &msgCollector{}

	w, err := New(gitDir)
	require.NoError(t, err)

	w.Start(collector.send)
	w.Stop()

	// Modify after stop - should NOT generate events
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "index"), []byte("changed"), 0o644))
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 0, collector.count(), "should not receive events after Stop")
}

func TestWatcher_MissingPath_NotAnError(t *testing.T) {
	// Create a git dir without optional files (MERGE_HEAD, etc.)
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	// Only create the bare minimum
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644))

	w, err := New(gitDir)
	require.NoError(t, err, "missing optional paths should not cause error")
	w.Stop()
}

func TestWatcher_IgnoresLockFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping filesystem watcher test in short mode")
	}

	gitDir := setupFakeGitDir(t)
	collector := &msgCollector{}

	w, err := New(gitDir)
	require.NoError(t, err)

	w.Start(collector.send)
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a .lock file - should be ignored
	require.NoError(t, os.WriteFile(filepath.Join(gitDir, "index.lock"), []byte("lock"), 0o644))
	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, 0, collector.count(), "lock files should be ignored")
}
