package git

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// realPath resolves symlinks for path comparison on macOS (/var -> /private/var).
func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

func TestParseWorktreePorcelain_Main(t *testing.T) {
	output := `worktree /home/user/project
HEAD abc1234567890123456789012345678901234abcd
branch refs/heads/main

`
	wts := parseWorktreePorcelain(output)
	require.Len(t, wts, 1)

	wt := wts[0]
	require.Equal(t, "/home/user/project", wt.Path)
	require.Equal(t, "abc1234567890123456789012345678901234abcd", wt.Head)
	require.Equal(t, "refs/heads/main", wt.Branch)
	require.False(t, wt.IsBare)
	require.False(t, wt.IsLocked)
}

func TestParseWorktreePorcelain_WithLinked(t *testing.T) {
	output := `worktree /home/user/project
HEAD abc1234567890123456789012345678901234abcd
branch refs/heads/main

worktree /home/user/project-feature
HEAD def4567890123456789012345678901234567defg
branch refs/heads/feature
locked

`
	wts := parseWorktreePorcelain(output)
	require.Len(t, wts, 2)

	require.Equal(t, "/home/user/project", wts[0].Path)
	require.Equal(t, "refs/heads/main", wts[0].Branch)
	require.False(t, wts[0].IsLocked)

	require.Equal(t, "/home/user/project-feature", wts[1].Path)
	require.Equal(t, "refs/heads/feature", wts[1].Branch)
	require.True(t, wts[1].IsLocked)
}

func TestParseWorktreePorcelain_Bare(t *testing.T) {
	output := `worktree /home/user/project.git
HEAD abc1234567890123456789012345678901234abcd
bare

`
	wts := parseWorktreePorcelain(output)
	require.Len(t, wts, 1)
	require.True(t, wts[0].IsBare)
}

func TestListWorktrees_ReturnsMain(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	wts, err := r.ListWorktrees(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, wts)

	// First worktree should be the main one
	require.Equal(t, realPath(t, r.path), realPath(t, wts[0].Path))
}

func TestAddWorktree_AppearsInList(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create a branch first
	_, err := r.runGit(ctx, "branch", "feature")
	require.NoError(t, err)

	wtPath := t.TempDir()
	err = r.AddWorktree(ctx, wtPath, "feature")
	require.NoError(t, err)

	wts, err := r.ListWorktrees(ctx)
	require.NoError(t, err)
	require.Len(t, wts, 2)

	// Find the linked worktree
	var found bool
	resolvedWtPath := realPath(t, wtPath)
	for _, wt := range wts {
		if realPath(t, wt.Path) == resolvedWtPath {
			found = true
			require.Equal(t, "refs/heads/feature", wt.Branch)
		}
	}
	require.True(t, found, "linked worktree not found in list")
}

func TestRemoveWorktree_GoneFromList(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create a branch and worktree
	_, err := r.runGit(ctx, "branch", "feature")
	require.NoError(t, err)

	wtPath := t.TempDir()
	err = r.AddWorktree(ctx, wtPath, "feature")
	require.NoError(t, err)

	// Remove it
	err = r.RemoveWorktree(ctx, wtPath, false)
	require.NoError(t, err)

	wts, err := r.ListWorktrees(ctx)
	require.NoError(t, err)
	require.Len(t, wts, 1) // Only main remains
}
