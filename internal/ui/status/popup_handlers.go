package status

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/platform"
	"github.com/mhersson/termagit/internal/ui/branchselect"
	"github.com/mhersson/termagit/internal/ui/commitselect"
	"github.com/mhersson/termagit/internal/ui/diffview"
	"github.com/mhersson/termagit/internal/ui/notification"
	"github.com/mhersson/termagit/internal/ui/popup"
)

// handlePopupAction processes the action from a closed popup.
func handlePopupAction(m Model, kind PopupKind, result popup.Result) (tea.Model, tea.Cmd) {
	switch kind {
	case PopupCommit:
		return handleCommitPopupAction(m, result)
	case PopupBranch:
		return handleBranchPopupAction(m, result)
	case PopupRebase:
		return handleRebasePopupAction(m, result)
	case PopupPush:
		return handlePushPopupAction(m, result)
	case PopupPull:
		return handlePullPopupAction(m, result)
	case PopupFetch:
		return handleFetchPopupAction(m, result)
	case PopupLog:
		return handleLogPopupAction(m, result)
	case PopupMerge:
		return handleMergePopupAction(m, result)
	case PopupCherryPick:
		return handleCherryPickPopupAction(m, result)
	case PopupRevert:
		return handleRevertPopupAction(m, result)
	case PopupStash:
		return handleStashPopupAction(m, result)
	case PopupReset:
		return handleResetPopupAction(m, result)
	case PopupTag:
		return handleTagPopupAction(m, result)
	case PopupRemote:
		return handleRemotePopupAction(m, result)
	case PopupWorktree:
		return handleWorktreePopupAction(m, result)
	case PopupBisect:
		return handleBisectPopupAction(m, result)
	case PopupIgnore:
		return handleIgnorePopupAction(m, result)
	case PopupDiff:
		return handleDiffPopupAction(m, result)
	case PopupMargin:
		return handleMarginPopupAction(m, result)
	case PopupHelp:
		return handleHelpPopupAction(m, result)
	case PopupYank:
		return handleYankPopupAction(m, result)
	case PopupRemoteConfig:
		return handleRemoteConfigPopupAction(m, result)
	case PopupBranchConfig:
		return handleBranchConfigPopupAction(m, result)
	default:
		return m, notifyAppCmd("Action: "+result.Action, notification.Info)
	}
}

// handleCommitPopupAction handles actions from the commit popup.
func handleCommitPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildCommitOpts(result)

	switch result.Action {
	case "c": // Commit
		if !opts.AllowEmpty && !opts.All && !hasStagedChanges(m) {
			return m, notifyAppCmd("No changes to commit.", notification.Warning)
		}
		return m, openCommitEditorCmd(opts, "commit")
	case "e": // Extend (amend without editing)
		if !opts.AllowEmpty && !opts.All && !hasStagedChanges(m) {
			return m, notifyAppCmd("No changes to commit.", notification.Warning)
		}
		opts.Amend = true
		return m, openCommitEditorCmd(opts, "extend")
	case "a": // Amend
		opts.Amend = true
		return m, openCommitEditorCmd(opts, "amend")
	case "w": // Reword
		opts.Amend = true
		return m, openCommitEditorCmd(opts, "reword")
	case "f": // Fixup
		return openCommitSelect(m, opts, commitSpecialFixup)
	case "s": // Squash
		return openCommitSelect(m, opts, commitSpecialSquash)
	case "A": // Alter
		return openCommitSelect(m, opts, commitSpecialAlter)
	case "n": // Augment
		return openCommitSelect(m, opts, commitSpecialAugment)
	case "W": // Revise
		return openCommitSelect(m, opts, commitSpecialRevise)
	case "F": // Instant Fixup
		return openCommitSelect(m, opts, commitSpecialInstantFixup)
	case "S": // Instant Squash
		return openCommitSelect(m, opts, commitSpecialInstantSquash)
	case "x": // Absorb — requires external git-absorb
		return m, notifyAppCmd("Absorb requires git-absorb to be installed", notification.Warning)
	default:
		return m, notifyAppCmd("Unknown commit action: "+result.Action, notification.Warning)
	}
}

// hasStagedChanges returns true if the model has a non-empty staged section.
func hasStagedChanges(m Model) bool {
	for i := range m.sections {
		if m.sections[i].Kind == SectionStaged && len(m.sections[i].Items) > 0 {
			return true
		}
	}
	return false
}

// buildCommitOpts builds CommitOpts from popup result switches and options.
func buildCommitOpts(result popup.Result) git.CommitOpts {
	return git.CommitOpts{
		All:          result.Switches["all"],
		AllowEmpty:   result.Switches["allow-empty"],
		Verbose:      result.Switches["verbose"],
		NoVerify:     result.Switches["no-verify"],
		ResetAuthor:  result.Switches["reset-author"],
		Signoff:      result.Switches["signoff"],
		Author:       result.Options["author"],
		GpgSign:      result.Options["gpg-sign"],
		ReuseMessage: result.Options["reuse-message"],
	}
}

// handleOpenCommitPopup opens the commit popup.
func handleOpenCommitPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewCommitPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupCommit
	return m, nil
}

// handleRebasePopupAction handles actions from the rebase popup.
func handleRebasePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inRebase := isInRebase(m.sections)

	if inRebase {
		return handleRebaseInProgressAction(m, result)
	}

	opts := buildRebaseOpts(result)

	switch result.Action {
	// Rebase onto group
	case "p": // pushRemote
		remote := m.head.PushRemote
		if remote == "" {
			remote = m.head.UpstreamRemote
		}
		if remote == "" {
			return m, notifyAppCmd("No push remote configured", notification.Warning)
		}
		target := remote + "/" + m.head.Branch
		opts.Onto = target
		return m, rebaseCmd(m.repo, opts)
	case "u": // @{upstream}
		remote := m.head.UpstreamRemote
		if remote == "" {
			return m, notifyAppCmd("No upstream configured", notification.Warning)
		}
		target := remote + "/" + m.head.UpstreamBranch
		opts.Onto = target
		return m, rebaseCmd(m.repo, opts)
	case "e": // elsewhere — select branch to rebase onto
		m.branchActionKind = branchActionRebaseElsewhere
		m.rebaseSpecialOpts = opts
		return m, loadAllBranchesCmd(m.repo)
	case "b": // base branch — rebase onto base (main/master detection)
		return m, notifyAppCmd("Base branch detection not configured", notification.Warning)

	// Rebase group
	case "i": // interactively — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialInteractive)
	case "s": // a subset — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialSubset)

	// Modify commits group
	case "m": // to modify a commit — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialModify)
	case "w": // to reword a commit — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialReword)
	case "d": // to remove a commit — needs commit selection
		return openRebaseCommitSelect(m, opts, rebaseSpecialDrop)
	case "f": // to autosquash
		target := m.head.UpstreamRemote + "/" + m.head.UpstreamBranch
		if m.head.UpstreamRemote == "" {
			return m, notifyAppCmd("No upstream configured for autosquash", notification.Warning)
		}
		return m, autosquashCmd(m.repo, opts, target)

	default:
		return m, notifyAppCmd("Unknown rebase action: "+result.Action, notification.Warning)
	}
}

// handleRebaseInProgressAction handles rebase actions when a rebase is already in progress.
func handleRebaseInProgressAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "r": // Continue
		return m, rebaseContinueCmd(m.repo)
	case "s": // Skip
		return m, rebaseSkipCmd(m.repo)
	case "e": // Edit todo
		return m, func() tea.Msg {
			return popup.OpenRebaseEditorMsg{}
		}
	case "a": // Abort
		return m, rebaseAbortCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown rebase action: "+result.Action, notification.Warning)
	}
}

