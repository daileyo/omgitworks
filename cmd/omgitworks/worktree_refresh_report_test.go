package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
)

// setStoredAligned overwrites one stored entry's Aligned value, standing in for
// configuration written by an older omgitworks whose projects root differed.
func setStoredAligned(t *testing.T, repoName, branch string, aligned bool) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	for i := range cfg.Repositories {
		if cfg.Repositories[i].Name != repoName {
			continue
		}
		for j := range cfg.Repositories[i].Worktrees {
			if cfg.Repositories[i].Worktrees[j].Branch == branch {
				cfg.Repositories[i].Worktrees[j].Aligned = aligned
				if err := config.Save(cfg); err != nil {
					t.Fatal(err)
				}
				return
			}
		}
	}
	t.Fatalf("fixture: no stored worktree %s/%s", repoName, branch)
}

// reportLines returns the output lines belonging to one repository's block.
func reportLines(out, repoName string) []string {
	var lines []string
	inBlock := false
	for _, line := range strings.Split(out, "\n") {
		switch {
		case line == "["+repoName+"]":
			inBlock = true
		case inBlock && strings.HasPrefix(line, "  "):
			lines = append(lines, strings.TrimSpace(line))
		default:
			inBlock = false
		}
	}
	return lines
}

func hasLine(lines []string, fields ...string) bool {
	return slices.ContainsFunc(lines, func(line string) bool {
		for _, f := range fields {
			if !strings.Contains(line, f) {
				return false
			}
		}
		return true
	})
}

