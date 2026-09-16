package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/repocontext"
	"github.com/daileyo/omgitworks/internal/xdg"
)

// chdirForTest moves into dir for the duration of the test, following the
// pattern in init_test.go. Tests using it must not call t.Parallel.
func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// setupTwoWorktreeTestRepos creates a workspace tracking two git repositories.
func setupTwoWorktreeTestRepos(t *testing.T, nameA, nameB string) (repoA, repoB string) {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(xdg.EnvConfigHome, "")
	t.Setenv(xdg.EnvDataHome, "")

	workspaceDir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(workspaceDir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	workspaceDir = resolved

	var paths []string
	for _, name := range []string{nameA, nameB} {
		dir := filepath.Join(workspaceDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
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
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v failed: %s\n%s", args, err, out)
			}
		}
		paths = append(paths, dir)
	}

	cfg := config.New(workspaceDir)
	cfg.Repositories = []config.Repository{
		{Name: nameA, Path: paths[0]},
		{Name: nameB, Path: paths[1]},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	return paths[0], paths[1]
}

func TestRunWorktreeAdd_ResolvesCurrentRepo(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")
	chdirForTest(t, repoDir)

	if err := runWorktreeAddCurrent("feat-x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := projectsPath(t, "my-repo", "feat-x")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("worktree not created at %s: %v", expected, err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if len(cfg.Repositories[0].Worktrees) != 1 {
		t.Fatalf("expected 1 recorded worktree, got %d", len(cfg.Repositories[0].Worktrees))
	}
	if got := cfg.Repositories[0].Worktrees[0].Branch; got != "feat-x" {
		t.Errorf("recorded branch %q, want feat-x", got)
	}
}

func TestRunWorktreeAdd_ResolvesFromSubdirectory(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")

	nested := filepath.Join(repoDir, "internal", "deep")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	chdirForTest(t, nested)

	if err := runWorktreeAddCurrent("feat-x"); err != nil {
		t.Fatalf("unexpected error from a nested directory: %v", err)
	}

	expected := projectsPath(t, "my-repo", "feat-x")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("worktree not created at %s: %v", expected, err)
	}
}

func TestRunWorktreeAdd_ResolvesFromInsideWorktree(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")

	// Create the first worktree the normal way, then work from inside it.
	if err := runWorktreeAdd("my-repo", "feat-x"); err != nil {
		t.Fatalf("failed to create the first worktree: %v", err)
	}
	firstWt := projectsPath(t, "my-repo", "feat-x")
	chdirForTest(t, firstWt)

	if err := runWorktreeAddCurrent("feat-y"); err != nil {
		t.Fatalf("unexpected error from inside a worktree: %v", err)
	}

	// The second worktree belongs to the owning repository, not the worktree.
	expected := projectsPath(t, "my-repo", "feat-y")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("worktree not created at %s: %v", expected, err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if cfg.Repositories[0].Path != repoDir {
		t.Errorf("worktree recorded against %q, want the owning repo %q", cfg.Repositories[0].Path, repoDir)
	}
	if len(cfg.Repositories[0].Worktrees) != 2 {
		t.Errorf("expected 2 recorded worktrees, got %d", len(cfg.Repositories[0].Worktrees))
	}
}

func TestRunWorktreeAdd_ExplicitRepoIgnoresCwd(t *testing.T) {
	repoA, _ := setupTwoWorktreeTestRepos(t, "repo-a", "repo-b")

	// Stand inside repo-a but name repo-b explicitly.
	chdirForTest(t, repoA)
	if err := runWorktreeAdd("repo-b", "feat-x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(projectsPath(t, "repo-b", "feat-x")); err != nil {
		t.Errorf("worktree not created for the named repo: %v", err)
	}
	if _, err := os.Stat(projectsPath(t, "repo-a", "feat-x")); err == nil {
		t.Error("worktree was created for the current repo; the explicit argument must win")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	for _, repo := range cfg.Repositories {
		if repo.Name == "repo-a" && len(repo.Worktrees) != 0 {
			t.Errorf("repo-a gained %d worktrees; it should be untouched", len(repo.Worktrees))
		}
		if repo.Name == "repo-b" && len(repo.Worktrees) != 1 {
			t.Errorf("repo-b has %d worktrees, want 1", len(repo.Worktrees))
		}
	}
}

func TestRunWorktreeAddCurrent_NotTracked(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	// A directory that is not inside any tracked repository.
	chdirForTest(t, t.TempDir())

	err := runWorktreeAddCurrent("feat-x")
	if !errors.Is(err, repocontext.ErrNotTracked) {
		t.Fatalf("got %v, want ErrNotTracked", err)
	}

	// No side effects: nothing created under the projects root.
	if _, statErr := os.Stat(projectsPath(t, "my-repo", "feat-x")); statErr == nil {
		t.Error("a worktree was created despite the resolution failure")
	}
}

func TestWorktreeAddCmd_Arity(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"zero args", []string{}, true},
		{"one arg (branch only)", []string{"feat-x"}, false},
		{"two args (repo and branch)", []string{"my-repo", "feat-x"}, false},
		{"three args", []string{"a", "b", "c"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := worktreeAddCmd.Args(worktreeAddCmd, tt.args)
			if tt.wantErr && err == nil {
				t.Errorf("Args(%v) = nil, want an error", tt.args)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Args(%v) = %v, want nil", tt.args, err)
			}
		})
	}
}