// buildRebaseOpts builds RebaseOpts from popup result switches and options.
func buildRebaseOpts(result popup.Result) git.RebaseOpts {
	return git.RebaseOpts{
		Interactive:               result.Switches["interactive"],
		Autosquash:                result.Switches["autosquash"],
		Autostash:                 result.Switches["autostash"],
		KeepEmpty:                 result.Switches["keep-empty"],
		UpdateRefs:                result.Switches["update-refs"],
		NoVerify:                  result.Switches["no-verify"],
		CommitterDateIsAuthorDate: result.Switches["committer-date-is-author-date"],
		IgnoreDate:                result.Switches["ignore-date"],
		RebaseMerges:              result.Options["rebase-merges"],
		GpgSign:                   result.Options["gpg-sign"],
	}
}

// openRebaseCommitSelect opens the commit select view for rebase actions that need a target commit.
func openRebaseCommitSelect(m Model, opts git.RebaseOpts, kind rebaseSpecialKind) (tea.Model, tea.Cmd) {
	m.rebaseSpecialOpts = opts
	m.rebaseSpecialKind = kind
	return m, loadCommitsForSelectCmd(m.repo)
}

// openCommitSelect initiates the commit select flow for special commit actions.
// It fetches recent commits; once loaded, the app switches to the commit select screen.
func openCommitSelect(m Model, opts git.CommitOpts, kind commitSpecialKind) (tea.Model, tea.Cmd) {
	m.commitSpecialOpts = opts
	m.commitSpecialKind = kind
	return m, loadCommitsForSelectCmd(m.repo)
}

// handleCommitSelected handles the user selecting a commit in the commit select view.
func handleCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	// Check if this is a rebase commit selection
	if m.rebaseSpecialKind != rebaseSpecialNone {
		return handleRebaseCommitSelected(m, msg)
	}

	// Check if this is a cherry-pick commit selection
	if m.cherryPickActionKind != cherryPickActionNone {
		return handleCherryPickCommitSelected(m, msg)
	}

	// Check if this is a revert commit selection
	if m.revertActionKind != revertActionNone {
		return handleRevertCommitSelected(m, msg)
	}

	// Check if this is a reset commit selection
	if m.resetActionKind != resetActionNone {
		return handleResetCommitSelected(m, msg)
	}

	// Check if this is a diff popup commit/stash selection
	if m.diffCommitKind != diffCommitNone {
		return handleDiffCommitSelected(m, msg)
	}

	opts := m.commitSpecialOpts
	kind := m.commitSpecialKind

	// Clear commit select state
	m.commitSpecialKind = commitSpecialNone
	m.commitSpecialOpts = git.CommitOpts{}

	switch kind {
	case commitSpecialFixup:
		opts.Fixup = msg.Hash
		opts.NoEdit = true
		return m, commitSpecialCmd(m.repo, opts)
	case commitSpecialSquash:
		opts.Squash = msg.Hash
		opts.NoEdit = true
		return m, commitSpecialCmd(m.repo, opts)
	case commitSpecialAugment:
		opts.Squash = msg.Hash
		return m, openCommitEditorCmd(opts, "augment")
	case commitSpecialAlter:
		opts.Fixup = "amend:" + msg.Hash
		return m, openCommitEditorCmd(opts, "alter")
	case commitSpecialRevise:
		opts.Fixup = "reword:" + msg.Hash
		return m, openCommitEditorCmd(opts, "revise")
	case commitSpecialInstantFixup:
		opts.Fixup = msg.Hash
		opts.NoEdit = true
		m.opInProgress = true
		return m, commitAndAutosquashCmd(m.repo, opts, msg.FullHash)
	case commitSpecialInstantSquash:
		opts.Squash = msg.Hash
		opts.NoEdit = true
		m.opInProgress = true
		return m, commitAndAutosquashCmd(m.repo, opts, msg.FullHash)
	default:
		return m, nil
	}
}

// handleRebaseCommitSelected handles the user selecting a commit for a rebase action.
func handleRebaseCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.rebaseSpecialOpts
	kind := m.rebaseSpecialKind

	// Clear rebase select state
	m.rebaseSpecialKind = rebaseSpecialNone
	m.rebaseSpecialOpts = git.RebaseOpts{}

	switch kind {
	case rebaseSpecialInteractive:
		opts.Interactive = true
		opts.Onto = msg.FullHash
		return m, interactiveRebaseCmd(m.repo, opts)
	case rebaseSpecialSubset:
		opts.Onto = msg.FullHash
		return m, rebaseCmd(m.repo, opts)
	case rebaseSpecialModify:
		return m, modifyCommitCmd(m.repo, msg.FullHash)
	case rebaseSpecialReword:
		return m, rewordCommitCmd(m.repo, msg.FullHash)
	case rebaseSpecialDrop:
		return m, dropCommitCmd(m.repo, msg.FullHash)
	default:
		return m, nil
	}
}

// handleOpenBranchPopup opens the branch popup.
func handleOpenBranchPopup(m Model) (tea.Model, tea.Cmd) {
	branch := m.head.Branch
	showConfig := branch != "" && !m.head.Detached
	hasUpstream := m.head.UpstreamRemote != ""
	p := popup.NewBranchPopup(m.tokens, nil, branch, showConfig, hasUpstream)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupBranch
	return m, nil
}

// handleOpenPushPopup opens the push popup.
func handleOpenPushPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewPushPopup(m.tokens, nil, popup.PushPopupParams{
		Branch:          m.head.Branch,
		IsDetached:      m.head.Detached,
		PushRemoteLabel: resolveRemoteLabel(m.head.PushRemote, m.head.PushBranch),
		UpstreamLabel:   resolveRemoteLabel(m.head.UpstreamRemote, m.head.UpstreamBranch),
	})
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupPush
	return m, nil
}

// handleOpenPullPopup opens the pull popup.
func handleOpenPullPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewPullPopup(m.tokens, nil, popup.PullPopupParams{
		Branch:          m.head.Branch,
		IsDetached:      m.head.Detached,
		PushRemoteLabel: resolveRemoteLabel(m.head.PushRemote, m.head.PushBranch),
		UpstreamLabel:   resolveRemoteLabel(m.head.UpstreamRemote, m.head.UpstreamBranch),
	})
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupPull
	return m, nil
}

// handleOpenFetchPopup opens the fetch popup.
func handleOpenFetchPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewFetchPopup(m.tokens, nil, popup.FetchPopupParams{
		PushRemoteLabel: resolveRemoteLabel(m.head.PushRemote, m.head.PushBranch),
		UpstreamLabel:   resolveRemoteLabel(m.head.UpstreamRemote, m.head.UpstreamBranch),
	})
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupFetch
	return m, nil
}

// handleOpenMergePopup opens the merge popup.
func handleOpenMergePopup(m Model) (tea.Model, tea.Cmd) {
	inMerge := isInMerge(m.sections)
	p := popup.NewMergePopup(m.tokens, nil, inMerge)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupMerge
	return m, nil
}

// handleOpenRebasePopup opens the rebase popup.
func handleOpenRebasePopup(m Model) (tea.Model, tea.Cmd) {
	inRebase := isInRebase(m.sections)
	p := popup.NewRebasePopup(m.tokens, nil, inRebase, m.head.Branch, "")
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRebase
	return m, nil
}

// handleOpenRevertPopup opens the revert popup.
func handleOpenRevertPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "revert")
	hasHunk := cursorOnHunk(m)
	p := popup.NewRevertPopup(m.tokens, nil, inProgress, hasHunk)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRevert
	return m, nil
}

// handleOpenCherryPickPopup opens the cherry-pick popup.
func handleOpenCherryPickPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "pick")
	p := popup.NewCherryPickPopup(m.tokens, nil, inProgress)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupCherryPick
	return m, nil
}

// handleOpenResetPopup opens the reset popup.
func handleOpenResetPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewResetPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupReset
	return m, nil
}

// handleOpenStashPopup opens the stash popup.
func handleOpenStashPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewStashPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupStash
	return m, nil
}

