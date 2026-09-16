package main

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
)

// removeBulk selects by tag and runs a bulk removal with confirmation skipped.
func removeBulk(t *testing.T, tag, branch string, opts removeOptions) (string, error) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeTargets(cfg, "", tag)
	if err != nil {
		t.Fatalf("selection failed: %v", err)
	}
	opts.Yes = true
	var buf bytes.Buffer
	runErr := runWorktreeRemove(cfg, repos, branch, opts, false, &buf, strings.NewReader(""))
	return buf.String(), runErr
}

// bulkRemoveFixture: three backend repos plus one untagged, all holding branch.
func bulkRemoveFixture(t *testing.T, branch string) map[string]string {
	t.Helper()
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-c", Tags: []string{"backend"}},
		taggedRepo{Name: "web-ui", Tags: []string{"frontend"}},
	)
	for _, name := range []string{"svc-a", "svc-b", "svc-c", "web-ui"} {
		addWorktreeTo(t, name, branch)
	}
	return paths
}

func TestWorktreeRemoveBulk_RemovesAcrossTag(t *testing.T) {
	resetRemoveFlags(t)
	bulkRemoveFixture(t, "feat-x")

	if _, err := removeBulk(t, "backend", "feat-x", removeOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, name := range []string{"svc-a", "svc-b", "svc-c"} {
		if _, err := os.Stat(projectsPath(t, name, "feat-x")); !os.IsNotExist(err) {
			t.Errorf("%s still has the worktree", name)
		}
	}
	// The untagged repository must be untouched.
	if _, err := os.Stat(projectsPath(t, "web-ui", "feat-x")); err != nil {
		t.Errorf("untagged repo's worktree was removed: %v", err)
	}
}

func TestWorktreeRemoveTargets_PrecedenceTable(t *testing.T) {
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "api-core", Tags: []string{"backend"}},
		taggedRepo{Name: "api-edge", Tags: []string{"backend"}},
		taggedRepo{Name: "web-ui", Tags: []string{"frontend"}},
		taggedRepo{Name: "api-legacy", Tags: nil},
	)

	tests := []struct {
		name    string
		pattern string
		tag     string
		cwd     string
		want    []string
	}{
		{"1 positional, no tag -> current repo", "", "", paths["web-ui"], []string{"web-ui"}},
		{"1 positional with tag -> all tagged", "", "backend", paths["web-ui"], []string{"api-core", "api-edge"}},
		{"2 positionals, no tag -> name pattern", "web", "", paths["api-core"], []string{"web-ui"}},
		{"2 positionals with tag -> name AND tag", "api", "backend", paths["web-ui"], []string{"api-core", "api-edge"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chdirForTest(t, tt.cwd)
			got, err := selectNames(t, tt.pattern, tt.tag)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("selected %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorktreeRemoveBulk_SkipsMissingBranch(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-c", Tags: []string{"backend"}},
	)
	// svc-b never had this branch.
	addWorktreeTo(t, "svc-a", "feat-x")
	addWorktreeTo(t, "svc-c", "feat-x")

	out, err := removeBulk(t, "backend", "feat-x", removeOptions{})
	if err != nil {
		t.Fatalf("a missing branch is a skip, not a failure: %v", err)
	}
	if !strings.Contains(out, "svc-b") || !strings.Contains(out, "Skipping") {
		t.Errorf("missing branch not reported as a skip:\n%s", out)
	}
	if !strings.Contains(out, "skipped 1") {
		t.Errorf("summary does not count the skip:\n%s", out)
	}
}

// Regression guard carried over from spec 26: a tag matching exactly one
// repository is still a bulk run, so a missing branch is a skip, not an error.
func TestWorktreeRemoveBulk_SingleMatchTagStillBulk(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t,
		taggedRepo{Name: "only-one", Tags: []string{"solo"}},
		taggedRepo{Name: "other", Tags: []string{"another"}},
	)

	out, err := removeBulk(t, "solo", "feat-missing", removeOptions{})
	if err != nil {
		t.Errorf("got %v; a single-match tag run must skip, not error", err)
	}
	if !strings.Contains(out, "skipped 1") {
		t.Errorf("summary not printed for a single-match tag run:\n%s", out)
	}
}

func TestWorktreeRemoveBulk_SummaryAndExit(t *testing.T) {
	t.Run("mixed run reports counts and fails", func(t *testing.T) {
		resetRemoveFlags(t)
		paths := setupTaggedFixture(t,
			taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
			taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
			taggedRepo{Name: "svc-c", Tags: []string{"backend"}},
		)
		addWorktreeTo(t, "svc-a", "feat-x")
		addWorktreeTo(t, "svc-b", "feat-x")
		// svc-c has no worktree for the branch -> skip.
		// Break svc-b's repository -> failure.
		if err := os.RemoveAll(paths["svc-b"]); err != nil {
			t.Fatalf("failed to break repo: %v", err)
		}

		out, err := removeBulk(t, "backend", "feat-x", removeOptions{})
		if !errors.Is(err, errPartialFailure) {
			t.Fatalf("got %v, want errPartialFailure", err)
		}
		for _, want := range []string{"Removed 1 worktree", "skipped 1", "1 failed", "svc-b"} {
			if !strings.Contains(out, want) {
				t.Errorf("summary missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("skips only returns nil", func(t *testing.T) {
		resetRemoveFlags(t)
		setupTaggedFixture(t,
			taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
			taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
		)
		if _, err := removeBulk(t, "backend", "never-existed", removeOptions{}); err != nil {
			t.Errorf("got %v, want nil when only skips occurred", err)
		}
	})
}

func TestWorktreeRemoveBulk_NoRestore(t *testing.T) {
	resetRemoveFlags(t)
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-c", Tags: []string{"backend"}},
	)
	for _, name := range []string{"svc-a", "svc-b", "svc-c"} {
		addWorktreeTo(t, name, "feat-x")
	}
	// Break the last repository so earlier removals precede a failure.
	if err := os.RemoveAll(paths["svc-c"]); err != nil {
		t.Fatalf("failed to break repo: %v", err)
	}

	if _, err := removeBulk(t, "backend", "feat-x", removeOptions{}); !errors.Is(err, errPartialFailure) {
		t.Fatalf("got %v, want errPartialFailure", err)
	}

	for _, name := range []string{"svc-a", "svc-b"} {
		if _, err := os.Stat(projectsPath(t, name, "feat-x")); !os.IsNotExist(err) {
			t.Errorf("%s was restored after a later failure; removals are not rolled back", name)
		}
	}
}

func TestWorktreeRemoveBulk_SingleSave(t *testing.T) {
	resetRemoveFlags(t)
	bulkRemoveFixture(t, "feat-x")

	if _, err := removeBulk(t, "backend", "feat-x", removeOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	for _, repo := range cfg.Repositories {
		if repo.Name == "web-ui" {
			continue
		}
		for _, wt := range repo.Worktrees {
			if wt.Branch == "feat-x" {
				t.Errorf("%s still records feat-x; a save was overwritten", repo.Name)
			}
		}
	}
}
