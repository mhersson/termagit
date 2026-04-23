package git

import (
	"context"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitHistory_ReturnsSubjectsNewestFirst(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "a.txt", "a", "First commit")
	addAndCommit(t, r, "b.txt", "b", "Second commit")

	subjects, err := r.CommitHistory(ctx, 10)
	require.NoError(t, err)
	require.Len(t, subjects, 3) // Initial + 2
	assert.Equal(t, "Second commit", subjects[0])
	assert.Equal(t, "First commit", subjects[1])
	assert.Equal(t, "Initial commit", subjects[2])
}

func TestCommitHistory_LimitsToN(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "a.txt", "a", "First commit")
	addAndCommit(t, r, "b.txt", "b", "Second commit")
	addAndCommit(t, r, "c.txt", "c", "Third commit")

	subjects, err := r.CommitHistory(ctx, 2)
	require.NoError(t, err)
	require.Len(t, subjects, 2)
	assert.Equal(t, "Third commit", subjects[0])
	assert.Equal(t, "Second commit", subjects[1])
}

func TestCommitHistory_EmptyRepo(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// newMemRepo creates an initial commit, so we get 1 entry
	subjects, err := r.CommitHistory(ctx, 10)
	require.NoError(t, err)
	require.Len(t, subjects, 1)
	assert.Equal(t, "Initial commit", subjects[0])
}

func TestCycler_Prev_ReturnsPreviousMessage(t *testing.T) {
	c := NewCycler([]string{"third", "second", "first"})

	msg := c.Prev("current text")
	assert.Equal(t, "third", msg)

	msg = c.Prev("third")
	assert.Equal(t, "second", msg)
}

func TestCycler_Prev_SavesCurrentBeforeFirstNav(t *testing.T) {
	c := NewCycler([]string{"third", "second", "first"})

	_ = c.Prev("my draft message")

	// Navigate back to start
	msg := c.Reset()
	assert.Equal(t, "my draft message", msg)
}

func TestCycler_Prev_ClampsAtOldest(t *testing.T) {
	c := NewCycler([]string{"only"})

	msg := c.Prev("current")
	assert.Equal(t, "only", msg)

	// Calling Prev again stays at the oldest
	msg = c.Prev("only")
	assert.Equal(t, "only", msg)
}

func TestCycler_Next_ReturnsTowardCurrent(t *testing.T) {
	c := NewCycler([]string{"third", "second", "first"})

	_ = c.Prev("current") // idx=0 -> "third"
	_ = c.Prev("third")   // idx=1 -> "second"

	msg := c.Next()
	assert.Equal(t, "third", msg) // back to idx=0
}

func TestCycler_Next_AtStart_RestoresOriginal(t *testing.T) {
	c := NewCycler([]string{"third", "second", "first"})

	_ = c.Prev("my draft") // idx=0

	msg := c.Next()
	assert.Equal(t, "my draft", msg) // back to original
}

func TestCycler_Reset_RestoresOriginal(t *testing.T) {
	c := NewCycler([]string{"third", "second", "first"})

	_ = c.Prev("my draft")
	_ = c.Prev("third")
	_ = c.Prev("second")

	msg := c.Reset()
	assert.Equal(t, "my draft", msg)
}

func TestCycler_Prev_EmptyMessages(t *testing.T) {
	c := NewCycler([]string{})

	msg := c.Prev("current")
	assert.Equal(t, "current", msg, "should return current when no messages")
}

func TestCommitMessagesForCycling_ReturnsFullMessages(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Create commits with multiline messages
	addAndCommitMsg(t, r, "a.txt", "a", "First commit\n\nThis is the body of the first commit.")
	addAndCommitMsg(t, r, "b.txt", "b", "Second commit\n\nThis is the body of the second commit.\n\nWith multiple paragraphs.")

	messages, err := r.CommitMessagesForCycling(ctx, 10)
	require.NoError(t, err)
	require.Len(t, messages, 3) // Initial + 2

	// Most recent first
	assert.Contains(t, messages[0], "Second commit")
	assert.Contains(t, messages[0], "multiple paragraphs")
	assert.Contains(t, messages[1], "First commit")
	assert.Contains(t, messages[1], "body of the first")
}

func TestCommitMessagesForCycling_LimitsToN(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "a.txt", "a", "First commit")
	addAndCommit(t, r, "b.txt", "b", "Second commit")
	addAndCommit(t, r, "c.txt", "c", "Third commit")

	messages, err := r.CommitMessagesForCycling(ctx, 2)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Contains(t, messages[0], "Third commit")
	assert.Contains(t, messages[1], "Second commit")
}

// addAndCommitMsg creates a file and commits with a custom message (including body).
func addAndCommitMsg(t *testing.T, r *Repository, name, content, message string) {
	t.Helper()

	wt, err := r.raw.Worktree()
	require.NoError(t, err)

	fs := wt.Filesystem
	f, err := fs.Create(name)
	require.NoError(t, err)
	_, err = f.Write([]byte(content))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)

	_, err = wt.Add(name)
	require.NoError(t, err)

	_, err = wt.Commit(message, &gogit.CommitOptions{})
	require.NoError(t, err)
}
