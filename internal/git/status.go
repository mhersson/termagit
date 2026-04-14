package git

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
)

// FileStatus represents the status of a file in the index or worktree.
type FileStatus rune

const (
	FileStatusNone      FileStatus = ' '
	FileStatusModified  FileStatus = 'M'
	FileStatusAdded     FileStatus = 'A'
	FileStatusDeleted   FileStatus = 'D'
	FileStatusRenamed   FileStatus = 'R'
	FileStatusCopied    FileStatus = 'C'
	FileStatusUpdated   FileStatus = 'U'
	FileStatusUntracked FileStatus = '?'
	FileStatusIgnored   FileStatus = '!'
	FileStatusNew       FileStatus = 'N' // New file (Neogit uses N for newly added)
	FileStatusChanged   FileStatus = 'T' // Type changed
)

// ModeText maps file status codes to their display text.
// These MUST match Neogit's config.lua exactly (lines 483-500).
var ModeText = map[string]string{
	"M":  "modified",
	"N":  "new file",
	"A":  "added",
	"D":  "deleted",
	"C":  "copied",
	"U":  "updated",
	"R":  "renamed",
	"T":  "changed",
	"DD": "unmerged",
	"AU": "unmerged",
	"UD": "unmerged",
	"UA": "unmerged",
	"DU": "unmerged",
	"AA": "unmerged",
	"UU": "unmerged",
	"?":  "",
}

// SubmoduleStatus describes the state of a submodule.
type SubmoduleStatus struct {
	CommitChanged       bool // C - commit changed
	HasTrackedChanges   bool // M - has tracked changes
	HasUntrackedChanges bool // U - has untracked changes
}

// FileModeChange holds file mode information from git status.
type FileModeChange struct {
	Head     string // mode in HEAD
	Index    string // mode in index
	Worktree string // mode in worktree
}

// StatusEntry represents a single file in git status output.
type StatusEntry struct {
	Path         string
	OrigPath     string // for renames/copies
	Staged       FileStatus
	Unstaged     FileStatus
	UnmergedMode string           // e.g. "UU", "AA" for unmerged files
	FileMode     *FileModeChange  // may be nil
	Submodule    *SubmoduleStatus // may be nil for non-submodules
}

// NewStatusEntry creates a StatusEntry with the given path and statuses.
// This is primarily useful for testing.
func NewStatusEntry(path string, staged, unstaged FileStatus) StatusEntry {
	return StatusEntry{
		Path:     path,
		Staged:   staged,
		Unstaged: unstaged,
	}
}

// StatusResult holds the parsed result of git status.
type StatusResult struct {
	Untracked []StatusEntry
	Unstaged  []StatusEntry
	Staged    []StatusEntry
}

// Status returns the current repository status.
// Uses git status --porcelain=2 for accurate results.
func (r *Repository) Status(ctx context.Context) (*StatusResult, error) {
	// Try using go-git first for in-memory repos
	wt, err := r.raw.Worktree()
	if err != nil {
		return nil, fmt.Errorf("get worktree: %w", err)
	}

	// Check if this is an in-memory repo (no path on disk)
	// by checking if the path is "/" (our sentinel for in-memory)
	if r.path == "/" {
		return r.statusFromGoGit(wt)
	}

	// For on-disk repos, use git status --porcelain=2 for accuracy
	return r.statusFromGit(ctx)
}

// statusFromGoGit uses go-git's Status API for in-memory repos.
func (r *Repository) statusFromGoGit(wt *gogit.Worktree) (*StatusResult, error) {
	status, err := wt.Status()
	if err != nil {
		return nil, fmt.Errorf("get status: %w", err)
	}

	result := &StatusResult{}

	for path, fileStatus := range status {
		entry := StatusEntry{Path: path}

		// Parse staging status
		switch fileStatus.Staging {
		case gogit.Added:
			entry.Staged = FileStatusNew
		case gogit.Modified:
			entry.Staged = FileStatusModified
		case gogit.Deleted:
			entry.Staged = FileStatusDeleted
		case gogit.Renamed:
			entry.Staged = FileStatusRenamed
		case gogit.Copied:
			entry.Staged = FileStatusCopied
		default:
			entry.Staged = FileStatusNone
		}

		// Parse worktree status
		switch fileStatus.Worktree {
		case gogit.Untracked:
			entry.Unstaged = FileStatusUntracked
		case gogit.Modified:
			entry.Unstaged = FileStatusModified
		case gogit.Deleted:
			entry.Unstaged = FileStatusDeleted
		default:
			entry.Unstaged = FileStatusNone
		}

		// Categorize the entry
		if entry.Unstaged == FileStatusUntracked {
			result.Untracked = append(result.Untracked, entry)
		} else if entry.Unstaged != FileStatusNone {
			result.Unstaged = append(result.Unstaged, entry)
		}

		if entry.Staged != FileStatusNone {
			result.Staged = append(result.Staged, entry)
		}
	}

	return result, nil
}

// statusFromGit uses git status --porcelain=2 for on-disk repos.
func (r *Repository) statusFromGit(ctx context.Context) (*StatusResult, error) {
	out, err := r.runGit(ctx, "status", "--porcelain=2")
	if err != nil {
		return nil, err
	}

	result := &StatusResult{}

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		entry, err := parsePorcelainLine(line)
		if err != nil {
			continue // Skip unparseable lines
		}
		if entry == nil {
			continue
		}

		// Categorize the entry
		if entry.Unstaged == FileStatusUntracked {
			result.Untracked = append(result.Untracked, *entry)
		} else if entry.Unstaged != FileStatusNone || entry.UnmergedMode != "" {
			result.Unstaged = append(result.Unstaged, *entry)
		}

		if entry.Staged != FileStatusNone {
			result.Staged = append(result.Staged, *entry)
		}
	}

	return result, nil
}

