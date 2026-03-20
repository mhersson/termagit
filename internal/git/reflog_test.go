package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitCommit creates a file and commits using git command (updates reflog).
func gitCommit(t *testing.T, dir, filename, content, message string) {
	t.Helper()

	filePath := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

	cmd := exec.Command("git", "add", filename)
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}

func TestReflogEntry_HasCorrectFields(t *testing.T) {
	entry := ReflogEntry{
		Oid:        "abc123def456abc123def456abc123def456abc1",
		Index:      0,
		AuthorName: "Test User",
		RefName:    "HEAD@{0}",
		RefSubject: "commit: add feature",
		RelDate:    "3 hours ago",
		CommitDate: "2024-01-15 10:30:00",
		Type:       "commit",
	}
	assert.Equal(t, "abc123def456abc123def456abc123def456abc1", entry.Oid)
	assert.Equal(t, 0, entry.Index)
	assert.Equal(t, "commit", entry.Type)
}

func TestParseReflogType_Commit(t *testing.T) {
	typ := parseReflogType("commit: add feature X")
	assert.Equal(t, "commit", typ)
}

func TestParseReflogType_CommitInitial(t *testing.T) {
	typ := parseReflogType("commit (initial): Initial commit")
	assert.Equal(t, "commit", typ)
}

func TestParseReflogType_CommitAmend(t *testing.T) {
	typ := parseReflogType("commit (amend): fix typo")
	assert.Equal(t, "amend", typ)
}

func TestParseReflogType_Merge(t *testing.T) {
	typ := parseReflogType("merge origin/main: Merge remote-tracking branch")
	assert.Equal(t, "merge", typ)
}

func TestParseReflogType_Checkout(t *testing.T) {
	typ := parseReflogType("checkout: moving from main to feature")
	assert.Equal(t, "checkout", typ)
}

func TestParseReflogType_Reset(t *testing.T) {
	typ := parseReflogType("reset: moving to HEAD~1")
	assert.Equal(t, "reset", typ)
}

func TestParseReflogType_Rebase(t *testing.T) {
	typ := parseReflogType("rebase (start): checkout abc123")
	assert.Equal(t, "rebase", typ)
}

func TestParseReflogType_RebaseInteractive(t *testing.T) {
	typ := parseReflogType("rebase -i (start): checkout abc123")
	assert.Equal(t, "rebase", typ)
}

func TestParseReflogType_CherryPick(t *testing.T) {
	typ := parseReflogType("cherry-pick: add feature")
	assert.Equal(t, "cherry-pick", typ)
}

func TestParseReflogType_Revert(t *testing.T) {
	typ := parseReflogType("revert: Revert \"add feature\"")
	assert.Equal(t, "revert", typ)
}

func TestParseReflogType_Pull(t *testing.T) {
	typ := parseReflogType("pull: Fast-forward")
	assert.Equal(t, "pull", typ)
}

func TestParseReflogType_Clone(t *testing.T) {
	typ := parseReflogType("clone: from git@github.com:user/repo.git")
	assert.Equal(t, "clone", typ)
}

func TestParseReflogType_Branch(t *testing.T) {
	typ := parseReflogType("branch: Created from HEAD")
	assert.Equal(t, "branch", typ)
}

func TestParseReflogType_Unknown(t *testing.T) {
	typ := parseReflogType("some unknown format")
	assert.Equal(t, "other", typ)
}

func TestReflog_ReturnsEntries(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create some commits using git command (go-git commits don't update reflog)
	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")
	gitCommit(t, r.path, "file2.txt", "content2", "Add file 2")

	entries, err := r.Reflog(ctx, "HEAD", 10)
	require.NoError(t, err)

	// Should have at least 2 entries (2 git commits - initial commit via go-git doesn't update reflog)
	require.GreaterOrEqual(t, len(entries), 2)

	// Most recent first (index 0)
	assert.Equal(t, 0, entries[0].Index)
	assert.Equal(t, 1, entries[1].Index)

	// OIDs should be 40 chars
	assert.Len(t, entries[0].Oid, 40)
}

func TestReflog_ParsesCommitType(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")

	entries, err := r.Reflog(ctx, "HEAD", 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// Most recent should be a commit
	assert.Equal(t, "commit", entries[0].Type)
}

func TestReflog_ParsesResetType(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit, then reset
	addAndCommit(t, r, "file1.txt", "content1", "Add file 1")
	cmd := exec.Command("git", "reset", "--soft", "HEAD~1")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	entries, err := r.Reflog(ctx, "HEAD", 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// Most recent should be a reset
	assert.Equal(t, "reset", entries[0].Type)
}

func TestReflog_ParsesCheckoutType(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a branch and checkout
	cmd := exec.Command("git", "checkout", "-b", "feature")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	entries, err := r.Reflog(ctx, "HEAD", 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// Most recent should be a checkout
	assert.Equal(t, "checkout", entries[0].Type)
}

func TestReflog_LimitsToN(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create several commits using git command
	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")
	gitCommit(t, r.path, "file2.txt", "content2", "Add file 2")
	gitCommit(t, r.path, "file3.txt", "content3", "Add file 3")

	entries, err := r.Reflog(ctx, "HEAD", 2)
	require.NoError(t, err)

	// Should be limited to 2
	assert.Len(t, entries, 2)
}

func TestReflog_DefaultsToHEAD(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")

	// Empty ref should default to HEAD
	entries, err := r.Reflog(ctx, "", 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	// RefName should be HEAD@{n}
	assert.Contains(t, entries[0].RefName, "HEAD@{")
}

func TestReflog_ReturnsRefSubject(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")

	entries, err := r.Reflog(ctx, "HEAD", 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// RefSubject should contain the full reflog message
	assert.NotEmpty(t, entries[0].RefSubject)
}

func TestReflog_ReturnsRelativeDate(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")

	entries, err := r.Reflog(ctx, "HEAD", 1)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// RelDate should be something like "X seconds ago" or similar
	assert.NotEmpty(t, entries[0].RelDate)
}

func TestReflog_BranchRef(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create commits on a branch using git command
	gitCommit(t, r.path, "file1.txt", "content1", "Add file 1")

	// Get branch name
	head, err := r.raw.Head()
	require.NoError(t, err)
	branchName := head.Name().Short()

	entries, err := r.Reflog(ctx, branchName, 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}
