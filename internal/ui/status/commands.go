package status

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/platform"
	"github.com/mhersson/termagit/internal/ui/notification"
	"github.com/mhersson/termagit/internal/ui/rebaseeditor"
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

		// Populate upstream tracking info
		if uRemote, uBranch, uErr := repo.CurrentUpstream(ctx); uErr == nil && uRemote != "" {
			head.UpstreamRemote = uRemote
			head.UpstreamBranch = uBranch

			upstreamRef := uRemote + "/" + uBranch
			if uOid, uSubject, err := repo.RefCommitInfo(ctx, upstreamRef); err == nil {
				head.UpstreamOid = uOid
				head.UpstreamSubject = uSubject
			}
		}

		// Populate push remote info
		if pRemote, pBranch, pErr := repo.CurrentPushRemote(ctx); pErr == nil && pRemote != "" {
			head.PushRemote = pRemote
			head.PushBranch = pBranch

			pushRef := pRemote + "/" + pBranch
			if pOid, pSubject, err := repo.RefCommitInfo(ctx, pushRef); err == nil {
				head.PushOid = pOid
				head.PushSubject = pSubject
			}
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
				sec := buildMergeSection(cfg, mergeBranch, mergeHead, mergeSubject)
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
			if err == nil {
				if state.Current != nil && !getSectionConfig(cfg, SectionBisect).Hidden {
					sections = append(sections, buildBisectDetailsSection(cfg, state.Current))
				}
				if len(state.Items) > 0 {
					sec := buildBisectSection(cfg, state)
					if sec != nil {
						sections = append(sections, *sec)
					}
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

		// 11. Recent commits
		if !getSectionConfig(cfg, SectionRecentCommits).Hidden {
			recentCount := 10
			if cfg != nil && cfg.UI.RecentCommitCount > 0 {
				recentCount = cfg.UI.RecentCommitCount
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
func buildMergeSection(cfg *config.Config, branch, head, subject string) *Section {
	if getSectionConfig(cfg, SectionSequencer).Hidden {
		return nil
	}
	title := "Merging"
	if branch != "" {
		title = "Merging " + branch
	}
	abbrev := head
	if len(abbrev) > 7 {
		abbrev = abbrev[:7]
	}
	return &Section{
		Kind:   SectionSequencer,
		Title:  title,
		Folded: getSectionConfig(cfg, SectionSequencer).Folded,
		Hidden: false,
		Items: []Item{{
			Action:        "merge",
			ActionHash:    abbrev,
			ActionSubject: subject,
		}},
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
		title += " (" + strconv.Itoa(state.Current) + "/" + strconv.Itoa(state.Total) + ")"
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
		title += " (" + strconv.Itoa(len(state.Items)) + ")"
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
// buildBisectDetailsSection builds the "Bisecting at" section showing the current commit.
func buildBisectDetailsSection(cfg *config.Config, current *git.LogEntry) Section {
	return Section{
		Kind:   SectionBisect,
		Title:  "Bisecting at",
		Folded: getSectionConfig(cfg, SectionBisect).Folded,
		Items:  []Item{{BisectDetail: current}},
	}
}

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
// Returns "remote/branch" or "" if no upstream is configured.
func getUpstreamRef(repo *git.Repository) string {
	remote, branch, err := repo.CurrentUpstream(context.Background())
	if err != nil || remote == "" || branch == "" {
		return ""
	}
	return remote + "/" + branch
}

// getPushRemoteRef returns the push remote ref for the current branch.
// Returns "remote/branch" or "" if no push remote is configured.
func getPushRemoteRef(repo *git.Repository) string {
	remote, branch, err := repo.CurrentPushRemote(context.Background())
	if err != nil || remote == "" || branch == "" {
		return ""
	}
	return remote + "/" + branch
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

func stageFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.StageFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// unstageFileCmd unstages a file.

func unstageFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.UnstageFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// discardFileCmd discards changes to a file.

func discardFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.DiscardFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// notifyAppCmd returns a command that sends a notification.NotifyMsg to the app layer.
func notifyAppCmd(msg string, kind notification.Kind) tea.Cmd {
	return func() tea.Msg {
		return notification.NotifyMsg{Message: msg, Kind: kind}
	}
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
func stageAllUnstagedCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		// git add -u stages all modified/deleted files (not untracked)
		err := repo.StageAll(context.Background())
		return operationDoneMsg{err: err}
	}
}

// unstageAllStagedCmd unstages all staged files.
func unstageAllStagedCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		err := repo.UnstageAll(context.Background())
		return operationDoneMsg{err: err}
	}
}

// yankToClipboardCmd copies text to clipboard using platform clipboard tools.

func yankToClipboardCmd(text string) tea.Cmd {
	return func() tea.Msg {
		err := platform.CopyToClipboard(text)
		return operationDoneMsg{err: err}
	}
}

// openTreeCmd opens the directory containing a file in the system file manager.
func openTreeCmd(repoPath, filePath string) tea.Cmd {
	return func() tea.Msg {
		dir := filepath.Dir(filepath.Join(repoPath, filePath))
		err := platform.Open(dir)
		return operationDoneMsg{err: err}
	}
}

// openInEditorCmd opens a file in the user's configured editor using tea.ExecProcess.
// This suspends the TUI, runs the editor, then resumes.
func openInEditorCmd(repoPath, filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	fullPath := filepath.Join(repoPath, filePath)
	c := exec.Command(editor, "--", fullPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return operationDoneMsg{err: err}
	})
}

// untrackFileCmd removes a file from the index (git rm --cached).
func untrackFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		err := repo.UntrackFile(context.Background(), path)
		return operationDoneMsg{err: err}
	}
}

// renameFileCmd renames a file (git mv).
//
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

// commitsLoadedMsg carries the recent commits for the commit select overlay.
type commitsLoadedMsg struct {
	commits []git.LogEntry
	err     error
}

// loadCommitsForSelectCmd fetches recent commits to populate the commit select view.
func loadCommitsForSelectCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		commits, err := repo.RecentCommits(context.Background(), 256)
		return commitsLoadedMsg{commits: commits, err: err}
	}
}

