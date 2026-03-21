package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ReflogEntry represents a single entry in the reflog.
type ReflogEntry struct {
	Oid        string // Full 40-char hash
	Index      int    // 0, 1, 2...
	AuthorName string
	RefName    string // "HEAD@{0}"
	RefSubject string // Full reflog subject (e.g., "commit: add feature")
	RelDate    string // "3 hours ago"
	CommitDate string
	Type       string // "commit", "merge", "rebase", "checkout", etc.
}

// Reflog returns reflog entries for the given ref.
// If ref is empty, defaults to HEAD.
// n limits the number of entries returned.
func (r *Repository) Reflog(ctx context.Context, ref string, n int) ([]ReflogEntry, error) {
	if ref == "" {
		ref = "HEAD"
	}

	// Format: %H%x1E%aN%x1E%gd%x1E%gs%x1E%cr%x1E%cd
	// %H  = full commit hash
	// %aN = author name
	// %gd = reflog selector (e.g., HEAD@{0})
	// %gs = reflog subject (e.g., "commit: add feature")
	// %cr = committer relative date
	// %cd = committer date
	// %x1E = record separator (ASCII 30)
	format := "%H%x1E%aN%x1E%gd%x1E%gs%x1E%cr%x1E%cd%x00"

	args := []string{"reflog", "show", "--format=" + format}

	if n > 0 {
		args = append(args, "-n", strconv.Itoa(n))
	}

	args = append(args, ref, "--")

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("reflog %s: %w", ref, err)
	}

	return parseReflogOutput(out), nil
}

// parseReflogOutput parses the output of git reflog with our custom format.
func parseReflogOutput(output string) []ReflogEntry {
	if output == "" {
		return nil
	}

	var entries []ReflogEntry

	// Split by NUL character
	records := strings.Split(output, "\x00")
	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		entry := parseReflogRecord(record)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

// parseReflogRecord parses a single reflog record.
// Format: %H%x1E%aN%x1E%gd%x1E%gs%x1E%cr%x1E%cd
func parseReflogRecord(record string) *ReflogEntry {
	parts := strings.Split(record, "\x1E")
	if len(parts) < 6 {
		return nil
	}

	entry := &ReflogEntry{
		Oid:        parts[0],
		AuthorName: parts[1],
		RefName:    parts[2],
		RefSubject: parts[3],
		RelDate:    parts[4],
		CommitDate: parts[5],
		Type:       parseReflogType(parts[3]),
	}

	// Parse index from RefName (e.g., "HEAD@{0}" -> 0)
	entry.Index = parseReflogIndex(parts[2])

	return entry
}

// parseReflogIndex extracts the index number from a reflog selector.
// e.g., "HEAD@{0}" -> 0, "main@{5}" -> 5
func parseReflogIndex(refName string) int {
	start := strings.Index(refName, "@{")
	if start == -1 {
		return 0
	}
	end := strings.Index(refName[start:], "}")
	if end == -1 {
		return 0
	}
	numStr := refName[start+2 : start+end]
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}
	return n
}

// parseReflogType extracts the type from a reflog subject.
// Examples:
//   - "commit: add feature" -> "commit"
//   - "commit (initial): Initial commit" -> "commit"
//   - "commit (amend): fix typo" -> "amend"
//   - "merge origin/main: Merge..." -> "merge"
//   - "checkout: moving from..." -> "checkout"
//   - "reset: moving to HEAD~1" -> "reset"
//   - "rebase (start): checkout abc123" -> "rebase"
//   - "rebase -i (start): checkout abc123" -> "rebase"
//   - "cherry-pick: add feature" -> "cherry-pick"
//   - "revert: Revert \"add feature\"" -> "revert"
//   - "pull: Fast-forward" -> "pull"
//   - "clone: from git@..." -> "clone"
//   - "branch: Created from HEAD" -> "branch"
func parseReflogType(subject string) string {
	// Handle special cases first
	if strings.HasPrefix(subject, "commit (amend)") {
		return "amend"
	}
	if strings.HasPrefix(subject, "commit (initial)") {
		return "commit"
	}
	if strings.HasPrefix(subject, "commit") {
		return "commit"
	}

	// Handle rebase variants (e.g., "rebase -i (start)")
	if strings.HasPrefix(subject, "rebase") {
		return "rebase"
	}

	// Standard format: "type: message" or "type word: message"
	colonIdx := strings.Index(subject, ":")
	if colonIdx == -1 {
		return "other"
	}

	prefix := strings.TrimSpace(subject[:colonIdx])

	// Handle "merge origin/main" -> "merge"
	if strings.HasPrefix(prefix, "merge ") || prefix == "merge" {
		return "merge"
	}

	// Handle "checkout" -> "checkout"
	knownTypes := []string{
		"checkout", "reset", "cherry-pick", "revert",
		"pull", "clone", "branch",
	}

	for _, t := range knownTypes {
		if prefix == t || strings.HasPrefix(prefix, t+" ") {
			return t
		}
	}

	return "other"
}
