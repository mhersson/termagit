package popup

import (
	"github.com/mhersson/termagit/internal/theme"
)

// HelpKeys contains the key bindings to display in the help popup.
type HelpKeys struct {
	// Popups
	CommitPopup     string
	BranchPopup     string
	PushPopup       string
	PullPopup       string
	FetchPopup      string
	MergePopup      string
	RebasePopup     string
	RevertPopup     string
	CherryPickPopup string
	ResetPopup      string
	StashPopup      string
	TagPopup        string
	RemotePopup     string
	WorktreePopup   string
	BisectPopup     string
	IgnorePopup     string
	DiffPopup       string
	LogPopup        string
	MarginPopup     string

	// Actions
	Stage   string
	Unstage string
	Discard string

	// Navigation
	MoveDown    string
	MoveUp      string
	Close       string
	Refresh     string
	NextSection string
	PrevSection string
	ToggleFold  string
}

// NewHelpPopup creates the help popup matching Neogit exactly.
// Source: neogit/lua/neogit/popups/help/init.lua
// This popup shows all key bindings from config, not hardcoded values.
func NewHelpPopup(tokens theme.Tokens, keys HelpKeys) Popup {
	p := New("Help", tokens)

	// Commands group (popup keys)
	p.AddActionGroup("Commands", []Action{
		{Key: keys.CommitPopup, Label: "Commit"},
		{Key: keys.BranchPopup, Label: "Branch"},
		{Key: keys.PushPopup, Label: "Push"},
		{Key: keys.PullPopup, Label: "Pull"},
		{Key: keys.FetchPopup, Label: "Fetch"},
		{Key: keys.MergePopup, Label: "Merge"},
		{Key: keys.RebasePopup, Label: "Rebase"},
		{Key: keys.RevertPopup, Label: "Revert"},
		{Key: keys.CherryPickPopup, Label: "Cherry Pick"},
		{Key: keys.ResetPopup, Label: "Reset"},
		{Key: keys.StashPopup, Label: "Stash"},
		{Key: keys.TagPopup, Label: "Tag"},
		{Key: keys.RemotePopup, Label: "Remote"},
		{Key: keys.WorktreePopup, Label: "Worktree"},
		{Key: keys.BisectPopup, Label: "Bisect"},
		{Key: keys.IgnorePopup, Label: "Ignore"},
		{Key: keys.DiffPopup, Label: "Diff"},
		{Key: keys.LogPopup, Label: "Log"},
		{Key: keys.MarginPopup, Label: "Margin"},
	})

	// Applying changes group
	p.AddActionGroup("Applying changes", []Action{
		{Key: keys.Stage, Label: "Stage"},
		{Key: keys.Unstage, Label: "Unstage"},
		{Key: keys.Discard, Label: "Discard"},
	})

	// Essential commands group
	p.AddActionGroup("Essential commands", []Action{
		{Key: keys.MoveDown, Label: "Move down"},
		{Key: keys.MoveUp, Label: "Move up"},
		{Key: keys.ToggleFold, Label: "Toggle fold"},
		{Key: keys.NextSection, Label: "Next section"},
		{Key: keys.PrevSection, Label: "Previous section"},
		{Key: keys.Refresh, Label: "Refresh"},
		{Key: keys.Close, Label: "Close"},
	})

	return p
}