// commitSpecialCmd runs a fixup/squash commit directly (no editor).
func commitSpecialCmd(repo *git.Repository, opts git.CommitOpts) tea.Cmd {
	return func() tea.Msg {
		_, err := repo.Commit(context.Background(), opts)
		return operationDoneMsg{err: err}
	}
}

// commitAndAutosquashCmd runs a fixup/squash commit then autosquash rebases.
func commitAndAutosquashCmd(repo *git.Repository, opts git.CommitOpts, targetFullHash string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		_, err := repo.Commit(ctx, opts)
		if err != nil {
			return operationDoneMsg{err: fmt.Errorf("commit: %w", err)}
		}

		err = repo.RebaseAutosquash(ctx, targetFullHash)
		if err != nil {
			return operationDoneMsg{err: fmt.Errorf("autosquash rebase: %w", err)}
		}
		return operationDoneMsg{}
	}
}

// rebaseCmd returns a command that runs a non-interactive rebase.
func rebaseCmd(repo *git.Repository, opts git.RebaseOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Rebase"}
		}
		err := repo.Rebase(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Rebase"}
	}
}

// interactiveRebaseCmd generates the todo entries and opens the rebase editor.
// opts.Onto is the commit the user selected — we use its parent as the rebase
// base so the selected commit itself appears as the first entry in the editor.
func interactiveRebaseCmd(repo *git.Repository, opts git.RebaseOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Rebase"}
		}
		ctx := context.Background()
		// Use parent of selected commit so it's included in the todo
		base := opts.Onto + "~1"
		entries, err := repo.GenerateRebaseTodo(ctx, base)
		if err != nil {
			return operationDoneMsg{err: err, op: "Rebase"}
		}
		return rebaseeditor.OpenRebaseEditorMsg{
			Entries:    entries,
			Base:       base,
			RebaseOpts: opts,
		}
	}
}

// rebaseContinueCmd returns a command that continues an in-progress rebase.
func rebaseContinueCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Rebase continue"}
		}
		err := repo.RebaseContinue(context.Background())
		return operationDoneMsg{err: err, op: "Rebase continue"}
	}
}

// rebaseSkipCmd returns a command that skips the current commit in a rebase.
func rebaseSkipCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Rebase skip"}
		}
		err := repo.RebaseSkip(context.Background())
		return operationDoneMsg{err: err, op: "Rebase skip"}
	}
}

// rebaseAbortCmd returns a command that aborts an in-progress rebase.
func rebaseAbortCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Rebase abort"}
		}
		err := repo.RebaseAbort(context.Background())
		return operationDoneMsg{err: err, op: "Rebase abort"}
	}
}

