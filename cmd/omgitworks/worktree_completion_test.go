package main

import (
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

// addBranch creates a branch in the repository at repoPath.
func addBranch(t *testing.T, repoPath, branch string) {
	t.Helper()
	cmd := exec.Command("git", "branch", branch)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch %s failed: %s\n%s", branch, err, out)
	}
}

func TestCompleteWorktreeAdd_BranchesWhenResolved(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")
	addBranch(t, repoDir, "feature-auth")
	chdirForTest(t, repoDir)

	got, directive := completeWorktreeAddArgs(nil, nil, "")

	if !slices.Contains(got, "feature-auth") {
		t.Errorf("completions %v, want them to contain feature-auth", got)
	}
	if slices.Contains(got, "my-repo") {
		t.Errorf("completions %v contain the repo name; branches were expected", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteWorktreeAdd_ReposWhenUnresolved(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")
	chdirForTest(t, t.TempDir())

	got, directive := completeWorktreeAddArgs(nil, nil, "")

	if !slices.Contains(got, "my-repo") {
		t.Errorf("completions %v, want the repo name as a fallback", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteWorktreeAdd_PrefixFilter(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")
	addBranch(t, repoDir, "feature-auth")
	addBranch(t, repoDir, "release-1")
	chdirForTest(t, repoDir)

	got, _ := completeWorktreeAddArgs(nil, nil, "feat")

	if !slices.Contains(got, "feature-auth") {
		t.Errorf("completions %v, want feature-auth for prefix 'feat'", got)
	}
	if slices.Contains(got, "release-1") {
		t.Errorf("completions %v include release-1, which does not match 'feat'", got)
	}
}

func TestCompleteWorktreeAdd_SecondArgument(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")
	addBranch(t, repoDir, "feature-auth")

	// Stand outside any tracked repository: the named repo must still drive
	// the second argument's completions.
	chdirForTest(t, t.TempDir())

	got, directive := completeWorktreeAddArgs(nil, []string{"my-repo"}, "")

	if !slices.Contains(got, "feature-auth") {
		t.Errorf("completions %v, want branches of the named repo", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteWorktreeAdd_BeyondTwoArgs(t *testing.T) {
	setupWorktreeTestRepo(t, "my-repo")

	got, _ := completeWorktreeAddArgs(nil, []string{"my-repo", "feat-x"}, "")
	if len(got) != 0 {
		t.Errorf("completions %v, want none past two arguments", got)
	}
}

func TestCompleteWorktreeRepoOrDot(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")

	t.Run("dot offered when resolved", func(t *testing.T) {
		chdirForTest(t, repoDir)

		got, directive := completeWorktreeRepoOrDot(nil, nil, "")

		if !slices.Contains(got, ".") {
			t.Errorf("completions %v, want them to contain '.'", got)
		}
		if !slices.Contains(got, "my-repo") {
			t.Errorf("completions %v, want repo names alongside '.'", got)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", directive)
		}
	})

	t.Run("dot withheld when unresolved", func(t *testing.T) {
		chdirForTest(t, t.TempDir())

		got, _ := completeWorktreeRepoOrDot(nil, nil, "")

		if slices.Contains(got, ".") {
			t.Errorf("completions %v offer '.' where it would fail", got)
		}
		if !slices.Contains(got, "my-repo") {
			t.Errorf("completions %v, want repo names to remain", got)
		}
	})

	t.Run("no completions after the first argument", func(t *testing.T) {
		chdirForTest(t, repoDir)

		got, _ := completeWorktreeRepoOrDot(nil, []string{"my-repo"}, "")
		if len(got) != 0 {
			t.Errorf("completions %v, want none once an argument is present", got)
		}
	})
}

func TestCompleteWorktreeRepoOrDot_PrefixFiltersDot(t *testing.T) {
	_, repoDir := setupWorktreeTestRepo(t, "my-repo")
	chdirForTest(t, repoDir)

	// "." does not start with "my", so it must be filtered out like any other
	// suggestion.
	got, _ := completeWorktreeRepoOrDot(nil, nil, "my")

	if slices.Contains(got, ".") {
		t.Errorf("completions %v include '.' for prefix 'my'", got)
	}
	if !slices.Contains(got, "my-repo") {
		t.Errorf("completions %v, want my-repo for prefix 'my'", got)
	}
}

func TestWorktreeCompletionsRegistered(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"add", worktreeAddCmd},
		{"list", worktreeListCmd},
		{"align", worktreeAlignCmd},
		{"refresh", worktreeRefreshCmd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.ValidArgsFunction == nil {
				t.Errorf("%s has no ValidArgsFunction registered", tt.name)
			}
		})
	}
}

func TestCompleteBranchNames_NotARepo(t *testing.T) {
	got, directive := completeBranchNames(filepath.Join(t.TempDir(), "nope"), "")
	if len(got) != 0 {
		t.Errorf("completions %v, want none for a non-repository", got)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteWorktreeAddTag(t *testing.T) {
	setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend", "shared"}},
		taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
		taggedRepo{Name: "web", Tags: []string{"frontend"}},
	)

	t.Run("suggests every tag once", func(t *testing.T) {
		got, directive := completeAllTags("")

		slices.Sort(got)
		want := []string{"backend", "frontend", "shared"}
		if !slices.Equal(got, want) {
			t.Errorf("completions %v, want %v deduplicated", got, want)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %v, want NoFileComp", directive)
		}
	})

	t.Run("filters on the typed prefix", func(t *testing.T) {
		got, _ := completeAllTags("back")
		if !slices.Equal(got, []string{"backend"}) {
			t.Errorf("completions %v, want [backend] for prefix 'back'", got)
		}
	})

	t.Run("registered on the tag flag", func(t *testing.T) {
		if worktreeAddCmd.Flag("tag") == nil {
			t.Fatal("worktree add has no --tag flag")
		}
	})
}
