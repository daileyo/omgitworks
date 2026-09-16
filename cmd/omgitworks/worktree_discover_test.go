package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
)

// gitWorktreeAdd creates a linked worktree with plain git, bypassing omgitworks
// entirely, the way a user working outside the tool would.
func gitWorktreeAdd(t *testing.T, repoDir, branch, wtPath string) {
	t.Helper()
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add %s failed: %s\n%s", branch, err, out)
	}
}

// breakWorktreeLink deletes a linked worktree's .git file while leaving its
// directory in place. Prune treats such an entry as dead, because the location
// git records for it no longer exists, but repair run from the main repository
// rewrites the file. It is the damage that makes repair-before-prune matter.
func breakWorktreeLink(t *testing.T, wtPath string) {
	t.Helper()
	if err := os.Remove(filepath.Join(wtPath, ".git")); err != nil {
		t.Fatalf("failed to remove .git file: %v", err)
	}
}

// gitWorktreePaths returns every linked worktree path git itself still records,
// including prunable ones, so tests can observe whether prune ran.
func gitWorktreePaths(t *testing.T, repoDir string) []string {
	t.Helper()
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree list failed: %s\n%s", err, out)
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if p, ok := strings.CutPrefix(line, "worktree "); ok && filepath.Clean(p) != filepath.Clean(repoDir) {
			paths = append(paths, p)
		}
	}
	return paths
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if filepath.Clean(p) == filepath.Clean(want) {
			return true
		}
	}
	return false
}

func TestBuildWorktreeEntries_SkipsMissingDirectories(t *testing.T) {
	workspace, repoDir := setupWorktreeTestRepo(t, "my-repo")
	live := filepath.Join(workspace, "live")
	gone := filepath.Join(workspace, "gone")
	gitWorktreeAdd(t, repoDir, "live", live)
	gitWorktreeAdd(t, repoDir, "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	wts, err := buildWorktreeEntries(repoDir, "my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(wts) != 1 || wts[0].Path != live || wts[0].Branch != "live" {
		t.Errorf("expected only the live worktree, got %+v", wts)
	}
	// Building entries reads git state only; the dead entry is still recorded.
	if !containsPath(gitWorktreePaths(t, repoDir), gone) {
		t.Error("buildWorktreeEntries must not prune git's own records")
	}
}

func TestBuildWorktreeEntries_NilWhenNoneSurvive(t *testing.T) {
	workspace, repoDir := setupWorktreeTestRepo(t, "my-repo")

	wts, err := buildWorktreeEntries(repoDir, "my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wts != nil {
		t.Errorf("repository without worktrees: expected nil, got %#v", wts)
	}

	gone := filepath.Join(workspace, "gone")
	gitWorktreeAdd(t, repoDir, "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	wts, err = buildWorktreeEntries(repoDir, "my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wts != nil {
		t.Errorf("all worktrees missing: expected nil so the field is omitted, got %#v", wts)
	}
}

func TestBuildWorktreeEntries_RecomputesAligned(t *testing.T) {
	workspace, repoDir := setupWorktreeTestRepo(t, "my-repo")
	loose := filepath.Join(workspace, "loose")
	aligned := projectsPath(t, "my-repo", "inside")
	gitWorktreeAdd(t, repoDir, "loose", loose)
	gitWorktreeAdd(t, repoDir, "inside", aligned)

	wts, err := buildWorktreeEntries(repoDir, "my-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := map[string]bool{}
	for _, wt := range wts {
		got[wt.Branch] = wt.Aligned
	}
	if got["loose"] || !got["inside"] {
		t.Errorf("expected loose unaligned and inside aligned, got %v", got)
	}
}

func TestSyncRepoWorktrees_PropagatesListError(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")
	stored := []config.Worktree{{Path: "/stale", Branch: "stale"}}
	repo := &config.Repository{Name: "my-repo", Path: repoDir, Worktrees: stored}
	breakRepo(t, repoDir)

	if _, err := buildWorktreeEntries(repoDir, "my-repo"); err == nil {
		t.Error("buildWorktreeEntries: expected an error for an unreadable repository")
	}
	if err := syncRepoWorktrees(repo); err == nil {
		t.Error("syncRepoWorktrees: expected the list error to be returned, not swallowed")
	}
	if len(repo.Worktrees) != 1 || repo.Worktrees[0].Path != "/stale" {
		t.Errorf("stored data must be left untouched on failure, got %+v", repo.Worktrees)
	}
}

func TestSyncRepoWorktrees_RepairsBeforePruning(t *testing.T) {
	workspace, repoDir := setupWorktreeTestRepo(t, "my-repo")
	wtPath := filepath.Join(workspace, "mislinked")
	gitWorktreeAdd(t, repoDir, "mislinked", wtPath)
	breakWorktreeLink(t, wtPath)

	repo := &config.Repository{Name: "my-repo", Path: repoDir}
	if err := syncRepoWorktrees(repo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Had prune run first, the entry would be gone from git and from storage.
	if len(repo.Worktrees) != 1 || repo.Worktrees[0].Path != wtPath {
		t.Fatalf("recoverable worktree should survive, got %+v", repo.Worktrees)
	}
	if _, err := os.Stat(filepath.Join(wtPath, ".git")); err != nil {
		t.Errorf("repair should have restored the .git file: %v", err)
	}
}

func TestSyncRepoWorktrees_PrunesDeadEntries(t *testing.T) {
	workspace, repoDir := setupWorktreeTestRepo(t, "my-repo")
	gone := filepath.Join(workspace, "gone")
	gitWorktreeAdd(t, repoDir, "gone", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	repo := &config.Repository{Name: "my-repo", Path: repoDir}
	if err := syncRepoWorktrees(repo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.Worktrees != nil {
		t.Errorf("expected nil worktrees, got %+v", repo.Worktrees)
	}
	if containsPath(gitWorktreePaths(t, repoDir), gone) {
		t.Error("prune should have removed git's record of the deleted worktree")
	}
}
