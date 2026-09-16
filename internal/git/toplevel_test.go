package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// initTestRepo creates a git repository with one empty commit and returns its
// symlink-resolved path, so comparisons are not defeated by /var -> /private/var.
func initTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	dir = resolved

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}
	return dir
}

func TestToplevel_RepoRoot(t *testing.T) {
	repo := initTestRepo(t)

	got, err := Toplevel(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ResolvePath(got) != repo {
		t.Errorf("Toplevel = %q, want %q", got, repo)
	}
}

func TestToplevel_Subdirectory(t *testing.T) {
	repo := initTestRepo(t)
	nested := filepath.Join(repo, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	got, err := Toplevel(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// git walks upward itself, so a directory three levels deep still reports
	// the checkout root rather than itself.
	if ResolvePath(got) != repo {
		t.Errorf("Toplevel from nested dir = %q, want %q", got, repo)
	}
}

func TestToplevel_NotAGitRepo(t *testing.T) {
	if _, err := Toplevel(t.TempDir()); err == nil {
		t.Error("expected an error outside a git repository, got nil")
	}
}

func TestListBranches(t *testing.T) {
	repo := initTestRepo(t)

	cmd := exec.Command("git", "branch", "feature-x")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch failed: %s\n%s", err, out)
	}

	branches, err := ListBranches(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := make(map[string]bool, len(branches))
	for _, b := range branches {
		found[b] = true
	}
	if !found["feature-x"] {
		t.Errorf("ListBranches = %v, want it to contain feature-x", branches)
	}
	if len(branches) != 2 {
		t.Errorf("ListBranches returned %d branches (%v), want 2", len(branches), branches)
	}
}

func TestListBranches_NotAGitRepo(t *testing.T) {
	if _, err := ListBranches(t.TempDir()); err == nil {
		t.Error("expected an error outside a git repository, got nil")
	}
}
