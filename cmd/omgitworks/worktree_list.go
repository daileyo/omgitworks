package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
)

var worktreeListCmd = &cobra.Command{
	Use:   "list [repo|.]",
	Short: "List worktrees across tracked repositories",
	Long: `List all git worktrees across tracked repositories, optionally filtered to a
single repository.

Each entry shows the repository name, branch, path, and whether the worktree
is aligned (inside the projects root) or unaligned.

Passing "." lists only the repository the current directory belongs to. Omitting
the argument continues to list every tracked repository.

Examples:
  gws worktree list                 # List all worktrees
  gws worktree list my-repo         # List worktrees for my-repo only
  gws worktree list .               # List worktrees for the current repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoArg := ""
		if len(args) == 1 {
			repoArg = args[0]
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		scope, err := worktreeScopeFor(cfg, repoArg)
		if err != nil {
			return err
		}
		return runWorktreeList(scope, os.Stdout)
	},
}

func init() {
	worktreeCmd.AddCommand(worktreeListCmd)
}

// worktreeListEntry holds data for a single row in worktree list output.
type worktreeListEntry struct {
	Repo    string
	Branch  string
	Path    string
	Aligned bool
}

func runWorktreeList(scope worktreeScope, stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var entries []worktreeListEntry
	for _, repo := range cfg.Repositories {
		if !scope.matches(&repo) {
			continue
		}
		for _, wt := range repo.Worktrees {
			entries = append(entries, worktreeListEntry{
				Repo:    repo.Name,
				Branch:  wt.Branch,
				Path:    wt.Path,
				Aligned: wt.Aligned,
			})
		}
	}

	if len(entries) == 0 {
		if scope.Label != "" {
			fmt.Fprintf(stdout, "No worktrees found for '%s'\n", scope.Label)
		} else {
			fmt.Fprintln(stdout, "No worktrees found")
		}
		return nil
	}

	// Calculate column widths
	maxRepo := 4   // "REPO"
	maxBranch := 6 // "BRANCH"
	maxPath := 4   // "PATH"

	for _, e := range entries {
		if len(e.Repo) > maxRepo {
			maxRepo = len(e.Repo)
		}
		if len(e.Branch) > maxBranch {
			maxBranch = len(e.Branch)
		}
		if len(e.Path) > maxPath {
			maxPath = len(e.Path)
		}
	}

	// Print header
	fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n", maxRepo, "REPO", maxBranch, "BRANCH", maxPath, "PATH", "STATUS")
	fmt.Fprintf(stdout, "%s  %s  %s  %s\n",
		strings.Repeat("-", maxRepo),
		strings.Repeat("-", maxBranch),
		strings.Repeat("-", maxPath),
		strings.Repeat("-", 6))

	// Print rows
	for _, e := range entries {
		status := "aligned"
		if !e.Aligned {
			status = "(unaligned)"
		}
		branch := e.Branch
		if branch == "" {
			branch = "(detached)"
		}
		fmt.Fprintf(stdout, "%-*s  %-*s  %-*s  %s\n", maxRepo, e.Repo, maxBranch, branch, maxPath, e.Path, status)
	}

	return nil
}
