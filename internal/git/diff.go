package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DiffOp represents a diff line operation.
type DiffOp rune

const (
	DiffOpContext DiffOp = ' ' // Context line (unchanged)
	DiffOpAdd     DiffOp = '+' // Added line
	DiffOpDelete  DiffOp = '-' // Deleted line
)

// DiffLine represents a single line in a diff hunk.
type DiffLine struct {
	Op        DiffOp
	Content   string
	OldLineNo int // Line number in old file (0 if added)
	NewLineNo int // Line number in new file (0 if deleted)
}

// Hunk represents a contiguous region of changes in a diff.
type Hunk struct {
	Header   string // The @@ line
	OldStart int    // Starting line in old file
	OldCount int    // Number of lines in old file
	NewStart int    // Starting line in new file
	NewCount int    // Number of lines in new file
	Lines    []DiffLine
	Length   int // Number of rendered lines (1 header + len(Lines))
}

// DiffKind indicates what is being compared.
type DiffKind int

const (
	DiffStaged   DiffKind = iota // Staged changes (index vs HEAD)
	DiffUnstaged                 // Unstaged changes (worktree vs index)
	DiffCommit                   // Changes in a commit
	DiffRange                    // Range diff (e.g. main..feature)
	DiffStash                    // Stash diff
)

// FileDiff represents the diff for a single file.
type FileDiff struct {
	Path     string
	OldPath  string // For renames
	Hunks    []Hunk
	IsBinary bool
	IsNew    bool
	IsDelete bool
	Kind     DiffKind
}

// StagedDiff returns the diff of staged changes.
// If path is empty, returns diff for all staged files.
func (r *Repository) StagedDiff(ctx context.Context, path string) ([]FileDiff, error) {
	args := []string{"diff", "--cached"}
	if path != "" {
		args = append(args, "--", path)
	}

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseDiffOutput(out, DiffStaged), nil
}

// UnstagedDiff returns the diff of unstaged changes.
// If path is empty, returns diff for all unstaged files.
func (r *Repository) UnstagedDiff(ctx context.Context, path string) ([]FileDiff, error) {
	args := []string{"diff"}
	if path != "" {
		args = append(args, "--", path)
	}

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseDiffOutput(out, DiffUnstaged), nil
}

// CommitDiff returns the diff for a specific commit.
func (r *Repository) CommitDiff(ctx context.Context, hash string) ([]FileDiff, error) {
	out, err := r.runGit(ctx, "show", "--format=", hash)
	if err != nil {
		return nil, err
	}

	return parseDiffOutput(out, DiffCommit), nil
}

// UntrackedDiff returns the diff for an untracked file.
func (r *Repository) UntrackedDiff(ctx context.Context, path string) (*FileDiff, error) {
	// git diff --no-index exits with 1 when there are differences, which is normal
	out, _, err := r.runGitFull(ctx, "diff", "--no-index", "/dev/null", path)
	if err != nil && !strings.Contains(err.Error(), "exit status 1") {
		return nil, err
	}

	diffs := parseDiffOutput(out, DiffUnstaged)
	if len(diffs) == 0 {
		return nil, nil
	}

	fd := &diffs[0]
	fd.IsNew = true
	fd.Path = path
	return fd, nil
}

// RangeDiff returns the diff for a range specification (e.g. "main..feature").
func (r *Repository) RangeDiff(ctx context.Context, rangeSpec string) ([]FileDiff, error) {
	out, err := r.runGit(ctx, "diff", rangeSpec)
	if err != nil {
		return nil, fmt.Errorf("range diff %s: %w", rangeSpec, err)
	}

	return parseDiffOutput(out, DiffRange), nil
}

// DiffStat returns file statistics for a diff command.
// The args are passed directly to git diff (e.g. "--cached", or a range spec).
func (r *Repository) DiffStat(ctx context.Context, args ...string) (*CommitOverview, error) {
	cmdArgs := make([]string, 0, 1+len(args)+1)
	cmdArgs = append(cmdArgs, "diff")
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs, "--stat")

	out, err := r.runGit(ctx, cmdArgs...)
	if err != nil {
		return nil, fmt.Errorf("diff stat: %w", err)
	}

	return parseStat(out), nil
}

// StashDiffStat returns file statistics for a stash entry.
func (r *Repository) StashDiffStat(ctx context.Context, index int) (*CommitOverview, error) {
	ref := fmt.Sprintf("stash@{%d}", index)
	out, err := r.runGit(ctx, "stash", "show", "--stat", ref)
	if err != nil {
		return nil, fmt.Errorf("stash diff stat %s: %w", ref, err)
	}

	return parseStat(out), nil
}