// handleOpenTagPopup opens the tag popup.
func handleOpenTagPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewTagPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupTag
	return m, nil
}

// handleOpenRemotePopup opens the remote popup.
func handleOpenRemotePopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewRemotePopup(m.tokens, nil, "origin")
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupRemote
	return m, nil
}

// handleOpenWorktreePopup opens the worktree popup.
func handleOpenWorktreePopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewWorktreePopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupWorktree
	return m, nil
}

// handleOpenBisectPopup opens the bisect popup.
func handleOpenBisectPopup(m Model) (tea.Model, tea.Cmd) {
	inProgress, finished := getBisectState(m.sections)
	p := popup.NewBisectPopup(m.tokens, nil, inProgress, finished)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupBisect
	return m, nil
}

// handleOpenIgnorePopup opens the ignore popup.
func handleOpenIgnorePopup(m Model) (tea.Model, tea.Cmd) {
	// Resolve global gitignore path (empty string if not configured)
	globalPath, _ := m.repo.GlobalIgnoreFile(context.Background())
	if globalPath != "" {
		// Make path relative to home for display, like Neogit
		if home, err := os.UserHomeDir(); err == nil {
			if rel, err := filepath.Rel(home, globalPath); err == nil {
				globalPath = "~/" + rel
			}
		}
	}
	p := popup.NewIgnorePopup(m.tokens, nil, globalPath)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupIgnore
	return m, nil
}

// handleOpenDiffPopup opens the diff popup.
func handleOpenDiffPopup(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	hasItem := item != nil
	commitSelected := item != nil && item.Commit != nil
	p := popup.NewDiffPopup(m.tokens, nil, hasItem, commitSelected)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupDiff
	return m, nil
}

// handleOpenLogPopup opens the log popup.
func handleOpenLogPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewLogPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupLog
	return m, nil
}

// handleOpenMarginPopup opens the margin popup.
func handleOpenMarginPopup(m Model) (tea.Model, tea.Cmd) {
	p := popup.NewMarginPopup(m.tokens, nil)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupMargin
	return m, nil
}

// handleOpenHelpPopup opens the help popup.
func handleOpenHelpPopup(m Model) (tea.Model, tea.Cmd) {
	keys := popup.HelpKeys{
		CommitPopup:     "c",
		BranchPopup:     "b",
		PushPopup:       "P",
		PullPopup:       "p",
		FetchPopup:      "f",
		MergePopup:      "m",
		RebasePopup:     "r",
		RevertPopup:     "v",
		CherryPickPopup: "A",
		ResetPopup:      "X",
		StashPopup:      "Z",
		TagPopup:        "t",
		RemotePopup:     "M",
		WorktreePopup:   "w",
		BisectPopup:     "B",
		IgnorePopup:     "i",
		DiffPopup:       "d",
		LogPopup:        "l",
		MarginPopup:     "L",
		Stage:           "s",
		Unstage:         "u",
		Discard:         "x",
		MoveDown:        "j",
		MoveUp:          "k",
		Close:           "q",
		Refresh:         "C-r",
		NextSection:     "C-n",
		PrevSection:     "C-p",
		ToggleFold:      "tab",
	}
	p := popup.NewHelpPopup(m.tokens, keys)
	p.SetSize(m.width, m.height)
	m.popup = &p
	m.popupKind = PopupHelp
	return m, nil
}

// openPopupByName opens a popup by its string name (used by commit view).
func openPopupByName(m Model, name string) (tea.Model, tea.Cmd) {
	switch name {
	case "commit":
		return handleOpenCommitPopup(m)
	case "branch":
		return handleOpenBranchPopup(m)
	case "push":
		return handleOpenPushPopup(m)
	case "pull":
		return handleOpenPullPopup(m)
	case "fetch":
		return handleOpenFetchPopup(m)
	case "merge":
		return handleOpenMergePopup(m)
	case "rebase":
		return handleOpenRebasePopup(m)
	case "revert":
		return handleOpenRevertPopup(m)
	case "cherry-pick":
		return handleOpenCherryPickPopup(m)
	case "reset":
		return handleOpenResetPopup(m)
	case "stash":
		return handleOpenStashPopup(m)
	case "tag":
		return handleOpenTagPopup(m)
	case "remote":
		return handleOpenRemotePopup(m)
	case "worktree":
		return handleOpenWorktreePopup(m)
	case "bisect":
		return handleOpenBisectPopup(m)
	case "ignore":
		return handleOpenIgnorePopup(m)
	case "diff":
		return handleOpenDiffPopup(m)
	case "log":
		return handleOpenLogPopup(m)
	}
	return m, nil
}

// isInMerge checks if there's an active merge.
func isInMerge(sections []Section) bool {
	for _, s := range sections {
		if s.Kind == SectionSequencer && len(s.Items) > 0 {
			for _, item := range s.Items {
				if item.Action == "merge" {
					return true
				}
			}
		}
	}
	return false
}

// isInRebase checks if there's an active rebase.
func isInRebase(sections []Section) bool {
	for _, s := range sections {
		if s.Kind == SectionRebase && len(s.Items) > 0 {
			return true
		}
	}
	return false
}

// isInSequencer checks if there's an active sequencer operation of the given type.
// resolveRemoteLabel builds "remote/branch" from the two parts.
// Returns empty string if either is empty, so callers fall back to the default label.
func resolveRemoteLabel(remote, branch string) string {
	if remote == "" || branch == "" {
		return ""
	}
	return remote + "/" + branch
}

func isInSequencer(sections []Section, action string) bool {
	for _, s := range sections {
		if s.Kind == SectionSequencer && len(s.Items) > 0 {
			for _, item := range s.Items {
				if item.Action == action {
					return true
				}
			}
		}
	}
	return false
}

// getBisectState returns whether bisect is in progress and if it's finished.
func getBisectState(sections []Section) (inProgress, finished bool) {
	for _, s := range sections {
		if s.Kind == SectionBisect && len(s.Items) > 0 {
			inProgress = true
			// Check if finished (implementation would check git bisect state)
			finished = false
			return
		}
	}
	return false, false
}

// handlePushPopupAction handles actions from the push popup.
func handlePushPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "C": // Configure — open branch select to pick branch to configure
		m.branchActionKind = branchActionBranchConfigure
		return m, loadLocalBranchesCmd(m.repo)
	case "e": // Elsewhere — select remote/branch to push to
		m.branchActionKind = branchActionPushElsewhere
		return m, loadAllBranchesCmd(m.repo)
	case "o": // Another branch — select source branch
		m.branchActionKind = branchActionPushOther
		return m, loadLocalBranchesCmd(m.repo)
	case "r": // Explicit refspec — text input
		return openBranchInput(m, inputPromptPushRefspec, "Push refspec: ")
	case "T": // A tag — text input for tag name
		return openBranchInput(m, inputPromptPushTag, "Push tag: ")
	}

	opts := buildPushOpts(result)
	remote, branch, setUpstream := resolvePushTarget(result.Action, m.head, m.repo)
	opts.Remote = remote
	opts.Branch = branch
	if setUpstream {
		opts.SetUpstream = true
	}
	applyPushActionOverrides(result.Action, &opts)

	if remote == "" {
		return m, notifyAppCmd("No remote configured for push", notification.Warning)
	}

	notifyMsg := "Pushing to " + remote + "/" + branch + "..."
	if opts.Tags && branch == "" {
		notifyMsg = "Pushing tags to " + remote + "..."
	}

	return m, tea.Batch(
		pushCmd(m.repo, opts),
		notifyAppCmd(notifyMsg, notification.Info),
	)
}

// buildPushOpts builds PushOpts from popup result switches.
func buildPushOpts(result popup.Result) git.PushOpts {
	return git.PushOpts{
		ForceWithLease: result.Switches["force-with-lease"],
		Force:          result.Switches["force"],
		NoVerify:       result.Switches["no-verify"],
		DryRun:         result.Switches["dry-run"],
		SetUpstream:    result.Switches["set-upstream"],
		Tags:           result.Switches["tags"],
		FollowTags:     result.Switches["follow-tags"],
	}
}

