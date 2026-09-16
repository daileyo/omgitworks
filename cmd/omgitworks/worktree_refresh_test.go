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
	"github.com/daileyo/omgitworks/internal/git"
)

// refreshAll runs a worktree refresh over every tracked repository, returning
// captured output.
func refreshAll(t *testing.T) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := runWorktreeRefreshCommand(nil, &buf)
	return buf.String(), err
}

// storedWorktrees reloads configuration from disk and returns the named
// repository's worktrees, so assertions see what was actually saved.
func storedWorktrees(t *testing.T, repoName string) []config.Worktree {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	for _, repo := range cfg.Repositories {
		if repo.Name == repoName {
			return repo.Worktrees
		}
	}
	t.Fatalf("repository %s not in configuration", repoName)
	return nil
}

func storedBranches(t *testing.T, repoName string) map[string]config.Worktree {
	t.Helper()
	byBranch := map[string]config.Worktree{}
	for _, wt := range storedWorktrees(t, repoName) {
		byBranch[wt.Branch] = wt
	}
	return byBranch
}

func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s\n%s", args, err, out)
	}
}

func TestWorktreeRefresh_DiscoversExternalAdd(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	ext := filepath.Join(t.TempDir(), "ext")
	gitWorktreeAdd(t, paths["svc-a"], "ext", ext)

	if len(storedWorktrees(t, "svc-a")) != 0 {
		t.Fatal("fixture: worktree should not be stored before refresh")
	}
	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wt, ok := storedBranches(t, "svc-a")["ext"]
	if !ok {
		t.Fatal("worktree created with plain git was not discovered")
	}
	if wt.Path != ext || wt.Aligned {
		t.Errorf("expected unaligned entry at %s, got %+v", ext, wt)
	}
}

func TestWorktreeRefresh_ClearsDeletedWorktree(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	addWorktreeTo(t, "svc-a", "feat-x")
	addWorktreeTo(t, "svc-a", "feat-y")
	gone := projectsPath(t, "svc-a", "feat-x")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := storedBranches(t, "svc-a")
	if _, ok := stored["feat-x"]; ok {
		t.Error("hand-deleted worktree is still stored")
	}
	if _, ok := stored["feat-y"]; !ok {
		t.Error("untouched worktree was dropped")
	}
	if containsPath(gitWorktreePaths(t, paths["svc-a"]), gone) {
		t.Error("prune should have removed git's record of the deleted worktree")
	}
}

func TestWorktreeRefresh_RepairBeforePrune(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	mislinked := filepath.Join(t.TempDir(), "mislinked")
	gitWorktreeAdd(t, paths["svc-a"], "mislinked", mislinked)
	breakWorktreeLink(t, mislinked)

	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := storedBranches(t, "svc-a")["mislinked"]; !ok {
		t.Error("recoverable worktree was discarded; prune must not run before repair")
	}
	if _, err := os.Stat(filepath.Join(mislinked, ".git")); err != nil {
		t.Errorf("repair should have restored the .git file: %v", err)
	}
}

func TestWorktreeRefresh_RecomputesAligned(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	loose := filepath.Join(t.TempDir(), "loose")
	gitWorktreeAdd(t, paths["svc-a"], "loose", loose)
	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if storedBranches(t, "svc-a")["loose"].Aligned {
		t.Fatal("fixture: worktree outside the projects root should start unaligned")
	}

	// Move it into the projects root with plain git, outside omgitworks.
	target := projectsPath(t, "svc-a", "loose")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	runGitIn(t, paths["svc-a"], "worktree", "move", loose, target)

	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wt := storedBranches(t, "svc-a")["loose"]
	if !wt.Aligned {
		t.Errorf("worktree moved into the projects root should be aligned, got %+v", wt)
	}
	if git.ResolvePath(wt.Path) != git.ResolvePath(target) {
		t.Errorf("stored path %s should follow the move to %s", wt.Path, target)
	}
}

func TestWorktreeRefresh_ClearsWhenEmpty(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	addWorktreeTo(t, "svc-a", "feat-x")
	runGitIn(t, paths["svc-a"], "worktree", "remove", projectsPath(t, "svc-a", "feat-x"))

	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if wts := storedWorktrees(t, "svc-a"); len(wts) != 0 {
		t.Errorf("expected no stored worktrees, got %+v", wts)
	}
}

