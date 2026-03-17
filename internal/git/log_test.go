package git

import (
	"context"
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
