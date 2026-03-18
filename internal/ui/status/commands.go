package status

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
)

// loadStatusCmd loads the HEAD state and all 12 sections.
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

		// Build sections in Neogit order
		var sections []Section

		// 1. Merge in progress (shown as sequencer)
		if repo.MergeInProgress() {
			mergeHead, mergeSubject, mergeBranch, _ := repo.ReadMergeState()
			if mergeHead != "" {
				sec := buildMergeSection(cfg, mergeBranch, mergeSubject)
				if sec != nil {
					sections = append(sections, *sec)
				}
			}
		}

		// 2. Rebase in progress
		if repo.RebaseInProgress() {
			state, err := repo.ReadRebaseTodo()
			if err == nil {
				sec := buildRebaseSection(cfg, state)
				if sec != nil {
					sections = append(sections, *sec)
				}
			}
		}

		// 3. Cherry-pick or Revert in progress (sequencer)
		if repo.CherryPickInProgress() || repo.RevertInProgress() {
			state, err := repo.SequencerState(ctx)
			if err == nil && len(state.Items) > 0 {
				sec := buildSequencerSection(cfg, state)
				if sec != nil {
					sections = append(sections, *sec)
				}
			}
		}

		// 4. Bisect in progress
		if repo.BisectInProgress() {
			state, err := repo.BisectState(ctx)
			if err == nil && len(state.Items) > 0 {
				sec := buildBisectSection(cfg, state)
				if sec != nil {
					sections = append(sections, *sec)
				}
			}
		}

		// 5. Untracked files
		if len(status.Untracked) > 0 && !getSectionConfig(cfg, SectionUntracked).Hidden {
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

		// 6. Unstaged changes
		if len(status.Unstaged) > 0 && !getSectionConfig(cfg, SectionUnstaged).Hidden {
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

		// 7. Staged changes
		if len(status.Staged) > 0 && !getSectionConfig(cfg, SectionStaged).Hidden {
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

		// 8. Stashes
		stashes, _ := repo.ListStashes(ctx)
		if len(stashes) > 0 && !getSectionConfig(cfg, SectionStashes).Hidden {
			items := make([]Item, len(stashes))
			for i := range stashes {
				items[i] = Item{Stash: &stashes[i]}
			}
			sections = append(sections, Section{
				Kind:   SectionStashes,
				Title:  "Stashes",
				Folded: getSectionConfig(cfg, SectionStashes).Folded,
				Hidden: getSectionConfig(cfg, SectionStashes).Hidden,
				Items:  items,
			})
		}

		// Get upstream info for commit sections
		upstreamRef := getUpstreamRef(repo)
		pushRemoteRef := getPushRemoteRef(repo)

		// 9. Unmerged into upstream (commits ahead)
		var unmergedUpstream []git.LogEntry
		if upstreamRef != "" {
			unmergedUpstream, _ = repo.LogAhead(ctx, upstreamRef, 256)
		}
		if len(unmergedUpstream) > 0 && !getSectionConfig(cfg, SectionUnmergedUpstream).Hidden {
			items := make([]Item, len(unmergedUpstream))
			for i := range unmergedUpstream {
				items[i] = Item{Commit: &unmergedUpstream[i]}
			}
			sections = append(sections, Section{
				Kind:   SectionUnmergedUpstream,
				Title:  "Unmerged into " + upstreamRef,
				Folded: getSectionConfig(cfg, SectionUnmergedUpstream).Folded,
				Hidden: getSectionConfig(cfg, SectionUnmergedUpstream).Hidden,
				Items:  items,
			})
		}

		// 10. Unpushed to push remote (only if different from upstream)
		if pushRemoteRef != "" && pushRemoteRef != upstreamRef {
			unpushed, _ := repo.LogAhead(ctx, pushRemoteRef, 256)
			if len(unpushed) > 0 && !getSectionConfig(cfg, SectionUnpushedPushRemote).Hidden {
				items := make([]Item, len(unpushed))
				for i := range unpushed {
					items[i] = Item{Commit: &unpushed[i]}
				}
				sections = append(sections, Section{
					Kind:   SectionUnpushedPushRemote,
					Title:  "Unpushed to " + pushRemoteRef,
					Folded: getSectionConfig(cfg, SectionUnpushedPushRemote).Folded,
					Hidden: getSectionConfig(cfg, SectionUnpushedPushRemote).Hidden,
					Items:  items,
				})
			}
		}

		// 11. Recent commits (only when unmerged upstream is empty)
		if len(unmergedUpstream) == 0 && !getSectionConfig(cfg, SectionRecentCommits).Hidden {
			recentCount := 10 // Default
			if cfg != nil && cfg.Sections.Recent.Folded {
				recentCount = 5 // Fewer when folded
			}
			recent, _ := repo.RecentCommits(ctx, recentCount)
			if len(recent) > 0 {
				items := make([]Item, len(recent))
				for i := range recent {
					items[i] = Item{Commit: &recent[i]}
				}
				sections = append(sections, Section{
					Kind:   SectionRecentCommits,
					Title:  "Recent Commits",
					Folded: getSectionConfig(cfg, SectionRecentCommits).Folded,
					Hidden: getSectionConfig(cfg, SectionRecentCommits).Hidden,
					Items:  items,
				})
			}
		}

		// 12. Unpulled from upstream (commits behind)
		if upstreamRef != "" {
			unpulled, _ := repo.LogBehind(ctx, upstreamRef, 256)
			if len(unpulled) > 0 && !getSectionConfig(cfg, SectionUnpulledUpstream).Hidden {
				items := make([]Item, len(unpulled))
				for i := range unpulled {
					items[i] = Item{Commit: &unpulled[i]}
				}
				sections = append(sections, Section{
					Kind:   SectionUnpulledUpstream,
					Title:  "Unpulled from " + upstreamRef,
					Folded: getSectionConfig(cfg, SectionUnpulledUpstream).Folded,
					Hidden: getSectionConfig(cfg, SectionUnpulledUpstream).Hidden,
					Items:  items,
				})
			}
		}

		// 13. Unpulled from push remote (only if different from upstream)
		if pushRemoteRef != "" && pushRemoteRef != upstreamRef {
			unpulled, _ := repo.LogBehind(ctx, pushRemoteRef, 256)
			if len(unpulled) > 0 && !getSectionConfig(cfg, SectionUnpulledPushRemote).Hidden {
				items := make([]Item, len(unpulled))
				for i := range unpulled {
					items[i] = Item{Commit: &unpulled[i]}
				}
				sections = append(sections, Section{
					Kind:   SectionUnpulledPushRemote,
					Title:  "Unpulled from " + pushRemoteRef,
					Folded: getSectionConfig(cfg, SectionUnpulledPushRemote).Folded,
					Hidden: getSectionConfig(cfg, SectionUnpulledPushRemote).Hidden,
					Items:  items,
				})
			}
		}

		return statusLoadedMsg{
			head:     head,
			sections: sections,
		}
	}
}

// buildMergeSection builds the merge sequencer section.
func buildMergeSection(cfg *config.Config, branch, subject string) *Section {
	if getSectionConfig(cfg, SectionSequencer).Hidden {
		return nil
	}
	title := "Merging"
	if branch != "" {
		title = "Merging " + branch
	}
	return &Section{
		Kind:   SectionSequencer,
		Title:  title,
		Folded: getSectionConfig(cfg, SectionSequencer).Folded,
		Hidden: false,
		Items:  nil, // Merge doesn't have items, just the header
	}
}

// buildRebaseSection builds the rebase section.
func buildRebaseSection(cfg *config.Config, state git.RebaseState) *Section {
	if getSectionConfig(cfg, SectionRebase).Hidden {
		return nil
	}

	title := "Rebasing"
	if state.Branch != "" {
		title = "Rebasing " + state.Branch
		if state.Onto != "" {
			onto := state.Onto
			if len(onto) > 7 {
				onto = onto[:7]
			}
			title += " onto " + onto
		}
	}
	if state.Total > 0 {
		title += " (" + itoa(state.Current) + "/" + itoa(state.Total) + ")"
	}

	var items []Item
	for _, entry := range state.Entries {
		items = append(items, Item{
			Action:        string(entry.Action),
			ActionHash:    entry.AbbrevHash,
			ActionSubject: entry.Subject,
			ActionDone:    entry.Done,
			ActionStopped: entry.Stopped,
		})
	}

	return &Section{
		Kind:   SectionRebase,
		Title:  title,
		Folded: getSectionConfig(cfg, SectionRebase).Folded,
		Hidden: false,
		Items:  items,
	}
}

// buildSequencerSection builds the cherry-pick or revert section.
func buildSequencerSection(cfg *config.Config, state git.SequencerState) *Section {
	if getSectionConfig(cfg, SectionSequencer).Hidden {
		return nil
	}

	var title string
	if state.Operation == "cherry-pick" {
		title = "Cherry Picking"
	} else {
		title = "Reverting"
	}
	if len(state.Items) > 0 {
		title += " (" + itoa(len(state.Items)) + ")"
	}

	var items []Item
	for _, entry := range state.Items {
		items = append(items, Item{
			Action:        entry.Action,
			ActionHash:    entry.AbbrevHash,
			ActionSubject: entry.Subject,
		})
	}

	return &Section{
		Kind:   SectionSequencer,
		Title:  title,
		Folded: getSectionConfig(cfg, SectionSequencer).Folded,
		Hidden: false,
		Items:  items,
	}
}

// buildBisectSection builds the bisect section.
func buildBisectSection(cfg *config.Config, state git.BisectState) *Section {
	if getSectionConfig(cfg, SectionBisect).Hidden {
		return nil
	}

	var items []Item
	for _, entry := range state.Items {
		items = append(items, Item{
			Action:        entry.Action,
			ActionHash:    entry.AbbrevHash,
			ActionSubject: entry.Subject,
		})
	}

	return &Section{
		Kind:   SectionBisect,
		Title:  "Bisecting Log",
		Folded: getSectionConfig(cfg, SectionBisect).Folded,
		Hidden: false,
		Items:  items,
	}
}

// getUpstreamRef returns the upstream tracking ref for the current branch.
func getUpstreamRef(repo *git.Repository) string {
	// This would need to query git config for branch.<name>.remote and branch.<name>.merge
	// For now, return empty - will be populated properly in a follow-up
	return ""
}

// getPushRemoteRef returns the push remote ref for the current branch.
func getPushRemoteRef(repo *git.Repository) string {
	// This would need to query git config for branch.<name>.pushRemote
	// For now, return empty - will be populated properly in a follow-up
	return ""
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
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

// loadPeekFileCmd loads file content for the peek preview pane.
//
//nolint:unused // Phase 4 - used in update.go
func loadPeekFileCmd(repoPath, filePath string) tea.Cmd {
	return func() tea.Msg {
		fullPath := filepath.Join(repoPath, filePath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return peekFileMsg{path: filePath, err: err}
		}
		return peekFileMsg{path: filePath, content: string(content)}
	}
}

// stageHunkCmd stages a specific hunk of a file.
func stageHunkCmd(repo *git.Repository, path string, hunk git.Hunk) tea.Cmd {
	return func() tea.Msg {
		patch := git.HunkToPatch(path, &hunk, false)
		err := repo.ApplyPatch(context.Background(), patch, "--cached")
		return operationDoneMsg{err: err}
	}
}

// unstageHunkCmd unstages a specific hunk of a file.
func unstageHunkCmd(repo *git.Repository, path string, hunk git.Hunk) tea.Cmd {
	return func() tea.Msg {
		patch := git.HunkToPatch(path, &hunk, true)
		err := repo.ApplyPatch(context.Background(), patch, "--cached")
		return operationDoneMsg{err: err}
	}
}

// discardHunkCmd discards a specific hunk of a file.
func discardHunkCmd(repo *git.Repository, path string, hunk git.Hunk) tea.Cmd {
	return func() tea.Msg {
		patch := git.HunkToPatch(path, &hunk, true)
		err := repo.ApplyPatch(context.Background(), patch)
		return operationDoneMsg{err: err}
	}
}

// stageAllUnstagedCmd stages all unstaged files.
//nolint:unused // Phase 4 - used in update.go
func stageAllUnstagedCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		// git add -u stages all modified/deleted files (not untracked)
		err := repo.StageAll(context.Background())
		return operationDoneMsg{err: err}
	}
}

