package status

import (
	"testing"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/ui/popup"
	"github.com/stretchr/testify/assert"
)

// --- Merge opts tests ---

func TestBuildMergeOpts_Defaults(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}
	opts := buildMergeOpts(result)
	assert.False(t, opts.FFOnly)
	assert.False(t, opts.NoFF)
	assert.False(t, opts.Squash)
	assert.False(t, opts.NoCommit)
	assert.Empty(t, opts.Strategy)
	assert.Empty(t, opts.StrategyOption)
	assert.Empty(t, opts.DiffAlgorithm)
	assert.Empty(t, opts.GpgSign)
}

func TestBuildMergeOpts_AllSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"ff-only": true,
			"no-ff":   true,
		},
		Options: map[string]string{
			"strategy":        "recursive",
			"strategy-option": "theirs",
			"Xdiff-algorithm": "patience",
			"gpg-sign":        "ABCD1234",
		},
	}
	opts := buildMergeOpts(result)
	assert.True(t, opts.FFOnly)
	assert.True(t, opts.NoFF)
	assert.Equal(t, "recursive", opts.Strategy)
	assert.Equal(t, "theirs", opts.StrategyOption)
	assert.Equal(t, "patience", opts.DiffAlgorithm)
	assert.Equal(t, "ABCD1234", opts.GpgSign)
}

// --- Cherry-pick opts tests ---

func TestBuildCherryPickOpts_Defaults(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}
	opts := buildCherryPickOpts(result)
	assert.False(t, opts.FF)
	assert.False(t, opts.ReferenceInMessage)
	assert.False(t, opts.Edit)
	assert.False(t, opts.Signoff)
	assert.Empty(t, opts.Strategy)
	assert.Empty(t, opts.GpgSign)
}

func TestBuildCherryPickOpts_AllSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"ff":      true,
			"x":       true,
			"edit":    true,
			"signoff": true,
		},
		Options: map[string]string{
			"mainline": "1",
			"strategy": "recursive",
			"gpg-sign": "KEY",
		},
	}
	opts := buildCherryPickOpts(result)
	assert.True(t, opts.FF)
	assert.True(t, opts.ReferenceInMessage)
	assert.True(t, opts.Edit)
	assert.True(t, opts.Signoff)
	assert.Equal(t, 1, opts.Mainline)
	assert.Equal(t, "recursive", opts.Strategy)
	assert.Equal(t, "KEY", opts.GpgSign)
}

// --- Revert opts tests ---

func TestBuildRevertOpts_Defaults(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}
	opts := buildRevertOpts(result)
	assert.False(t, opts.Edit)
	assert.False(t, opts.NoEdit)
	assert.False(t, opts.Signoff)
	assert.Empty(t, opts.Strategy)
	assert.Empty(t, opts.GpgSign)
}

func TestBuildRevertOpts_AllSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"edit":    true,
			"no-edit": true,
			"signoff": true,
		},
		Options: map[string]string{
			"mainline": "2",
			"strategy": "ours",
			"gpg-sign": "SIGN",
		},
	}
	opts := buildRevertOpts(result)
	assert.True(t, opts.Edit)
	assert.True(t, opts.NoEdit)
	assert.True(t, opts.Signoff)
	assert.Equal(t, 2, opts.Mainline)
	assert.Equal(t, "ours", opts.Strategy)
	assert.Equal(t, "SIGN", opts.GpgSign)
}

// --- Stash opts tests ---

func TestBuildStashOpts_Defaults(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}
	opts := buildStashOpts(result)
	assert.False(t, opts.IncludeUntracked)
	assert.False(t, opts.All)
	assert.False(t, opts.KeepIndex)
	assert.Empty(t, opts.Message)
}

func TestBuildStashOpts_AllSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"include-untracked": true,
			"all":               true,
		},
		Options: map[string]string{},
	}
	opts := buildStashOpts(result)
	assert.True(t, opts.IncludeUntracked)
	assert.True(t, opts.All)
}

// --- Tag opts tests ---