// applyPushActionOverrides applies action-specific overrides to push opts.
// Some actions imply certain options regardless of switch state.
func applyPushActionOverrides(action string, opts *git.PushOpts) {
	switch action {
	case "t": // all tags - implies Tags option
		opts.Tags = true
	}
}

// resolvePushTarget returns the remote and branch for a push action key.
// When the action is "p" or "u" and no remote is configured, it attempts to
// resolve a sensible default remote and signals that --set-upstream should be
// used so the upstream tracking branch is created automatically.
func resolvePushTarget(action string, head HeadState, repo *git.Repository) (remote, branch string, setUpstream bool) {
	switch action {
	case "p": // pushRemote
		remote = head.PushRemote
		if remote == "" {
			remote, _ = repo.SmartDefaultRemote(context.Background())
			setUpstream = true
		}
		return remote, head.Branch, setUpstream
	case "u": // @{upstream}
		remote = head.UpstreamRemote
		if remote == "" {
			remote, _ = repo.SmartDefaultRemote(context.Background())
			setUpstream = true
		}
		return remote, head.Branch, setUpstream
	case "t": // all tags
		return defaultRemote(head), "", false
	case "m": // matching branches
		return defaultRemote(head), "", false
	default:
		return defaultRemote(head), head.Branch, false
	}
}

// defaultRemote returns the best remote to use for push operations.
func defaultRemote(head HeadState) string {
	if head.PushRemote != "" {
		return head.PushRemote
	}
	if head.UpstreamRemote != "" {
		return head.UpstreamRemote
	}
	return "origin"
}

// pushCmd creates a command that executes a git push.
func pushCmd(repo *git.Repository, opts git.PushOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Push"}
		}
		var err error
		if opts.Tags && opts.Branch == "" {
			err = repo.PushTags(context.Background(), opts.Remote)
		} else {
			err = repo.Push(context.Background(), opts)
		}
		return operationDoneMsg{err: err, op: "Push"}
	}
}

// handlePullPopupAction handles actions from the pull popup.
func handlePullPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildPullOpts(result)
	remote, branch := resolvePullTarget(result.Action, m.head)
	opts.Remote = remote
	opts.Branch = branch

	if remote == "" {
		return m, notifyAppCmd("No remote configured for pull", notification.Warning)
	}

	return m, tea.Batch(
		pullCmd(m.repo, opts),
		notifyAppCmd("Pulling from "+remote+"/"+branch+"...", notification.Info),
	)
}

// buildPullOpts builds PullOpts from popup result switches.
func buildPullOpts(result popup.Result) git.PullOpts {
	return git.PullOpts{
		Rebase:    result.Switches["rebase"],
		FFOnly:    result.Switches["ff-only"],
		Tags:      result.Switches["tags"],
		Autostash: result.Switches["autostash"],
	}
}

// resolvePullTarget returns the remote and branch for a pull action key.
func resolvePullTarget(action string, head HeadState) (remote, branch string) {
	switch action {
	case "p": // pushRemote
		return head.PushRemote, head.Branch
	case "u": // @{upstream}
		return head.UpstreamRemote, head.Branch
	default:
		return defaultRemote(head), head.Branch
	}
}

// pullCmd creates a command that executes a git pull.
func pullCmd(repo *git.Repository, opts git.PullOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Pull"}
		}
		err := repo.Pull(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Pull"}
	}
}

// handleFetchPopupAction handles actions from the fetch popup.
func handleFetchPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildFetchOpts(result)

	switch result.Action {
	case "p": // pushRemote
		opts.Remote = m.head.PushRemote
	case "u": // upstream
		opts.Remote = m.head.UpstreamRemote
	default:
		opts.Remote = defaultRemote(m.head)
	}

	if opts.Remote == "" {
		return m, notifyAppCmd("No remote configured for fetch", notification.Warning)
	}

	return m, tea.Batch(
		fetchCmd(m.repo, opts),
		notifyAppCmd("Fetching from "+opts.Remote+"...", notification.Info),
	)
}

// buildFetchOpts builds FetchOpts from popup result switches.
func buildFetchOpts(result popup.Result) git.FetchOpts {
	return git.FetchOpts{
		Prune: result.Switches["prune"],
		Tags:  result.Switches["tags"],
	}
}

// fetchCmd creates a command that executes a git fetch.
func fetchCmd(repo *git.Repository, opts git.FetchOpts) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return operationDoneMsg{err: fmt.Errorf("no repository"), op: "Fetch"}
		}
		err := repo.Fetch(context.Background(), opts)
		return operationDoneMsg{err: err, op: "Fetch"}
	}
}

// handleLogPopupAction handles actions from the log popup.
func handleLogPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildLogOpts(result)

	// Log actions
	switch result.Action {
	case "l": // current branch
		branch := m.head.Branch
		if branch == "" {
			branch = "HEAD"
		}
		return m, loadLogCmd(m.repo, opts, branch)

	case "h": // HEAD
		return m, loadLogCmd(m.repo, opts, "HEAD")

	case "u": // related (upstream)
		if m.head.UpstreamRemote == "" {
			return m, notifyAppCmd("No upstream configured", notification.Warning)
		}
		branch := m.head.UpstreamRemote + "/" + m.head.UpstreamBranch
		return m, loadLogCmd(m.repo, opts, branch)

	case "L": // local branches
		opts.All = false
		opts.Branch = ""
		return m, loadLogCmd(m.repo, opts, m.head.Branch)

	case "b": // all branches
		opts.All = true
		return m, loadLogCmd(m.repo, opts, "")

	case "a": // all references
		opts.All = true
		opts.Decorate = true
		return m, loadLogCmd(m.repo, opts, "")

	// Reflog actions
	case "r": // current branch reflog
		branch := m.head.Branch
		if branch == "" {
			branch = "HEAD"
		}
		return m, loadReflogCmd(m.repo, branch)

	case "H": // HEAD reflog
		return m, loadReflogCmd(m.repo, "HEAD")

	case "O": // other reflog — prompt for ref
		return openBranchInput(m, inputPromptReflogRef, "Reflog for ref: ")

	case "o": // other branch log — open branch select
		m.branchActionKind = branchActionLogOtherBranch
		return m, loadAllBranchesCmd(m.repo)

	default:
		return m, notifyAppCmd("Unknown log action: "+result.Action, notification.Warning)
	}
}

// buildLogOpts builds LogOpts from popup result switches and options.
func buildLogOpts(result popup.Result) git.LogOpts {
	maxCount := 256
	if maxStr, ok := result.Options["max-count"]; ok && maxStr != "" {
		if n, err := parseMaxCount(maxStr); err == nil {
			maxCount = n
		}
	}

	return git.LogOpts{
		MaxCount:    maxCount,
		Author:      result.Options["author"],
		Grep:        result.Options["grep"],
		Since:       result.Options["since"],
		Until:       result.Options["until"],
		NoMerges:    result.Switches["no-merges"],
		FirstParent: result.Switches["first-parent"],
		Reverse:     result.Switches["reverse"],
		Graph:       result.Switches["graph"],
		Decorate:    result.Switches["decorate"],
	}
}

// parseMaxCount parses the max-count string to an int.
func parseMaxCount(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// loadLogCmd loads commits and opens the log view.
func loadLogCmd(repo *git.Repository, opts git.LogOpts, branch string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return notification.NotifyMsg{Message: "No repository", Kind: notification.Error}
		}

		opts.Branch = branch
		opts.Decorate = true // Always show decorations in log view

		commits, hasMore, err := repo.Log(context.Background(), opts)
		if err != nil {
			return notification.NotifyMsg{Message: "Failed to load log: " + err.Error(), Kind: notification.Error}
		}

		return OpenLogViewMsg{
			Commits: commits,
			HasMore: hasMore,
			Branch:  branch,
			Opts:    &opts,
		}
	}
}

