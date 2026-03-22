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

func TestParseHunks_LengthIsOnesPlusLines(t *testing.T) {
	diff := `@@ -1,3 +1,4 @@
 line1
 line2
+new line
 line3
`
	hunks := parseHunks(diff)
	require.Len(t, hunks, 1)

	// Length = 1 (header) + len(Lines)
	assert.Equal(t, 1+len(hunks[0].Lines), hunks[0].Length)
	assert.Equal(t, 5, hunks[0].Length) // 1 header + 4 lines
}

func TestParseHunks_BinaryDiff(t *testing.T) {
	diff := `diff --git a/image.png b/image.png
Binary files a/image.png and b/image.png differ
`
	hunks := parseHunks(diff)
	assert.Empty(t, hunks, "binary files should have no hunks")
}

func TestParseHunks_BinaryFile(t *testing.T) {
	// Verify that parseFileDiff sets IsBinary for binary files
	diff := `diff --git a/photo.jpg b/photo.jpg
Binary files a/photo.jpg and b/photo.jpg differ
`
	fd := parseFileDiff(diff)
	require.NotNil(t, fd)
	assert.True(t, fd.IsBinary, "binary file should have IsBinary set")
	assert.Empty(t, fd.Hunks, "binary file should have no hunks")
}

func TestParseHunks_RenamedFile(t *testing.T) {
	// Verify that parseFileDiff sets OldPath for renamed files
	diff := `diff --git a/before.txt b/after.txt
similarity index 100%
rename from before.txt
rename to after.txt
`
	fd := parseFileDiff(diff)
	require.NotNil(t, fd)
	assert.Equal(t, "after.txt", fd.Path)
	assert.Equal(t, "before.txt", fd.OldPath, "OldPath should be set for renames")
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

func TestApplyPatch_StagesHunk(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a file with multiple lines and commit it
	addAndCommit(t, r, "multi.txt", "line1\nline2\nline3\nline4\nline5\n", "add multi.txt")

	// Modify lines 2 and 4 (creates unstaged changes)
	addFile(t, r, "multi.txt", "line1\nLINE2\nline3\nLINE4\nline5\n")

	// Get the unstaged diff to get hunks
	diffs, err := r.UnstagedDiff(ctx, "multi.txt")
	require.NoError(t, err)
	require.NotEmpty(t, diffs, "expected at least one file diff")
	require.NotEmpty(t, diffs[0].Hunks, "expected at least one hunk")

	// Generate patch for the first hunk and apply it to stage
	patch := HunkToPatch("multi.txt", &diffs[0].Hunks[0], false)
	err = r.ApplyPatch(ctx, patch, "--cached")
	require.NoError(t, err)

	// Verify the hunk is now staged
	status, err := r.Status(ctx)
	require.NoError(t, err)

	found := false
	for _, e := range status.Staged {
		if e.Path() == "multi.txt" {
			found = true
		}
	}
	assert.True(t, found, "expected multi.txt in staged entries")
}

func TestApplyPatch_UnstagesHunk(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a file and commit it
	addAndCommit(t, r, "unstage.txt", "aaa\nbbb\nccc\n", "initial")

	// Modify and stage the change
	addFile(t, r, "unstage.txt", "aaa\nBBB\nccc\n")
	_, err := r.runGit(ctx, "add", "unstage.txt")
	require.NoError(t, err)

	// Get the staged diff
	diffs, err := r.StagedDiff(ctx, "unstage.txt")
	require.NoError(t, err)
	require.NotEmpty(t, diffs)
	require.NotEmpty(t, diffs[0].Hunks)

	// Reverse-apply the patch to unstage the hunk
	patch := HunkToPatch("unstage.txt", &diffs[0].Hunks[0], true)
	err = r.ApplyPatch(ctx, patch, "--cached")
	require.NoError(t, err)

	// Verify unstaged: file should no longer be in Staged, but should be in Unstaged
	status, err := r.Status(ctx)
	require.NoError(t, err)
	for _, e := range status.Staged {
		if e.Path() == "unstage.txt" {
			t.Fatal("unstage.txt should not be in staged entries after unstaging hunk")
		}
	}
	foundUnstaged := false
	for _, e := range status.Unstaged {
		if e.Path() == "unstage.txt" {
			foundUnstaged = true
		}
	}
	assert.True(t, foundUnstaged, "unstage.txt should still have unstaged changes")
}

func TestApplyPatch_DiscardsHunk(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a file and commit
	addAndCommit(t, r, "discard.txt", "one\ntwo\nthree\n", "initial")

	// Modify worktree
	addFile(t, r, "discard.txt", "one\nTWO\nthree\n")

	// Get the unstaged diff
	diffs, err := r.UnstagedDiff(ctx, "discard.txt")
	require.NoError(t, err)
	require.NotEmpty(t, diffs)
	require.NotEmpty(t, diffs[0].Hunks)

	// Reverse-apply (no --cached) to discard the worktree change
	patch := HunkToPatch("discard.txt", &diffs[0].Hunks[0], true)
	err = r.ApplyPatch(ctx, patch)
	require.NoError(t, err)

	// Verify worktree change is gone — file should not appear in any status list
	status, err := r.Status(ctx)
	require.NoError(t, err)
	for _, e := range status.Unstaged {
		if e.Path() == "discard.txt" {
			t.Fatal("discard.txt should not appear in unstaged after discarding all changes")
		}
	}
}

// DiffKind constant tests

func TestDiffKind_Constants(t *testing.T) {
	assert.Equal(t, DiffKind(0), DiffStaged)
	assert.Equal(t, DiffKind(1), DiffUnstaged)
	assert.Equal(t, DiffKind(2), DiffCommit)
	assert.Equal(t, DiffKind(3), DiffRange)
	assert.Equal(t, DiffKind(4), DiffStash)
}

// RangeDiff tests

func TestRangeDiff_ParsesOutput(t *testing.T) {
	output := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 line1
 line2
+added
 line3
`
	diffs := ParseDiffOutput(output, DiffRange)
	require.Len(t, diffs, 1)
	assert.Equal(t, DiffRange, diffs[0].Kind)
	assert.Equal(t, "file.go", diffs[0].Path)
	require.Len(t, diffs[0].Hunks, 1)
}

// DiffStat tests

func TestDiffStat_ParsesStatOutput(t *testing.T) {
	// parseStat is tested indirectly through parseCommitOverview,
	// but verify the new DiffRange/DiffStash kinds propagate correctly.
	output := `diff --git a/file.go b/file.go
index abc..def 100644
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 line1
+added
 line3
`
	diffs := ParseDiffOutput(output, DiffStash)
	require.Len(t, diffs, 1)
	assert.Equal(t, DiffStash, diffs[0].Kind)
}
