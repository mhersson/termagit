package cmdlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_CreatesLogFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	_, err = os.Stat(path)
	assert.NoError(t, err, "log file should exist")
}

func TestAppend_WritesValidNDJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)

	entry := Entry{
		Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Command:    "git status",
		Dir:        "/repo",
		ExitCode:   0,
		Stdout:     "output",
		Stderr:     "",
		DurationMs: 42,
	}

	err = logger.Append(entry)
	require.NoError(t, err)
	_ = logger.Close()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded Entry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, entry.Command, decoded.Command)
	assert.Equal(t, entry.ExitCode, decoded.ExitCode)
	assert.Equal(t, entry.DurationMs, decoded.DurationMs)
}

func TestAppend_ThreadSafe_NoConcurrentRace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			entry := Entry{
				Timestamp: time.Now(),
				Command:   "cmd",
				ExitCode:  n,
			}
			_ = logger.Append(entry)
		}(i)
	}
	wg.Wait()
}

func TestAppend_NilLogger_DoesNotPanic(t *testing.T) {
	var logger *Logger
	entry := Entry{Command: "test"}

	// Should not panic
	err := logger.Append(entry)
	assert.NoError(t, err)
}

func TestAppend_RotatesWhenMaxSizeExceeded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Small max size to trigger rotation quickly
	logger, err := New(path, 100, 3)
	require.NoError(t, err)

	// Write enough data to exceed maxBytes
	for i := 0; i < 10; i++ {
		entry := Entry{
			Timestamp: time.Now(),
			Command:   "git status --long-option-to-make-it-bigger",
			Dir:       "/some/long/path/to/repository",
			ExitCode:  0,
		}
		err = logger.Append(entry)
		require.NoError(t, err)
	}
	_ = logger.Close()

	// Check that rotation file exists
	_, err = os.Stat(path + ".1")
	assert.NoError(t, err, "rotated file should exist")
}

func TestRotate_KeepsMaxCopies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Very small size, keep only 2 rotated copies
	logger, err := New(path, 50, 2)
	require.NoError(t, err)

	// Write many entries to trigger multiple rotations
	for i := 0; i < 50; i++ {
		entry := Entry{
			Timestamp: time.Now(),
			Command:   "git status --long-option",
			Dir:       "/path",
		}
		_ = logger.Append(entry)
	}
	_ = logger.Close()

	// Should have at most .1 and .2
	_, err1 := os.Stat(path + ".1")
	_, err2 := os.Stat(path + ".2")
	_, err3 := os.Stat(path + ".3")

	assert.NoError(t, err1, ".1 should exist")
	assert.NoError(t, err2, ".2 should exist")
	assert.True(t, os.IsNotExist(err3), ".3 should not exist")
}

func TestClose_FlushesAndCloses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)

	entry := Entry{Timestamp: time.Now(), Command: "test"}
	_ = logger.Append(entry)

	err = logger.Close()
	assert.NoError(t, err)

	// Should be able to read the data after close
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test")
}

func TestReadRecent_ReturnsNewestFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)

	for i := 1; i <= 5; i++ {
		entry := Entry{
			Timestamp: time.Date(2024, 1, i, 0, 0, 0, 0, time.UTC),
			Command:   "cmd",
			ExitCode:  i,
		}
		_ = logger.Append(entry)
	}
	_ = logger.Close()

	entries, err := ReadRecent(path, 10)
	require.NoError(t, err)
	require.Len(t, entries, 5)

	// Newest first means highest exit code first (day 5, 4, 3, 2, 1)
	assert.Equal(t, 5, entries[0].ExitCode)
	assert.Equal(t, 1, entries[4].ExitCode)
}

func TestReadRecent_LimitsToN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		_ = logger.Append(Entry{Timestamp: time.Now(), Command: "cmd"})
	}
	_ = logger.Close()

	entries, err := ReadRecent(path, 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestReadRecent_MissingFile_ReturnsEmpty(t *testing.T) {
	entries, err := ReadRecent("/nonexistent/path.log", 10)
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestReadRecent_CombinesCurrentAndRotated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	// Create rotated file first
	rotated := path + ".1"
	f, err := os.Create(rotated)
	require.NoError(t, err)
	oldEntry := Entry{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Command: "old", ExitCode: 1}
	data, _ := json.Marshal(oldEntry)
	_, _ = f.Write(data)
	_, _ = f.Write([]byte("\n"))
	_ = f.Close()

	// Create current log file
	logger, err := New(path, 1024*1024, 3)
	require.NoError(t, err)
	newEntry := Entry{Timestamp: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Command: "new", ExitCode: 2}
	_ = logger.Append(newEntry)
	_ = logger.Close()

	entries, err := ReadRecent(path, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Newest first
	assert.Equal(t, 2, entries[0].ExitCode)
	assert.Equal(t, 1, entries[1].ExitCode)
}