// loadReflogCmd loads reflog entries and opens the reflog view.
func loadReflogCmd(repo *git.Repository, ref string) tea.Cmd {
	return func() tea.Msg {
		if repo == nil {
			return notification.NotifyMsg{Message: "No repository", Kind: notification.Error}
		}

		entries, err := repo.Reflog(context.Background(), ref, 256)
		if err != nil {
			return notification.NotifyMsg{Message: "Failed to load reflog: " + err.Error(), Kind: notification.Error}
		}

		return OpenReflogViewMsg{
			Entries: entries,
			Ref:     ref,
		}
	}
}

// --- Branch popup action handling ---

// handleBranchPopupAction handles actions from the branch popup.
func handleBranchPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	// Branch selection actions
	case "b": // checkout branch/revision
		m.branchActionKind = branchActionCheckout
		return m, loadAllBranchesCmd(m.repo)
	case "l": // checkout local branch
		m.branchActionKind = branchActionCheckoutLocal
		return m, loadLocalBranchesCmd(m.repo)
	case "r": // checkout recent branch
		m.branchActionKind = branchActionCheckoutRecent
		return m, loadRecentBranchesCmd(m.repo)
	case "D": // delete
		m.branchActionKind = branchActionDelete
		return m, loadLocalBranchesCmd(m.repo)

	// Text input actions
	case "c": // new branch + checkout
		return openBranchInput(m, inputPromptNewBranchCheckout, "Create and checkout branch: ")
	case "n": // new branch no checkout
		return openBranchInput(m, inputPromptNewBranch, "Create branch: ")
	case "s": // spin-off
		return openBranchInput(m, inputPromptSpinOff, "Spin-off branch name: ")
	case "S": // spin-out
		return openBranchInput(m, inputPromptSpinOut, "Spin-out branch name: ")
	case "m": // rename
		return openBranchInput(m, inputPromptRename, "Rename "+m.head.Branch+" to: ")

	// Immediate actions
	case "X": // reset to upstream
		if m.head.UpstreamRemote == "" {
			return m, notifyAppCmd("No upstream configured for "+m.head.Branch, notification.Warning)
		}
		return m, resetBranchToUpstreamCmd(m.repo)

	case "w", "W": // Worktree — prompt for path
		return openBranchInput(m, inputPromptWorktreePath, "Worktree path: ")
	case "C": // Configure — select branch to configure
		m.branchActionKind = branchActionBranchConfigure
		return m, loadLocalBranchesCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown branch action: "+result.Action, notification.Warning)
	}
}

// handleBranchSelected processes a branch selection result.
func handleBranchSelected(m Model, msg branchselect.SelectedMsg) (tea.Model, tea.Cmd) {
	kind := m.branchActionKind
	m.branchActionKind = branchActionNone

	switch kind {
	case branchActionCheckout, branchActionCheckoutLocal, branchActionCheckoutRecent:
		return m, checkoutBranchCmd(m.repo, msg.Name)
	case branchActionDelete:
		return m, deleteBranchCmd(m.repo, msg.Name)
	case branchActionPushElsewhere:
		opts := buildPushOpts(popup.Result{Switches: map[string]bool{}, Options: map[string]string{}})
		opts.Remote = msg.Name
		opts.Branch = m.head.Branch
		return m, pushCmd(m.repo, opts)
	case branchActionPushOther:
		opts := buildPushOpts(popup.Result{Switches: map[string]bool{}, Options: map[string]string{}})
		remote, _ := m.repo.SmartDefaultRemote(context.Background())
		opts.Remote = remote
		opts.Branch = msg.Name
		return m, pushCmd(m.repo, opts)
	case branchActionRebaseElsewhere:
		opts := m.rebaseSpecialOpts
		opts.Onto = msg.Name
		m.rebaseSpecialOpts = git.RebaseOpts{}
		return m, rebaseCmd(m.repo, opts)
	case branchActionLogOtherBranch:
		logOpts := git.LogOpts{MaxCount: 256, Decorate: true}
		return m, loadLogCmd(m.repo, logOpts, msg.Name)
	case branchActionBranchConfigure:
		return m, openBranchConfigCmd(m.repo, msg.Name)
	case branchActionMergeBranch:
		opts := m.mergeOpts
		opts.Branch = msg.Name
		kind := m.mergeActionKind
		m.mergeActionKind = mergeActionNone
		m.mergeOpts = git.MergeOpts{}
		switch kind {
		case mergeActionEdit:
			// merge + edit: just pass through, the editor will open
		case mergeActionNoCommit:
			opts.NoCommit = true
		case mergeActionSquash:
			opts.Squash = true
		case mergeActionDissolve:
			opts.Squash = true
			opts.NoCommit = true
		default:
			// mergeActionMerge, mergeActionAbsorb — default merge
		}
		return m, tea.Batch(
			mergeCmd(m.repo, opts),
			notifyAppCmd("Merging "+msg.Name+"...", notification.Info),
		)
	case branchActionWorktreeCheckout:
		return m, worktreeAddCmd(m.repo, "", msg.Name)
	case branchActionDonate:
		hashes := m.donateHashes
		opts := m.cherryPickOpts
		m.donateHashes = nil
		m.cherryPickOpts = git.CherryPickOpts{}
		return m, cherryPickDonateCmd(m.repo, hashes, m.head.Branch, msg.Name, opts)
	case branchActionMergePreview:
		return m, mergePreviewCmd(m.repo, msg.Name)
	case branchActionDiffRangeFrom:
		m.diffRangeFrom = msg.Name
		m.branchActionKind = branchActionDiffRangeTo
		return m, loadAllBranchesCmd(m.repo)
	case branchActionDiffRangeTo:
		rangeSpec := m.diffRangeFrom + ".." + msg.Name
		m.diffRangeFrom = ""
		return m, func() tea.Msg {
			return diffview.OpenDiffViewMsg{
				Source: diffview.DiffSource{Kind: git.DiffRange, Range: rangeSpec},
			}
		}
	default:
		return m, nil
	}
}

// --- Merge popup action handling ---

// handleMergePopupAction handles actions from the merge popup.
func handleMergePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inMerge := isInMerge(m.sections)

	if inMerge {
		switch result.Action {
		case "m": // Commit merge
			return m, mergeCommitCmd(m.repo)
		case "a": // Abort merge
			return m, mergeAbortCmd(m.repo)
		default:
			return m, notifyAppCmd("Unknown merge action: "+result.Action, notification.Warning)
		}
	}

	opts := buildMergeOpts(result)

	var kind mergeActionKind
	switch result.Action {
	case "m":
		kind = mergeActionMerge
	case "e":
		kind = mergeActionEdit
	case "n":
		kind = mergeActionNoCommit
	case "a":
		kind = mergeActionAbsorb
	case "p": // Preview merge
		m.mergeActionKind = mergeActionNone
		m.branchActionKind = branchActionMergePreview
		return m, loadAllBranchesCmd(m.repo)
	case "s":
		kind = mergeActionSquash
	case "i":
		kind = mergeActionDissolve
	default:
		return m, notifyAppCmd("Unknown merge action: "+result.Action, notification.Warning)
	}

	m.mergeActionKind = kind
	m.mergeOpts = opts
	m.branchActionKind = branchActionMergeBranch
	return m, loadAllBranchesCmd(m.repo)
}

// buildMergeOpts builds MergeOpts from popup result switches and options.
func buildMergeOpts(result popup.Result) git.MergeOpts {
	return git.MergeOpts{
		FFOnly:         result.Switches["ff-only"],
		NoFF:           result.Switches["no-ff"],
		Strategy:       result.Options["strategy"],
		StrategyOption: result.Options["strategy-option"],
		DiffAlgorithm:  result.Options["Xdiff-algorithm"],
		GpgSign:        result.Options["gpg-sign"],
	}
}