// modifyCommitCmd returns a command that modifies a commit (stop for editing).
func modifyCommitCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Modify commit"}
		}
		err := repo.ModifyCommit(context.Background(), hash)
		return operationDoneMsg{err: err, op: "Modify commit"}
	}
}

// rewordCommitCmd returns a command that rewords a commit message.
func rewordCommitCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Reword commit"}
		}
		err := repo.RewordCommit(context.Background(), hash, "")
		return operationDoneMsg{err: err, op: "Reword commit"}
	}
}

// dropCommitCmd returns a command that drops a commit from history.
func dropCommitCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Drop commit"}
		}
		err := repo.DropCommit(context.Background(), hash)
		return operationDoneMsg{err: err, op: "Drop commit"}
	}
}

// autosquashCmd returns a command that runs autosquash rebase.
func autosquashCmd(repo *git.Repository, opts git.RebaseOpts, target string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Autosquash"}
		}
		opts.Onto = target
		err := repo.Autosquash(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Autosquash"}
	}
}

// --- Branch commands ---

// loadLocalBranchesCmd loads local branches.
func loadLocalBranchesCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return branchesLoadedMsg{err: fmt.Errorf("no repository")}
		}
		branches, err := repo.ListBranches(context.Background())
		return branchesLoadedMsg{branches: branches, err: err}
	}
}

// loadAllBranchesCmd loads local and remote branches.
func loadAllBranchesCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return branchesLoadedMsg{err: fmt.Errorf("no repository")}
		}
		ctx := context.Background()
		local, err := repo.ListBranches(ctx)
		if err != nil {
			return branchesLoadedMsg{err: err}
		}
		remote, err := repo.ListRemoteBranches(ctx)
		if err != nil {
			return branchesLoadedMsg{err: err}
		}
		return branchesLoadedMsg{branches: append(local, remote...)}
	}
}

// loadRecentBranchesCmd loads branches sorted by most recently checked out.
func loadRecentBranchesCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return branchesLoadedMsg{err: fmt.Errorf("no repository")}
		}
		branches, err := repo.RecentBranches(context.Background())
		return branchesLoadedMsg{branches: branches, err: err}
	}
}

// loadStashesForSelectCmd loads stashes as LogEntry items for the commit select view.
func loadStashesForSelectCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return commitsLoadedMsg{err: fmt.Errorf("no repository")}
		}
		stashes, err := repo.ListStashes(context.Background())
		if err != nil {
			return commitsLoadedMsg{err: err}
		}
		entries := make([]git.LogEntry, len(stashes))
		for i, s := range stashes {
			entries[i] = git.LogEntry{
				Hash:            s.Hash,
				AbbreviatedHash: s.Name,
				Subject:         s.Message,
			}
		}
		return commitsLoadedMsg{commits: entries}
	}
}

// checkoutBranchCmd checks out a branch.
func checkoutBranchCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Checkout"}
		}
		err := repo.Checkout(context.Background(), name)
		return operationDoneMsg{err: err, op: "Checkout " + name}
	}
}

// createAndCheckoutBranchCmd creates a new branch and checks it out.
func createAndCheckoutBranchCmd(repo *git.Repository, name, base string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Create branch"}
		}
		err := repo.CheckoutNewBranch(context.Background(), name, base)
		return operationDoneMsg{err: err, op: "Create and checkout " + name}
	}
}

// createBranchCmd creates a new branch without checking it out.
func createBranchCmd(repo *git.Repository, name, base string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Create branch"}
		}
		err := repo.CreateBranch(context.Background(), name, base)
		return operationDoneMsg{err: err, op: "Create " + name}
	}
}

// deleteBranchCmd deletes a branch.
func deleteBranchCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Delete branch"}
		}
		err := repo.DeleteBranch(context.Background(), name, false)
		return operationDoneMsg{err: err, op: "Delete " + name}
	}
}

// renameBranchCmd renames a branch.
func renameBranchCmd(repo *git.Repository, oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Rename branch"}
		}
		err := repo.RenameBranch(context.Background(), oldName, newName)
		return operationDoneMsg{err: err, op: "Rename " + oldName + " to " + newName}
	}
}

// spinOffBranchCmd creates a spin-off branch.
func spinOffBranchCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Spin-off"}
		}
		err := repo.SpinOffBranch(context.Background(), name)
		return operationDoneMsg{err: err, op: "Spin-off " + name}
	}
}

