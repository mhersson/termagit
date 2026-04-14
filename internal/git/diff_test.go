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
		if e.Path == "multi.txt" {
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
		if e.Path == "unstage.txt" {
			t.Fatal("unstage.txt should not be in staged entries after unstaging hunk")
		}
	}
	foundUnstaged := false
	for _, e := range status.Unstaged {
		if e.Path == "unstage.txt" {
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
		if e.Path == "discard.txt" {
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

func TestDiffStat_DoesNotCorruptCallerSlice(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a slice with spare capacity
	backing := make([]string, 2, 4)
	backing[0] = "--cached"
	backing[1] = "--"
	args := backing[:2]

	// Place a sentinel in the spare capacity
	backing = backing[:3]
	backing[2] = "UNTOUCHED"

	// DiffStat will fail (no staged files), but that's fine —
	// we only care that the backing array wasn't corrupted.
	_, _ = r.DiffStat(ctx, args...)

	assert.Equal(t, "UNTOUCHED", backing[2],
		"DiffStat must not overwrite caller's backing array")
}

func TestApplyPatch_DoesNotCorruptCallerSlice(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Create a slice with spare capacity
	backing := make([]string, 2, 4)
	backing[0] = "--cached"
	backing[1] = "--stat"
	extraArgs := backing[:2]

	// Place a sentinel in the spare capacity
	backing = backing[:3]
	backing[2] = "UNTOUCHED"

	// ApplyPatch will fail (empty patch), but that's fine —
	// we only care that the backing array wasn't corrupted.
	_ = r.ApplyPatch(ctx, "", extraArgs...)

	assert.Equal(t, "UNTOUCHED", backing[2],
		"ApplyPatch must not overwrite caller's backing array")
}

// LineRangeToPatch tests

// buildTestHunk builds a Hunk with the given lines for testing.
// Each line is specified as "<op><content>" where op is '+', '-', or ' '.
func buildTestHunk(header string, oldStart, oldCount, newStart, newCount int, rawLines []string) *Hunk {
	hunk := &Hunk{
		Header:   header,
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
	}
	for _, raw := range rawLines {
		if len(raw) == 0 {
			continue
		}
		op := raw[0]
		content := raw[1:]
		var dl DiffLine
		switch op {
		case '+':
			dl = DiffLine{Op: DiffOpAdd, Content: content}
		case '-':
			dl = DiffLine{Op: DiffOpDelete, Content: content}
		default:
			dl = DiffLine{Op: DiffOpContext, Content: content}
		}
		hunk.Lines = append(hunk.Lines, dl)
	}
	hunk.Length = 1 + len(hunk.Lines)
	return hunk
}

func TestLineRangeToPatch_AllLinesSelected_EqualsHunkToPatch(t *testing.T) {
	// A hunk with context, delete, and add lines.
	// Selecting all lines should produce the same result as HunkToPatch.
	hunk := buildTestHunk(
		"@@ -1,4 +1,4 @@",
		1, 4, 1, 4,
		[]string{" line1", "-oldLine2", "+newLine2", " line3", " line4"},
	)

	// All lines selected: indices 0..4 (inclusive)
	got := LineRangeToPatch("file.txt", hunk, 0, len(hunk.Lines)-1, false)
	want := HunkToPatch("file.txt", hunk, false)

	assert.Equal(t, want, got)
}

func TestLineRangeToPatch_PartialAddSelection_DropsUnselectedAdds(t *testing.T) {
	// Hunk: 2 context, 3 add lines.
	// Select only the first add (index 2); the other two adds (indices 3,4) are dropped.
	hunk := buildTestHunk(
		"@@ -1,2 +1,5 @@",
		1, 2, 1, 5,
		[]string{" context1", " context2", "+add1", "+add2", "+add3"},
	)

	// Select only line index 2 (+add1)
	got := LineRangeToPatch("file.txt", hunk, 2, 2, false)

	// Expected: context1, context2, +add1 (add2 and add3 dropped)
	// OldCount = 2 context = 2
	// NewCount = 2 context + 1 selected add = 3
	expected := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1,2 +1,3 @@\n" +
		" context1\n" +
		" context2\n" +
		"+add1\n"

	assert.Equal(t, expected, got)
}

func TestLineRangeToPatch_PartialDeleteSelection_UnselectedBecomesContext(t *testing.T) {
	// Hunk: 1 context, 3 delete lines, 1 context.
	// Select only the second delete (index 2); others become context.
	hunk := buildTestHunk(
		"@@ -1,5 +1,2 @@",
		1, 5, 1, 2,
		[]string{" ctx1", "-del1", "-del2", "-del3", " ctx2"},
	)

	// Select only line index 2 (-del2)
	got := LineRangeToPatch("file.txt", hunk, 2, 2, false)

	// Expected: ctx1, -del1→context, -del2 (kept as delete), -del3→context, ctx2
	// Output lines: " ctx1", " del1", "-del2", " del3", " ctx2"
	// OldCount = ctx1 + del1(as context in old) + del2(delete in old) + del3(as context in old) + ctx2 = 5
	// NewCount = ctx1 + del1(now context in new) + del3(now context in new) + ctx2 = 4
	// (del2 is deleted so not in new)
	expected := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1,5 +1,4 @@\n" +
		" ctx1\n" +
		" del1\n" +
		"-del2\n" +
		" del3\n" +
		" ctx2\n"

	assert.Equal(t, expected, got)
}

func TestLineRangeToPatch_MixedSelection(t *testing.T) {
	// Hunk: context, delete, add, delete, add, context
	// Select indices 1..3 (first delete, add, second delete) but not second add (index 4)
	hunk := buildTestHunk(
		"@@ -1,4 +1,4 @@",
		1, 4, 1, 4,
		[]string{" ctx", "-del1", "+add1", "-del2", "+add2", " ctx2"},
	)

	// Select indices 1..3: -del1, +add1, -del2
	// +add2 (index 4) is NOT selected → dropped
	got := LineRangeToPatch("file.txt", hunk, 1, 3, false)

	// Output lines: ctx, -del1, +add1, -del2, ctx2  (add2 dropped)
	// OldCount = ctx + del1 + del2 + ctx2 = 4
	// NewCount = ctx + add1 + ctx2 = 3  (add2 dropped, del2 removed)
	expected := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1,4 +1,3 @@\n" +
		" ctx\n" +
		"-del1\n" +
		"+add1\n" +
		"-del2\n" +
		" ctx2\n"

	assert.Equal(t, expected, got)
}

func TestLineRangeToPatch_ReverseMode(t *testing.T) {
	// Reverse mode: ops are swapped and old/new counts swapped in header.
	// Hunk: context, delete, add, context
	hunk := buildTestHunk(
		"@@ -1,3 +1,3 @@",
		1, 3, 1, 3,
		[]string{" ctx1", "-oldLine", "+newLine", " ctx2"},
	)

	// Select all lines (reverse=true)
	got := LineRangeToPatch("file.txt", hunk, 0, len(hunk.Lines)-1, true)
	want := HunkToPatch("file.txt", hunk, true)

	assert.Equal(t, want, got)
}

func TestLineRangeToPatch_NilHunk_ReturnsEmpty(t *testing.T) {
	got := LineRangeToPatch("file.txt", nil, 0, 0, false)
	assert.Equal(t, "", got)
}

func TestLineRangeToPatch_Integration_PartialStage(t *testing.T) {
	// Integration: stage only a subset of lines from a hunk using LineRangeToPatch + ApplyPatch.
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Commit a file with 5 lines.
	addAndCommit(t, r, "partial.txt", "line1\nline2\nline3\nline4\nline5\n", "initial")

	// Modify all 5 lines in the worktree.
	addFile(t, r, "partial.txt", "LINE1\nLINE2\nLINE3\nLINE4\nLINE5\n")

	// Get the unstaged diff — expect a single hunk.
	diffs, err := r.UnstagedDiff(ctx, "partial.txt")
	require.NoError(t, err)
	require.NotEmpty(t, diffs)
	require.NotEmpty(t, diffs[0].Hunks)

	hunk := &diffs[0].Hunks[0]

	// Find indices of the first delete/add pair (line1→LINE1).
	// Typically: -line1, +LINE1, -line2, +LINE2, ...
	// We select only the first two lines (index 0 and 1: -line1, +LINE1).
	patch := LineRangeToPatch("partial.txt", hunk, 0, 1, false)
	require.NotEmpty(t, patch)

	err = r.ApplyPatch(ctx, patch, "--cached")
	require.NoError(t, err, "applying partial line range patch to stage")

	// partial.txt should now appear in both Staged (partial) and Unstaged.
	status, err := r.Status(ctx)
	require.NoError(t, err)

	foundStaged := false
	for _, e := range status.Staged {
		if e.Path == "partial.txt" {
			foundStaged = true
		}
	}
	assert.True(t, foundStaged, "partial.txt should have partially staged changes")
}

func TestLineRangeToPatch_Integration_PartialUnstage(t *testing.T) {
	// Integration: unstage only a subset of lines from a staged hunk using LineRangeToPatch + ApplyPatch.
	// We stage a file that has an added line followed by a deletion. By selecting only the
	// delete lines (not the adds) we can produce a well-formed reverse patch that git can apply.
	// Specifically: commit "aaa\nbbb\n", then stage "bbb\n" (deletes aaa), so the staged diff
	// shows "-aaa". Selecting index 0 (-aaa) with reverse=true produces a patch that re-adds
	// "aaa" to the index.
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Commit a file with two lines.
	addAndCommit(t, r, "unstage_partial.txt", "aaa\nbbb\n", "initial")

	// Stage a deletion of the first line (keep only bbb).
	addFile(t, r, "unstage_partial.txt", "bbb\n")
	_, err := r.runGit(ctx, "add", "unstage_partial.txt")
	require.NoError(t, err)

	// Verify fully staged.
	status, err := r.Status(ctx)
	require.NoError(t, err)
	foundStaged := false
	for _, e := range status.Staged {
		if e.Path == "unstage_partial.txt" {
			foundStaged = true
		}
	}
	require.True(t, foundStaged, "unstage_partial.txt should be fully staged")

	// Get staged diff — should show "-aaa" (the deleted first line).
	diffs, err := r.StagedDiff(ctx, "unstage_partial.txt")
	require.NoError(t, err)
	require.NotEmpty(t, diffs)
	require.NotEmpty(t, diffs[0].Hunks)

	hunk := &diffs[0].Hunks[0]

	// Find the delete line index.
	delIdx := -1
	for i, l := range hunk.Lines {
		if l.Op == DiffOpDelete {
			delIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, delIdx, 0, "expected at least one delete line in staged diff")

	// Select the delete line for unstage (reverse=true). This re-adds "aaa" to the index,
	// effectively undoing the staged deletion.
	patch := LineRangeToPatch("unstage_partial.txt", hunk, delIdx, delIdx, true)
	require.NotEmpty(t, patch)

	err = r.ApplyPatch(ctx, patch, "--cached")
	require.NoError(t, err, "applying partial line range patch to unstage")

	// After unstaging the deletion, the file should no longer appear as staged
	// (the index now matches HEAD).
	status, err = r.Status(ctx)
	require.NoError(t, err)

	foundStaged = false
	for _, e := range status.Staged {
		if e.Path == "unstage_partial.txt" {
			foundStaged = true
		}
	}
	assert.False(t, foundStaged, "unstage_partial.txt should no longer be staged after undoing the deletion")

	// The deletion should now appear as an unstaged change.
	foundUnstaged := false
	for _, e := range status.Unstaged {
		if e.Path == "unstage_partial.txt" {
			foundUnstaged = true
		}
	}
	assert.True(t, foundUnstaged, "unstage_partial.txt should appear as an unstaged change")
}

func TestLineRangeToPatch_PartialReverseUnstage(t *testing.T) {
	// Unstaging a partial selection: reverse=true, select only one of two adds.
	// Hunk (staged diff): context, +add1, +add2, context
	hunk := buildTestHunk(
		"@@ -1,2 +1,4 @@",
		1, 2, 1, 4,
		[]string{" ctx1", "+add1", "+add2", " ctx2"},
	)

	// Select only index 1 (+add1) with reverse=true to unstage just add1.
	// In reverse: +add1 (selected) becomes -add1, +add2 (unselected) dropped.
	// Output (reversed): ctx1, -add1, ctx2
	// For unstage patch: old is the staged state (has add1, add2), new is partially unstaged.
	// Reversed header: old↔new from the perspective of git apply -R --cached
	// With reverse=true and only add1 selected:
	//   - Forward OldCount = ctx1 + ctx2 = 2, NewCount = ctx1 + add1 + ctx2 = 3
	//   - Reversed header: @@ -3 +2 @@ i.e. @@ -NewCount +OldCount @@
	got := LineRangeToPatch("file.txt", hunk, 1, 1, true)

	// Reversed: @@ -NewStart,NewCount +OldStart,OldCount @@
	// Forward NewCount(selected) = 2 ctx + 1 add = 3, OldCount(selected) = 2 ctx
	// Reversed header: @@ -1,3 +1,2 @@
	expected := "diff --git a/file.txt b/file.txt\n" +
		"--- a/file.txt\n" +
		"+++ b/file.txt\n" +
		"@@ -1,3 +1,2 @@\n" +
		" ctx1\n" +
		"-add1\n" +
		" ctx2\n"

	assert.Equal(t, expected, got)
}