// --- Cherry-pick popup action handling ---

// handleCherryPickPopupAction handles actions from the cherry-pick popup.
func handleCherryPickPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "pick")

	if inProgress {
		switch result.Action {
		case "A": // Continue
			return m, cherryPickContinueCmd(m.repo)
		case "s": // Skip
			return m, cherryPickSkipCmd(m.repo)
		case "a": // Abort
			return m, cherryPickAbortCmd(m.repo)
		default:
			return m, notifyAppCmd("Unknown cherry-pick action: "+result.Action, notification.Warning)
		}
	}

	opts := buildCherryPickOpts(result)

	var kind cherryPickActionKind
	switch result.Action {
	case "A":
		kind = cherryPickActionPick
	case "a":
		kind = cherryPickActionApply
	case "h":
		kind = cherryPickActionHarvest
	case "m":
		kind = cherryPickActionSquash
	case "d":
		kind = cherryPickActionDonate
	case "n":
		kind = cherryPickActionSpinout
	case "s":
		kind = cherryPickActionSpinoff
	default:
		return m, notifyAppCmd("Unknown cherry-pick action: "+result.Action, notification.Warning)
	}

	m.cherryPickActionKind = kind
	m.cherryPickOpts = opts
	return m, loadCommitsForSelectCmd(m.repo)
}

// buildCherryPickOpts builds CherryPickOpts from popup result switches and options.
func buildCherryPickOpts(result popup.Result) git.CherryPickOpts {
	mainline := 0
	if ml, ok := result.Options["mainline"]; ok && ml != "" {
		if n, err := parseMaxCount(ml); err == nil {
			mainline = n
		}
	}

	return git.CherryPickOpts{
		Mainline:           mainline,
		Strategy:           result.Options["strategy"],
		GpgSign:            result.Options["gpg-sign"],
		FF:                 result.Switches["ff"],
		ReferenceInMessage: result.Switches["x"],
		Edit:               result.Switches["edit"],
		Signoff:            result.Switches["signoff"],
	}
}

// handleCherryPickCommitSelected handles the user selecting a commit for a cherry-pick action.
func handleCherryPickCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.cherryPickOpts
	kind := m.cherryPickActionKind

	// Clear cherry-pick select state
	m.cherryPickActionKind = cherryPickActionNone
	m.cherryPickOpts = git.CherryPickOpts{}

	hashes := []string{msg.FullHash}

	switch kind {
	case cherryPickActionPick:
		return m, cherryPickCmd(m.repo, hashes, opts)
	case cherryPickActionApply:
		return m, cherryPickApplyCmd(m.repo, hashes, opts)
	case cherryPickActionHarvest:
		// Harvest: cherry-pick but don't remove from source
		return m, cherryPickCmd(m.repo, hashes, opts)
	case cherryPickActionSquash:
		// Squash: cherry-pick --no-commit (apply changes without commit)
		return m, cherryPickApplyCmd(m.repo, hashes, opts)
	case cherryPickActionDonate:
		m.donateHashes = hashes
		m.cherryPickOpts = opts
		m.branchActionKind = branchActionDonate
		return m, loadAllBranchesCmd(m.repo)
	case cherryPickActionSpinout:
		// Spinout: create new branch at HEAD, cherry-pick commits onto it
		m.cherryPickOpts = opts
		m.donateHashes = hashes
		return openBranchInput(m, inputPromptCherryPickSpinout, "New branch name: ")
	case cherryPickActionSpinoff:
		// Spinoff: like spinout but also reset current branch back
		m.cherryPickOpts = opts
		m.donateHashes = hashes
		return openBranchInput(m, inputPromptCherryPickSpinoff, "New branch name: ")
	default:
		return m, nil
	}
}

// --- Revert popup action handling ---

// handleRevertPopupAction handles actions from the revert popup.
func handleRevertPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	inProgress := isInSequencer(m.sections, "revert")

	if inProgress {
		switch result.Action {
		case "v": // Continue
			return m, revertContinueCmd(m.repo)
		case "s": // Skip
			return m, revertSkipCmd(m.repo)
		case "a": // Abort
			return m, revertAbortCmd(m.repo)
		default:
			return m, notifyAppCmd("Unknown revert action: "+result.Action, notification.Warning)
		}
	}

	opts := buildRevertOpts(result)

	switch result.Action {
	case "v": // Commit(s) — needs commit select
		m.revertActionKind = revertActionCommit
		m.revertOpts = opts
		return m, loadCommitsForSelectCmd(m.repo)
	case "V": // Changes (no commit) — needs commit select
		m.revertActionKind = revertActionChanges
		m.revertOpts = opts
		return m, loadCommitsForSelectCmd(m.repo)
	case "h": // Hunk — revert the current hunk
		return handleRevertHunk(m)
	default:
		return m, notifyAppCmd("Unknown revert action: "+result.Action, notification.Warning)
	}
}

// buildRevertOpts builds RevertOpts from popup result switches and options.
func buildRevertOpts(result popup.Result) git.RevertOpts {
	mainline := 0
	if ml, ok := result.Options["mainline"]; ok && ml != "" {
		if n, err := parseMaxCount(ml); err == nil {
			mainline = n
		}
	}

	return git.RevertOpts{
		Mainline: mainline,
		Strategy: result.Options["strategy"],
		GpgSign:  result.Options["gpg-sign"],
		Edit:     result.Switches["edit"],
		NoEdit:   result.Switches["no-edit"],
		Signoff:  result.Switches["signoff"],
	}
}

// handleRevertCommitSelected handles the user selecting a commit for a revert action.
func handleRevertCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	opts := m.revertOpts
	kind := m.revertActionKind

	// Clear revert select state
	m.revertActionKind = revertActionNone
	m.revertOpts = git.RevertOpts{}

	hashes := []string{msg.FullHash}

	switch kind {
	case revertActionCommit:
		return m, revertCmd(m.repo, hashes, opts)
	case revertActionChanges:
		return m, revertChangesCmd(m.repo, hashes, opts)
	default:
		return m, nil
	}
}

// --- Stash popup action handling ---

// handleStashPopupAction handles actions from the stash popup.
func handleStashPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildStashOpts(result)

	switch result.Action {
	// Stash group
	case "z": // both
		return m, stashPushCmd(m.repo, opts)
	case "i": // index
		opts.KeepIndex = true
		return m, stashPushCmd(m.repo, opts)
	case "w": // worktree
		// Stash only worktree changes (keep index)
		opts.KeepIndex = true
		return m, stashPushCmd(m.repo, opts)
	case "x": // keeping index
		opts.KeepIndex = true
		return m, stashPushCmd(m.repo, opts)
	case "P": // push (with message)
		return openBranchInput(m, inputPromptStashMessage, "Stash message: ")

	// Snapshot group
	case "Z": // snapshot both
		return m, stashSnapshotCmd(m.repo, opts, "snapshot")
	case "I": // snapshot index
		opts.KeepIndex = true
		return m, stashSnapshotCmd(m.repo, opts, "index snapshot")
	case "W": // snapshot worktree
		return m, stashSnapshotCmd(m.repo, opts, "worktree snapshot")
	case "r": // to wip ref
		return m, stashWipRefCmd(m.repo, opts)

	// Use group
	case "p": // pop
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashPopCmd(m.repo, idx)
	case "a": // apply
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashApplyCmd(m.repo, idx)
	case "d": // drop
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashDropCmd(m.repo, idx)

	// Inspect group
	case "l": // list — open stash list view
		return m, loadStashListCmd(m.repo)
	case "v": // show — open commit view for the stash
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, openStashInCommitViewCmd(m.repo, idx)

	// Transform group
	case "b": // branch
		return openBranchInput(m, inputPromptStashBranch, "Stash branch name: ")
	case "B": // branch here
		return openBranchInput(m, inputPromptStashBranch, "Stash branch name: ")
	case "m": // rename
		return openBranchInput(m, inputPromptStashRename, "New stash message: ")
	case "f": // format patch
		idx, ok := getStashIndex(m)
		if !ok {
			idx = 0
		}
		return m, stashFormatPatchCmd(m.repo, idx)

	default:
		return m, notifyAppCmd("Unknown stash action: "+result.Action, notification.Warning)
	}
}

