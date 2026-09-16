package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
	"github.com/daileyo/omgitworks/internal/xdg"
)

var flagDryRun bool

var worktreeAlignCmd = &cobra.Command{
	Use:   "align [repo|.]",
	Short: "Move unaligned worktrees into the projects root",
	Long: `Move all unaligned worktrees into the XDG projects root
using git worktree move (requires Git 2.17+).

When a repo name argument is provided, only that repo's worktrees are aligned.
Without an argument, all repos are processed.

If two worktrees would produce the same directory name, a -dup-NN suffix is
appended (where NN is 00-99).

Passing "." aligns only the repository the current directory belongs to.
Omitting the argument continues to process every tracked repository.

Examples:
  gws worktree align                # Align all repos
  gws worktree align my-repo        # Align only my-repo
  gws worktree align .              # Align only the current repo
  gws worktree align --dry-run      # Preview moves without executing`,
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
		return runWorktreeAlign(scope, flagDryRun)
	},
}

func init() {
	worktreeAlignCmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Preview moves without executing them")
	worktreeCmd.AddCommand(worktreeAlignCmd)
}

// alignPlan describes a single worktree move operation.
type alignPlan struct {
	RepoName string
	RepoPath string
	Branch   string
	From     string
	To       string
	Renamed  bool // true if a -dup-NN suffix was applied
}