// spinOutBranchCmd creates a spin-out branch.
func spinOutBranchCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Spin-out"}
		}
		err := repo.SpinOutBranch(context.Background(), name)
		return operationDoneMsg{err: err, op: "Spin-out " + name}
	}
}

// pushRefspecCmd pushes an explicit refspec to the default remote.
func pushRefspecCmd(repo *git.Repository, refspec string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Push refspec"}
		}
		ctx := context.Background()
		remote, _ := repo.SmartDefaultRemote(ctx)
		if remote == "" {
			return operationDoneMsg{err: fmt.Errorf("no remote configured"), op: "Push refspec"}
		}
		err := repo.Push(ctx, git.PushOpts{Remote: remote, Branch: refspec})
		return operationDoneMsg{err: err, op: "Push " + refspec}
	}
}

// pushTagCmd pushes a tag to the default remote.
func pushTagCmd(repo *git.Repository, tag string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Push tag"}
		}
		ctx := context.Background()
		remote, _ := repo.SmartDefaultRemote(ctx)
		if remote == "" {
			return operationDoneMsg{err: fmt.Errorf("no remote configured"), op: "Push tag"}
		}
		err := repo.PushTag(ctx, remote, tag)
		return operationDoneMsg{err: err, op: "Push tag " + tag}
	}
}

// resetBranchToUpstreamCmd resets the current branch to its upstream.
func resetBranchToUpstreamCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Reset branch"}
		}
		ctx := context.Background()
		remote, branch, err := repo.CurrentUpstream(ctx)
		if err != nil {
			return operationDoneMsg{err: err, op: "Reset branch"}
		}
		if remote == "" {
			return operationDoneMsg{err: fmt.Errorf("no upstream configured"), op: "Reset branch"}
		}
		err = repo.Reset(ctx, remote+"/"+branch, git.ResetHard)
		return operationDoneMsg{err: err, op: "Reset branch to " + remote + "/" + branch}
	}
}

// --- Merge commands ---

// mergeCmd creates a command that executes a git merge.
func mergeCmd(repo *git.Repository, opts git.MergeOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Merge"}
		}
		err := repo.Merge(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Merge"}
	}
}

// mergePreviewCmd shows what would be merged from a branch using the log view.
func mergePreviewCmd(repo *git.Repository, branch string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Merge preview"}
		}
		ctx := context.Background()
		// Show commits in branch that are not in HEAD
		opts := git.LogOpts{MaxCount: 256, Decorate: true, Branch: "HEAD.." + branch}
		commits, _, err := repo.Log(ctx, opts)
		if err != nil {
			return operationDoneMsg{err: err, op: "Merge preview"}
		}
		if len(commits) == 0 {
			return notification.NotifyMsg{
				Message: "Nothing to merge from " + branch,
				Kind:    notification.Info,
			}
		}
		return OpenLogViewMsg{
			Commits: commits,
			HasMore: false,
			Branch:  "merge preview: " + branch,
		}
	}
}

// mergeCommitCmd creates a command that commits an in-progress merge.
func mergeCommitCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Merge commit"}
		}
		err := repo.MergeCommit(context.Background())
		return operationDoneMsg{err: err, op: "Merge commit"}
	}
}

// mergeAbortCmd creates a command that aborts an in-progress merge.
func mergeAbortCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Merge abort"}
		}
		err := repo.MergeAbort(context.Background())
		return operationDoneMsg{err: err, op: "Merge abort"}
	}
}

// --- Cherry-pick commands ---

// cherryPickCmd creates a command that cherry-picks commits.
func cherryPickCmd(repo *git.Repository, hashes []string, opts git.CherryPickOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick"}
		}
		err := repo.CherryPick(context.Background(), hashes, opts)
		return operationDoneMsg{err: err, op: "Cherry-pick"}
	}
}

// cherryPickDonateCmd creates a command that donates cherry-picked commits to another branch.
func cherryPickDonateCmd(repo *git.Repository, hashes []string, src, dst string, opts git.CherryPickOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick donate"}
		}
		err := repo.CherryPickDonate(context.Background(), hashes, src, dst, opts)
		return operationDoneMsg{err: err, op: "Cherry-pick donate"}
	}
}

