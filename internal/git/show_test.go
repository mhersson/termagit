package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitOverview_ParsesFileStat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	// Create a temp repo with a commit that modifies multiple files
	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create initial commit
	writeFile(t, dir, "file1.txt", "initial content\n")
	writeFile(t, dir, "file2.txt", "another file\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "initial commit")

	// Second commit with modifications
	writeFile(t, dir, "file1.txt", "initial content\nmore content\neven more\n")
	writeFile(t, dir, "file2.txt", "another file\nwith changes\n")
	writeFile(t, dir, "file3.txt", "brand new file\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "make changes")

	// Get the commit hash
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	hash := strings.TrimSpace(string(out))

	repo, err := Open(dir, nil)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	overview, err := repo.CommitOverview(context.Background(), hash)
	if err != nil {
		t.Fatalf("CommitOverview failed: %v", err)
	}

	// Should have 3 files
	if len(overview.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(overview.Files))
	}

	// Should have a summary line
	if overview.Summary == "" {
		t.Error("expected non-empty summary")
	}
	if !strings.Contains(overview.Summary, "changed") {
		t.Errorf("summary should contain 'changed': %s", overview.Summary)
	}

	// Check file entries
	paths := make(map[string]bool)
	for _, f := range overview.Files {
		paths[f.Path] = true
		if f.Changes == "" {
			t.Errorf("file %s has empty changes", f.Path)
		}
	}

	if !paths["file1.txt"] {
		t.Error("expected file1.txt in overview")
	}
	if !paths["file3.txt"] {
		t.Error("expected file3.txt in overview")
	}
}

func TestCommitOverview_HandlesBinaryFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	// Create a binary file (PNG header)
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(dir, "image.png"), binaryData, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "add binary")

	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	hash := strings.TrimSpace(string(out))

	repo, err := Open(dir, nil)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	overview, err := repo.CommitOverview(context.Background(), hash)
	if err != nil {
		t.Fatalf("CommitOverview failed: %v", err)
	}

	if len(overview.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(overview.Files))
	}

	f := overview.Files[0]
	if f.Path != "image.png" {
		t.Errorf("expected path image.png, got %s", f.Path)
	}
	if !f.IsBinary {
		t.Error("expected IsBinary to be true for PNG file")
	}
}

func TestVerifyCommit_ReturnsSignatureInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell-out test in short mode")
	}

	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	runCmd(t, dir, "git", "config", "user.name", "Test User")

	writeFile(t, dir, "file.txt", "content\n")
	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "unsigned commit")

	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	hash := strings.TrimSpace(string(out))

	repo, err := Open(dir, nil)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	sig, err := repo.VerifyCommit(context.Background(), hash)
	if err != nil {
		t.Fatalf("VerifyCommit failed: %v", err)
	}

	// Unsigned commit should have "none" status
	if sig.Status != "none" {
		t.Errorf("expected status 'none' for unsigned commit, got %q", sig.Status)
	}
}

// Helper functions
func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2024-01-01T12:00:00Z", "GIT_COMMITTER_DATE=2024-01-01T12:00:00Z")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}
