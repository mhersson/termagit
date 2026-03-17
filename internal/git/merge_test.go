package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerge_NoFF_CreatesExplicitMergeCommit(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a branch with a commit
	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "feature.txt", "feature content", "Feature commit")

	// Go back to master
	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)

	// Merge with --no-ff to force a merge commit
	err = r.Merge(ctx, MergeOpts{
		Branch: "feature",
		NoFF:   true,
	})
	require.NoError(t, err)

	// Verify feature.txt exists
	_, err = os.Stat(filepath.Join(r.path, "feature.txt"))
	assert.NoError(t, err, "feature.txt should exist after merge")

	// Verify a merge commit was created (HEAD should have 2 parents)
	out, err := r.runGit(ctx, "log", "-1", "--format=%P")
	require.NoError(t, err)
	// A merge commit has two parent hashes separated by space
	parents := len(splitNonEmpty(trimOutput(out), " "))
	assert.Equal(t, 2, parents, "merge commit should have 2 parents")
}

func TestMergeAbort_ClearsMergeHead(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create conflicting changes
	addAndCommitDisk(t, r, "conflict.txt", "original", "Add file")

	_, err := r.runGit(ctx, "checkout", "-b", "feature")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "feature version", "Feature change")

	_, err = r.runGit(ctx, "checkout", "master")
	require.NoError(t, err)
	addAndCommitDisk(t, r, "conflict.txt", "main version", "Main change")

	// Merge should fail with conflict
	err = r.Merge(ctx, MergeOpts{Branch: "feature"})
	require.Error(t, err, "merge should fail due to conflict")

	// MERGE_HEAD should exist
	require.True(t, r.MergeInProgress(), "merge should be in progress")

	// Abort the merge
	err = r.MergeAbort(ctx)
	require.NoError(t, err)

	// MERGE_HEAD should be gone
	require.False(t, r.MergeInProgress(), "merge should not be in progress after abort")
}

// splitNonEmpty splits a string and returns only non-empty parts.
func splitNonEmpty(s, sep string) []string {
	var result []string
	for _, part := range splitStr(s, sep) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitStr(s, sep string) []string {
	if s == "" {
		return nil
	}
	return append([]string{}, splitParts(s, sep)...)
}

func splitParts(s, sep string) []string {
	result := []string{}
	for {
		idx := indexOf(s, sep)
		if idx < 0 {
			result = append(result, s)
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
