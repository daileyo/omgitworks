package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
)

// resetRemoveFlags clears command state between cases.
func resetRemoveFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		flagWorktreeRemoveTags = nil
		flagWorktreeRemoveForce = false
		flagWorktreeRemoveDryRun = false
		flagWorktreeRemoveYes = false
	})
}

// addWorktreeTo creates a worktree for branch in the named repository via the
// real command path, so fixtures match production state.
func addWorktreeTo(t *testing.T, repoName, branch string) {
	t.Helper()
	if err := runWorktreeAdd(repoName, branch); err != nil {
		t.Fatalf("fixture: failed to add worktree %s/%s: %v", repoName, branch, err)
	}
}

// removeOne runs a single-repository removal, returning captured output.
func removeOne(t *testing.T, repoName, branch string, opts removeOptions) (string, error) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeTargets(cfg, repoName, "")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	runErr := runWorktreeRemove(cfg, repos, branch, opts, true, &buf, strings.NewReader(""))
	return buf.String(), runErr
}

// lockWorktree writes git's lock marker for a worktree.
func lockWorktree(t *testing.T, repoPath, worktreePath, reason string) {
	t.Helper()
	base := filepath.Base(worktreePath)
	dir := filepath.Join(repoPath, ".git", "worktrees", base)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("no git worktree metadata at %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "locked"), []byte(reason), 0644); err != nil {
		t.Fatalf("failed to lock worktree: %v", err)
	}
}

func TestRunWorktreeRemove_Individual(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-x")
	wtPath := projectsPath(t, "svc-a", "feat-x")

	if _, err := removeOne(t, "svc-a", "feat-x", removeOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory still present at %s", wtPath)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	for _, wt := range cfg.Repositories[0].Worktrees {
		if wt.Branch == "feat-x" {
			t.Error("configuration still records the removed worktree")
		}
	}
}

func TestRunWorktreeRemove_CurrentRepo(t *testing.T) {
	resetRemoveFlags(t)
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-x")
	chdirForTest(t, paths["svc-a"])

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeTargets(cfg, "", "")
	if err != nil {
		t.Fatalf("resolver selection failed: %v", err)
	}
	var buf bytes.Buffer
	if err := runWorktreeRemove(cfg, repos, "feat-x", removeOptions{}, true, &buf, strings.NewReader("")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(projectsPath(t, "svc-a", "feat-x")); !os.IsNotExist(err) {
		t.Error("worktree was not removed via the current-repository form")
	}
}

func TestRunWorktreeRemove_ExactBranchMatch(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-auth")

	// "feat" must not match a worktree on "feat-auth".
	_, err := removeOne(t, "svc-a", "feat", removeOptions{})
	var notFound *worktreeNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("got %v, want worktreeNotFoundError for a partial branch name", err)
	}
	if _, statErr := os.Stat(projectsPath(t, "svc-a", "feat-auth")); statErr != nil {
		t.Errorf("the worktree was removed by a partial match: %v", statErr)
	}
}

func TestRunWorktreeRemove_NoSuchBranch(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})

	_, err := removeOne(t, "svc-a", "nope", removeOptions{})
	if err == nil {
		t.Fatal("expected an error for a branch with no worktree")
	}
	for _, want := range []string{"svc-a", "nope"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}

func TestRunWorktreeRemove_LockedSkipped(t *testing.T) {
	for _, force := range []bool{false, true} {
		name := "without force"
		if force {
			name = "with force"
		}
		t.Run(name, func(t *testing.T) {
			resetRemoveFlags(t)
			paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
			addWorktreeTo(t, "svc-a", "feat-x")
			wtPath := projectsPath(t, "svc-a", "feat-x")
			lockWorktree(t, paths["svc-a"], wtPath, "in review")

			out, err := removeOne(t, "svc-a", "feat-x", removeOptions{Force: force})
			if err != nil {
				t.Fatalf("a locked worktree should be skipped, not fail: %v", err)
			}
			// --force covers dirty worktrees only; a lock is absolute.
			if _, statErr := os.Stat(wtPath); statErr != nil {
				t.Errorf("locked worktree was removed (force=%v): %v", force, statErr)
			}
			if !strings.Contains(out, "locked") || !strings.Contains(out, "in review") {
				t.Errorf("lock and reason not reported:\n%s", out)
			}
		})
	}
}