func TestBuildTagOpts_Defaults(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}
	opts := buildTagOpts(result)
	assert.False(t, opts.Force)
	assert.False(t, opts.Annotate)
	assert.False(t, opts.Sign)
	assert.Empty(t, opts.LocalUser)
}

func TestBuildTagOpts_AllSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"force":    true,
			"annotate": true,
			"sign":     true,
		},
		Options: map[string]string{
			"local-user": "KEYID",
		},
	}
	opts := buildTagOpts(result)
	assert.True(t, opts.Force)
	assert.True(t, opts.Annotate)
	assert.True(t, opts.Sign)
	assert.Equal(t, "KEYID", opts.LocalUser)
}

// --- Bisect opts tests ---

func TestBuildBisectOpts_Defaults(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}
	opts := buildBisectOpts(result)
	assert.False(t, opts.NoCheckout)
	assert.False(t, opts.FirstParent)
}

func TestBuildBisectOpts_AllSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"no-checkout":  true,
			"first-parent": true,
		},
		Options: map[string]string{},
	}
	opts := buildBisectOpts(result)
	assert.True(t, opts.NoCheckout)
	assert.True(t, opts.FirstParent)
}

// --- Reset mode mapping tests ---

func TestResetModeForAction(t *testing.T) {
	tests := []struct {
		action   string
		expected git.ResetMode
	}{
		{"m", git.ResetMixed},
		{"s", git.ResetSoft},
		{"h", git.ResetHard},
		{"k", git.ResetKeep},
		{"i", git.ResetIndex},
		{"w", git.ResetWorktree},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			mode, ok := resetModeForAction(tt.action)
			assert.True(t, ok)
			assert.Equal(t, tt.expected, mode)
		})
	}

	_, ok := resetModeForAction("z")
	assert.False(t, ok)
}

// --- Command function tests (non-nil check) ---

func TestMergeCmd_ReturnsNonNil(t *testing.T) {
	cmd := mergeCmd(nil, git.MergeOpts{Branch: "main"})
	assert.NotNil(t, cmd)
}

func TestCherryPickCmd_ReturnsNonNil(t *testing.T) {
	cmd := cherryPickCmd(nil, []string{"abc123"}, git.CherryPickOpts{})
	assert.NotNil(t, cmd)
}

func TestRevertCmd_ReturnsNonNil(t *testing.T) {
	cmd := revertCmd(nil, []string{"abc123"}, git.RevertOpts{})
	assert.NotNil(t, cmd)
}

func TestStashPushCmd_ReturnsNonNil(t *testing.T) {
	cmd := stashPushCmd(nil, git.StashOpts{})
	assert.NotNil(t, cmd)
}

func TestStashPopCmd_ReturnsNonNil(t *testing.T) {
	cmd := stashPopCmd(nil, 0)
	assert.NotNil(t, cmd)
}

func TestStashApplyCmd_ReturnsNonNil(t *testing.T) {
	cmd := stashApplyCmd(nil, 0)
	assert.NotNil(t, cmd)
}

func TestStashDropCmd_ReturnsNonNil(t *testing.T) {
	cmd := stashDropCmd(nil, 0)
	assert.NotNil(t, cmd)
}

func TestResetCmd_ReturnsNonNil(t *testing.T) {
	cmd := resetCmd(nil, "HEAD~1", git.ResetMixed)
	assert.NotNil(t, cmd)
}

func TestTagCreateCmd_ReturnsNonNil(t *testing.T) {
	cmd := tagCreateCmd(nil, "v1.0", "HEAD", git.TagOpts{})
	assert.NotNil(t, cmd)
}

func TestTagDeleteCmd_ReturnsNonNil(t *testing.T) {
	cmd := tagDeleteCmd(nil, "v1.0")
	assert.NotNil(t, cmd)
}

func TestRemoteAddCmd_ReturnsNonNil(t *testing.T) {
	cmd := remoteAddCmd(nil, "origin", "https://example.com/repo.git")
	assert.NotNil(t, cmd)
}

func TestRemoteRemoveCmd_ReturnsNonNil(t *testing.T) {
	cmd := remoteRemoveCmd(nil, "origin")
	assert.NotNil(t, cmd)
}

