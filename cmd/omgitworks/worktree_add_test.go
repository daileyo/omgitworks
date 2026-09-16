package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/xdg"
)

// setupWorktreeTestRepo creates a workspace with a git repo that has an initial commit.
func setupWorktreeTestRepo(t *testing.T, repoName string) (workspaceDir string, repoDir string) {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(xdg.EnvConfigHome, "")
	t.Setenv(xdg.EnvDataHome, "")

	workspaceDir = t.TempDir()
	resolved, err := filepath.EvalSymlinks(workspaceDir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	workspaceDir = resolved

	repoDir = filepath.Join(workspaceDir, repoName)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s\n%s", args, err, out)
		}
	}

	cfg := config.New(workspaceDir)
	cfg.Repositories = []config.Repository{
		{Name: repoName, Path: repoDir},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	return workspaceDir, repoDir
}

func TestRunWorktreeAdd_Success(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	err := runWorktreeAdd("my-repo", "feature-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify worktree directory was created
	wtPath := projectsPath(t, "my-repo", "feature-x")
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree directory should exist at %s: %v", wtPath, err)
	}

	// Verify config was updated
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	repo := cfg.Repositories[0]
	if len(repo.Worktrees) != 1 {
		t.Fatalf("expected 1 worktree in config, got %d", len(repo.Worktrees))
	}
	if repo.Worktrees[0].Branch != "feature-x" {
		t.Errorf("expected branch 'feature-x', got '%s'", repo.Worktrees[0].Branch)
	}
	if !repo.Worktrees[0].Aligned {
		t.Error("worktree should be aligned (inside the projects root)")
	}
}

func TestRunWorktreeAdd_CreatesWtDir(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	// The repo's projects dir should not exist yet
	wtDir := projectsPath(t, "my-repo")
	if _, err := os.Stat(wtDir); err == nil {
		t.Fatal("projects dir should not exist before add")
	}

	err := runWorktreeAdd("my-repo", "new-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The repo's projects dir should now exist
	if _, err := os.Stat(wtDir); err != nil {
		t.Errorf("projects dir should exist after add: %v", err)
	}
}

func TestRunWorktreeAdd_UnknownRepo(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	err := runWorktreeAdd("nonexistent", "feat")
	if err == nil {
		t.Fatal("expected error for unknown repo")
	}
	if got := err.Error(); got != "no repository found matching 'nonexistent'" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestRunWorktreeAdd_DuplicateBranch(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	// First add should succeed
	if err := runWorktreeAdd("my-repo", "dupe-branch"); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Second add with same branch should fail
	err := runWorktreeAdd("my-repo", "dupe-branch")
	if err == nil {
		t.Fatal("expected error for duplicate branch")
	}
	if got := err.Error(); !contains(got, "already exists") {
		t.Errorf("error should mention already exists, got: %s", got)
	}
}

func TestRunWorktreeAdd_BranchWithSlash(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	err := runWorktreeAdd("my-repo", "feature/auth-flow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify nested directory was created
	wtPath := projectsPath(t, "my-repo", "feature", "auth-flow")
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree directory should exist at %s: %v", wtPath, err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestWorktreeAdd_RepairsBeforeRebuild pins a deliberate behavior change: the
// add path used to list worktrees without repairing or pruning first, so it
// stored whatever git reported, dead entries included. It now shares the
// repair → prune → rebuild sequence with refresh and align.
//
// A surviving entry alone cannot tell the two apart — listing without repair
// still reports a worktree whose .git file is missing — so the test asserts the
// effects only repair and prune produce.
func TestWorktreeAdd_RepairsBeforeRebuild(t *testing.T) {
	workspace, repoDir := setupWorktreeTestRepo(t, "my-repo")
	mislinked := filepath.Join(workspace, "mislinked")
	gone := filepath.Join(workspace, "gone")
	gitWorktreeAdd(t, repoDir, "mislinked", mislinked)
	gitWorktreeAdd(t, repoDir, "gone", gone)
	breakWorktreeLink(t, mislinked)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	if err := runWorktreeAdd("my-repo", "feature-x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Repair ran: the broken link was restored rather than left for prune.
	if _, err := os.Stat(filepath.Join(mislinked, ".git")); err != nil {
		t.Errorf("repair should have restored the mislinked worktree's .git file: %v", err)
	}
	// Prune ran: git no longer records the deleted worktree.
	if containsPath(gitWorktreePaths(t, repoDir), gone) {
		t.Error("prune should have removed git's record of the deleted worktree")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	stored := map[string]bool{}
	for _, wt := range cfg.Repositories[0].Worktrees {
		stored[wt.Branch] = true
	}
	if !stored["mislinked"] || !stored["feature-x"] || stored["gone"] {
		t.Errorf("expected mislinked and feature-x stored without gone, got %v", stored)
	}
}
