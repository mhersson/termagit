package status

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
)

// loadStatusCmd loads the HEAD state and core sections.
func loadStatusCmd(repo *git.Repository, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Load HEAD info
		branch, subject, err := repo.HeadInfo(ctx)
		if err != nil {
			return statusLoadedMsg{err: err}
		}

		oid, err := repo.HeadOID(ctx)
		if err != nil {
			return statusLoadedMsg{err: err}
		}

		head := HeadState{
			Branch:    branch,
			Oid:       oid,
			AbbrevOid: abbreviateOID(oid),
			Subject:   subject,
			Detached:  branch == "HEAD",
		}

		// Load git status
		status, err := repo.Status(ctx)
		if err != nil {
			return statusLoadedMsg{head: head, err: err}
		}

		// Build sections
		var sections []Section

		// Untracked files
		if len(status.Untracked) > 0 || !getSectionConfig(cfg, SectionUntracked).Hidden {
			items := make([]Item, len(status.Untracked))
			for i := range status.Untracked {
				items[i] = Item{Entry: &status.Untracked[i]}
			}
			sections = append(sections, Section{
				Kind:   SectionUntracked,
				Title:  "Untracked files",
				Folded: getSectionConfig(cfg, SectionUntracked).Folded,
				Hidden: getSectionConfig(cfg, SectionUntracked).Hidden,
				Items:  items,
			})
		}

		// Unstaged changes
		if len(status.Unstaged) > 0 || !getSectionConfig(cfg, SectionUnstaged).Hidden {
			items := make([]Item, len(status.Unstaged))
			for i := range status.Unstaged {
				items[i] = Item{Entry: &status.Unstaged[i]}
			}
			sections = append(sections, Section{
				Kind:   SectionUnstaged,
				Title:  "Unstaged changes",
				Folded: getSectionConfig(cfg, SectionUnstaged).Folded,
				Hidden: getSectionConfig(cfg, SectionUnstaged).Hidden,
				Items:  items,
			})
		}

		// Staged changes
		if len(status.Staged) > 0 || !getSectionConfig(cfg, SectionStaged).Hidden {
			items := make([]Item, len(status.Staged))
			for i := range status.Staged {
				items[i] = Item{Entry: &status.Staged[i]}
			}
			sections = append(sections, Section{
				Kind:   SectionStaged,
				Title:  "Staged changes",
				Folded: getSectionConfig(cfg, SectionStaged).Folded,
				Hidden: getSectionConfig(cfg, SectionStaged).Hidden,
				Items:  items,
			})
		}

		return statusLoadedMsg{
			head:     head,
			sections: sections,
		}
	}
}

// loadHunksCmd loads diff hunks for a file.
func loadHunksCmd(repo *git.Repository, sIdx, iIdx int, entry *git.StatusEntry, kind git.DiffKind) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var hunks []git.Hunk

		switch kind {
		case git.DiffStaged:
			diffs, err := repo.StagedDiff(ctx, entry.Path())
			if err != nil {
				return hunksLoadedMsg{sectionIdx: sIdx, itemIdx: iIdx, err: err}
			}
			if len(diffs) > 0 {
				hunks = diffs[0].Hunks
			}

		case git.DiffUnstaged:
			// Check if untracked
			if entry.Unstaged == git.FileStatusUntracked {
				diff, err := repo.UntrackedDiff(ctx, entry.Path())
				if err != nil {
					return hunksLoadedMsg{sectionIdx: sIdx, itemIdx: iIdx, err: err}
				}
				if diff != nil {
					hunks = diff.Hunks
				}
			} else {
				diffs, err := repo.UnstagedDiff(ctx, entry.Path())
				if err != nil {
					return hunksLoadedMsg{sectionIdx: sIdx, itemIdx: iIdx, err: err}
				}
				if len(diffs) > 0 {
					hunks = diffs[0].Hunks
				}
			}
		}

		return hunksLoadedMsg{
			sectionIdx: sIdx,
			itemIdx:    iIdx,
			hunks:      hunks,
		}
	}
}

// stageFileCmd stages a file.
//
//nolint:unused // Phase 4
func stageFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.StageFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// unstageFileCmd unstages a file.
//
//nolint:unused // Phase 4
func unstageFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.UnstageFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// discardFileCmd discards changes to a file.
//
//nolint:unused // Phase 4
func discardFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.DiscardFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// notifyCmd returns a command that sends notificationExpiredMsg after duration.
//
//nolint:unused // Phase 4
func notifyCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return notificationExpiredMsg{}
	})
}

// abbreviateOID returns the first 7 characters of an OID.
func abbreviateOID(oid string) string {
	if len(oid) >= 7 {
		return oid[:7]
	}
	return oid
}

// getSectionConfig returns the config for a section kind.
func getSectionConfig(cfg *config.Config, kind SectionKind) config.SectionConfig {
	if cfg == nil {
		return config.SectionConfig{}
	}

	switch kind {
	case SectionSequencer:
		return cfg.Sections.Sequencer
	case SectionRebase:
		return cfg.Sections.Rebase
	case SectionBisect:
		return cfg.Sections.Bisect
	case SectionUntracked:
		return cfg.Sections.Untracked
	case SectionUnstaged:
		return cfg.Sections.Unstaged
	case SectionStaged:
		return cfg.Sections.Staged
	case SectionStashes:
		return cfg.Sections.Stashes
	case SectionUnmergedUpstream:
		return cfg.Sections.UnmergedUpstream
	case SectionUnpushedPushRemote:
		return cfg.Sections.UnmergedPushRemote
	case SectionRecentCommits:
		return cfg.Sections.Recent
	case SectionUnpulledUpstream:
		return cfg.Sections.UnpulledUpstream
	case SectionUnpulledPushRemote:
		return cfg.Sections.UnpulledPushRemote
	default:
		return config.SectionConfig{}
	}
}
