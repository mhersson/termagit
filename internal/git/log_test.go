package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ref parsing tests

func TestParseRefs_ParsesLocalBranch(t *testing.T) {
	refs := parseRefs("HEAD -> main", nil)
	require.Len(t, refs, 2)

	// First should be HEAD
	assert.Equal(t, "HEAD", refs[0].Name)
	assert.Equal(t, RefKindHead, refs[0].Kind)

	// Second should be main (local branch)
	assert.Equal(t, "main", refs[1].Name)
	assert.Equal(t, RefKindLocal, refs[1].Kind)
}

func TestParseRefs_ParsesRemoteBranch(t *testing.T) {
	refs := parseRefs("origin/main", []string{"origin"})
	require.Len(t, refs, 1)

	assert.Equal(t, "main", refs[0].Name)
	assert.Equal(t, RefKindRemote, refs[0].Kind)
	assert.Equal(t, "origin", refs[0].Remote)
}

func TestParseRefs_ParsesTag(t *testing.T) {
	refs := parseRefs("tag: v1.0.0", nil)
	require.Len(t, refs, 1)

	assert.Equal(t, "v1.0.0", refs[0].Name)
	assert.Equal(t, RefKindTag, refs[0].Kind)
}

func TestParseRefs_ParsesMultipleRefs(t *testing.T) {
	refs := parseRefs("HEAD -> main, origin/main, tag: v1.0.0", []string{"origin"})
	require.Len(t, refs, 4)

	// HEAD, main, origin/main, tag: v1.0.0
	assert.Equal(t, RefKindHead, refs[0].Kind)
	assert.Equal(t, RefKindLocal, refs[1].Kind)
	assert.Equal(t, RefKindRemote, refs[2].Kind)
	assert.Equal(t, RefKindTag, refs[3].Kind)
}

func TestParseRefs_EmptyString_ReturnsEmpty(t *testing.T) {
	refs := parseRefs("", nil)
	assert.Empty(t, refs)
}

// LogEntry tests

func TestLogEntry_HasCorrectFields(t *testing.T) {
	entry := LogEntry{
		Hash:            "abc123def456",
		AbbreviatedHash: "abc123d",
		Subject:         "Test commit",
		AuthorName:      "Test User",
	}
	assert.Equal(t, "abc123d", entry.AbbreviatedHash)
}

// Repository log methods

func TestRecentCommits_ReturnsInitialCommit(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	commits, err := r.RecentCommits(ctx, 10)
	require.NoError(t, err)

	// Should have at least the initial commit
	require.GreaterOrEqual(t, len(commits), 1)
	assert.Equal(t, "Initial commit", commits[0].Subject)
}

func TestRecentCommits_ReturnsMultipleCommits(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create some more commits
	addAndCommit(t, r, "file1.txt", "content1", "Add file 1")
	addAndCommit(t, r, "file2.txt", "content2", "Add file 2")

	commits, err := r.RecentCommits(ctx, 10)
	require.NoError(t, err)

	// Should have 3 commits: initial + 2 new ones
	require.Len(t, commits, 3)

	// Most recent first
	assert.Equal(t, "Add file 2", commits[0].Subject)
	assert.Equal(t, "Add file 1", commits[1].Subject)
}

func TestRecentCommits_LimitsToN(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create some more commits
	addAndCommit(t, r, "file1.txt", "content1", "Add file 1")
	addAndCommit(t, r, "file2.txt", "content2", "Add file 2")
	addAndCommit(t, r, "file3.txt", "content3", "Add file 3")

	commits, err := r.RecentCommits(ctx, 2)
	require.NoError(t, err)

	// Should be limited to 2
	require.Len(t, commits, 2)
}

func TestCommitMessage_ReturnsSubject(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Get the initial commit hash
	head, err := r.raw.Head()
	require.NoError(t, err)

	subject, err := r.CommitMessage(ctx, head.Hash().String())
	require.NoError(t, err)
	assert.Equal(t, "Initial commit", subject)
}