// parseSubmoduleStatus parses the 4-character submodule status field.
// Format: "N..." for non-submodule, "S<c><m><u>" for submodule.
func parseSubmoduleStatus(status string) *SubmoduleStatus {
	if len(status) < 4 {
		return nil
	}
	if status[0] == 'N' {
		return nil
	}
	return &SubmoduleStatus{
		CommitChanged:       status[1] == 'C',
		HasTrackedChanges:   status[2] == 'M',
		HasUntrackedChanges: status[3] == 'U',
	}
}

// parsePorcelainLine parses a single line from git status --porcelain=2 output.
// Returns nil, nil for ignored/header lines.
func parsePorcelainLine(line string) (*StatusEntry, error) {
	if len(line) < 2 {
		return nil, nil
	}

	kind := line[0]
	rest := line[2:] // Skip kind and space

	switch kind {
	case '1': // Ordinary change
		return parseKind1(rest)
	case '2': // Renamed/copied
		return parseKind2(rest)
	case 'u': // Unmerged
		return parseKindU(rest)
	case '?': // Untracked
		return &StatusEntry{
			Path:     rest,
			Staged:   FileStatusNone,
			Unstaged: FileStatusUntracked,
		}, nil
	case '!': // Ignored
		return nil, nil
	case '#': // Header
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown porcelain kind: %c", kind)
	}
}

// parseKind1 parses an ordinary change line.
// Format: <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
func parseKind1(rest string) (*StatusEntry, error) {
	parts := strings.Fields(rest)
	if len(parts) < 8 {
		return nil, fmt.Errorf("invalid kind 1 line: %s", rest)
	}

	xy := parts[0]
	sub := parts[1]
	mH := parts[2]
	mI := parts[3]
	mW := parts[4]
	hH := parts[5]
	// hI := parts[6] // not used directly
	path := strings.Join(parts[7:], " ")

	entry := &StatusEntry{
		Path: path,
		FileMode: &FileModeChange{
			Head:     mH,
			Index:    mI,
			Worktree: mW,
		},
		Submodule: parseSubmoduleStatus(sub),
	}

	// Parse XY status
	if len(xy) >= 1 {
		entry.Staged = parseFileStatusChar(xy[0], hH)
	}
	if len(xy) >= 2 {
		entry.Unstaged = parseFileStatusChar(xy[1], "")
	}

	return entry, nil
}

// parseKind2 parses a renamed/copied line.
// Format: <XY> <sub> <mH> <mI> <mW> <hH> <hI> <score> <path><TAB><origPath>
// Note: The path is the last space-separated field before the tab.
func parseKind2(rest string) (*StatusEntry, error) {
	// Find the tab separator between path and origPath
	tabIdx := strings.Index(rest, "\t")
	if tabIdx == -1 {
		return nil, fmt.Errorf("invalid kind 2 line, no tab: %s", rest)
	}

	beforeTab := rest[:tabIdx]
	origPath := rest[tabIdx+1:]

	// Parse the fields before the tab (space-separated)
	parts := strings.Fields(beforeTab)
	if len(parts) < 9 {
		return nil, fmt.Errorf("invalid kind 2 line fields (got %d, need 9): %s", len(parts), rest)
	}

	xy := parts[0]
	sub := parts[1]
	mH := parts[2]
	mI := parts[3]
	mW := parts[4]
	// hH := parts[5]
	// hI := parts[6]
	// score := parts[7]
	path := strings.Join(parts[8:], " ")

	entry := &StatusEntry{
		Path:     path,
		OrigPath: origPath,
		FileMode: &FileModeChange{
			Head:     mH,
			Index:    mI,
			Worktree: mW,
		},
		Submodule: parseSubmoduleStatus(sub),
	}

	// Parse XY status
	if len(xy) >= 1 {
		entry.Staged = parseFileStatusChar(xy[0], "")
	}
	if len(xy) >= 2 {
		entry.Unstaged = parseFileStatusChar(xy[1], "")
	}

	return entry, nil
}

// parseKindU parses an unmerged line.
// Format: <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
func parseKindU(rest string) (*StatusEntry, error) {
	parts := strings.Fields(rest)
	if len(parts) < 10 {
		return nil, fmt.Errorf("invalid kind u line: %s", rest)
	}

	xy := parts[0]
	// sub := parts[1]
	// Modes and hashes: parts[2:9]
	path := strings.Join(parts[9:], " ")

	return &StatusEntry{
		Path:         path,
		Staged:       FileStatusNone,
		UnmergedMode: xy,
		Unstaged:     FileStatusUpdated,
	}, nil
}

// parseFileStatusChar converts a single status character to FileStatus.
// hH is the HEAD hash, used to detect new files (all zeros = new file).
func parseFileStatusChar(c byte, hH string) FileStatus {
	switch c {
	case 'M':
		return FileStatusModified
	case 'A':
		// Check if this is actually a new file
		if hH != "" && isZeroHash(hH) {
			return FileStatusNew
		}
		return FileStatusNew // Default to New for Added
	case 'D':
		return FileStatusDeleted
	case 'R':
		return FileStatusRenamed
	case 'C':
		return FileStatusCopied
	case 'U':
		return FileStatusUpdated
	case 'T':
		return FileStatusChanged
	case '?':
		return FileStatusUntracked
	case '!':
		return FileStatusIgnored
	case '.', ' ':
		return FileStatusNone
	default:
		return FileStatusNone
	}
}

// isZeroHash returns true if the hash is all zeros (indicating a new file).
func isZeroHash(h string) bool {
	for _, c := range h {
		if c != '0' {
			return false
		}
	}
	return len(h) > 0
}
