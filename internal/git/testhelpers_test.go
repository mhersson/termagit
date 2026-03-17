package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/require"

	"github.com/go-git/go-billy/v5/memfs"
)

// testSignature returns a consistent author/committer for tests.
func testSignature() *object.Signature {
	return &object.Signature{
		Name:  "Test User",
		Email: "test@example.com",
		When:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}
}

// newMemRepo creates an in-memory go-git repository for fast tests.
// The repository is initialized with an empty commit on main branch.
func newMemRepo(t *testing.T) *Repository {
	t.Helper()

	fs := memfs.New()
	storage := memory.NewStorage()

	raw, err := git.Init(storage, fs)
	require.NoError(t, err, "init in-memory repo")

	// Create initial commit so HEAD exists
	wt, err := raw.Worktree()
	require.NoError(t, err, "get worktree")

	// Create a dummy file for initial commit
	f, err := fs.Create("README.md")
	require.NoError(t, err, "create README")
	_, err = f.Write([]byte("# Test Repo\n"))
	require.NoError(t, err, "write README")
	require.NoError(t, f.Close(), "close README")

	_, err = wt.Add("README.md")
	require.NoError(t, err, "stage README")

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author:    testSignature(),
		Committer: testSignature(),
	})
	require.NoError(t, err, "initial commit")

	return &Repository{
		raw:    raw,
		path:   "/",
		gitDir: "/.git",
		logger: nil,
	}
}

// addFile creates or overwrites a file in the repository worktree.
// For in-memory repos, the path is relative to the filesystem root.
// For on-disk repos, the path is relative to the worktree root.
func addFile(t *testing.T, r *Repository, path, content string) {
	t.Helper()

	wt, err := r.raw.Worktree()
	require.NoError(t, err, "get worktree")

	fs := wt.Filesystem

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		require.NoError(t, fs.MkdirAll(dir, 0o755), "create parent dir")
	}

	f, err := fs.Create(path)
	require.NoError(t, err, "create file %s", path)

	_, err = f.Write([]byte(content))
	require.NoError(t, err, "write file %s", path)
	require.NoError(t, f.Close(), "close file %s", path)
}

// stageFile stages a file in the repository.
func stageFile(t *testing.T, r *Repository, path string) {
	t.Helper()

	wt, err := r.raw.Worktree()
	require.NoError(t, err, "get worktree")

	_, err = wt.Add(path)
	require.NoError(t, err, "stage file %s", path)
}

// addAndCommit creates a file, stages it, and commits with the given message.
// Returns the commit hash.
func addAndCommit(t *testing.T, r *Repository, path, content, message string) plumbing.Hash {
	t.Helper()

	addFile(t, r, path, content)
	stageFile(t, r, path)

	wt, err := r.raw.Worktree()
	require.NoError(t, err, "get worktree")

	hash, err := wt.Commit(message, &git.CommitOptions{
		Author:    testSignature(),
		Committer: testSignature(),
	})
	require.NoError(t, err, "commit")

	return hash
}

// newTempRepo creates a real on-disk repository for integration tests.
// The repository is created in a temporary directory that is cleaned up
// when the test completes.
// Use this for tests that require real filesystem operations or subprocess calls.
func newTempRepo(t *testing.T) *Repository {
	t.Helper()

	dir := t.TempDir()

	raw, err := git.PlainInit(dir, false)
	require.NoError(t, err, "plain init temp repo")

	// Create initial commit
	wt, err := raw.Worktree()
	require.NoError(t, err, "get worktree")

	readmePath := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("# Test Repo\n"), 0o644), "write README")

	_, err = wt.Add("README.md")
	require.NoError(t, err, "stage README")

	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author:    testSignature(),
		Committer: testSignature(),
	})
	require.NoError(t, err, "initial commit")

	return &Repository{
		raw:    raw,
		path:   dir,
		gitDir: filepath.Join(dir, ".git"),
		logger: nil,
	}
}

// skipInShort skips the test if running with -short flag.
// Use for tests that shell out to git or require real filesystem operations.
func skipInShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}
}

// Tests for test helpers - validates our test infrastructure

func TestNewMemRepo_CreatesValidRepository(t *testing.T) {
	r := newMemRepo(t)

	// Should have a valid go-git repo
	require.NotNil(t, r.raw)

	// HEAD should exist and point to main/master
	head, err := r.raw.Head()
	require.NoError(t, err)
	require.True(t, head.Name().IsBranch())
}

func TestAddFile_CreatesFileInWorktree(t *testing.T) {
	r := newMemRepo(t)
	addFile(t, r, "test.txt", "hello world")

	wt, err := r.raw.Worktree()
	require.NoError(t, err)

	f, err := wt.Filesystem.Open("test.txt")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	buf := make([]byte, 100)
	n, err := f.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(buf[:n]))
}

func TestAddAndCommit_CreatesCommit(t *testing.T) {
	r := newMemRepo(t)
	hash := addAndCommit(t, r, "new.txt", "content", "Add new file")

	// Should be able to retrieve the commit
	commit, err := r.raw.CommitObject(hash)
	require.NoError(t, err)
	require.Equal(t, "Add new file", commit.Message)
}

func TestNewTempRepo_CreatesOnDiskRepository(t *testing.T) {
	r := newTempRepo(t)

	// Should have a real path
	require.DirExists(t, r.path)
	require.DirExists(t, r.gitDir)

	// Should have a valid go-git repo
	head, err := r.raw.Head()
	require.NoError(t, err)
	require.True(t, head.Name().IsBranch())
}
