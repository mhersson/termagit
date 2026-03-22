package diffview

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/git"
)

// loadDiffCmd returns a command that loads diff data based on the source.
func (m Model) loadDiffCmd() tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return DiffDataLoadedMsg{}
		}

		ctx := context.Background()

		var files []git.FileDiff
		var stats *git.CommitOverview
		var err error

		switch m.source.Kind {
		case git.DiffStaged:
			files, err = m.repo.StagedDiff(ctx, m.source.Path)
			if err == nil {
				stats, _ = m.repo.DiffStat(ctx, "--cached")
			}

		case git.DiffUnstaged:
			files, err = m.repo.UnstagedDiff(ctx, m.source.Path)

		case git.DiffCommit:
			files, err = m.repo.CommitDiff(ctx, m.source.Commit)
			if err == nil {
				stats, _ = m.repo.CommitOverview(ctx, m.source.Commit)
			}

		case git.DiffRange:
			files, err = m.repo.RangeDiff(ctx, m.source.Range)
			if err == nil {
				stats, _ = m.repo.DiffStat(ctx, m.source.Range)
			}

		case git.DiffStash:
			idx, ok := parseStashIndex(m.source.Stash)
			if !ok {
				return DiffDataLoadedMsg{Err: fmt.Errorf("invalid stash ref: %s", m.source.Stash)}
			}
			patch, patchErr := m.repo.StashShowPatch(ctx, idx)
			if patchErr != nil {
				return DiffDataLoadedMsg{Err: patchErr}
			}
			files = git.ParseDiffOutput(patch, git.DiffStash)
			stats, _ = m.repo.StashDiffStat(ctx, idx)
		}

		if err != nil {
			return DiffDataLoadedMsg{Err: err}
		}

		return DiffDataLoadedMsg{
			Files: files,
			Stats: stats,
		}
	}
}

// stageHunkCmd stages a single hunk.
func stageHunkCmd(repo *git.Repository, path string, hunk *git.Hunk) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		patch := git.HunkToPatch(path, hunk, false)
		err := repo.ApplyPatch(ctx, patch, "--cached")
		return HunkStagedMsg{Err: err}
	}
}

// unstageHunkCmd unstages a single hunk.
func unstageHunkCmd(repo *git.Repository, path string, hunk *git.Hunk) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		patch := git.HunkToPatch(path, hunk, true)
		err := repo.ApplyPatch(ctx, patch, "--cached")
		return HunkStagedMsg{Err: err}
	}
}

// parseStashIndex extracts the numeric index from a stash ref like "stash@{0}".
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
