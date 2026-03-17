package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DiffLine tests

func TestDiffLine_OpIsCorrect(t *testing.T) {
	line := DiffLine{Op: DiffOpAdd, Content: "new line"}
	assert.Equal(t, DiffOpAdd, line.Op)
}

// Hunk tests

func TestHunk_HasCorrectCounts(t *testing.T) {
	hunk := Hunk{
		OldStart: 1, OldCount: 3,
		NewStart: 1, NewCount: 4,
	}
	assert.Equal(t, 3, hunk.OldCount)
	assert.Equal(t, 4, hunk.NewCount)
}

// parseHunks tests (pure function)

func TestParseHunks_SimpleAddition(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
index abc123..def456 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
 line2
+new line
 line3
`
	hunks := parseHunks(diff)
	require.Len(t, hunks, 1)

	hunk := hunks[0]
	assert.Equal(t, 1, hunk.OldStart)
	assert.Equal(t, 3, hunk.OldCount)
	assert.Equal(t, 1, hunk.NewStart)
	assert.Equal(t, 4, hunk.NewCount)

	// Find the added line
	var foundAdd bool
	for _, line := range hunk.Lines {
		if line.Op == DiffOpAdd && line.Content == "new line" {
			foundAdd = true
			break
		}
	}
	assert.True(t, foundAdd, "should find added line")
}

func TestParseHunks_SimpleDeletion(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,4 +1,3 @@
 line1
-deleted
 line2
 line3
`
	hunks := parseHunks(diff)
	require.Len(t, hunks, 1)

	var foundDel bool
	for _, line := range hunks[0].Lines {
		if line.Op == DiffOpDelete && line.Content == "deleted" {
			foundDel = true
			break
		}
	}
	assert.True(t, foundDel, "should find deleted line")
}

func TestParseHunks_MultipleHunks(t *testing.T) {
	diff := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line1
+add1
 line2
 line3
@@ -10,3 +11,4 @@
 line10
+add2
 line11
 line12
`
	hunks := parseHunks(diff)
	require.Len(t, hunks, 2)

	assert.Equal(t, 1, hunks[0].OldStart)
	assert.Equal(t, 10, hunks[1].OldStart)
}

func TestParseHunks_NoCount_DefaultsToOne(t *testing.T) {
	diff := `--- a/file.txt
+++ b/file.txt
@@ -5 +5 @@
-old
+new
`
	hunks := parseHunks(diff)
	require.Len(t, hunks, 1)

	assert.Equal(t, 5, hunks[0].OldStart)
	assert.Equal(t, 1, hunks[0].OldCount)
	assert.Equal(t, 5, hunks[0].NewStart)
	assert.Equal(t, 1, hunks[0].NewCount)
}

func TestParseHunks_BinaryDiff(t *testing.T) {
	diff := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ
`
	hunks := parseHunks(diff)
	assert.Empty(t, hunks, "binary files should have no hunks")
}

// FileDiff tests

func TestParseFileDiff_DetectsNewFile(t *testing.T) {
	diff := `diff --git a/new.txt b/new.txt
new file mode 100644
index 0000000..abc1234
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,2 @@
+line1
+line2
`
	fd := parseFileDiff(diff)
	require.NotNil(t, fd)
	assert.True(t, fd.IsNew)
	assert.Equal(t, "new.txt", fd.Path)
}

func TestParseFileDiff_DetectsDeletedFile(t *testing.T) {
	diff := `diff --git a/old.txt b/old.txt
deleted file mode 100644
index abc1234..0000000
--- a/old.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line1
-line2
`
	fd := parseFileDiff(diff)
	require.NotNil(t, fd)
	assert.True(t, fd.IsDelete)
	assert.Equal(t, "old.txt", fd.Path)
}

func TestParseFileDiff_DetectsRename(t *testing.T) {
	diff := `diff --git a/old.txt b/new.txt
similarity index 100%
rename from old.txt
rename to new.txt
`
	fd := parseFileDiff(diff)
	require.NotNil(t, fd)
	assert.Equal(t, "new.txt", fd.Path)
	assert.Equal(t, "old.txt", fd.OldPath)
}

func TestParseFileDiff_DetectsBinary(t *testing.T) {
	diff := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ
`
	fd := parseFileDiff(diff)
	require.NotNil(t, fd)
	assert.True(t, fd.IsBinary)
}

// Repository diff methods

func TestStagedDiff_ReturnsEmpty_WhenNothingStaged(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	diffs, err := r.StagedDiff(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestStagedDiff_ReturnsDiff_WhenFileStaged(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Modify and stage a file
	addFile(t, r, "README.md", "# Modified\n\nNew content\n")
	stageFile(t, r, "README.md")

	diffs, err := r.StagedDiff(ctx, "")
	require.NoError(t, err)
	require.Len(t, diffs, 1)

	assert.Equal(t, "README.md", diffs[0].Path)
	assert.NotEmpty(t, diffs[0].Hunks)
}

func TestUnstagedDiff_ReturnsEmpty_WhenNoChanges(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	diffs, err := r.UnstagedDiff(ctx, "")
	require.NoError(t, err)
	assert.Empty(t, diffs)
}

func TestUnstagedDiff_ReturnsDiff_WhenFileModified(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Modify a file without staging
	addFile(t, r, "README.md", "# Modified\n\nUnstaged changes\n")

	diffs, err := r.UnstagedDiff(ctx, "")
	require.NoError(t, err)
	require.Len(t, diffs, 1)

	assert.Equal(t, "README.md", diffs[0].Path)
	assert.NotEmpty(t, diffs[0].Hunks)
}

func TestUntrackedDiff_ReturnsDiff_ForNewFile(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create an untracked file
	addFile(t, r, "untracked.txt", "new file content\n")

	diff, err := r.UntrackedDiff(ctx, "untracked.txt")
	require.NoError(t, err)
	require.NotNil(t, diff)

	assert.Equal(t, "untracked.txt", diff.Path)
	assert.True(t, diff.IsNew)
}
