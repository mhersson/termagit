package git

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RefKind indicates the type of a git reference.
type RefKind int

const (
	RefKindLocal  RefKind = iota // Local branch
	RefKindRemote                // Remote tracking branch
	RefKindTag                   // Tag
	RefKindHead                  // HEAD pointer
)

// Ref represents a git reference (branch, tag, etc).
type Ref struct {
	Name   string  // Reference name (e.g., "main", "v1.0.0")
	Kind   RefKind // Type of reference
	Remote string  // Remote name for remote refs (e.g., "origin")
}

// LogEntry represents a commit in the log.
type LogEntry struct {
	Hash            string // Full 40-char commit hash
	AbbreviatedHash string // Short hash (usually 7 chars)
	Subject         string // First line of commit message
	Body            string // Rest of commit message
	AuthorName      string
	AuthorEmail     string
	AuthorDate      string // ISO 8601 format
	CommitterName   string
	CommitterEmail  string
	CommitterDate   string
	Refs            []Ref    // Branches/tags pointing at this commit
	When            time.Time // Parsed from AuthorDate
	RefName         string    // Raw decoration string from git
	ParentHashes    string    // Space-separated parent commit hashes
}

// LogOpts controls what commits are returned by Log.
type LogOpts struct {
	MaxCount    int    // Maximum commits to return (0 = unlimited)
	Offset      int    // Number of commits to skip
	Path        string // Filter by file path
	Author      string // Filter by author
	Grep        string // Filter by commit message
	Since       string // Filter commits after this date
	Until       string // Filter commits before this date
	NoMerges    bool   // Exclude merge commits
	FirstParent bool   // Follow only first parent
	Reverse     bool   // Oldest first
	All         bool   // All branches
	Decorate    bool   // Include refs
	Graph       bool   // Include ASCII graph
	Branch      string // Specific branch to show
}

// Log returns commits matching the given options.
// Returns (entries, hasMore, error).
func (r *Repository) Log(ctx context.Context, opts LogOpts) ([]LogEntry, bool, error) {
	// Build git log command
	// Format: %H|%h|%P|%s|%an|%ae|%aI|%cn|%ce|%cI|%d
	format := "%H|%h|%P|%s|%an|%ae|%aI|%cn|%ce|%cI|%d%x00"

	args := []string{"log", "--format=" + format}

	// Request one extra to detect if there are more
	maxCount := opts.MaxCount
	if maxCount > 0 {
		args = append(args, fmt.Sprintf("-n%d", maxCount+1))
	}

	if opts.Offset > 0 {
		args = append(args, fmt.Sprintf("--skip=%d", opts.Offset))
	}

	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}

	if opts.Grep != "" {
		args = append(args, "--grep="+opts.Grep)
	}

	if opts.Since != "" {
		args = append(args, "--since="+opts.Since)
	}

	if opts.Until != "" {
		args = append(args, "--until="+opts.Until)
	}

	if opts.NoMerges {
		args = append(args, "--no-merges")
	}

	if opts.FirstParent {
		args = append(args, "--first-parent")
	}

	if opts.Reverse {
		args = append(args, "--reverse")
	}

	if opts.All {
		args = append(args, "--all")
	}

	if opts.Decorate {
		args = append(args, "--decorate")
	}

	// Note: --graph is NOT passed to git because it prepends graph
	// characters (containing "|") to each line, breaking the "|"-delimited
	// format parser. The Graph flag is preserved in LogOpts for future
	// internal graph rendering in the UI (like Neogit does).

	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}

	if opts.Path != "" {
		args = append(args, "--", opts.Path)
	}

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, false, err
	}

	// Get remotes for ref parsing
	remotes, _ := r.listRemotes(ctx)

	entries := parseLogOutput(out, remotes)

	// Check if there are more
	hasMore := false
	if maxCount > 0 && len(entries) > maxCount {
		hasMore = true
		entries = entries[:maxCount]
	}

	return entries, hasMore, nil
}

// CommitDetail returns full information about a single commit.
func (r *Repository) CommitDetail(ctx context.Context, hash string) (*LogEntry, error) {
	// Format with body: %H|%h|%P|%s|%an|%ae|%aI|%cn|%ce|%cI|%d|%B
	format := "%H|%h|%P|%s|%an|%ae|%aI|%cn|%ce|%cI|%d%x00%B"

	out, err := r.runGit(ctx, "log", "-1", "--format="+format, hash)
	if err != nil {
		return nil, err
	}

	remotes, _ := r.listRemotes(ctx)
	entries := parseLogOutputWithBody(out, remotes)
	if len(entries) == 0 {
		return nil, fmt.Errorf("commit not found: %s", hash)
	}

	return &entries[0], nil
}

// RecentCommits returns the N most recent commits.
func (r *Repository) RecentCommits(ctx context.Context, n int) ([]LogEntry, error) {
	entries, _, err := r.Log(ctx, LogOpts{MaxCount: n})
	return entries, err
}

// LogAhead returns commits in ref..HEAD (commits ahead of ref).
func (r *Repository) LogAhead(ctx context.Context, ref string, max int) ([]LogEntry, error) {
	args := []string{"log", "--format=%H|%h|%s|%an|%ae|%aI%x00", ref + "..HEAD"}
	if max > 0 {
		args = append(args, fmt.Sprintf("-n%d", max))
	}

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseLogOutput(out, nil), nil
}

// LogBehind returns commits in HEAD..ref (commits behind HEAD).
func (r *Repository) LogBehind(ctx context.Context, ref string, max int) ([]LogEntry, error) {
	args := []string{"log", "--format=%H|%h|%s|%an|%ae|%aI%x00", "HEAD.." + ref}
	if max > 0 {
		args = append(args, fmt.Sprintf("-n%d", max))
	}

	out, err := r.runGit(ctx, args...)
	if err != nil {
		return nil, err
	}

	return parseLogOutput(out, nil), nil
}

