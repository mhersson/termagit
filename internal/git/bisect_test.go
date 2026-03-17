package git

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBisectStart_CreatesBisectLog(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create several commits to bisect through
	addAndCommit(t, r, "a.txt", "a", "commit a")
	addAndCommit(t, r, "b.txt", "b", "commit b")
	addAndCommit(t, r, "c.txt", "c", "commit c")

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	// Get first commit hash
	out, err := r.runGit(ctx, "rev-list", "--reverse", "HEAD")
	require.NoError(t, err)
	firstCommit := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])

	err = r.BisectStart(ctx, head, []string{firstCommit}, BisectOpts{})
	require.NoError(t, err)

	require.True(t, r.BisectInProgress())
}

func TestBisectGoodBad_ProgressesBisect(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	// Create enough commits for bisect to iterate
	addAndCommit(t, r, "a.txt", "a", "commit a")
	addAndCommit(t, r, "b.txt", "b", "commit b")
	addAndCommit(t, r, "c.txt", "c", "commit c")
	addAndCommit(t, r, "d.txt", "d", "commit d")

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	out, err := r.runGit(ctx, "rev-list", "--reverse", "HEAD")
	require.NoError(t, err)
	firstCommit := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])

	err = r.BisectStart(ctx, head, []string{firstCommit}, BisectOpts{})
	require.NoError(t, err)

	// Mark current as good -- bisect should continue
	currentHead, err := r.HeadOID(ctx)
	require.NoError(t, err)

	err = r.BisectGood(ctx, currentHead)
	require.NoError(t, err)

	// Should still be in bisect
	require.True(t, r.BisectInProgress())
}

func TestBisectReset_EndsBisect(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "a.txt", "a", "commit a")
	addAndCommit(t, r, "b.txt", "b", "commit b")

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	out, err := r.runGit(ctx, "rev-list", "--reverse", "HEAD")
	require.NoError(t, err)
	firstCommit := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])

	err = r.BisectStart(ctx, head, []string{firstCommit}, BisectOpts{})
	require.NoError(t, err)
	require.True(t, r.BisectInProgress())

	err = r.BisectReset(ctx)
	require.NoError(t, err)
	require.False(t, r.BisectInProgress())
}

func TestBisectSkip_SkipsCommit(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	addAndCommit(t, r, "a.txt", "a", "commit a")
	addAndCommit(t, r, "b.txt", "b", "commit b")
	addAndCommit(t, r, "c.txt", "c", "commit c")
	addAndCommit(t, r, "d.txt", "d", "commit d")

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	out, err := r.runGit(ctx, "rev-list", "--reverse", "HEAD")
	require.NoError(t, err)
	firstCommit := strings.TrimSpace(strings.Split(strings.TrimSpace(out), "\n")[0])

	err = r.BisectStart(ctx, head, []string{firstCommit}, BisectOpts{})
	require.NoError(t, err)

	currentHead, err := r.HeadOID(ctx)
	require.NoError(t, err)

	err = r.BisectSkip(ctx, currentHead)
	require.NoError(t, err)

	// Should still be in bisect
	require.True(t, r.BisectInProgress())
}
