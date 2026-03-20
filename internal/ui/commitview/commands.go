package commitview

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/git"
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

		// Load diffs
		diffs, err := m.repo.CommitDiff(ctx, m.commitID)
		if err != nil {
			// Non-fatal - continue without diffs
			diffs = nil
		}

		// Filter diffs if filter is specified
		if len(m.filter) > 0 && len(diffs) > 0 {
			diffs = filterDiffs(diffs, m.filter)
		}

		// Optionally verify signature (skip for now, can be controlled by config)
		// signature, _ := m.repo.VerifyCommit(ctx, m.commitID)

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