func TestLog_WithMaxCount(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create commits
	addAndCommit(t, r, "file1.txt", "content1", "Commit 1")
	addAndCommit(t, r, "file2.txt", "content2", "Commit 2")

	opts := LogOpts{MaxCount: 1}
	entries, hasMore, err := r.Log(ctx, opts)
	require.NoError(t, err)

	require.Len(t, entries, 1)
	assert.True(t, hasMore, "should have more commits")
}

func TestLog_WithGrep(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create commits with different messages
	addAndCommit(t, r, "file1.txt", "content1", "Feature: add login")
	addAndCommit(t, r, "file2.txt", "content2", "Bug: fix crash")
	addAndCommit(t, r, "file3.txt", "content3", "Feature: add logout")

	opts := LogOpts{Grep: "Feature"}
	entries, _, err := r.Log(ctx, opts)
	require.NoError(t, err)

	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Contains(t, e.Subject, "Feature")
	}
}

func TestCommitDetail_ReturnsFullInfo(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a commit with a body
	addAndCommit(t, r, "detail.txt", "content", "Subject line\n\nThis is the body")

	commits, err := r.RecentCommits(ctx, 1)
	require.NoError(t, err)
	require.Len(t, commits, 1)

	detail, err := r.CommitDetail(ctx, commits[0].Hash)
	require.NoError(t, err)

	assert.Equal(t, commits[0].Hash, detail.Hash)
	assert.NotEmpty(t, detail.AuthorName)
	assert.NotEmpty(t, detail.AuthorEmail)
}

func TestLog_WhenFieldIsParsedFromAuthorDate(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "when.txt", "content", "Test when field")

	entries, _, err := r.Log(ctx, LogOpts{MaxCount: 1})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.False(t, entries[0].When.IsZero(), "When should be parsed from AuthorDate")
}

func TestLog_RefNameIsPopulatedForDecoratedCommits(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "refname.txt", "content", "Test refname field")

	entries, _, err := r.Log(ctx, LogOpts{MaxCount: 1, Decorate: true})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// HEAD commit should have decoration (branch name at minimum)
	assert.NotEmpty(t, entries[0].RefName, "RefName should be populated for HEAD commit")
}

func TestLog_GraphOptionAddsGraphFlag(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "graph.txt", "content", "Test graph")

	opts := LogOpts{MaxCount: 5, Graph: true}
	entries, _, err := r.Log(ctx, opts)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "should return commits with graph option")
}