func TestRunWorktreeRemove_BranchSurvives(t *testing.T) {
	resetRemoveFlags(t)
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-keep")

	if _, err := removeOne(t, "svc-a", "feat-keep", removeOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := exec.Command("git", "branch", "--list", "feat-keep")
	cmd.Dir = paths["svc-a"]
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --list failed: %v", err)
	}
	if !strings.Contains(string(out), "feat-keep") {
		t.Error("the branch was deleted along with its worktree")
	}
}

func TestRunWorktreeRemove_DirtyNeedsForce(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-dirty")
	wtPath := projectsPath(t, "svc-a", "feat-dirty")

	if err := os.WriteFile(filepath.Join(wtPath, "scratch.txt"), []byte("wip\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if _, err := removeOne(t, "svc-a", "feat-dirty", removeOptions{}); err == nil {
		t.Fatal("removal succeeded despite untracked changes and no --force")
	}
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("worktree should survive a refused removal: %v", err)
	}

	if _, err := removeOne(t, "svc-a", "feat-dirty", removeOptions{Force: true}); err != nil {
		t.Fatalf("forced removal failed: %v", err)
	}
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree still present after a forced removal")
	}
}

func TestRunWorktreeRemove_CleansEmptyDirs(t *testing.T) {
	t.Run("nested branch leaves no empty parent", func(t *testing.T) {
		resetRemoveFlags(t)
		setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
		addWorktreeTo(t, "svc-a", "hotfix/urgent")

		nestedParent := projectsPath(t, "svc-a", "hotfix")
		if _, err := os.Stat(nestedParent); err != nil {
			t.Fatalf("fixture: expected %s to exist: %v", nestedParent, err)
		}

		if _, err := removeOne(t, "svc-a", "hotfix/urgent", removeOptions{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(nestedParent); !os.IsNotExist(err) {
			t.Errorf("empty %s was left behind", nestedParent)
		}
	})

	t.Run("directory holding another worktree is kept", func(t *testing.T) {
		resetRemoveFlags(t)
		setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
		addWorktreeTo(t, "svc-a", "feat-x")
		addWorktreeTo(t, "svc-a", "feat-y")

		if _, err := removeOne(t, "svc-a", "feat-x", removeOptions{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		repoDir := projectsPath(t, "svc-a")
		if _, err := os.Stat(repoDir); err != nil {
			t.Errorf("repo projects dir removed while it still holds feat-y: %v", err)
		}
		if _, err := os.Stat(projectsPath(t, "svc-a", "feat-y")); err != nil {
			t.Errorf("the surviving worktree was disturbed: %v", err)
		}
	})
}

func TestWorktreeRemoveCmd_Alias(t *testing.T) {
	// Resolve through cobra's lookup rather than reading the Aliases field, so
	// the test proves behavior rather than restating configuration.
	cmd, _, err := worktreeCmd.Find([]string{"rm"})
	if err != nil {
		t.Fatalf("cobra could not resolve 'rm': %v", err)
	}
	if cmd != worktreeRemoveCmd {
		t.Errorf("'rm' resolved to %q, want the remove command", cmd.Name())
	}
}

func TestCompleteWorktreeRemove(t *testing.T) {
	resetRemoveFlags(t)
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-auth")
	addWorktreeTo(t, "svc-a", "feat-login")

	t.Run("suggests only branches that have worktrees", func(t *testing.T) {
		chdirForTest(t, paths["svc-a"])
		got, directive := completeRemovableBranches(nil, nil, "")

		for _, want := range []string{"feat-auth", "feat-login"} {
			if !slices.Contains(got, want) {
				t.Errorf("completions %v, want them to contain %s", got, want)
			}
		}
		// The repository has other branches (main) with no worktree.
		if slices.Contains(got, "main") {
			t.Errorf("completions %v include a branch with no worktree", got)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", directive)
		}
	})

	t.Run("filters on the typed prefix", func(t *testing.T) {
		chdirForTest(t, paths["svc-a"])
		got, _ := completeRemovableBranches(nil, nil, "feat-log")
		if !slices.Equal(got, []string{"feat-login"}) {
			t.Errorf("completions %v, want [feat-login]", got)
		}
	})

	t.Run("falls back to repo names outside a tracked repo", func(t *testing.T) {
		chdirForTest(t, t.TempDir())
		got, _ := completeRemovableBranches(nil, nil, "")
		if !slices.Contains(got, "svc-a") {
			t.Errorf("completions %v, want repo names as a fallback", got)
		}
	})
}

// Regression guard: cleanup must stay inside the repository's own projects
// directory. A string prefix is not containment — projects/svc-a-old has the
// prefix projects/svc-a but belongs to another repository, and was once
// deleted when a stray svc-a worktree inside it was removed.
func TestRunWorktreeRemove_CleanupRespectsSiblingRepo(t *testing.T) {
	resetRemoveFlags(t)
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: nil},
		taggedRepo{Name: "svc-a-old", Tags: nil},
	)

	siblingDir := projectsPath(t, "svc-a-old")
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatalf("failed to create sibling projects dir: %v", err)
	}

	// A worktree of svc-a placed inside the sibling repository's directory.
	stray := filepath.Join(siblingDir, "feat-stray")
	cmd := exec.Command("git", "worktree", "add", "-b", "feat-stray", stray)
	cmd.Dir = paths["svc-a"]
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %s\n%s", err, out)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	for i := range cfg.Repositories {
		if cfg.Repositories[i].Name == "svc-a" {
			cfg.Repositories[i].Worktrees = []config.Worktree{{Path: stray, Branch: "feat-stray"}}
		}
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Target svc-a unambiguously through the resolver: "svc-a" as a name
	// pattern would also match svc-a-old.
	chdirForTest(t, paths["svc-a"])
	cfg, _ = config.Load()
	repos, err := selectWorktreeTargets(cfg, "", "")
	if err != nil {
		t.Fatalf("selection failed: %v", err)
	}
	var buf bytes.Buffer
	if err := runWorktreeRemove(cfg, repos, "feat-stray", removeOptions{}, true, &buf, strings.NewReader("")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Fatal("fixture: the stray worktree was not removed")
	}
	if _, err := os.Stat(siblingDir); err != nil {
		t.Errorf("cleanup deleted another repository's projects directory %s", siblingDir)
	}
}

// TestRunWorktreeRemove_RefreshUsesSharedRules pins the behavior remove picked
// up when its re-discovery moved onto the shared helper. It used to list and
// store worktrees without repairing or pruning, so a sibling worktree whose
// directory had been deleted stayed recorded after an unrelated removal.
//
// As in the add-path test, surviving entries alone cannot tell the old and new
// behavior apart, so the test asserts what only repair and prune produce.
func TestRunWorktreeRemove_RefreshUsesSharedRules(t *testing.T) {
	resetRemoveFlags(t)
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	repoDir := paths["svc-a"]
	addWorktreeTo(t, "svc-a", "feat-x")

	outside := t.TempDir()
	mislinked := filepath.Join(outside, "mislinked")
	gone := filepath.Join(outside, "gone")
	gitWorktreeAdd(t, repoDir, "mislinked", mislinked)
	gitWorktreeAdd(t, repoDir, "gone", gone)
	breakWorktreeLink(t, mislinked)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	if _, err := removeOne(t, "svc-a", "feat-x", removeOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(mislinked, ".git")); err != nil {
		t.Errorf("repair should have restored the mislinked worktree's .git file: %v", err)
	}
	if containsPath(gitWorktreePaths(t, repoDir), gone) {
		t.Error("prune should have removed git's record of the deleted worktree")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	stored := map[string]bool{}
	for _, wt := range cfg.Repositories[0].Worktrees {
		stored[wt.Branch] = true
	}
	if !stored["mislinked"] || stored["gone"] || stored["feat-x"] {
		t.Errorf("expected only mislinked stored, got %v", stored)
	}
}