// ParseDiffOutput parses raw git diff output into FileDiff structs.
func ParseDiffOutput(output string, kind DiffKind) []FileDiff {
	return parseDiffOutput(output, kind)
}

// parseDiffOutput parses the output of git diff into FileDiff structs.
func parseDiffOutput(output string, kind DiffKind) []FileDiff {
	if output == "" {
		return nil
	}

	var diffs []FileDiff

	// Split by "diff --git" headers
	parts := splitDiffOutput(output)

	for _, part := range parts {
		if part == "" {
			continue
		}

		fd := parseFileDiff(part)
		if fd != nil {
			fd.Kind = kind
			diffs = append(diffs, *fd)
		}
	}

	return diffs
}

// splitDiffOutput splits diff output by file boundaries.
func splitDiffOutput(output string) []string {
	// Split on "diff --git" but keep the marker
	parts := strings.Split(output, "diff --git ")
	var result []string
	for _, p := range parts {
		if p != "" {
			result = append(result, "diff --git "+p)
		}
	}
	return result
}

// parseFileDiff parses a single file's diff.
func parseFileDiff(diff string) *FileDiff {
	if diff == "" {
		return nil
	}

	fd := &FileDiff{}

	lines := strings.Split(diff, "\n")
	if len(lines) == 0 {
		return nil
	}

	// Parse the header line: diff --git a/path b/path
	header := lines[0]
	fd.Path = parseGitDiffPath(header)

	// Look for metadata lines
	for i, line := range lines {
		if strings.HasPrefix(line, "new file") {
			fd.IsNew = true
		} else if strings.HasPrefix(line, "deleted file") {
			fd.IsDelete = true
		} else if strings.HasPrefix(line, "rename from ") {
			fd.OldPath = strings.TrimPrefix(line, "rename from ")
		} else if strings.HasPrefix(line, "rename to ") {
			fd.Path = strings.TrimPrefix(line, "rename to ")
		} else if strings.HasPrefix(line, "Binary files") {
			fd.IsBinary = true
			return fd
		} else if strings.HasPrefix(line, "@@") {
			// Found first hunk, parse hunks from here
			fd.Hunks = parseHunks(strings.Join(lines[i:], "\n"))
			break
		}
	}

	return fd
}

// parseGitDiffPath extracts the path from "diff --git a/path b/path".
func parseGitDiffPath(header string) string {
	// Format: diff --git a/path b/path
	parts := strings.SplitN(header, " b/", 2)
	if len(parts) < 2 {
		return ""
	}
	// Remove trailing newline if present
	return strings.TrimSpace(parts[1])
}

// parseHunks parses hunk sections from diff output.
func parseHunks(diff string) []Hunk {
	if diff == "" {
		return nil
	}

	var hunks []Hunk
	var currentHunk *Hunk

	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Start of a new hunk
			if currentHunk != nil {
				currentHunk.Length = 1 + len(currentHunk.Lines)
				hunks = append(hunks, *currentHunk)
			}
			currentHunk = parseHunkHeader(line)
		} else if currentHunk != nil && len(line) > 0 {
			// Add line to current hunk
			dl := parseDiffLine(line, currentHunk)
			if dl != nil {
				currentHunk.Lines = append(currentHunk.Lines, *dl)
			}
		}
	}

	// Add final hunk
	if currentHunk != nil {
		currentHunk.Length = 1 + len(currentHunk.Lines)
		hunks = append(hunks, *currentHunk)
	}

	return hunks
}