// buildStashOpts builds StashOpts from popup result switches.
func buildStashOpts(result popup.Result) git.StashOpts {
	return git.StashOpts{
		IncludeUntracked: result.Switches["include-untracked"],
		All:              result.Switches["all"],
	}
}

// --- Reset popup action handling ---

// handleResetPopupAction handles actions from the reset popup.
func handleResetPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "f": // file — reset the file under cursor
		path, ok := getCursorFilePath(m)
		if !ok {
			return m, notifyAppCmd("No file selected", notification.Warning)
		}
		return m, resetFileCmd(m.repo, path)
	case "b": // branch — select commit to reset branch to
		m.resetActionKind = resetActionBranch
		m.resetMode = git.ResetMixed
		return m, loadCommitsForSelectCmd(m.repo)
	default:
		// m/s/h/k/i/w — reset modes, need commit select for target
		mode, ok := resetModeForAction(result.Action)
		if !ok {
			return m, notifyAppCmd("Unknown reset action: "+result.Action, notification.Warning)
		}
		m.resetActionKind = resetActionBranch
		m.resetMode = mode
		return m, loadCommitsForSelectCmd(m.repo)
	}
}

// resetModeForAction maps a reset popup action key to a git.ResetMode.
func resetModeForAction(action string) (git.ResetMode, bool) {
	switch action {
	case "m":
		return git.ResetMixed, true
	case "s":
		return git.ResetSoft, true
	case "h":
		return git.ResetHard, true
	case "k":
		return git.ResetKeep, true
	case "i":
		return git.ResetIndex, true
	case "w":
		return git.ResetWorktree, true
	default:
		return "", false
	}
}

// handleResetCommitSelected handles the user selecting a commit for a reset action.
func handleResetCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	mode := m.resetMode

	// Clear reset select state
	m.resetActionKind = resetActionNone
	m.resetMode = ""

	return m, resetCmd(m.repo, msg.FullHash, mode)
}

// --- Tag popup action handling ---

// handleTagPopupAction handles actions from the tag popup.
func handleTagPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildTagOpts(result)
	m.tagOpts = opts

	switch result.Action {
	case "t": // create tag
		return openBranchInput(m, inputPromptTagName, "Tag name: ")
	case "r": // release tag
		opts.Annotate = true
		m.tagOpts = opts
		return openBranchInput(m, inputPromptTagRelease, "Release tag name: ")
	case "x": // delete tag
		return openBranchInput(m, inputPromptTagDelete, "Delete tag: ")
	case "p": // prune
		remote := defaultRemote(m.head)
		return m, tagPruneCmd(m.repo, remote)
	default:
		return m, notifyAppCmd("Unknown tag action: "+result.Action, notification.Warning)
	}
}

// buildTagOpts builds TagOpts from popup result switches and options.
func buildTagOpts(result popup.Result) git.TagOpts {
	return git.TagOpts{
		Force:     result.Switches["force"],
		Annotate:  result.Switches["annotate"],
		Sign:      result.Switches["sign"],
		LocalUser: result.Options["local-user"],
	}
}

// --- Remote popup action handling ---

// handleRemotePopupAction handles actions from the remote popup.
func handleRemotePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "a": // Add
		return openBranchInput(m, inputPromptRemoteName, "Remote name: ")
	case "r": // Rename
		return openBranchInput(m, inputPromptRemoteRename, "Rename remote to: ")
	case "x": // Remove
		return openBranchInput(m, inputPromptRemoteRemove, "Remove remote: ")
	case "C": // Configure
		return openBranchInput(m, inputPromptRemoteConfigure, "Configure remote: ")
	case "p": // Prune stale branches
		return openBranchInput(m, inputPromptRemotePrune, "Prune remote: ")
	case "P": // Prune stale refspecs
		return openBranchInput(m, inputPromptRemotePrune, "Prune refspecs for remote: ")
	case "b": // Update default branch
		return openBranchInput(m, inputPromptRemoteSetHead, "Remote for set-head: ")
	case "z": // Unshallow
		return m, fetchUnshallowCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown remote action: "+result.Action, notification.Warning)
	}
}

// --- Worktree popup action handling ---

// handleWorktreePopupAction handles actions from the worktree popup.
func handleWorktreePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "w": // Checkout — select branch for worktree
		m.branchActionKind = branchActionWorktreeCheckout
		return m, loadAllBranchesCmd(m.repo)
	case "W": // Create — prompt for path
		return openBranchInput(m, inputPromptWorktreeCreate, "Worktree path: ")
	case "g": // Goto — prompt for path
		return m, notifyAppCmd("Goto worktree: switch not supported in terminal", notification.Info)
	case "m": // Move — prompt for destination
		return openBranchInput(m, inputPromptWorktreeMove, "Move worktree to: ")
	case "D": // Delete — prompt for path
		return openBranchInput(m, inputPromptWorktreeDelete, "Delete worktree path: ")
	default:
		return m, notifyAppCmd("Unknown worktree action: "+result.Action, notification.Warning)
	}
}

// --- Bisect popup action handling ---

// handleBisectPopupAction handles actions from the bisect popup.
func handleBisectPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	opts := buildBisectOpts(result)

	switch result.Action {
	case "B": // Start
		return m, bisectStartCmd(m.repo, opts)
	case "S": // Scripted / Run script
		return openBranchInput(m, inputPromptBisectScript, "Bisect script: ")
	case "b": // Bad
		hash := getCommitHashAtCursor(m)
		return m, bisectBadCmd(m.repo, hash)
	case "g": // Good
		hash := getCommitHashAtCursor(m)
		return m, bisectGoodCmd(m.repo, hash)
	case "s": // Skip
		hash := getCommitHashAtCursor(m)
		return m, bisectSkipCmd(m.repo, hash)
	case "r": // Reset
		return m, bisectResetCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown bisect action: "+result.Action, notification.Warning)
	}
}

// buildBisectOpts builds BisectOpts from popup result switches.
func buildBisectOpts(result popup.Result) git.BisectOpts {
	return git.BisectOpts{
		NoCheckout:  result.Switches["no-checkout"],
		FirstParent: result.Switches["first-parent"],
	}
}

// --- Ignore popup action handling ---

// handleIgnorePopupAction handles actions from the ignore popup.
func handleIgnorePopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	path, ok := getCursorFilePath(m)
	if !ok {
		return m, notifyAppCmd("No file selected to ignore", notification.Warning)
	}

	pattern := git.IgnorePatternForPath(path)

	var scope git.IgnoreScope
	switch result.Action {
	case "t": // shared at top-level
		scope = git.IgnoreScopeTopLevel
	case "s": // shared in sub-directory
		scope = git.IgnoreScopeSubdir
	case "p": // privately for this repository
		scope = git.IgnoreScopePrivate
	case "g": // globally for this user
		scope = git.IgnoreScopeGlobal
	default:
		return m, notifyAppCmd("Unknown ignore action: "+result.Action, notification.Warning)
	}

	return m, ignoreCmd(m.repo, pattern, scope)
}

// --- Diff popup action handling ---