func TestRemoteRenameCmd_ReturnsNonNil(t *testing.T) {
	cmd := remoteRenameCmd(nil, "origin", "upstream")
	assert.NotNil(t, cmd)
}

func TestRemotePruneCmd_ReturnsNonNil(t *testing.T) {
	cmd := remotePruneCmd(nil, "origin")
	assert.NotNil(t, cmd)
}

func TestBisectStartCmd_ReturnsNonNil(t *testing.T) {
	cmd := bisectStartCmd(nil, git.BisectOpts{})
	assert.NotNil(t, cmd)
}

func TestBisectGoodCmd_ReturnsNonNil(t *testing.T) {
	cmd := bisectGoodCmd(nil, "abc123")
	assert.NotNil(t, cmd)
}

func TestBisectBadCmd_ReturnsNonNil(t *testing.T) {
	cmd := bisectBadCmd(nil, "abc123")
	assert.NotNil(t, cmd)
}

func TestBisectResetCmd_ReturnsNonNil(t *testing.T) {
	cmd := bisectResetCmd(nil)
	assert.NotNil(t, cmd)
}

func TestIgnoreCmd_ReturnsNonNil(t *testing.T) {
	cmd := ignoreCmd(nil, "/some-file", git.IgnoreScopeTopLevel)
	assert.NotNil(t, cmd)
}

func TestWorktreeAddCmd_ReturnsNonNil(t *testing.T) {
	cmd := worktreeAddCmd(nil, "/tmp/wt", "main")
	assert.NotNil(t, cmd)
}

func TestWorktreeRemoveCmd_ReturnsNonNil(t *testing.T) {
	cmd := worktreeRemoveCmd(nil, "/tmp/wt")
	assert.NotNil(t, cmd)
}

func TestMergeCommitCmd_ReturnsNonNil(t *testing.T) {
	cmd := mergeCommitCmd(nil)
	assert.NotNil(t, cmd)
}

func TestMergeAbortCmd_ReturnsNonNil(t *testing.T) {
	cmd := mergeAbortCmd(nil)
	assert.NotNil(t, cmd)
}

func TestCherryPickContinueCmd_ReturnsNonNil(t *testing.T) {
	cmd := cherryPickContinueCmd(nil)
	assert.NotNil(t, cmd)
}

func TestCherryPickAbortCmd_ReturnsNonNil(t *testing.T) {
	cmd := cherryPickAbortCmd(nil)
	assert.NotNil(t, cmd)
}

func TestRevertContinueCmd_ReturnsNonNil(t *testing.T) {
	cmd := revertContinueCmd(nil)
	assert.NotNil(t, cmd)
}

func TestRevertAbortCmd_ReturnsNonNil(t *testing.T) {
	cmd := revertAbortCmd(nil)
	assert.NotNil(t, cmd)
}

// --- Nil-repo error handling tests ---

func TestMergeCmd_NilRepo(t *testing.T) {
	cmd := mergeCmd(nil, git.MergeOpts{Branch: "main"})
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Merge", done.op)
}

func TestCherryPickCmd_NilRepo(t *testing.T) {
	cmd := cherryPickCmd(nil, []string{"abc"}, git.CherryPickOpts{})
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Cherry-pick", done.op)
}

func TestRevertCmd_NilRepo(t *testing.T) {
	cmd := revertCmd(nil, []string{"abc"}, git.RevertOpts{})
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Revert", done.op)
}

func TestResetCmd_NilRepo(t *testing.T) {
	cmd := resetCmd(nil, "HEAD~1", git.ResetMixed)
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Reset", done.op)
}

func TestStashPushCmd_NilRepo(t *testing.T) {
	cmd := stashPushCmd(nil, git.StashOpts{})
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Stash push", done.op)
}

func TestStashPopCmd_NilRepo(t *testing.T) {
	cmd := stashPopCmd(nil, 0)
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Stash pop", done.op)
}

func TestTagCreateCmd_NilRepo(t *testing.T) {
	cmd := tagCreateCmd(nil, "v1.0", "HEAD", git.TagOpts{})
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Tag create", done.op)
}

