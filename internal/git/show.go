package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// CommitOverview contains file statistics from git show --stat.
type CommitOverview struct {
	Summary string // "3 files changed, 12 insertions(+), 2 deletions(-)"
	Files   []CommitOverviewFile
}

// CommitOverviewFile represents a single file in the commit overview.
type CommitOverviewFile struct {
	Path       string // file path
	Changes    string // number of changes (e.g. "12")
	Insertions string // "++++" chars
	Deletions  string // "----" chars
	IsBinary   bool
}

// CommitSignature represents GPG signature verification info.
type CommitSignature struct {
	Status  string // good/bad/missing/none/unknown-error
	KeyID   string
	Summary string
}

// CommitOverview returns file statistics for a commit.
func (r *Repository) CommitOverview(ctx context.Context, hash string) (*CommitOverview, error) {
	out, err := r.runGit(ctx, "show", "--stat", "--oneline", hash)
	if err != nil {
		return nil, fmt.Errorf("commit overview %s: %w", hash, err)
	}

	return parseCommitOverview(out), nil
}

// VerifyCommit checks GPG signature of a commit.
func (r *Repository) VerifyCommit(ctx context.Context, hash string) (*CommitSignature, error) {
	// Use git verify-commit which returns exit 1 for unsigned commits
	out, _, err := r.runGitFull(ctx, "verify-commit", "--raw", hash)

	sig := &CommitSignature{Status: "none"}

	if err != nil {
		// Exit 1 with no output means unsigned commit
		if out == "" || strings.Contains(out, "error: no signature found") {
			return sig, nil
		}
		// Parse the raw output for signature info
		return parseSignatureOutput(out), nil
	}

	// Successfully verified - parse output
	if out != "" {
		return parseSignatureOutput(out), nil
	}

	return sig, nil
}

// Regex patterns for parsing git show --stat output.
var (
	// Matches file stat lines: " path | changes +++---" or " path | Bin ..."
	statFileRegex = regexp.MustCompile(`^\s*(.+?)\s+\|\s+(\d+)\s*(\+*)(\-*)`)
	// Matches binary files: " path | Bin ..."
	statBinaryRegex = regexp.MustCompile(`^\s*(.+?)\s+\|\s+(Bin.*)`)
	// Matches summary line: "N files changed, ..."
	statSummaryRegex = regexp.MustCompile(`^\s*(\d+ files? changed.*)`)
)

func parseCommitOverview(output string) *CommitOverview {
	lines := strings.Split(output, "\n")
	// Skip the first line (oneline commit info)
	if len(lines) > 1 {
		return parseStat(strings.Join(lines[1:], "\n"))
	}
	return &CommitOverview{}
}

// parseStat parses --stat output (without a header line) into a CommitOverview.
func parseStat(output string) *CommitOverview {
	overview := &CommitOverview{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		// Check for summary line
		if matches := statSummaryRegex.FindStringSubmatch(line); matches != nil {
			overview.Summary = strings.TrimSpace(matches[1])
			continue
		}

		// Check for binary file
		if matches := statBinaryRegex.FindStringSubmatch(line); matches != nil {
			overview.Files = append(overview.Files, CommitOverviewFile{
				Path:     strings.TrimSpace(matches[1]),
				Changes:  matches[2],
				IsBinary: true,
			})
			continue
		}

		// Check for regular file stat
		if matches := statFileRegex.FindStringSubmatch(line); matches != nil {
			overview.Files = append(overview.Files, CommitOverviewFile{
				Path:       strings.TrimSpace(matches[1]),
				Changes:    matches[2],
				Insertions: matches[3],
				Deletions:  matches[4],
			})
		}
	}

	return overview
}

func parseSignatureOutput(output string) *CommitSignature {
	sig := &CommitSignature{Status: "none"}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[GNUPG:] GOODSIG") {
			sig.Status = "good"
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 3 {
				sig.KeyID = parts[2]
			}
			if len(parts) >= 4 {
				sig.Summary = parts[3]
			}
		} else if strings.HasPrefix(line, "[GNUPG:] BADSIG") {
			sig.Status = "bad"
			parts := strings.SplitN(line, " ", 4)
			if len(parts) >= 3 {
				sig.KeyID = parts[2]
			}
		} else if strings.HasPrefix(line, "[GNUPG:] ERRSIG") {
			sig.Status = "unknown-error"
		} else if strings.HasPrefix(line, "[GNUPG:] NO_PUBKEY") {
			sig.Status = "missing"
			parts := strings.SplitN(line, " ", 3)
			if len(parts) >= 3 {
				sig.KeyID = parts[2]
			}
		}
	}

	return sig
}