// RefCommitInfo returns the full OID and subject of the commit at the tip of a ref.
func (r *Repository) RefCommitInfo(ctx context.Context, ref string) (oid, subject string, err error) {
	out, err := r.runGit(ctx, "log", "-1", "--format=%H|%s", ref)
	if err != nil {
		return "", "", fmt.Errorf("ref commit info %s: %w", ref, err)
	}

	line := strings.TrimSpace(out)
	if line == "" {
		return "", "", fmt.Errorf("no commit found for ref %s", ref)
	}

	parts := strings.SplitN(line, "|", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("unexpected format for ref %s: %s", ref, line)
	}

	return parts[0], parts[1], nil
}

// CommitMessage returns just the subject line of a commit.
func (r *Repository) CommitMessage(ctx context.Context, hash string) (string, error) {
	out, err := r.runGit(ctx, "log", "-1", "--format=%s", hash)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HeadCommitMessage returns the full commit message (subject + body) of HEAD.
func (r *Repository) HeadCommitMessage(ctx context.Context) (string, error) {
	msg, _, err := r.logOp(ctx, "git log -1 --format=%B HEAD", func() (string, string, error) {
		head, err := r.raw.Head()
		if err != nil {
			return "", "", fmt.Errorf("get HEAD: %w", err)
		}

		commit, err := r.raw.CommitObject(head.Hash())
		if err != nil {
			return "", "", fmt.Errorf("get commit: %w", err)
		}

		return strings.TrimSpace(commit.Message), "", nil
	})
	return msg, err
}

// listRemotes returns the list of remote names.
func (r *Repository) listRemotes(ctx context.Context) ([]string, error) {
	out, err := r.runGit(ctx, "remote")
	if err != nil {
		return nil, err
	}

	var remotes []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			remotes = append(remotes, line)
		}
	}
	return remotes, nil
}

// parseLogOutput parses the output of git log with our custom format.
func parseLogOutput(output string, remotes []string) []LogEntry {
	if output == "" {
		return nil
	}

	var entries []LogEntry

	// Split by NUL character
	records := strings.Split(output, "\x00")
	for _, record := range records {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}

		entry := parseLogRecord(record, remotes)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

// parseLogOutputWithBody parses log output that includes the body.
func parseLogOutputWithBody(output string, remotes []string) []LogEntry {
	if output == "" {
		return nil
	}

	var entries []LogEntry

	// Split by NUL character (between header and body)
	parts := strings.SplitN(output, "\x00", 2)
	if len(parts) == 0 {
		return nil
	}

	entry := parseLogRecord(parts[0], remotes)
	if entry != nil {
		if len(parts) > 1 {
			entry.Body = strings.TrimSpace(parts[1])
		}
		entries = append(entries, *entry)
	}

	return entries
}

// parseLogRecord parses a single log record.
// Format: %H|%h|%P|%s|%an|%ae|%aI|%cn|%ce|%cI|%d
func parseLogRecord(record string, remotes []string) *LogEntry {
	parts := strings.Split(record, "|")
	if len(parts) < 7 {
		return nil
	}

	entry := &LogEntry{
		Hash:            parts[0],
		AbbreviatedHash: parts[1],
		ParentHashes:    parts[2],
		Subject:         parts[3],
		AuthorName:      parts[4],
		AuthorEmail:     parts[5],
		AuthorDate:      parts[6],
	}

	// Parse When from AuthorDate (ISO 8601 / RFC3339)
	if t, err := time.Parse(time.RFC3339, parts[6]); err == nil {
		entry.When = t
	}

	// Optional fields
	if len(parts) > 7 {
		entry.CommitterName = parts[7]
	}
	if len(parts) > 8 {
		entry.CommitterEmail = parts[8]
	}
	if len(parts) > 9 {
		entry.CommitterDate = parts[9]
	}
	if len(parts) > 10 {
		decoration := strings.TrimSpace(parts[10])
		entry.RefName = decoration
		// Remove parentheses from decoration
		decoration = strings.TrimPrefix(decoration, "(")
		decoration = strings.TrimSuffix(decoration, ")")
		entry.Refs = parseRefs(decoration, remotes)
	}

	return entry
}

// parseRefs parses the decoration string (e.g., "HEAD -> main, origin/main, tag: v1.0.0").
func parseRefs(decoration string, remotes []string) []Ref {
	if decoration == "" {
		return nil
	}

	var refs []Ref

	// Split by comma
	parts := strings.Split(decoration, ", ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for "HEAD -> branch"
		if strings.HasPrefix(part, "HEAD") {
			refs = append(refs, Ref{Name: "HEAD", Kind: RefKindHead})
			// Check if there's a branch reference
			if idx := strings.Index(part, "-> "); idx != -1 {
				branchName := part[idx+3:]
				refs = append(refs, Ref{Name: branchName, Kind: RefKindLocal})
			}
			continue
		}

		// Check for tag
		if strings.HasPrefix(part, "tag: ") {
			tagName := strings.TrimPrefix(part, "tag: ")
			refs = append(refs, Ref{Name: tagName, Kind: RefKindTag})
			continue
		}

		// Check for remote branch
		isRemote := false
		for _, remote := range remotes {
			prefix := remote + "/"
			if strings.HasPrefix(part, prefix) {
				branchName := strings.TrimPrefix(part, prefix)
				refs = append(refs, Ref{Name: branchName, Kind: RefKindRemote, Remote: remote})
				isRemote = true
				break
			}
		}

		if !isRemote {
			// Local branch
			refs = append(refs, Ref{Name: part, Kind: RefKindLocal})
		}
	}

	return refs
}
