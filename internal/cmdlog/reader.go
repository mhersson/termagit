package cmdlog

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
)

// ReadRecent reads the n most recent entries from the log file, newest first.
// Combines entries from the current log and any rotated files.
func ReadRecent(path string, n int) ([]Entry, error) {
	var all []Entry

	// Read from current file
	entries, err := readFile(path)
	if err != nil {
		return nil, err
	}
	all = append(all, entries...)

	// Read from rotated files (.1, .2, etc.)
	const maxRotatedFiles = 10
	for i := 1; i <= maxRotatedFiles; i++ {
		rotated := fmt.Sprintf("%s.%d", path, i)
		entries, err := readFile(rotated)
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			break
		}
		all = append(all, entries...)
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp.After(all[j].Timestamp)
	})

	// Limit to n
	if len(all) > n {
		all = all[:n]
	}

	return all, nil
}

func readFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return entries, nil
}
