package watcher

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

// RepoChangedMsg is sent when the watcher detects a change in the git directory.
type RepoChangedMsg struct{}

// Watcher monitors the .git directory for changes and sends RepoChangedMsg.
type Watcher struct {
	gitDir  string
	send    func(tea.Msg)
	watcher *fsnotify.Watcher
	done    chan struct{}
	once    sync.Once
}

// watchedFiles are the specific files inside .git/ that we monitor.
var watchedFiles = []string{
	"index",
	"HEAD",
	"MERGE_HEAD",
	"CHERRY_PICK_HEAD",
	"REVERT_HEAD",
	"BISECT_START",
}

// ignoreSuffixes are file suffixes that should be ignored.
var ignoreSuffixes = []string{".lock", "~"}

// New creates a new Watcher for the given .git directory.
// Missing optional paths (MERGE_HEAD, etc.) are silently skipped.
func New(gitDir string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Watch the .git directory itself (catches new file creation like MERGE_HEAD)
	if err := fw.Add(gitDir); err != nil {
		_ = fw.Close()
		return nil, err
	}

	// Watch subdirectories that may appear during rebase
	for _, sub := range []string{"rebase-merge", "rebase-apply"} {
		p := filepath.Join(gitDir, sub)
		// Ignore errors - directories may not exist
		_ = fw.Add(p)
	}

	return &Watcher{
		gitDir:  gitDir,
		watcher: fw,
		done:    make(chan struct{}),
	}, nil
}

// Start begins watching. send is called with RepoChangedMsg on relevant changes.
func (w *Watcher) Start(send func(tea.Msg)) {
	w.send = send
	go w.loop()
}

// Stop terminates watching and blocks until the goroutine exits.
func (w *Watcher) Stop() {
	w.once.Do(func() {
		close(w.done)
		_ = w.watcher.Close()
	})
}

// loop is the main event loop that reads fsnotify events and debounces them.
func (w *Watcher) loop() {
	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-w.done:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			if w.shouldIgnore(event.Name) {
				continue
			}

			if !w.isRelevant(event.Name) {
				continue
			}

			// Debounce: reset timer on each relevant event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.NewTimer(200 * time.Millisecond)
			debounceCh = debounceTimer.C

		case <-debounceCh:
			debounceCh = nil
			if w.send != nil {
				w.send(RepoChangedMsg{})
			}

		case _, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			// Silently ignore watcher errors
		}
	}
}

// shouldIgnore returns true if the file should be ignored (lock files, temp files).
func (w *Watcher) shouldIgnore(path string) bool {
	name := filepath.Base(path)
	for _, suffix := range ignoreSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// isRelevant returns true if the changed file is one we care about.
func (w *Watcher) isRelevant(path string) bool {
	name := filepath.Base(path)

	// Check watched files
	for _, f := range watchedFiles {
		if name == f {
			return true
		}
	}

	// Check if it's inside rebase-merge/ or rebase-apply/
	rel, err := filepath.Rel(w.gitDir, path)
	if err != nil {
		return false
	}
	if strings.HasPrefix(rel, "rebase-merge") || strings.HasPrefix(rel, "rebase-apply") {
		return true
	}

	return false
}