// cherryPickSpinoutCmd creates a new branch at HEAD and cherry-picks commits onto it.
func cherryPickSpinoutCmd(repo *git.Repository, hashes []string, branchName string, opts git.CherryPickOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick spinout"}
		}
		ctx := context.Background()
		// Create new branch at HEAD
		if err := repo.CreateBranch(ctx, branchName, "HEAD"); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinout"}
		}
		// Checkout the new branch
		if err := repo.Checkout(ctx, branchName); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinout"}
		}
		// Cherry-pick the commits
		if err := repo.CherryPick(ctx, hashes, opts); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinout"}
		}
		return operationDoneMsg{op: "Cherry-pick spinout to " + branchName}
	}
}

// cherryPickSpinoffCmd creates a new branch, cherry-picks commits onto it,
// then resets the original branch back.
func cherryPickSpinoffCmd(repo *git.Repository, hashes []string, originalBranch, branchName string, opts git.CherryPickOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick spinoff"}
		}
		ctx := context.Background()
		// Create new branch at HEAD
		if err := repo.CreateBranch(ctx, branchName, "HEAD"); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinoff"}
		}
		// Checkout the new branch
		if err := repo.Checkout(ctx, branchName); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinoff"}
		}
		// Cherry-pick the commits
		if err := repo.CherryPick(ctx, hashes, opts); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinoff"}
		}
		// Reset original branch to exclude the cherry-picked commits.
		// Move it back by N commits.
		target := fmt.Sprintf("%s~%d", originalBranch, len(hashes))
		if err := repo.MoveBranch(ctx, originalBranch, target); err != nil {
			return operationDoneMsg{err: err, op: "Cherry-pick spinoff (reset original)"}
		}
		return operationDoneMsg{op: "Cherry-pick spinoff to " + branchName}
	}
}

// cherryPickApplyCmd creates a command that applies cherry-pick changes without committing.
func cherryPickApplyCmd(repo *git.Repository, hashes []string, opts git.CherryPickOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick apply"}
		}
		err := repo.CherryPickApply(context.Background(), hashes, opts)
		return operationDoneMsg{err: err, op: "Cherry-pick apply"}
	}
}

// cherryPickContinueCmd creates a command that continues an in-progress cherry-pick.
func cherryPickContinueCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick continue"}
		}
		err := repo.CherryPickContinue(context.Background())
		return operationDoneMsg{err: err, op: "Cherry-pick continue"}
	}
}

// cherryPickSkipCmd creates a command that skips the current commit in a cherry-pick.
func cherryPickSkipCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick skip"}
		}
		err := repo.CherryPickSkip(context.Background())
		return operationDoneMsg{err: err, op: "Cherry-pick skip"}
	}
}

// cherryPickAbortCmd creates a command that aborts an in-progress cherry-pick.
func cherryPickAbortCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Cherry-pick abort"}
		}
		err := repo.CherryPickAbort(context.Background())
		return operationDoneMsg{err: err, op: "Cherry-pick abort"}
	}
}

// --- Revert commands ---

// revertCmd creates a command that reverts commits.
func revertCmd(repo *git.Repository, hashes []string, opts git.RevertOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Revert"}
		}
		err := repo.Revert(context.Background(), hashes, opts)
		return operationDoneMsg{err: err, op: "Revert"}
	}
}

// revertChangesCmd creates a command that applies reverse changes without committing.
func revertChangesCmd(repo *git.Repository, hashes []string, opts git.RevertOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Revert changes"}
		}
		err := repo.RevertChanges(context.Background(), hashes, opts)
		return operationDoneMsg{err: err, op: "Revert changes"}
	}
}

// revertContinueCmd creates a command that continues an in-progress revert.
func revertContinueCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Revert continue"}
		}
		err := repo.RevertContinue(context.Background())
		return operationDoneMsg{err: err, op: "Revert continue"}
	}
}

// revertSkipCmd creates a command that skips the current commit in a revert.
func revertSkipCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Revert skip"}
		}
		err := repo.RevertSkip(context.Background())
		return operationDoneMsg{err: err, op: "Revert skip"}
	}
}

// revertAbortCmd creates a command that aborts an in-progress revert.
func revertAbortCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Revert abort"}
		}
		err := repo.RevertAbort(context.Background())
		return operationDoneMsg{err: err, op: "Revert abort"}
	}
}

// --- Stash commands ---

// stashPushCmd creates a command that pushes to the stash.
func stashPushCmd(repo *git.Repository, opts git.StashOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash push"}
		}
		err := repo.Stash(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Stash push"}
	}
}

// stashPopCmd creates a command that pops a stash entry.
func stashPopCmd(repo *git.Repository, index int) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash pop"}
		}
		err := repo.StashPop(context.Background(), index)
		return operationDoneMsg{err: err, op: "Stash pop"}
	}
}