func runWorktreeAlign(scope worktreeScope, dryRun bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var plans []alignPlan

	for i := range cfg.Repositories {
		repo := &cfg.Repositories[i]
		if !scope.matches(repo) {
			continue
		}

		// Repair broken links first, then prune truly dead entries
		_ = git.RepairWorktrees(repo.Path)
		_ = git.PruneWorktrees(repo.Path)

		wtDir, err := xdg.RepoProjectsDir(repo.Name)
		if err != nil {
			return err
		}
		// Track names used in this repo's projects dir to detect conflicts
		usedNames := make(map[string]bool)

		// Pre-populate with existing aligned worktrees
		for _, wt := range repo.Worktrees {
			if wt.Aligned {
				relPath, err := filepath.Rel(wtDir, wt.Path)
				if err == nil {
					usedNames[relPath] = true
				}
			}
		}

		for _, wt := range repo.Worktrees {
			if wt.Aligned {
				continue
			}

			// Skip locked worktrees — they can't be moved
			if locked, reason := git.IsWorktreeLocked(repo.Path, wt.Path); locked {
				msg := "locked"
				if reason != "" {
					msg = fmt.Sprintf("locked: %s", reason)
				}
				fmt.Printf("Skipping [%s] %s — %s\n", repo.Name, wt.Branch, msg)
				continue
			}

			targetName := wt.Branch
			targetPath := filepath.Join(wtDir, targetName)
			renamed := false

			// Handle naming conflicts
			if usedNames[targetName] {
				renamed = true
				for dup := 0; dup < 100; dup++ {
					candidate := fmt.Sprintf("%s-dup-%02d", targetName, dup)
					if !usedNames[candidate] {
						targetName = candidate
						targetPath = filepath.Join(wtDir, targetName)
						break
					}
				}
			}
			usedNames[targetName] = true

			plans = append(plans, alignPlan{
				RepoName: repo.Name,
				RepoPath: repo.Path,
				Branch:   wt.Branch,
				From:     wt.Path,
				To:       targetPath,
				Renamed:  renamed,
			})
		}
	}

	if len(plans) == 0 {
		fmt.Println("All worktrees are already aligned")
		return nil
	}

	// Display plan
	verb := "Moving"
	if dryRun {
		verb = "Would move"
		fmt.Println("Dry run — no changes will be made:")
		fmt.Println()
	}

	// The destination changed in this version, so say where things are going
	// before listing the moves rather than after.
	if hasLegacyPlan(plans) {
		if root, err := xdg.ProjectsDir(); err == nil {
			fmt.Printf("Worktrees now live in the projects root: %s\n\n", root)
		}
	}

	for _, p := range plans {
		suffix := ""
		if p.Renamed {
			suffix = "  (renamed to avoid conflict)"
		}
		fmt.Printf("%s [%s] %s\n  from: %s\n  to:   %s%s\n\n", verb, p.RepoName, p.Branch, p.From, p.To, suffix)
	}

	if dryRun {
		fmt.Printf("Total: %d %s to align\n", len(plans), pluralize(len(plans), "worktree", "worktrees"))
		return nil
	}

	// Execute moves
	var errors []string
	moved := 0

	for _, p := range plans {
		// Create the repo's projects directory if needed
		wtDir, err := xdg.RepoProjectsDir(p.RepoName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("  %s/%s: %v", p.RepoName, p.Branch, err))
			continue
		}
		if err := os.MkdirAll(wtDir, 0755); err != nil {
			errors = append(errors, fmt.Sprintf("  %s/%s: failed to create projects dir: %v", p.RepoName, p.Branch, err))
			continue
		}

		// Ensure parent directories exist for branches with slashes
		parentDir := filepath.Dir(p.To)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			errors = append(errors, fmt.Sprintf("  %s/%s: failed to create parent dir: %v", p.RepoName, p.Branch, err))
			continue
		}

		if err := git.MoveWorktree(p.RepoPath, p.From, p.To); err != nil {
			errors = append(errors, fmt.Sprintf("  %s/%s: %v", p.RepoName, p.Branch, err))
			continue
		}
		moved++

		// Tidying the home directory is the point of the move, so clean up the
		// legacy directory once its last worktree has left.
		if removed := removeEmptyLegacyDir(p.From); removed != "" {
			fmt.Printf("Removed empty %s\n", removed)
		}
	}

	// Re-discover worktrees for affected repos and save
	affectedRepos := make(map[string]bool)
	for _, p := range plans {
		affectedRepos[p.RepoPath] = true
	}
	for i := range cfg.Repositories {
		repo := &cfg.Repositories[i]
		if !affectedRepos[repo.Path] {
			continue
		}
		// Repair broken links, then prune truly dead entries
		_ = git.RepairWorktrees(repo.Path)
		_ = git.PruneWorktrees(repo.Path)
		entries, err := git.ListWorktrees(repo.Path)
		if err != nil {
			continue
		}
		wts := make([]config.Worktree, len(entries))
		for j, e := range entries {
			wts[j] = config.Worktree{
				Path:    e.Path,
				Branch:  e.Branch,
				Aligned: git.IsAligned(e.Path, repo.Name),
			}
		}
		repo.Worktrees = wts
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Aligned %d %s\n", moved, pluralize(moved, "worktree", "worktrees"))
	if len(errors) > 0 {
		fmt.Printf("\n%d %s:\n%s\n", len(errors), pluralize(len(errors), "error", "errors"), strings.Join(errors, "\n"))
	}

	return nil
}

// legacyWtSuffix is the pre-XDG convention: worktrees lived in a sibling
// directory named <repo>.wt next to the repository itself.
const legacyWtSuffix = ".wt"

// isLegacyWtPath reports whether a worktree still sits in a <repo>.wt directory.
func isLegacyWtPath(worktreePath string) bool {
	for dir := filepath.Dir(worktreePath); ; dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, legacyWtSuffix) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
	}
}

// removeEmptyLegacyDir removes the <repo>.wt directory a worktree was moved out
// of, once nothing is left in it, and returns the path removed. Directories that
// still hold anything are left alone — the goal is to remove the husk, never to
// discard something the user still has there.
func removeEmptyLegacyDir(movedFrom string) string {
	dir := filepath.Dir(movedFrom)
	if !strings.HasSuffix(dir, legacyWtSuffix) {
		return ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return ""
	}
	if err := os.Remove(dir); err != nil {
		return ""
	}
	return dir
}

// hasLegacyPlan reports whether any planned move starts in a pre-XDG .wt
// directory, meaning the user is seeing the new location for the first time.
func hasLegacyPlan(plans []alignPlan) bool {
	for _, p := range plans {
		if isLegacyWtPath(p.From) {
			return true
		}
	}
	return false
}
