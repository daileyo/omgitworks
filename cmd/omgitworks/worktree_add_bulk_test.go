package main

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
)

// bulkAdd selects by tag and runs the bulk creation, returning captured stdout
// and the resulting error.
func bulkAdd(t *testing.T, tag, branch string) (string, error) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeAddTargets(cfg, "", tag)
	if err != nil {
		t.Fatalf("selection failed: %v", err)
	}

	var runErr error
	out := captureStdoutStr(func() {
		runErr = runWorktreeAddBulk(cfg, repos, branch)
	})
	return out, runErr
}

// threeBackendFixture tracks three backend repositories and one untagged.
func threeBackendFixture(t *testing.T) map[string]string {
	t.Helper()
	return setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-c", Tags: []string{"backend"}},
		taggedRepo{Name: "web-ui", Tags: []string{"frontend"}},
	)
}

// breakRepo removes a repository's directory so git operations against it fail.
func breakRepo(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("failed to break repo: %v", err)
	}
}

func TestWorktreeAddBulk_CreatesForEachTagged(t *testing.T) {
	threeBackendFixture(t)

	if _, err := bulkAdd(t, "backend", "feat-x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"svc-a", "svc-b", "svc-c"} {
		if _, err := os.Stat(projectsPath(t, name, "feat-x")); err != nil {
			t.Errorf("no worktree created for tagged repo %s: %v", name, err)
		}
	}
	if _, err := os.Stat(projectsPath(t, "web-ui", "feat-x")); err == nil {
		t.Error("a worktree was created for the untagged repo")
	}
}

func TestWorktreeAddBulk_SingleSave(t *testing.T) {
	threeBackendFixture(t)

	if _, err := bulkAdd(t, "backend", "feat-x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Reload from disk: every repository in the run must be recorded. A
	// per-repository save would leave only the last one intact.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	for _, repo := range cfg.Repositories {
		if repo.Name == "web-ui" {
			continue
		}
		found := false
		for _, wt := range repo.Worktrees {
			if wt.Branch == "feat-x" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s has no recorded worktree for feat-x; an earlier save was overwritten", repo.Name)
		}
	}
}

func TestWorktreeAddBulk_BranchWithSlash(t *testing.T) {
	threeBackendFixture(t)

	if _, err := bulkAdd(t, "backend", "hotfix/urgent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"svc-a", "svc-b", "svc-c"} {
		if _, err := os.Stat(projectsPath(t, name, "hotfix", "urgent")); err != nil {
			t.Errorf("nested worktree not created for %s: %v", name, err)
		}
	}
}

func TestWorktreeAddBulk_SkipsExisting(t *testing.T) {
	threeBackendFixture(t)

	// Give svc-b the branch already.
	if err := runWorktreeAdd("svc-b", "feat-x"); err != nil {
		t.Fatalf("failed to pre-create worktree: %v", err)
	}

	out, err := bulkAdd(t, "backend", "feat-x")
	if err != nil {
		t.Fatalf("a skip must not be a failure, got: %v", err)
	}

	if !strings.Contains(out, "Skipping [svc-b]") {
		t.Errorf("skip not announced during the run:\n%s", out)
	}
	if !strings.Contains(out, "skipped 1") {
		t.Errorf("summary does not report the skip:\n%s", out)
	}
	for _, name := range []string{"svc-a", "svc-c"} {
		if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); statErr != nil {
			t.Errorf("%s should still have been created: %v", name, statErr)
		}
	}
}

func TestWorktreeAddBulk_ContinuesPastFailure(t *testing.T) {
	paths := threeBackendFixture(t)
	breakRepo(t, paths["svc-b"])

	_, err := bulkAdd(t, "backend", "feat-x")
	if !errors.Is(err, errPartialFailure) {
		t.Fatalf("got %v, want errPartialFailure", err)
	}

	// svc-c comes after the failing svc-b and must still be created.
	if _, statErr := os.Stat(projectsPath(t, "svc-c", "feat-x")); statErr != nil {
		t.Errorf("the run aborted at the failure instead of continuing: %v", statErr)
	}
	if _, statErr := os.Stat(projectsPath(t, "svc-a", "feat-x")); statErr != nil {
		t.Errorf("svc-a should have been created: %v", statErr)
	}
}