func TestWorktreeRefresh_LeavesOtherDataUntouched(t *testing.T) {
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-b", Tags: []string{"frontend"}},
	)
	gitWorktreeAdd(t, paths["svc-a"], "ext", filepath.Join(t.TempDir(), "ext"))

	// Stored user data refresh must not re-detect.
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	for i := range cfg.Repositories {
		cfg.Repositories[i].User = "Stored Name"
		cfg.Repositories[i].Email = "stored@example.com"
	}
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	// An untracked repository beside the tracked ones must not be discovered.
	untracked := filepath.Join(filepath.Dir(paths["svc-a"]), "untracked")
	if err := os.MkdirAll(untracked, 0755); err != nil {
		t.Fatal(err)
	}
	runGitIn(t, untracked, "init")

	// A status cache that must not be cleared.
	cachePath, err := git.GetCachePath()
	if err != nil {
		t.Fatal(err)
	}
	statusCache := git.NewCache(git.DefaultTTL)
	statusCache.Set(paths["svc-a"], &git.Status{Branch: "main"})
	if err := statusCache.Save(cachePath); err != nil {
		t.Fatal(err)
	}
	cacheBefore, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Repositories) != 2 {
		t.Fatalf("repository list changed: %d repositories", len(after.Repositories))
	}
	for i, repo := range after.Repositories {
		want := cfg.Repositories[i]
		if repo.Name != want.Name || repo.Path != want.Path || repo.User != want.User ||
			repo.Email != want.Email || strings.Join(repo.Tags, ",") != strings.Join(want.Tags, ",") {
			t.Errorf("non-worktree data changed for %s: got %+v", want.Name, repo)
		}
	}
	if _, ok := storedBranches(t, "svc-a")["ext"]; !ok {
		t.Error("fixture sanity: worktree data itself should still have been refreshed")
	}
	cacheAfter, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("status cache was removed: %v", err)
	}
	if !bytes.Equal(cacheBefore, cacheAfter) {
		t.Error("status cache contents changed")
	}
}

func TestWorktreeRefresh_PartialFailure(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"}, taggedRepo{Name: "svc-b"})
	addWorktreeTo(t, "svc-a", "feat-x")
	ext := filepath.Join(t.TempDir(), "ext")
	gitWorktreeAdd(t, paths["svc-b"], "ext", ext)
	breakRepo(t, paths["svc-a"]) // first in the run

	out, err := refreshAll(t)

	if !errors.Is(err, errPartialFailure) {
		t.Fatalf("expected errPartialFailure so the command exits non-zero, got %v", err)
	}
	if !strings.Contains(out, "1 error") || !strings.Contains(out, "svc-a:") {
		t.Errorf("failure should be reported by repository name, got:\n%s", out)
	}
	if _, ok := storedBranches(t, "svc-b")["ext"]; !ok {
		t.Error("the repository after the failure was not refreshed")
	}
	if _, ok := storedBranches(t, "svc-a")["feat-x"]; !ok {
		t.Error("the failed repository's stored data should be left untouched")
	}
}

func TestWorktreeRefresh_SucceedsWithZeroExit(t *testing.T) {
	setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	if _, err := refreshAll(t); err != nil {
		t.Errorf("a run with no failures must return nil, got %v", err)
	}
}

// TestWorktreeRefresh_SingleSave mirrors TestWorktreeAddBulk_SingleSave. It
// catches the harmful form of a per-repository save — each repository loading
// and saving its own copy of the configuration, so a later save overwrites an
// earlier repository's result. It cannot count writes; that the save happens
// once is confirmed by inspection of runWorktreeRefresh.
func TestWorktreeRefresh_SingleSave(t *testing.T) {
	names := []string{"svc-a", "svc-b", "svc-c"}
	paths := setupTaggedFixture(t,
		taggedRepo{Name: names[0]}, taggedRepo{Name: names[1]}, taggedRepo{Name: names[2]})
	outside := t.TempDir()
	for _, name := range names {
		gitWorktreeAdd(t, paths[name], "ext-"+name, filepath.Join(outside, name))
	}

	if _, err := refreshAll(t); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range names {
		if _, ok := storedBranches(t, name)["ext-"+name]; !ok {
			t.Errorf("%s has no stored worktree; an earlier result was overwritten", name)
		}
	}
}

func TestWorktreeRefreshCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range worktreeCmd.Commands() {
		if sub == worktreeRefreshCmd {
			found = true
		}
	}
	if !found {
		t.Error("refresh is not registered under worktree")
	}
}
