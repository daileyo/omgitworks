package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoWithWorktree creates a repository with one worktree on branch and
// returns both paths, with symlinks resolved.
func repoWithWorktree(t *testing.T, branch string) (repo, worktree string) {
	t.Helper()
	repo = initTestRepo(t)

	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	worktree = filepath.Join(parent, branch)

	cmd := exec.Command("git", "worktree", "add", "-b", branch, worktree)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %s\n%s", err, out)
	}
	return repo, worktree
}

// branchExists reports whether the branch is still present in the repository.
func branchExists(t *testing.T, repo, branch string) bool {
	t.Helper()
	cmd := exec.Command("git", "branch", "--list", branch)
	cmd.Dir = repo
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --list failed: %v", err)
	}
	return strings.Contains(string(out), branch)
}

func TestRemoveWorktree(t *testing.T) {
	repo, wt := repoWithWorktree(t, "feat-x")

	if err := RemoveWorktree(repo, wt, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree directory still present at %s", wt)
	}

	entries, err := ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees failed: %v", err)
	}
	for _, e := range entries {
		if e.Branch == "feat-x" {
			t.Errorf("git still reports a worktree for feat-x at %s", e.Path)
		}
	}
}

func TestRemoveWorktree_DirtyRefused(t *testing.T) {
	repo, wt := repoWithWorktree(t, "feat-dirty")

	// A tracked file with uncommitted changes.
	tracked := filepath.Join(wt, "file.txt")
	if err := os.WriteFile(tracked, []byte("v1\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "add file"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = wt
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(tracked, []byte("v2 uncommitted\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// git must refuse without force.
	if err := RemoveWorktree(repo, wt, false); err == nil {
		t.Fatal("removal succeeded with uncommitted changes and no force")
	}
	if _, err := os.Stat(wt); err != nil {
		t.Errorf("worktree should still exist after a refused removal: %v", err)
	}

	// And succeed with it.
	if err := RemoveWorktree(repo, wt, true); err != nil {
		t.Fatalf("forced removal failed: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Error("worktree still present after a forced removal")
	}
}

func TestRemoveWorktree_UntrackedRefused(t *testing.T) {
	repo, wt := repoWithWorktree(t, "feat-untracked")

	if err := os.WriteFile(filepath.Join(wt, "scratch.txt"), []byte("notes\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// The guard is git's own, and it covers untracked files too.
	if err := RemoveWorktree(repo, wt, false); err == nil {
		t.Fatal("removal succeeded with an untracked file and no force")
	}
	if _, err := os.Stat(wt); err != nil {
		t.Errorf("worktree should still exist: %v", err)
	}

	if err := RemoveWorktree(repo, wt, true); err != nil {
		t.Fatalf("forced removal failed: %v", err)
	}
}

func TestRemoveWorktree_BranchSurvives(t *testing.T) {
	repo, wt := repoWithWorktree(t, "feat-keep")

	if err := RemoveWorktree(repo, wt, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Removing a checkout must never discard the work on it.
	if !branchExists(t, repo, "feat-keep") {
		t.Error("branch feat-keep was deleted along with its worktree")
	}
}

func TestRemoveWorktree_ErrorKeepsGitReason(t *testing.T) {
	repo, wt := repoWithWorktree(t, "feat-reason")

	if err := os.WriteFile(filepath.Join(wt, "scratch.txt"), []byte("x\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	err := RemoveWorktree(repo, wt, false)
	if err == nil {
		t.Fatal("expected a refusal")
	}

	msg := err.Error()
	if !strings.Contains(msg, "failed to remove worktree") {
		t.Errorf("error lost its context wrapper: %s", msg)
	}
	// git's own explanation must survive, since it is the only reason the user gets.
	if !strings.Contains(msg, "untracked") && !strings.Contains(msg, "contains modified") {
		t.Errorf("error replaced git's reason instead of wrapping it: %s", msg)
	}
}

func TestRemoveWorktree_NotAWorktree(t *testing.T) {
	repo := initTestRepo(t)

	if err := RemoveWorktree(repo, filepath.Join(t.TempDir(), "nope"), false); err == nil {
		t.Error("expected an error removing a path that is not a worktree")
	}
}