func TestWorktreeAddBulk_SummaryCounts(t *testing.T) {
	paths := threeBackendFixture(t)

	// svc-a already has it (skip), svc-b is broken (failure), svc-c succeeds.
	if err := runWorktreeAdd("svc-a", "feat-x"); err != nil {
		t.Fatalf("failed to pre-create worktree: %v", err)
	}
	breakRepo(t, paths["svc-b"])

	out, err := bulkAdd(t, "backend", "feat-x")
	if !errors.Is(err, errPartialFailure) {
		t.Fatalf("got %v, want errPartialFailure", err)
	}

	for _, want := range []string{"Created 1 worktree", "skipped 1", "1 failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("summary missing %q:\n%s", want, out)
		}
	}
}

func TestWorktreeAddBulk_NamesFailedRepos(t *testing.T) {
	paths := threeBackendFixture(t)
	breakRepo(t, paths["svc-b"])

	out, _ := bulkAdd(t, "backend", "feat-x")

	if !strings.Contains(out, "svc-b") {
		t.Errorf("error block does not name the failed repo:\n%s", out)
	}
	if !strings.Contains(out, "failed to create worktree") {
		t.Errorf("error block does not give the underlying reason:\n%s", out)
	}
	if !strings.Contains(out, "1 error") {
		t.Errorf("error block has no heading:\n%s", out)
	}
}

func TestWorktreeAddBulk_ExitStatus(t *testing.T) {
	t.Run("all created returns nil", func(t *testing.T) {
		threeBackendFixture(t)
		if _, err := bulkAdd(t, "backend", "feat-x"); err != nil {
			t.Errorf("got %v, want nil", err)
		}
	})

	t.Run("creations plus skips returns nil", func(t *testing.T) {
		threeBackendFixture(t)
		if err := runWorktreeAdd("svc-a", "feat-x"); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		if _, err := bulkAdd(t, "backend", "feat-x"); err != nil {
			t.Errorf("got %v, want nil: skips are not failures", err)
		}
	})

	t.Run("any failure returns the sentinel", func(t *testing.T) {
		paths := threeBackendFixture(t)
		breakRepo(t, paths["svc-b"])
		_, err := bulkAdd(t, "backend", "feat-x")
		if !errors.Is(err, errPartialFailure) {
			t.Errorf("got %v, want errPartialFailure", err)
		}
	})
}

func TestWorktreeAddBulk_NoRollback(t *testing.T) {
	paths := threeBackendFixture(t)
	breakRepo(t, paths["svc-c"]) // the last repo fails

	if _, err := bulkAdd(t, "backend", "feat-x"); !errors.Is(err, errPartialFailure) {
		t.Fatalf("got %v, want errPartialFailure", err)
	}

	// Earlier successes are retained on disk and in the saved configuration.
	for _, name := range []string{"svc-a", "svc-b"} {
		if _, err := os.Stat(projectsPath(t, name, "feat-x")); err != nil {
			t.Errorf("%s was rolled back after a later failure: %v", name, err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	recorded := 0
	for _, repo := range cfg.Repositories {
		for _, wt := range repo.Worktrees {
			if wt.Branch == "feat-x" {
				recorded++
			}
		}
	}
	if recorded != 2 {
		t.Errorf("%d worktrees recorded, want 2 retained after the failure", recorded)
	}
}

func TestWorktreeAddBulk_SingleRepoKeepsExistingErrors(t *testing.T) {
	threeBackendFixture(t)

	// The single-repository path must keep returning the duplicate error rather
	// than reporting a skip, preserving the pre-existing contract.
	if err := runWorktreeAdd("svc-a", "feat-x"); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	err := runWorktreeAdd("svc-a", "feat-x")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("got %v, want the existing 'already exists' error", err)
	}
}
