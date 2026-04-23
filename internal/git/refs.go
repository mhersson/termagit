package git

import (
	"context"
	"fmt"
	"strings"
)

// RefType classifies a git reference.
type RefType string

const (
	RefTypeLocalBranch  RefType = "local_branch"
	RefTypeRemoteBranch RefType = "remote_branch"
	RefTypeTag          RefType = "tag"
)

// RefEntry represents a single git reference with metadata.
type RefEntry struct {
	Name            string  // Short name (e.g., "main", "feature/x", "v1.0")
	UnambiguousName string  // Full ref path (e.g., "main", "origin/main", "tags/v1.0")
	Type            RefType // local_branch, remote_branch, tag
	Remote          string  // Remote name for remote branches (e.g., "origin")
	Head            bool    // true if this is the current HEAD
	Oid             string  // Full commit hash
	AbbrevOid       string  // Abbreviated hash (7 chars)
	Subject         string  // Tip commit subject line
	UpstreamName    string  // Upstream ref name (for local branches with tracking)
	UpstreamStatus  string  // "" | "+" | "-" | "<>" | "=" | "<" | ">"
}

// RefsResult holds all refs grouped by type.
type RefsResult struct {
	LocalBranches  []RefEntry
	RemoteBranches map[string][]RefEntry // keyed by remote name
	Tags           []RefEntry
}

// ListRefs returns all refs (local branches, remote branches, tags)
// with upstream information for local branches.
func (r *Repository) ListRefs(ctx context.Context) (*RefsResult, error) {
	// Use tab as field delimiter. Each record is one line.
	format := "%(HEAD)\t%(objectname)\t%(objectname:short)\t%(refname)\t%(refname:short)\t%(upstream:trackshort)\t%(upstream:short)\t%(subject)"

	out, err := r.runGit(ctx, "for-each-ref", "--format="+format)
	if err != nil {
		return nil, fmt.Errorf("list refs: %w", err)
	}

	result := &RefsResult{
		RemoteBranches: make(map[string][]RefEntry),
	}

	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.SplitN(line, "\t", 8)
		if len(fields) < 8 {
			continue
		}

		entry := RefEntry{
			Head:           fields[0] == "*",
			Oid:            fields[1],
			AbbrevOid:      fields[2],
			UpstreamStatus: fields[5],
			UpstreamName:   fields[6],
			Subject:        fields[7],
		}

		refName := fields[3]   // e.g., "refs/heads/main"
		shortName := fields[4] // e.g., "main"

		switch {
		case strings.HasPrefix(refName, "refs/heads/"):
			entry.Type = RefTypeLocalBranch
			entry.Name = shortName
			entry.UnambiguousName = shortName
			result.LocalBranches = append(result.LocalBranches, entry)

		case strings.HasPrefix(refName, "refs/remotes/"):
			entry.Type = RefTypeRemoteBranch
			// Extract remote name and branch from refs/remotes/<remote>/<branch>
			rest := strings.TrimPrefix(refName, "refs/remotes/")
			remote, branch, ok := strings.Cut(rest, "/")
			if !ok {
				continue // malformed
			}
			entry.Remote = remote
			entry.Name = branch
			entry.UnambiguousName = remote + "/" + branch
			result.RemoteBranches[remote] = append(result.RemoteBranches[remote], entry)

		case strings.HasPrefix(refName, "refs/tags/"):
			entry.Type = RefTypeTag
			entry.Name = shortName
			entry.UnambiguousName = "tags/" + shortName
			result.Tags = append(result.Tags, entry)
		}
	}

	return result, nil
}