func TestLogAhead_ReturnsEmpty_WhenNoAheadCommits(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// HEAD is at the same position as HEAD (trivial case)
	head, err := r.raw.Head()
	require.NoError(t, err)

	entries, err := r.LogAhead(ctx, head.Hash().String(), 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLogBehind_ReturnsEmpty_WhenNoBehindCommits(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// HEAD is at the same position as HEAD (trivial case)
	head, err := r.raw.Head()
	require.NoError(t, err)

	entries, err := r.LogBehind(ctx, head.Hash().String(), 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLog_Offset_Paginates(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create 5 commits (initial + 4 more = 5 total)
	addAndCommit(t, r, "file1.txt", "c1", "Commit 1")
	addAndCommit(t, r, "file2.txt", "c2", "Commit 2")
	addAndCommit(t, r, "file3.txt", "c3", "Commit 3")
	addAndCommit(t, r, "file4.txt", "c4", "Commit 4")

	// Log with Offset=2 should skip the 2 most recent
	opts := LogOpts{Offset: 2}
	entries, _, err := r.Log(ctx, opts)
	require.NoError(t, err)

	// Should have 3 remaining (5 total - 2 skipped)
	require.Len(t, entries, 3)
	// Most recent of the remaining should be "Commit 2"
	assert.Equal(t, "Commit 2", entries[0].Subject)
}

func TestLog_HasMore_TrueWhenMoreExist(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create 5 commits total (initial + 4)
	addAndCommit(t, r, "file1.txt", "c1", "Commit 1")
	addAndCommit(t, r, "file2.txt", "c2", "Commit 2")
	addAndCommit(t, r, "file3.txt", "c3", "Commit 3")
	addAndCommit(t, r, "file4.txt", "c4", "Commit 4")

	opts := LogOpts{MaxCount: 3}
	entries, hasMore, err := r.Log(ctx, opts)
	require.NoError(t, err)

	require.Len(t, entries, 3)
	assert.True(t, hasMore, "should indicate more commits exist")
}

func TestLog_HasMore_FalseOnLastPage(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create 3 commits total (initial + 2)
	addAndCommit(t, r, "file1.txt", "c1", "Commit 1")
	addAndCommit(t, r, "file2.txt", "c2", "Commit 2")

	opts := LogOpts{MaxCount: 5}
	entries, hasMore, err := r.Log(ctx, opts)
	require.NoError(t, err)

	require.Len(t, entries, 3)
	assert.False(t, hasMore, "should indicate no more commits")
}

func TestLog_RefDecoration_LocalBranch(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "branch.txt", "content", "Branch commit")

	opts := LogOpts{MaxCount: 1, Decorate: true}
	entries, _, err := r.Log(ctx, opts)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// HEAD commit should have a local branch ref
	var foundLocal bool
	for _, ref := range entries[0].Refs {
		if ref.Kind == RefKindLocal {
			foundLocal = true
			break
		}
	}
	assert.True(t, foundLocal, "should find a local branch ref in decorations")
}

func TestLog_RefDecoration_Tag(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "tag.txt", "content", "Tag commit")

	// Create a lightweight tag
	cmd := exec.Command("git", "tag", "v1.0.0")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	opts := LogOpts{MaxCount: 1, Decorate: true}
	entries, _, err := r.Log(ctx, opts)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	var foundTag bool
	for _, ref := range entries[0].Refs {
		if ref.Kind == RefKindTag && ref.Name == "v1.0.0" {
			foundTag = true
			break
		}
	}
	assert.True(t, foundTag, "should find tag v1.0.0 in decorations")
}

func TestLog_RefDecoration_HEAD(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "head.txt", "content", "HEAD commit")

	opts := LogOpts{MaxCount: 1, Decorate: true}
	entries, _, err := r.Log(ctx, opts)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	var foundHEAD bool
	for _, ref := range entries[0].Refs {
		if ref.Kind == RefKindHead {
			foundHEAD = true
			break
		}
	}
	assert.True(t, foundHEAD, "should find HEAD ref in decorations")
}

func TestLog_EmptyRepo_ReturnsEmpty(t *testing.T) {
	skipInShort(t)

	// Create a repo with no commits at all
	dir := t.TempDir()
	cmd := exec.Command("git", "init", dir)
	require.NoError(t, cmd.Run())

	repo, err := Open(dir, nil)
	require.NoError(t, err)

	entries, _, err := repo.Log(context.Background(), LogOpts{MaxCount: 10})
	// An empty repo may return an error from git log (no commits), or empty results
	if err == nil {
		assert.Empty(t, entries)
	}
}

func TestRefCommitInfo_ReturnsOIDAndSubject(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// HEAD is on the default branch with "Initial commit"
	head, err := r.raw.Head()
	require.NoError(t, err)
	expectedOID := head.Hash().String()
	branch := head.Name().Short()

	oid, subject, err := r.RefCommitInfo(ctx, branch)
	require.NoError(t, err)
	assert.Equal(t, expectedOID, oid)
	assert.Equal(t, "Initial commit", subject)
}

func TestRefCommitInfo_ReturnsErrorForInvalidRef(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	_, _, err := r.RefCommitInfo(ctx, "nonexistent-ref")
	assert.Error(t, err)
}

func TestLog_When_IsParsed(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "when.txt", "content", "When test")

	entries, _, err := r.Log(ctx, LogOpts{MaxCount: 1})
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.False(t, entries[0].When.IsZero(), "When should be parsed from AuthorDate")
}