// stashApplyCmd creates a command that applies a stash entry.
func stashApplyCmd(repo *git.Repository, index int) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash apply"}
		}
		err := repo.StashApply(context.Background(), index)
		return operationDoneMsg{err: err, op: "Stash apply"}
	}
}

// stashDropCmd creates a command that drops a stash entry.
func stashDropCmd(repo *git.Repository, index int) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash drop"}
		}
		err := repo.StashDrop(context.Background(), index)
		return operationDoneMsg{err: err, op: "Stash drop"}
	}
}

// stashBranchCmd creates a command that creates a branch from a stash entry.
func stashBranchCmd(repo *git.Repository, name string, index int) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash branch"}
		}
		err := repo.StashBranch(context.Background(), name, index)
		return operationDoneMsg{err: err, op: "Stash branch"}
	}
}

// stashRenameCmd creates a command that renames a stash entry.
func stashRenameCmd(repo *git.Repository, index int, newName string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash rename"}
		}
		err := repo.StashRename(context.Background(), index, newName)
		return operationDoneMsg{err: err, op: "Stash rename"}
	}
}

// stashSnapshotCmd creates a stash snapshot (stash push + stash apply to restore state).
func stashSnapshotCmd(repo *git.Repository, opts git.StashOpts, label string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash snapshot"}
		}
		ctx := context.Background()
		opts.Message = label
		// Push to stash
		if err := repo.Stash(ctx, opts); err != nil {
			return operationDoneMsg{err: err, op: "Stash snapshot"}
		}
		// Apply the stash immediately to restore working state
		if err := repo.StashApply(ctx, 0); err != nil {
			return operationDoneMsg{err: err, op: "Stash snapshot (apply)"}
		}
		return operationDoneMsg{op: "Stash snapshot"}
	}
}

// stashWipRefCmd creates a stash commit and stores it under refs/wip/<branch>.
func stashWipRefCmd(repo *git.Repository, opts git.StashOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash WIP ref"}
		}
		err := repo.StashCreateWipRef(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Stash WIP ref"}
	}
}

// openStashInCommitViewMsg is sent to open a commit view overlay for a stash entry.
type openStashInCommitViewMsg struct {
	ref string
}

// openStashInCommitViewCmd emits a message to open the commit view for a stash.
func openStashInCommitViewCmd(_ *git.Repository, index int) tea.Cmd {
	return func() tea.Msg {
		ref := fmt.Sprintf("stash@{%d}", index)
		return openStashInCommitViewMsg{ref: ref}
	}
}

// stashFormatPatchCmd creates a patch file from a stash entry.
func stashFormatPatchCmd(repo *git.Repository, index int) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Stash format-patch"}
		}
		ctx := context.Background()
		patch, err := repo.StashShowPatch(ctx, index)
		if err != nil {
			return operationDoneMsg{err: err, op: "Stash format-patch"}
		}
		filename := fmt.Sprintf("stash-%d.patch", index)
		if err := os.WriteFile(filename, []byte(patch), 0o600); err != nil {
			return operationDoneMsg{err: fmt.Errorf("write patch file: %w", err), op: "Stash format-patch"}
		}
		return operationDoneMsg{op: "Stash format-patch: wrote " + filename}
	}
}

// --- Reset commands ---

// resetCmd creates a command that executes a git reset.
func resetCmd(repo *git.Repository, target string, mode git.ResetMode) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Reset"}
		}
		err := repo.Reset(context.Background(), target, mode)
		return operationDoneMsg{err: err, op: "Reset"}
	}
}

// resetFileCmd creates a command that resets a single file.
func resetFileCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Reset file"}
		}
		err := repo.ResetFile(context.Background(), path, "HEAD")
		return operationDoneMsg{err: err, op: "Reset file"}
	}
}

// --- Tag commands ---

// tagCreateCmd creates a command that creates a tag.
func tagCreateCmd(repo *git.Repository, name, hash string, opts git.TagOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Tag create"}
		}
		err := repo.CreateTag(context.Background(), name, hash, opts)
		return operationDoneMsg{err: err, op: "Tag create"}
	}
}

// tagDeleteCmd creates a command that deletes a tag.
func tagDeleteCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Tag delete"}
		}
		err := repo.DeleteTag(context.Background(), name)
		return operationDoneMsg{err: err, op: "Tag delete"}
	}
}

