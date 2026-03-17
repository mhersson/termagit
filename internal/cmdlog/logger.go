package cmdlog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Logger handles appending command entries to a log file with rotation.
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	writer   *bufio.Writer
	path     string
	maxBytes int64
	keep     int
	size     int64
}

// New creates a new Logger that writes to the given path.
// maxBytes controls when rotation occurs, keep controls how many rotated files to retain.
func New(path string, maxBytes int64, keep int) (*Logger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("stat log file: %w", err)
	}

	return &Logger{
		file:     f,
		writer:   bufio.NewWriter(f),
		path:     path,
		maxBytes: maxBytes,
		keep:     keep,
		size:     info.Size(),
	}, nil
}

// Append adds an entry to the log. Thread-safe. Nil-safe.
func (l *Logger) Append(e Entry) error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	data = append(data, '\n')

	n, err := l.writer.Write(data)
	if err != nil {
		return fmt.Errorf("write entry: %w", err)
	}
	l.size += int64(n)

	if l.size >= l.maxBytes {
		if err := l.rotate(); err != nil {
			return fmt.Errorf("rotate: %w", err)
		}
	}

	return nil
}

// Close flushes and closes the log file.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.writer.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return l.file.Close()
}

func (l *Logger) rotate() error {
	if err := l.writer.Flush(); err != nil {
		return err
	}
	if err := l.file.Close(); err != nil {
		return err
	}

	// Shift existing rotated files
	for i := l.keep; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", l.path, i)
		newPath := fmt.Sprintf("%s.%d", l.path, i+1)
		if i == l.keep {
			_ = os.Remove(old)
		} else {
			_ = os.Rename(old, newPath)
		}
	}

	// Move current to .1
	if err := os.Rename(l.path, l.path+".1"); err != nil {
		return err
	}

	// Open fresh file
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	l.file = f
	l.writer = bufio.NewWriter(f)
	l.size = 0
	return nil
}