func TestWorktreeRefreshReport_ListsAllThreeChangeKinds(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	addWorktreeTo(t, "svc-a", "feat-gone")
	addWorktreeTo(t, "svc-a", "feat-stale")
	addWorktreeTo(t, "svc-a", "feat-switch")

	gone := projectsPath(t, "svc-a", "feat-gone")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	ext := filepath.Join(t.TempDir(), "ext")
	gitWorktreeAdd(t, paths["svc-a"], "feat-ext", ext)
	setStoredAligned(t, "svc-a", "feat-stale", false)
	switched := projectsPath(t, "svc-a", "feat-switch")
	runGitIn(t, switched, "checkout", "-q", "-b", "feat-switched")

	out, err := refreshAll(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := reportLines(out, "svc-a")
	for _, want := range [][]string{
		{"added", "feat-ext", ext, "unaligned"},
		{"removed", "feat-gone", gone},
		{"realigned", "feat-stale", "now aligned"},
		{"branch", "feat-switched", switched, "was feat-switch"},
	} {
		if !hasLine(lines, want...) {
			t.Errorf("no report line containing %q in:\n%s", want, out)
		}
	}
	if len(lines) != 4 {
		t.Errorf("expected exactly 4 change lines, got %d:\n%s", len(lines), out)
	}
	if !strings.HasSuffix(out, "\nRefreshed 1 repository, 1 changed\n") {
		t.Errorf("unexpected summary:\n%q", out)
	}
}

func TestWorktreeRefreshReport_KeysOnPath(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"})
	outside := t.TempDir()
	first := filepath.Join(outside, "detached-1")
	second := filepath.Join(outside, "detached-2")
	runGitIn(t, paths["svc-a"], "worktree", "add", "--detach", first)
	runGitIn(t, paths["svc-a"], "worktree", "add", "--detach", second)

	out, err := refreshAll(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := reportLines(out, "svc-a")
	if !hasLine(lines, "added", "(detached)", first) || !hasLine(lines, "added", "(detached)", second) {
		t.Errorf("both detached worktrees should be reported as added:\n%s", out)
	}
	if n := len(storedWorktrees(t, "svc-a")); n != 2 {
		t.Errorf("expected 2 stored detached worktrees, got %d", n)
	}

	// Removing one must report exactly that one: keyed on branch, both entries
	// share "" and the removal would be missed or misattributed.
	runGitIn(t, paths["svc-a"], "worktree", "remove", first)
	out, err = refreshAll(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines = reportLines(out, "svc-a")
	if len(lines) != 1 || !hasLine(lines, "removed", "(detached)", first) {
		t.Errorf("expected one removal of %s, got:\n%s", first, out)
	}
}

func TestWorktreeRefreshReport_SilentWhenUnchanged(t *testing.T) {
	paths := setupTaggedFixture(t, taggedRepo{Name: "svc-a"}, taggedRepo{Name: "svc-b"})
	addWorktreeTo(t, "svc-b", "feat-accurate") // stored data already matches git
	gitWorktreeAdd(t, paths["svc-a"], "feat-ext", filepath.Join(t.TempDir(), "ext"))

	out, err := refreshAll(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "[svc-b]") || strings.Contains(out, "feat-accurate") {
		t.Errorf("unchanged repository should produce no lines:\n%s", out)
	}
	if !strings.Contains(out, "[svc-a]") {
		t.Errorf("fixture sanity: changed repository should be reported:\n%s", out)
	}

	// With nothing left to change, only the summary is printed.
	out, err = refreshAll(t)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "\nRefreshed 2 repositories, 0 changed\n"; out != want {
		t.Errorf("output = %q, want only the summary %q", out, want)
	}
}

func TestWorktreeRefreshReport_SummaryCounts(t *testing.T) {
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "changed-1"}, taggedRepo{Name: "unchanged"},
		taggedRepo{Name: "failed"}, taggedRepo{Name: "changed-2"})
	outside := t.TempDir()
	gitWorktreeAdd(t, paths["changed-1"], "ext-1", filepath.Join(outside, "1"))
	gitWorktreeAdd(t, paths["changed-2"], "ext-2", filepath.Join(outside, "2"))
	breakRepo(t, paths["failed"])

	out, err := refreshAll(t)

	if !errors.Is(err, errPartialFailure) {
		t.Fatalf("expected errPartialFailure, got %v", err)
	}
	if !strings.Contains(out, "\nRefreshed 3 repositories, 2 changed, 1 failed\n") {
		t.Errorf("unexpected summary:\n%s", out)
	}
	if strings.Contains(out, "[failed]") || strings.Contains(out, "[unchanged]") {
		t.Errorf("only changed repositories get a change block:\n%s", out)
	}
}

func TestDiffWorktrees(t *testing.T) {
	a := config.Worktree{Path: "/wt/a", Branch: "a", Aligned: true}
	b := config.Worktree{Path: "/wt/b", Branch: "b"}
	c := config.Worktree{Path: "/wt/c", Branch: "c"}

	t.Run("no change", func(t *testing.T) {
		if diffWorktrees([]config.Worktree{a, b}, []config.Worktree{a, b}).changed() {
			t.Error("identical lists must not report a change")
		}
	})

	t.Run("nil and empty are the same", func(t *testing.T) {
		if diffWorktrees(nil, []config.Worktree{}).changed() {
			t.Error("nil and empty must not report a change")
		}
	})

	t.Run("moved worktree is a removal plus an addition", func(t *testing.T) {
		moved := config.Worktree{Path: "/projects/a", Branch: "a", Aligned: true}
		got := diffWorktrees([]config.Worktree{a}, []config.Worktree{moved})
		if len(got.Removed) != 1 || got.Removed[0] != a || len(got.Added) != 1 || got.Added[0] != moved {
			t.Errorf("got %+v", got)
		}
		if len(got.Realigned)+len(got.Rebranched) != 0 {
			t.Errorf("a path change must not also count as realigned or rebranched: %+v", got)
		}
	})

	t.Run("realignment and branch change on one entry", func(t *testing.T) {
		after := config.Worktree{Path: b.Path, Branch: "b2", Aligned: true}
		got := diffWorktrees([]config.Worktree{b}, []config.Worktree{after})
		if len(got.Realigned) != 1 || len(got.Rebranched) != 1 || got.Rebranched[0].Previous != "b" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("order follows the input", func(t *testing.T) {
		got := diffWorktrees([]config.Worktree{c, a}, []config.Worktree{b})
		if len(got.Removed) != 2 || got.Removed[0] != c || got.Removed[1] != a {
			t.Errorf("removals out of order: %+v", got.Removed)
		}
	})
}
