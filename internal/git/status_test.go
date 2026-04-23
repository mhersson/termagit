package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mhersson/termagit/internal/cmdlog"
)

// ModeText tests - must match Neogit exactly

func TestModeText_MatchesNeogit(t *testing.T) {
	// These exact values are from Neogit's config.lua lines 483-500
	expected := map[string]string{
		"M":  "modified",
		"N":  "new file",
		"A":  "added",
		"D":  "deleted",
		"C":  "copied",
		"U":  "updated",
		"R":  "renamed",
		"T":  "changed",
		"DD": "unmerged",
		"AU": "unmerged",
		"UD": "unmerged",
		"UA": "unmerged",
		"DU": "unmerged",
		"AA": "unmerged",
		"UU": "unmerged",
		"?":  "",
	}

	for code, text := range expected {
		assert.Equal(t, text, ModeText[code], "ModeText[%q]", code)
	}
}

// StatusEntry tests

func TestStatusEntry_Path_ReturnsPath(t *testing.T) {
	e := StatusEntry{Path: "foo.txt"}
	assert.Equal(t, "foo.txt", e.Path)
}

func TestStatusEntry_OrigPath_ReturnsOrigPath(t *testing.T) {
	e := StatusEntry{Path: "new.txt", OrigPath: "old.txt"}
	assert.Equal(t, "old.txt", e.OrigPath)
}

// Status parsing tests

func TestStatus_ReturnsUntracked(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Add an untracked file
	addFile(t, r, "untracked.txt", "content")

	result, err := r.Status(ctx)
	require.NoError(t, err)

	require.Len(t, result.Untracked, 1)
	assert.Equal(t, "untracked.txt", result.Untracked[0].Path)
	assert.Equal(t, FileStatusUntracked, result.Untracked[0].Unstaged)
}

func TestStatus_ReturnsStagedFile(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Add and stage a new file
	addFile(t, r, "staged.txt", "content")
	stageFile(t, r, "staged.txt")

	result, err := r.Status(ctx)
	require.NoError(t, err)

	require.Len(t, result.Staged, 1)
	assert.Equal(t, "staged.txt", result.Staged[0].Path)
	assert.Equal(t, FileStatusNew, result.Staged[0].Staged)
}

func TestStatus_ReturnsUnstagedModification(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Modify an existing committed file
	addFile(t, r, "README.md", "modified content")

	result, err := r.Status(ctx)
	require.NoError(t, err)

	require.Len(t, result.Unstaged, 1)
	assert.Equal(t, "README.md", result.Unstaged[0].Path)
	assert.Equal(t, FileStatusModified, result.Unstaged[0].Unstaged)
}

func TestStatus_ReturnsStagedModification(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	// Modify and stage an existing committed file
	addFile(t, r, "README.md", "modified content")
	stageFile(t, r, "README.md")

	result, err := r.Status(ctx)
	require.NoError(t, err)

	require.Len(t, result.Staged, 1)
	assert.Equal(t, "README.md", result.Staged[0].Path)
	assert.Equal(t, FileStatusModified, result.Staged[0].Staged)
}

func TestStatus_ReturnsDeletedFile(t *testing.T) {
	r := newTempRepo(t)
	ctx := context.Background()

	// Commit a file then delete it from worktree
	addAndCommit(t, r, "todelete.txt", "content", "Add file to delete")

	// Delete the file from filesystem
	wt, err := r.raw.Worktree()
	require.NoError(t, err)
	_, err = wt.Remove("todelete.txt")
	require.NoError(t, err)

	result, err := r.Status(ctx)
	require.NoError(t, err)

	require.Len(t, result.Staged, 1)
	assert.Equal(t, "todelete.txt", result.Staged[0].Path)
	assert.Equal(t, FileStatusDeleted, result.Staged[0].Staged)
}

func TestStatus_EmptyRepository_ReturnsEmpty(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	result, err := r.Status(ctx)
	require.NoError(t, err)

	assert.Empty(t, result.Untracked)
	assert.Empty(t, result.Unstaged)
	assert.Empty(t, result.Staged)
}

func TestStatus_RenamedFile_HasOrigPath(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	// Create and commit a file
	addAndCommit(t, r, "original.txt", "some content\n", "Add original file")

	// Rename via git mv
	cmd := exec.Command("git", "mv", "original.txt", "renamed.txt")
	cmd.Dir = r.path
	require.NoError(t, cmd.Run())

	result, err := r.Status(ctx)
	require.NoError(t, err)

	// The renamed file should appear in staged with OrigPath set
	var found bool
	for _, e := range result.Staged {
		if e.Path == "renamed.txt" {
			found = true
			assert.Equal(t, "original.txt", e.OrigPath, "OrigPath should be the old filename")
			assert.Equal(t, FileStatusRenamed, e.Staged)
			break
		}
	}
	assert.True(t, found, "renamed file should appear in staged entries")
}

func TestStatus_LogsEntry(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	logPath := filepath.Join(t.TempDir(), "status.log")
	logger, err := cmdlog.New(logPath, 1<<20, 2)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	r.logger = logger

	_, err = r.Status(ctx)
	require.NoError(t, err)

	// Flush and read back
	require.NoError(t, logger.Close())

	entries, err := cmdlog.ReadRecent(logPath, 10)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "Status should log at least one command")
	// Should have logged a git status command
	assert.Contains(t, entries[0].Command, "status")
}

// SubmoduleStatus parsing tests