// hunkHeaderRegex matches "@@ -old,count +new,count @@" patterns.
var hunkHeaderRegex = regexp.MustCompile(`@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseHunkHeader parses a hunk header line like "@@ -1,3 +1,4 @@".
func parseHunkHeader(line string) *Hunk {
	matches := hunkHeaderRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	hunk := &Hunk{Header: line}

	// Parse old start and count
	hunk.OldStart, _ = strconv.Atoi(matches[1])
	if matches[2] != "" {
		hunk.OldCount, _ = strconv.Atoi(matches[2])
	} else {
		hunk.OldCount = 1 // Default to 1 if not specified
	}

	// Parse new start and count
	hunk.NewStart, _ = strconv.Atoi(matches[3])
	if matches[4] != "" {
		hunk.NewCount, _ = strconv.Atoi(matches[4])
	} else {
		hunk.NewCount = 1 // Default to 1 if not specified
	}

	return hunk
}

// parseDiffLine parses a single diff line.
func parseDiffLine(line string, _ *Hunk) *DiffLine {
	if len(line) == 0 {
		return nil
	}

	op := line[0]
	content := ""
	if len(line) > 1 {
		content = line[1:]
	}

	dl := &DiffLine{Content: content}

	switch op {
	case '+':
		dl.Op = DiffOpAdd
	case '-':
		dl.Op = DiffOpDelete
	case ' ':
		dl.Op = DiffOpContext
	case '\\':
		// "\ No newline at end of file"
		return nil
	default:
		// Unknown line type, skip
		return nil
	}

	return dl
}

// HunkToPatch generates a unified diff patch from a hunk.
// This is used for staging/unstaging individual hunks.
func HunkToPatch(path string, hunk *Hunk, reverse bool) string {
	if hunk == nil {
		return ""
	}

	var sb strings.Builder

	// Write diff header
	fmt.Fprintf(&sb, "diff --git a/%s b/%s\n", path, path)
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)

	// Write hunk header
	if reverse {
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
			hunk.NewStart, hunk.NewCount, hunk.OldStart, hunk.OldCount)
	} else {
		sb.WriteString(hunk.Header)
		sb.WriteString("\n")
	}

	// Write lines
	for _, line := range hunk.Lines {
		if reverse {
			// Reverse the operation
			switch line.Op {
			case DiffOpAdd:
				sb.WriteByte('-')
			case DiffOpDelete:
				sb.WriteByte('+')
			default:
				sb.WriteByte(byte(line.Op))
			}
		} else {
			sb.WriteByte(byte(line.Op))
		}
		sb.WriteString(line.Content)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// LineRangeToPatch generates a unified diff patch from a subset of lines within
// a hunk. startLine and endLine are 0-based inclusive indices into hunk.Lines.
//
// Selection rules:
//   - Context lines are always included.
//   - Selected delete (-) lines remain as deletions.
//   - Unselected delete (-) lines are converted to context lines.
//   - Selected add (+) lines remain as additions.
//   - Unselected add (+) lines are dropped entirely.
//
// OldCount and NewCount are recomputed based on the lines that appear in the
// output. When reverse is true, the patch is inverted (delete↔add, old↔new)
// for use with unstage or discard operations.
func LineRangeToPatch(path string, hunk *Hunk, startLine, endLine int, reverse bool) string {
	if hunk == nil {
		return ""
	}

	// Build the output lines and compute counts.
	type outLine struct {
		op      byte
		content string
	}
	var lines []outLine
	oldCount := 0
	newCount := 0

	for i, dl := range hunk.Lines {
		selected := i >= startLine && i <= endLine

		switch dl.Op {
		case DiffOpContext:
			lines = append(lines, outLine{' ', dl.Content})
			oldCount++
			newCount++
		case DiffOpDelete:
			if selected {
				// Keep as deletion: exists in old, removed from new.
				lines = append(lines, outLine{'-', dl.Content})
				oldCount++
			} else {
				// Demote to context: kept in both old and new.
				lines = append(lines, outLine{' ', dl.Content})
				oldCount++
				newCount++
			}
		case DiffOpAdd:
			if selected {
				// Keep as addition: new only.
				lines = append(lines, outLine{'+', dl.Content})
				newCount++
			}
			// Unselected add: dropped entirely.
		}
	}

	var sb strings.Builder

	// Write diff header.
	fmt.Fprintf(&sb, "diff --git a/%s b/%s\n", path, path)
	fmt.Fprintf(&sb, "--- a/%s\n", path)
	fmt.Fprintf(&sb, "+++ b/%s\n", path)

	// Write hunk header with recomputed counts.
	if reverse {
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
			hunk.NewStart, newCount, hunk.OldStart, oldCount)
	} else {
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
			hunk.OldStart, oldCount, hunk.NewStart, newCount)
	}

	// Write lines, reversing ops when needed.
	for _, l := range lines {
		op := l.op
		if reverse {
			switch op {
			case '+':
				op = '-'
			case '-':
				op = '+'
			}
		}
		sb.WriteByte(op)
		sb.WriteString(l.content)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// ApplyPatch applies a unified diff patch via `git apply`.
// Extra args are passed directly (e.g. "--cached", "-R").
func (r *Repository) ApplyPatch(ctx context.Context, patch string, extraArgs ...string) error {
	args := make([]string, 0, 1+len(extraArgs))
	args = append(args, "apply")
	args = append(args, extraArgs...)
	_, err := r.runGitWithStdin(ctx, patch, args...)
	if err != nil {
		return fmt.Errorf("apply patch: %w", err)
	}
	return nil
}