// tagPruneCmd creates a command that prunes remote tags.
func tagPruneCmd(repo *git.Repository, remote string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Tag prune"}
		}
		err := repo.PruneRemoteTags(context.Background(), remote)
		return operationDoneMsg{err: err, op: "Tag prune"}
	}
}

// --- Remote commands ---

// remoteAddCmd creates a command that adds a remote.
func remoteAddCmd(repo *git.Repository, name, url string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Remote add"}
		}
		err := repo.AddRemote(context.Background(), name, url)
		return operationDoneMsg{err: err, op: "Remote add"}
	}
}

// remoteRemoveCmd creates a command that removes a remote.
func remoteRemoveCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Remote remove"}
		}
		err := repo.RemoveRemote(context.Background(), name)
		return operationDoneMsg{err: err, op: "Remote remove"}
	}
}

// remoteRenameCmd creates a command that renames a remote.
func remoteRenameCmd(repo *git.Repository, oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Remote rename"}
		}
		err := repo.RenameRemote(context.Background(), oldName, newName)
		return operationDoneMsg{err: err, op: "Remote rename"}
	}
}

// remotePruneCmd creates a command that prunes stale remote tracking branches.
func remotePruneCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Remote prune"}
		}
		err := repo.PruneRemote(context.Background(), name)
		return operationDoneMsg{err: err, op: "Remote prune"}
	}
}

// remoteSetHeadCmd auto-detects and sets the default branch for a remote.
func remoteSetHeadCmd(repo *git.Repository, name string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Remote set-head"}
		}
		err := repo.SetRemoteHead(context.Background(), name)
		return operationDoneMsg{err: err, op: "Remote set-head"}
	}
}

// fetchUnshallowCmd converts a shallow clone to a full clone.
func fetchUnshallowCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Fetch unshallow"}
		}
		err := repo.FetchUnshallow(context.Background())
		return operationDoneMsg{err: err, op: "Fetch unshallow"}
	}
}

// --- Worktree commands ---

// worktreeAddCmd creates a command that adds a worktree.
func worktreeAddCmd(repo *git.Repository, path, branch string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Worktree add"}
		}
		err := repo.AddWorktree(context.Background(), path, branch)
		return operationDoneMsg{err: err, op: "Worktree add"}
	}
}

// worktreeRemoveCmd creates a command that removes a worktree.
func worktreeRemoveCmd(repo *git.Repository, path string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Worktree remove"}
		}
		err := repo.RemoveWorktree(context.Background(), path, false)
		return operationDoneMsg{err: err, op: "Worktree remove"}
	}
}

// worktreeMoveCmd creates a command that moves a worktree.
func worktreeMoveCmd(repo *git.Repository, oldPath, newPath string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Worktree move"}
		}
		err := repo.MoveWorktree(context.Background(), oldPath, newPath)
		return operationDoneMsg{err: err, op: "Worktree move"}
	}
}

// --- Bisect commands ---

// bisectStartCmd creates a command that starts a bisect session.
func bisectStartCmd(repo *git.Repository, opts git.BisectOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Bisect start"}
		}
		err := repo.BisectStart(context.Background(), "HEAD", nil, opts)
		return operationDoneMsg{err: err, op: "Bisect start"}
	}
}

// bisectGoodCmd creates a command that marks a commit as good.
func bisectGoodCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Bisect good"}
		}
		err := repo.BisectGood(context.Background(), hash)
		return operationDoneMsg{err: err, op: "Bisect good"}
	}
}

// bisectBadCmd creates a command that marks a commit as bad.
func bisectBadCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Bisect bad"}
		}
		err := repo.BisectBad(context.Background(), hash)
		return operationDoneMsg{err: err, op: "Bisect bad"}
	}
}

// bisectSkipCmd creates a command that marks a commit as untestable.
func bisectSkipCmd(repo *git.Repository, hash string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Bisect skip"}
		}
		err := repo.BisectSkip(context.Background(), hash)
		return operationDoneMsg{err: err, op: "Bisect skip"}
	}
}

// bisectResetCmd creates a command that resets/ends a bisect session.
func bisectResetCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Bisect reset"}
		}
		err := repo.BisectReset(context.Background())
		return operationDoneMsg{err: err, op: "Bisect reset"}
	}
}

