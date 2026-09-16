package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
)

var (
	// flagWorktreeRefreshTags backs --tag. It is a slice so a repeated flag is
	// detectable and can be rejected, as for add and remove.
	flagWorktreeRefreshTags   []string
	flagWorktreeRefreshDryRun bool
)

var worktreeRefreshCmd = &cobra.Command{
	Use:   "refresh [repo|.]",
	Short: "Re-sync stored worktree data with git",
	Long: `Re-sync stored worktree data with what git reports.

For each repository, runs git worktree repair, then git worktree prune, then
git worktree list, and rebuilds the stored worktree entries from the result.
Worktrees created or deleted outside omgitworks are picked up, and whether
each worktree sits in the projects root is recomputed. Each repository whose
stored data changed is listed with what changed, followed by a summary.

This updates worktree data only. Discovering new repositories, re-detecting
git users, and clearing the status cache remain with 'omgw refresh'.

Repair runs before prune so that a worktree whose link is broken but whose
directory still exists is fixed rather than discarded. Use --dry-run to see
what would change: it runs neither repair nor prune, since both change git
state, and saves nothing.

Targeting:

  refresh                   every tracked repo
  refresh <repo>            every repo matching the name pattern
  refresh .                 the repo of the current directory
  refresh -t <tag>          every repo carrying the tag
  refresh <repo> -t <tag>   repos matching the name AND the tag

Unlike 'worktree add' and 'worktree remove', omitting the repo argument means
every repository, not the current one, and a name pattern matching several
repositories refreshes all of them.

Examples:
  gws worktree refresh                   # Re-sync every repo
  gws worktree refresh my-repo           # Only repos matching my-repo
  gws worktree refresh .                 # Only the current repo
  gws worktree refresh -t backend        # Every repo tagged backend
  gws worktree refresh --dry-run         # Preview without changing anything`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := runWorktreeRefreshCommand(args, os.Stdout)
		if err != nil && !errors.Is(err, errPartialFailure) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return err
	},
}

func init() {
	worktreeRefreshCmd.Flags().StringArrayVarP(&flagWorktreeRefreshTags, "tag", "t", nil,
		"Select repositories by tag (single value; not repeatable)")
	worktreeRefreshCmd.Flags().BoolVar(&flagWorktreeRefreshDryRun, "dry-run", false,
		"Preview changes without writing them")
	worktreeRefreshCmd.ValidArgsFunction = completeWorktreeRepoOrDot
	_ = worktreeRefreshCmd.RegisterFlagCompletionFunc("tag",
		func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeAllTags(toComplete)
		})
	// A partial failure prints its own report; other errors print themselves in RunE.
	worktreeRefreshCmd.SilenceErrors = true
	worktreeRefreshCmd.SilenceUsage = true
	worktreeCmd.AddCommand(worktreeRefreshCmd)
}