// handleDiffPopupAction handles actions from the diff popup.
func handleDiffPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "d": // this — expand inline diff for current item
		return handleToggle(m)
	case "h": // this..HEAD — show diff for commit vs HEAD
		item, _ := getCurrentItem(m)
		if item != nil && item.Commit != nil {
			return m, func() tea.Msg {
				return diffview.OpenDiffViewMsg{
					Source: diffview.DiffSource{
						Kind:  git.DiffRange,
						Range: item.Commit.Hash + "..HEAD",
					},
				}
			}
		}
		return m, notifyAppCmd("No commit selected", notification.Warning)
	case "r": // range — pick two refs, then open diff for from..to
		m.branchActionKind = branchActionDiffRangeFrom
		return m, loadAllBranchesCmd(m.repo)
	case "u": // unstaged
		return m, func() tea.Msg {
			return diffview.OpenDiffViewMsg{
				Source: diffview.DiffSource{Kind: git.DiffUnstaged},
			}
		}
	case "s": // staged
		return m, func() tea.Msg {
			return diffview.OpenDiffViewMsg{
				Source: diffview.DiffSource{Kind: git.DiffStaged},
			}
		}
	case "w": // worktree
		return m, func() tea.Msg {
			return diffview.OpenDiffViewMsg{
				Source: diffview.DiffSource{Kind: git.DiffRange, Range: "HEAD"},
			}
		}
	case "c": // Commit — pick a commit, then open diff view
		m.diffCommitKind = diffCommitShow
		return m, loadCommitsForSelectCmd(m.repo)
	case "t": // Stash — pick a stash, then open diff view
		m.diffCommitKind = diffCommitStash
		return m, loadStashesForSelectCmd(m.repo)
	default:
		return m, notifyAppCmd("Unknown diff action: "+result.Action, notification.Warning)
	}
}

// handleDiffCommitSelected handles the user selecting a commit/stash for the diff popup.
func handleDiffCommitSelected(m Model, msg commitselect.SelectedMsg) (tea.Model, tea.Cmd) {
	kind := m.diffCommitKind
	m.diffCommitKind = diffCommitNone

	switch kind {
	case diffCommitShow:
		return m, func() tea.Msg {
			return diffview.OpenDiffViewMsg{
				Source: diffview.DiffSource{Kind: git.DiffCommit, Commit: msg.FullHash},
			}
		}
	case diffCommitStash:
		return m, func() tea.Msg {
			return diffview.OpenDiffViewMsg{
				Source: diffview.DiffSource{Kind: git.DiffStash, Stash: msg.Hash},
			}
		}
	default:
		return m, nil
	}
}

// --- Margin popup action handling ---

// handleMarginPopupAction handles actions from the margin popup.
func handleMarginPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	case "g": // Refresh buffer
		return m, loadStatusCmd(m.repo, m.cfg)
	case "L": // Toggle visibility
		m.margin.Visible = !m.margin.Visible
		label := "hidden"
		if m.margin.Visible {
			label = "visible"
		}
		return m, notifyAppCmd("Margin: "+label, notification.Info)
	case "l": // Cycle style
		m.margin.Style = (m.margin.Style + 1) % dateStyleCount
		styles := []string{"relative short", "relative long", "local datetime"}
		return m, notifyAppCmd("Margin style: "+styles[m.margin.Style], notification.Info)
	case "d": // Toggle details
		m.margin.Details = !m.margin.Details
		label := "off"
		if m.margin.Details {
			label = "on"
		}
		return m, notifyAppCmd("Margin details: "+label, notification.Info)
	case "x": // Toggle shortstat
		m.margin.Shortstat = !m.margin.Shortstat
		label := "off"
		if m.margin.Shortstat {
			label = "on"
		}
		return m, notifyAppCmd("Margin shortstat: "+label, notification.Info)
	default:
		return m, notifyAppCmd("Unknown margin action: "+result.Action, notification.Warning)
	}
}

// --- Remote config popup action handling ---

// handleRemoteConfigPopupAction handles actions from the remote config popup.
// The popup returns config key-value pairs that should be written to git config.
func handleRemoteConfigPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	if len(result.Config) == 0 {
		return m, nil
	}
	return m, setRemoteConfigCmd(m.repo, result.Config)
}

// --- Branch config popup action handling ---

// handleBranchConfigPopupAction handles actions from the branch config popup.
// The popup returns config key-value pairs that should be written to git config.
func handleBranchConfigPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	if len(result.Config) == 0 {
		return m, nil
	}
	return m, setBranchConfigCmd(m.repo, result.Config)
}

// --- Help popup action handling ---

// handleHelpPopupAction handles actions from the help popup.
// The help popup returns the key that was pressed. We dispatch that key
// to open the corresponding popup, matching Neogit behaviour where selecting
// an item in the help popup triggers that action.
func handleHelpPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	switch result.Action {
	// Popup keys — open the corresponding popup
	case "c":
		return handleOpenCommitPopup(m)
	case "b":
		return handleOpenBranchPopup(m)
	case "P":
		return handleOpenPushPopup(m)
	case "p":
		return handleOpenPullPopup(m)
	case "f":
		return handleOpenFetchPopup(m)
	case "m":
		return handleOpenMergePopup(m)
	case "r":
		return handleOpenRebasePopup(m)
	case "v":
		return handleOpenRevertPopup(m)
	case "A":
		return handleOpenCherryPickPopup(m)
	case "X":
		return handleOpenResetPopup(m)
	case "Z":
		return handleOpenStashPopup(m)
	case "t":
		return handleOpenTagPopup(m)
	case "M":
		return handleOpenRemotePopup(m)
	case "w":
		return handleOpenWorktreePopup(m)
	case "B":
		return handleOpenBisectPopup(m)
	case "i":
		return handleOpenIgnorePopup(m)
	case "d":
		return handleOpenDiffPopup(m)
	case "l":
		return handleOpenLogPopup(m)
	case "L":
		return handleOpenMarginPopup(m)

	// Stage/Unstage/Discard — these are action labels, not keys in the help popup
	// The help popup uses the actual key bindings as action keys
	case "s":
		return handleStage(m)
	case "u":
		return handleUnstage(m)
	case "x":
		return handleDiscardStart(m)

	// Essential commands — refresh is the only actionable one from help
	case "C-r":
		return m, loadStatusCmd(m.repo, m.cfg)
	default:
		// Navigation keys (j, k, tab, C-n, C-p, q) are listed for reference
		// in the help popup but don't need dispatch — they apply after the
		// popup is already closed.
		return m, nil
	}
}

// --- Yank popup action handling ---

// handleYankPopupAction handles actions from the yank popup.
// Each action copies a specific piece of information to the clipboard.
func handleYankPopupAction(m Model, result popup.Result) (tea.Model, tea.Cmd) {
	text := yankValue(m, result.Action)
	if text == "" {
		return m, notifyAppCmd("Nothing to yank", notification.Warning)
	}

	return m, func() tea.Msg {
		if err := platform.CopyToClipboard(text); err != nil {
			return notification.NotifyMsg{
				Message: "Failed to copy to clipboard: " + err.Error(),
				Kind:    notification.Error,
			}
		}
		return notification.NotifyMsg{
			Message: "Yanked: " + text,
			Kind:    notification.Info,
		}
	}
}

// yankValue determines what text to yank based on the action key.
func yankValue(m Model, action string) string {
	item, _ := getCurrentItem(m)

	switch action {
	case "Y": // Hash
		if item != nil && item.Commit != nil {
			return item.Commit.Hash
		}
		return m.head.Oid
	case "s": // Subject
		if item != nil && item.Commit != nil {
			return item.Commit.Subject
		}
		return m.head.Subject
	case "m": // Message (subject and body)
		if item != nil && item.Commit != nil {
			msg := item.Commit.Subject
			if item.Commit.Body != "" {
				msg += "\n\n" + item.Commit.Body
			}
			return msg
		}
		return m.head.Subject
	case "b": // Message body
		if item != nil && item.Commit != nil {
			return item.Commit.Body
		}
		return ""
	case "u": // URL
		if m.head.UpstreamRemote != "" && m.head.Branch != "" {
			return m.head.UpstreamRemote + "/" + m.head.Branch
		}
		return ""
	case "a": // Author
		if item != nil && item.Commit != nil {
			return item.Commit.AuthorName
		}
		return ""
	case "t": // Tags
		return m.head.Tag
	default:
		return ""
	}
}
