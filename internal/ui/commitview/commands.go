package commitview

import (
	"context"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/git"
)

// loadCommitDataCmd returns a command that loads commit data.
func (m Model) loadCommitDataCmd() tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return CommitDataLoadedMsg{Err: nil}
		}

		ctx := context.Background()

		// Load commit info
		info, err := m.repo.CommitDetail(ctx, m.commitID)
		if err != nil {
			return CommitDataLoadedMsg{Err: err}
		}

		// Load file overview
		overview, err := m.repo.CommitOverview(ctx, m.commitID)
		if err != nil {
			// Non-fatal - continue without overview
			overview = nil
		}

		// Load diffs — stash refs need special handling
		var diffs []git.FileDiff
		if idx, ok := parseStashIndex(m.commitID); ok {
			patch, patchErr := m.repo.StashShowPatch(ctx, idx)
			if patchErr == nil {
				diffs = git.ParseDiffOutput(patch, git.DiffCommit)
			}
		} else {
			diffs, _ = m.repo.CommitDiff(ctx, m.commitID)
		}

		// Filter diffs if filter is specified
		if len(m.filter) > 0 && len(diffs) > 0 {
			diffs = filterDiffs(diffs, m.filter)
		}

		return CommitDataLoadedMsg{
			Info:     info,
			Overview: overview,
			Diffs:    diffs,
		}
	}
}

// filterDiffs filters diffs to only include matching paths.
func filterDiffs(diffs []git.FileDiff, filter []string) []git.FileDiff {
	if len(filter) == 0 {
		return diffs
	}

	filterSet := make(map[string]bool)
	for _, f := range filter {
		filterSet[f] = true
	}

	var result []git.FileDiff
	for _, d := range diffs {
		if filterSet[d.Path] {
			result = append(result, d)
		}
	}
	return result
}

// parseStashIndex extracts the numeric index from a stash ref like "stash@{0}".
// Returns the index and true if the ref is a stash ref, or 0 and false otherwise.
func parseStashIndex(ref string) (int, bool) {
	if !strings.HasPrefix(ref, "stash@{") || !strings.HasSuffix(ref, "}") {
		return 0, false
	}
	idxStr := ref[7 : len(ref)-1]
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return 0, false
	}
	return idx, true
}