func TestParseSubmoduleStatus_ReturnsNil_ForNonSubmodule(t *testing.T) {
	result := parseSubmoduleStatus("N...")
	assert.Nil(t, result)
}

func TestParseSubmoduleStatus_ParsesCommitChanged(t *testing.T) {
	result := parseSubmoduleStatus("SC..")
	require.NotNil(t, result)
	assert.True(t, result.CommitChanged)
	assert.False(t, result.HasTrackedChanges)
	assert.False(t, result.HasUntrackedChanges)
}

func TestParseSubmoduleStatus_ParsesTrackedChanges(t *testing.T) {
	result := parseSubmoduleStatus("S.M.")
	require.NotNil(t, result)
	assert.False(t, result.CommitChanged)
	assert.True(t, result.HasTrackedChanges)
	assert.False(t, result.HasUntrackedChanges)
}

func TestParseSubmoduleStatus_ParsesUntrackedChanges(t *testing.T) {
	result := parseSubmoduleStatus("S..U")
	require.NotNil(t, result)
	assert.False(t, result.CommitChanged)
	assert.False(t, result.HasTrackedChanges)
	assert.True(t, result.HasUntrackedChanges)
}

func TestParseSubmoduleStatus_ParsesAllFlags(t *testing.T) {
	result := parseSubmoduleStatus("SCMU")
	require.NotNil(t, result)
	assert.True(t, result.CommitChanged)
	assert.True(t, result.HasTrackedChanges)
	assert.True(t, result.HasUntrackedChanges)
}

// Porcelain v2 parsing tests

func TestParsePorcelainV2_Kind1_Ordinary(t *testing.T) {
	// Kind 1: ordinary change
	// Format: 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
	line := "1 M. N... 100644 100644 100644 abc1234 def5678 modified.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "modified.txt", entry.Path)
	assert.Equal(t, FileStatusModified, entry.Staged)
	assert.Equal(t, FileStatusNone, entry.Unstaged)
}

func TestParsePorcelainV2_Kind1_StagedAndUnstaged(t *testing.T) {
	// File is both staged and has unstaged changes
	line := "1 MM N... 100644 100644 100644 abc1234 def5678 both.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "both.txt", entry.Path)
	assert.Equal(t, FileStatusModified, entry.Staged)
	assert.Equal(t, FileStatusModified, entry.Unstaged)
}

func TestParsePorcelainV2_Kind2_Renamed(t *testing.T) {
	// Kind 2: renamed/copied
	// Format: 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <score> <path><TAB><origPath>
	line := "2 R. N... 100644 100644 100644 abc1234 def5678 R100 new.txt\told.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "new.txt", entry.Path)
	assert.Equal(t, "old.txt", entry.OrigPath)
	assert.Equal(t, FileStatusRenamed, entry.Staged)
}

func TestParsePorcelainV2_KindUntracked(t *testing.T) {
	// Kind ?: untracked
	line := "? untracked.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "untracked.txt", entry.Path)
	assert.Equal(t, FileStatusUntracked, entry.Unstaged)
}

func TestParsePorcelainV2_KindU_Unmerged(t *testing.T) {
	// Kind u: unmerged
	// Format: u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
	line := "u UU N... 100644 100644 100644 100644 abc1234 def5678 ghi9012 conflicted.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "conflicted.txt", entry.Path)
	assert.Equal(t, "UU", entry.UnmergedMode)
}

func TestParsePorcelainV2_NewFile_UsesN(t *testing.T) {
	// When a file is new (hH is all zeros), staged mode should be N
	line := "1 A. N... 000000 100644 100644 0000000000000000000000000000000000000000 abc1234 newfile.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	// Per Neogit, new files use "N" for mode, not "A"
	assert.Equal(t, FileStatusNew, entry.Staged)
}

func TestParseKind1_PathWithSpaces(t *testing.T) {
	// Filenames containing spaces must be preserved in full.
	// Format: 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
	line := "1 M. N... 100644 100644 100644 abc1234 def5678 Everforest Dark Medium"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "Everforest Dark Medium", entry.Path)
}

func TestParseKind2_PathWithSpaces(t *testing.T) {
	// Renamed files whose new name contains spaces must have the full path preserved.
	// Format: 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <score> <path><TAB><origPath>
	line := "2 R. N... 100644 100644 100644 abc1234 def5678 R100 Everforest Dark Medium\told.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "Everforest Dark Medium", entry.Path)
	assert.Equal(t, "old.txt", entry.OrigPath)
}

func TestParseKindU_PathWithSpaces(t *testing.T) {
	// Unmerged files whose path contains spaces must have the full path preserved.
	// Format: u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
	line := "u UU N... 100644 100644 100644 100644 abc1234 def5678 ghi9012 Everforest Dark Medium"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "Everforest Dark Medium", entry.Path)
	assert.Equal(t, "UU", entry.UnmergedMode)
}

func TestParseKind2_OrigPathWithSpaces(t *testing.T) {
	// The origPath (source of rename) may also contain spaces and must be preserved in full.
	// Format: 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <score> <path><TAB><origPath>
	line := "2 R. N... 100644 100644 100644 abc1234 def5678 R100 new.txt\tMy Old File.txt"
	entry, err := parsePorcelainLine(line)
	require.NoError(t, err)
	require.NotNil(t, entry)

	assert.Equal(t, "new.txt", entry.Path)
	assert.Equal(t, "My Old File.txt", entry.OrigPath)
	assert.Equal(t, FileStatusRenamed, entry.Staged)
}