// runWorktreeRefreshCommand resolves the targeted repositories from the
// arguments and flags, then refreshes them.
func runWorktreeRefreshCommand(args []string, stdout io.Writer) error {
	tag, err := singleTagValue(flagWorktreeRefreshTags, "--tag")
	if err != nil {
		return err
	}
	repoArg := ""
	if len(args) == 1 {
		repoArg = args[0]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	repos, err := selectWorktreeRefreshTargets(cfg, repoArg, tag)
	if err != nil {
		return err
	}
	return runWorktreeRefresh(cfg, repos, flagWorktreeRefreshDryRun, stdout)
}

// selectWorktreeRefreshTargets resolves refresh's targeting to a repository set.
//
// It deliberately does not use selectWorktreeTargets. That precedence table
// serves add and remove, where an omitted repository means the current one and
// an ambiguous name pattern is an error. Refresh is a bulk metadata operation
// like list and align: an omitted repository means every repository, and a
// pattern selects every match. So the base set comes from worktreeScopeFor, the
// model list and align share, and a tag narrows it with AND semantics.
func selectWorktreeRefreshTargets(cfg *config.Config, repoArg, tag string) ([]*config.Repository, error) {
	scope, err := worktreeScopeFor(cfg, repoArg)
	if err != nil {
		return nil, err
	}

	var repos []*config.Repository
	for _, repo := range allRepositories(cfg) {
		if scope.matches(repo) {
			repos = append(repos, repo)
		}
	}
	if tag != "" {
		repos = filterByTag(repos, tag)
	}

	if len(repos) > 0 {
		return repos, nil
	}
	switch {
	case repoArg == currentRepoArg && tag != "":
		return nil, fmt.Errorf("current repository '%s' is not tagged '%s'", scope.Label, tag)
	case repoArg != "" && tag != "":
		return nil, fmt.Errorf("no repository found matching '%s' and tagged '%s'", repoArg, tag)
	case tag != "":
		return nil, fmt.Errorf("no repository found tagged '%s'", tag)
	case repoArg != "":
		return nil, fmt.Errorf("no repository found matching '%s'", repoArg)
	default:
		// No filters and nothing tracked: there is simply nothing to refresh.
		return nil, nil
	}
}

// runWorktreeRefresh re-syncs stored worktree data for the given repositories.
//
// It touches worktree data only. Repository discovery, git user detection, and
// the status cache belong to the full 'omgw refresh' and are deliberately not
// called from here.
//
// A dry run reports the same changes without making them: it lists worktrees
// but skips repair and prune, which change git state, and saves nothing.
//
// A repository that fails is reported and skipped; the rest still run.
// Configuration is saved once, after every repository has been attempted: a
// save per repository would let a later save of stale data overwrite an
// earlier repository's result.
func runWorktreeRefresh(cfg *config.Config, repos []*config.Repository, dryRun bool, stdout io.Writer) error {
	var (
		refreshed int
		changed   int
		failures  []string
	)

	if dryRun {
		fmt.Fprintln(stdout, "Dry run — no changes will be made:")
		fmt.Fprintln(stdout, "git worktree repair and prune were not run, so a worktree that repair would fix is shown as it currently stands.")
		fmt.Fprintln(stdout)
	}

	for _, repo := range repos {
		// Snapshot before syncing: the helper replaces the field in place.
		before := slices.Clone(repo.Worktrees)
		after, err := refreshedWorktrees(repo, dryRun)
		if err != nil {
			failures = append(failures, fmt.Sprintf("  %s: %v", repo.Name, err))
			continue
		}
		refreshed++

		changes := diffWorktrees(before, after)
		if changes.changed() {
			changed++
			printWorktreeChanges(stdout, repo.Name, changes)
		}
	}

	if !dryRun {
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	}

	printWorktreeRefreshSummary(stdout, refreshed, changed, len(failures), dryRun)

	if len(failures) > 0 {
		fmt.Fprintf(stdout, "\n%d %s:\n%s\n",
			len(failures), pluralize(len(failures), "error", "errors"),
			strings.Join(failures, "\n"))
		return errPartialFailure
	}

	return nil
}

// refreshedWorktrees returns what a repository's stored worktrees are, or in a
// dry run would be, after a refresh. Only a real run touches git state or the
// repository's stored entries.
func refreshedWorktrees(repo *config.Repository, dryRun bool) ([]config.Worktree, error) {
	if dryRun {
		return buildWorktreeEntries(repo.Path, repo.Name)
	}
	if err := syncRepoWorktrees(repo); err != nil {
		return nil, err
	}
	return repo.Worktrees, nil
}

// worktreeChanges is the difference between a repository's stored worktrees
// before and after a refresh.
type worktreeChanges struct {
	Added      []config.Worktree
	Removed    []config.Worktree
	Realigned  []config.Worktree // after-state entries whose Aligned value flipped
	Rebranched []branchChange
}

// branchChange records a worktree still at the same path but now on a
// different branch, as happens after a checkout inside the worktree.
type branchChange struct {
	Worktree config.Worktree // after-state entry
	Previous string
}

func (c worktreeChanges) changed() bool {
	return len(c.Added)+len(c.Removed)+len(c.Realigned)+len(c.Rebranched) > 0
}

// diffWorktrees compares stored worktree entries keyed on path. Branch cannot
// be the key: a detached HEAD has none, so two detached worktrees would
// collapse into one. A worktree moved to a new path is therefore reported as
// one removal and one addition, which is what happened to the stored entries.
//
// Output order follows the input slices, which follow git's listing, so the
// report is deterministic.
func diffWorktrees(before, after []config.Worktree) worktreeChanges {
	previous := make(map[string]config.Worktree, len(before))
	for _, wt := range before {
		previous[wt.Path] = wt
	}
	current := make(map[string]bool, len(after))

	var changes worktreeChanges
	for _, wt := range after {
		current[wt.Path] = true
		old, existed := previous[wt.Path]
		switch {
		case !existed:
			changes.Added = append(changes.Added, wt)
		default:
			if old.Aligned != wt.Aligned {
				changes.Realigned = append(changes.Realigned, wt)
			}
			if old.Branch != wt.Branch {
				changes.Rebranched = append(changes.Rebranched, branchChange{Worktree: wt, Previous: old.Branch})
			}
		}
	}
	for _, wt := range before {
		if !current[wt.Path] {
			changes.Removed = append(changes.Removed, wt)
		}
	}
	return changes
}

// printWorktreeChanges prints one repository's changes, one line per entry.
func printWorktreeChanges(w io.Writer, repoName string, c worktreeChanges) {
	fmt.Fprintf(w, "[%s]\n", repoName)
	for _, wt := range c.Added {
		fmt.Fprintf(w, "  added      %s  %s  %s\n", branchLabel(wt.Branch), wt.Path, alignmentLabel(wt.Aligned))
	}
	for _, wt := range c.Removed {
		fmt.Fprintf(w, "  removed    %s  %s\n", branchLabel(wt.Branch), wt.Path)
	}
	for _, wt := range c.Realigned {
		fmt.Fprintf(w, "  realigned  %s  %s  now %s\n", branchLabel(wt.Branch), wt.Path, alignmentLabel(wt.Aligned))
	}
	for _, bc := range c.Rebranched {
		fmt.Fprintf(w, "  branch     %s  %s  was %s\n", branchLabel(bc.Worktree.Branch), bc.Worktree.Path, branchLabel(bc.Previous))
	}
}

// printWorktreeRefreshSummary prints the closing count line. Refreshed counts
// repositories that synced successfully; failures are counted separately,
// following the add and remove summaries. A dry run frames the same counts as
// what would happen.
func printWorktreeRefreshSummary(w io.Writer, refreshed, changed, failed int, dryRun bool) {
	repositories := pluralize(refreshed, "repository", "repositories")
	if dryRun {
		fmt.Fprintf(w, "\nWould refresh %d %s, %d would change", refreshed, repositories, changed)
	} else {
		fmt.Fprintf(w, "\nRefreshed %d %s, %d changed", refreshed, repositories, changed)
	}
	if failed > 0 {
		fmt.Fprintf(w, ", %d failed", failed)
	}
	fmt.Fprintln(w)
}

func branchLabel(branch string) string {
	if branch == "" {
		return "(detached)"
	}
	return branch
}

func alignmentLabel(aligned bool) string {
	if aligned {
		return "aligned"
	}
	return "unaligned"
}