// bisectRunCmd creates a command that runs an automated bisect with a script.
func bisectRunCmd(repo *git.Repository, script string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Bisect run"}
		}
		err := repo.BisectRun(context.Background(), script, nil)
		return operationDoneMsg{err: err, op: "Bisect run"}
	}
}

// --- Ignore commands ---

// ignoreCmd creates a command that adds an ignore rule.
func ignoreCmd(repo *git.Repository, pattern string, scope git.IgnoreScope) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Ignore"}
		}
		err := repo.AddIgnoreRule(context.Background(), pattern, scope)
		return operationDoneMsg{err: err, op: "Ignore"}
	}
}

// openRemoteConfigCmd reads git config values for a remote and returns a message to open the popup.
func openRemoteConfigCmd(repo *git.Repository, remote string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		keys := []string{
			"remote." + remote + ".url",
			"remote." + remote + ".fetch",
			"remote." + remote + ".pushurl",
			"remote." + remote + ".push",
			"remote." + remote + ".tagOpt",
		}
		values := make(map[string]string)
		for _, k := range keys {
			v, err := repo.GetConfigValue(ctx, k)
			if err != nil {
				return notification.NotifyMsg{
					Message: "Failed to read config: " + err.Error(),
					Kind:    notification.Error,
				}
			}
			values[k] = v
		}
		return remoteConfigLoadedMsg{remote: remote, values: values}
	}
}

// openBranchConfigCmd loads git config values for a branch and sends them to open the popup.
func openBranchConfigCmd(repo *git.Repository, branch string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		keys := []string{
			"branch." + branch + ".description",
			"branch." + branch + ".merge",
			"branch." + branch + ".remote",
			"branch." + branch + ".rebase",
			"branch." + branch + ".pushRemote",
			"pull.rebase",
			"remote.pushDefault",
		}
		values := make(map[string]string)
		for _, k := range keys {
			v, err := repo.GetConfigValue(ctx, k)
			if err != nil {
				return notification.NotifyMsg{
					Message: "Failed to read config: " + err.Error(),
					Kind:    notification.Error,
				}
			}
			values[k] = v
		}

		// Load remotes for pushRemote/pushDefault choices
		remotes, err := repo.ListRemotes(ctx)
		if err != nil {
			return branchConfigLoadedMsg{branch: branch, values: values, err: err}
		}
		remoteNames := make([]string, len(remotes))
		for i, r := range remotes {
			remoteNames[i] = r.Name
		}

		// Load local and global pull.rebase
		pullRebase := values["pull.rebase"]
		if pullRebase == "" {
			pullRebase = "false"
		}
		globalPullRebase, _ := repo.GetGlobalConfigValue(ctx, "pull.rebase")

		return branchConfigLoadedMsg{
			branch:           branch,
			values:           values,
			remotes:          remoteNames,
			pullRebase:       pullRebase,
			globalPullRebase: globalPullRebase,
		}
	}
}

// setBranchConfigCmd writes changed git config values for a branch.
func setBranchConfigCmd(repo *git.Repository, configValues map[string]string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		for key, value := range configValues {
			if value == "" {
				continue
			}
			if err := repo.SetConfigValue(ctx, key, value); err != nil {
				return operationDoneMsg{err: err, op: "Branch configure"}
			}
		}
		return operationDoneMsg{op: "Branch configure"}
	}
}

// setRemoteConfigCmd writes changed git config values for a remote.
func setRemoteConfigCmd(repo *git.Repository, configValues map[string]string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		for key, value := range configValues {
			if value == "" {
				continue
			}
			if err := repo.SetConfigValue(ctx, key, value); err != nil {
				return operationDoneMsg{err: err, op: "Remote configure"}
			}
		}
		return operationDoneMsg{op: "Remote configure"}
	}
}

// loadRefsCmd loads all refs and remotes for the refs view.
func loadRefsCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		refs, err := repo.ListRefs(ctx)
		if err != nil {
			return notification.NotifyMsg{Message: "Failed to load refs: " + err.Error(), Kind: notification.Error}
		}
		remotes, _ := repo.ListRemotes(ctx)
		return OpenRefsViewMsg{Refs: refs, Remotes: remotes}
	}
}

// loadStashListCmd loads all stashes for the stash list view.
func loadStashListCmd(repo *git.Repository) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		stashes, err := repo.ListStashes(ctx)
		if err != nil {
			return notification.NotifyMsg{Message: "Failed to load stashes: " + err.Error(), Kind: notification.Error}
		}
		return OpenStashListMsg{Stashes: stashes}
	}
}