// unstageAllStagedCmd unstages all staged files.
//nolint:unused // Phase 4 - used in update.go
func unstageAllStagedCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		err := repo.UnstageAll(context.Background())
		return operationDoneMsg{err: err}
	}
}

// yankToClipboardCmd copies text to clipboard using OSC 52 escape sequence.
// This works in most modern terminals.
//nolint:unused // Phase 4 - used in update.go
func yankToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		// OSC 52 clipboard escape sequence
		// Format: ESC ] 52 ; c ; <base64-encoded-text> BEL
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		fmt.Printf("\033]52;c;%s\007", encoded)
		return operationDoneMsg{err: nil}
	}
}

// openTreeCmd opens the directory containing a file in the system file manager.
//nolint:unused // Phase 4 - used in update.go
func openTreeCmd(repoPath, filePath string) tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Dir(filepath.Join(repoPath, filePath))

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", dir)
		case "linux":
			cmd = exec.Command("xdg-open", dir)
		case "windows":
			cmd = exec.Command("explorer", dir)
		default:
			return operationDoneMsg{err: fmt.Errorf("unsupported platform: %s", runtime.GOOS)}
		}

		err := cmd.Start()
		return operationDoneMsg{err: err}
	}
}

// openInEditorCmd opens a file in the user's configured editor.
//nolint:unused // Phase 4 - used in update.go
func openInEditorCmd(repoPath, filePath string) tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}

		fullPath := filepath.Join(repoPath, filePath)
		cmd := exec.Command(editor, fullPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// This will block until the editor exits
		// In a TUI app, we need to handle this specially
		return operationDoneMsg{err: fmt.Errorf("editor opening not yet implemented in TUI")}
	}
}

// untrackFileCmd removes a file from the index (git rm --cached).
//nolint:unused // Phase 4 - used in update.go
func untrackFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.UntrackFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// renameFileCmd renames a file (git mv).
//
//nolint:unused // Phase 4 - used when rename prompt is implemented
func renameFileCmd(repo *git.Repository, oldPath, newPath string) tea.Cmd {
	return func() tea.Msg {
		err := repo.RenameFile(context.Background(), oldPath, newPath)
		return operationDoneMsg{err: err}
	}
}

// openCommitEditorCmd returns a command to open the commit editor.
func openCommitEditorCmd(opts git.CommitOpts, action string) tea.Cmd {
	return func() tea.Msg {
		return openCommitEditorMsg{opts: opts, action: action}
	}
}

// openCommitEditorMsg is sent to open the commit editor.
type openCommitEditorMsg struct {
	opts   git.CommitOpts
	action string
}
