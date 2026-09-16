package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/repocontext"
	"github.com/daileyo/omgitworks/internal/xdg"
)

// setupScopeFixture tracks two git repositories, each with one worktree created
// outside the projects root so it counts as unaligned. It returns both repo
// paths and reloads config so callers see the recorded worktrees.
func setupScopeFixture(t *testing.T, nameA, nameB string) (repoA, repoB string) {
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

	cfg := config.New(workspaceDir)

	var paths []string
	for _, name := range []string{nameA, nameB} {
		dir := filepath.Join(workspaceDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create repo dir: %v", err)
		}
		for _, args := range [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@test.com"},
			{"git", "config", "user.name", "Test"},
			{"git", "commit", "--allow-empty", "-m", "init"},
		} {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v failed: %s\n%s", args, err, out)
			}
		}

		// An unaligned worktree: outside the projects root, so align has work.
		branch := "feat-" + name
		wtPath := filepath.Join(workspaceDir, "loose", name)
		cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git worktree add failed: %s\n%s", err, out)
		}

		cfg.Repositories = append(cfg.Repositories, config.Repository{
			Name:      name,
			Path:      dir,
			Worktrees: []config.Worktree{{Path: wtPath, Branch: branch, Aligned: false}},
		})
		paths = append(paths, dir)
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	return paths[0], paths[1]
}

// scopeFor builds a scope from the loaded config, failing the test on error.
func scopeFor(t *testing.T, arg string) worktreeScope {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	scope, err := worktreeScopeFor(cfg, arg)
	if err != nil {
		t.Fatalf("worktreeScopeFor(%q) failed: %v", arg, err)
	}
	return scope
}

func TestRunWorktreeList_NoArgListsAllRepos(t *testing.T) {
	setupScopeFixture(t, "repo-a", "repo-b")

	var buf bytes.Buffer
	if err := runWorktreeList(worktreeScope{}, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// Omitting the argument still means every tracked repository.
	for _, want := range []string{"repo-a", "repo-b"} {
		if !strings.Contains(out, want) {
			t.Errorf("bare list omitted %s:\n%s", want, out)
		}
	}
}

func TestRunWorktreeList_DotScopesToCurrentRepo(t *testing.T) {
	repoA, _ := setupScopeFixture(t, "repo-a", "repo-b")
	chdirForTest(t, repoA)

	var buf bytes.Buffer
	if err := runWorktreeList(scopeFor(t, "."), &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "feat-repo-a") {
		t.Errorf("list . omitted the current repo's worktree:\n%s", out)
	}
	if strings.Contains(out, "feat-repo-b") {
		t.Errorf("list . included another repo's worktree:\n%s", out)
	}
}

func TestRunWorktreeList_DotNotTreatedAsNamePattern(t *testing.T) {
	// A tracked repository whose name contains a period. If "." were passed
	// through to pattern matching, this repo would match it.
	repoA, _ := setupScopeFixture(t, "repo-a", "my.repo")
	chdirForTest(t, repoA)

	scope := scopeFor(t, ".")
	if scope.NamePattern != "" {
		t.Errorf("scope.NamePattern = %q, want empty: '.' must not become a pattern", scope.NamePattern)
	}
	if scope.RepoPath == "" {
		t.Error("scope.RepoPath is empty; '.' should resolve to an exact path")
	}

	var buf bytes.Buffer
	if err := runWorktreeList(scope, &buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(buf.String(), "my.repo") {
		t.Errorf("'.' matched a repo whose name contains a period:\n%s", buf.String())
	}
}

func TestRunWorktreeAlign_NoArgProcessesAllRepos(t *testing.T) {
	setupScopeFixture(t, "repo-a", "repo-b")

	output := captureStdoutStr(func() {
		if err := runWorktreeAlign(worktreeScope{}, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, want := range []string{"repo-a", "repo-b"} {
		if !strings.Contains(output, want) {
			t.Errorf("bare align skipped %s:\n%s", want, output)
		}
	}
}

func TestRunWorktreeAlign_DotScopesToCurrentRepo(t *testing.T) {
	repoA, _ := setupScopeFixture(t, "repo-a", "repo-b")
	chdirForTest(t, repoA)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	var repoBWtPath string
	for _, repo := range cfg.Repositories {
		if repo.Name == "repo-b" {
			repoBWtPath = repo.Worktrees[0].Path
		}
	}

	output := captureStdoutStr(func() {
		if err := runWorktreeAlign(scopeFor(t, "."), false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "repo-a") {
		t.Errorf("align . skipped the current repo:\n%s", output)
	}
	if strings.Contains(output, "repo-b") {
		t.Errorf("align . planned a move for another repo:\n%s", output)
	}
	// The out-of-scope repo's worktree must still be where it was.
	if _, err := os.Stat(repoBWtPath); err != nil {
		t.Errorf("repo-b's worktree was moved despite being out of scope: %v", err)
	}
}

func TestRunWorktreeAlign_DotHonorsDryRun(t *testing.T) {
	repoA, _ := setupScopeFixture(t, "repo-a", "repo-b")
	chdirForTest(t, repoA)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	originalPath := cfg.Repositories[0].Worktrees[0].Path

	output := captureStdoutStr(func() {
		if err := runWorktreeAlign(scopeFor(t, "."), true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Would move") {
		t.Errorf("dry run should say 'Would move', got:\n%s", output)
	}
	if !strings.Contains(output, "Dry run") {
		t.Errorf("dry run should announce itself, got:\n%s", output)
	}
	if _, err := os.Stat(originalPath); err != nil {
		t.Errorf("dry run moved the worktree: %v", err)
	}
}

func TestWorktreeDotResolutionFailure(t *testing.T) {
	setupScopeFixture(t, "repo-a", "repo-b")
	chdirForTest(t, t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	_, err = worktreeScopeFor(cfg, ".")
	if !errors.Is(err, repocontext.ErrNotTracked) {
		t.Fatalf("got %v, want ErrNotTracked", err)
	}

	// Both subcommands must surface the same error for ".".
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"list", func() error { return worktreeListCmd.RunE(worktreeListCmd, []string{"."}) }},
		{"align", func() error { return worktreeAlignCmd.RunE(worktreeAlignCmd, []string{"."}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, repocontext.ErrNotTracked) {
				t.Errorf("%s . returned %v, want ErrNotTracked", tc.name, err)
			}
		})
	}
}

func TestWorktreeScopeFor_NamePattern(t *testing.T) {
	setupScopeFixture(t, "repo-a", "repo-b")

	scope := scopeFor(t, "repo-a")
	if scope.NamePattern != "repo-a" {
		t.Errorf("NamePattern = %q, want repo-a", scope.NamePattern)
	}
	if scope.RepoPath != "" {
		t.Errorf("RepoPath = %q, want empty for a name pattern", scope.RepoPath)
	}
}

func TestWorktreeScopeFor_Empty(t *testing.T) {
	setupScopeFixture(t, "repo-a", "repo-b")

	scope := scopeFor(t, "")
	if (scope != worktreeScope{}) {
		t.Errorf("empty argument produced %+v, want the zero scope", scope)
	}
}