func TestTagDeleteCmd_NilRepo(t *testing.T) {
	cmd := tagDeleteCmd(nil, "v1.0")
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Tag delete", done.op)
}

func TestRemoteAddCmd_NilRepo(t *testing.T) {
	cmd := remoteAddCmd(nil, "origin", "https://example.com")
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Remote add", done.op)
}

func TestRemoteRemoveCmd_NilRepo(t *testing.T) {
	cmd := remoteRemoveCmd(nil, "origin")
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Remote remove", done.op)
}

func TestBisectStartCmd_NilRepo(t *testing.T) {
	cmd := bisectStartCmd(nil, git.BisectOpts{})
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Bisect start", done.op)
}

func TestBisectResetCmd_NilRepo(t *testing.T) {
	cmd := bisectResetCmd(nil)
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Bisect reset", done.op)
}

func TestIgnoreCmd_NilRepo(t *testing.T) {
	cmd := ignoreCmd(nil, "/file", git.IgnoreScopeTopLevel)
	msg := cmd()
	done := msg.(operationDoneMsg)
	assert.Error(t, done.err)
	assert.Equal(t, "Ignore", done.op)
}

// --- getStashIndex helper test ---

func TestGetStashIndex(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionStashes, Items: []Item{
				{Stash: &git.StashEntry{Index: 0}},
				{Stash: &git.StashEntry{Index: 1}},
				{Stash: &git.StashEntry{Index: 2}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 1, Hunk: -1, Line: -1},
	}
	idx, ok := getStashIndex(m)
	assert.True(t, ok)
	assert.Equal(t, 1, idx)
}

func TestGetStashIndex_NoStash(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionUntracked, Items: []Item{
				{Entry: &git.StatusEntry{}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	_, ok := getStashIndex(m)
	assert.False(t, ok)
}

// --- getCursorFilePath helper test ---

func TestGetCursorFilePath(t *testing.T) {
	entry := git.NewStatusEntry("test.go", git.FileStatusModified, git.FileStatusNone)
	m := Model{
		sections: []Section{
			{Kind: SectionUnstaged, Items: []Item{
				{Entry: &entry},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	path, ok := getCursorFilePath(m)
	assert.True(t, ok)
	assert.Equal(t, "test.go", path)
}

func TestGetCursorFilePath_NoFile(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionRecentCommits, Items: []Item{
				{Commit: &git.LogEntry{Hash: "abc"}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	_, ok := getCursorFilePath(m)
	assert.False(t, ok)
}

// --- Yank value tests ---

func TestYankValue_Hash_FromCommit(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionRecentCommits, Items: []Item{
				{Commit: &git.LogEntry{Hash: "abc123def456"}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	assert.Equal(t, "abc123def456", yankValue(m, "Y"))
}

func TestYankValue_Hash_FallsBackToHead(t *testing.T) {
	m := Model{
		head: HeadState{Oid: "head-oid"},
	}
	assert.Equal(t, "head-oid", yankValue(m, "Y"))
}

func TestYankValue_Subject_FromCommit(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionRecentCommits, Items: []Item{
				{Commit: &git.LogEntry{Subject: "fix: something"}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	assert.Equal(t, "fix: something", yankValue(m, "s"))
}

func TestYankValue_Author_FromCommit(t *testing.T) {
	m := Model{
		sections: []Section{
			{Kind: SectionRecentCommits, Items: []Item{
				{Commit: &git.LogEntry{AuthorName: "Jane Doe"}},
			}},
		},
		cursor: Cursor{Section: 0, Item: 0, Hunk: -1, Line: -1},
	}
	assert.Equal(t, "Jane Doe", yankValue(m, "a"))
}

func TestYankValue_Tag_FromHead(t *testing.T) {
	m := Model{
		head: HeadState{Tag: "v1.0.0"},
	}
	assert.Equal(t, "v1.0.0", yankValue(m, "t"))
}

func TestYankValue_Unknown_ReturnsEmpty(t *testing.T) {
	m := Model{}
	assert.Equal(t, "", yankValue(m, "z"))
}
