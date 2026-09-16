package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
)

// flagWorktreeRefreshTags backs --tag. It is a slice so a repeated flag is
// detectable and can be rejected, as for add and remove.
var flagWorktreeRefreshTags []string

var worktreeRefreshCmd = &cobra.Command{
	Use:   "refresh [repo|.]",
	Short: "Re-sync stored worktree data with git",
	Long: `Re-sync stored worktree data with what git reports.

For each repository, runs git worktree repair, then git worktree prune, then
git worktree list, and rebuilds the stored worktree entries from the result.
Worktrees created or deleted outside omgitworks are picked up, and whether
each worktree sits in the projects root is recomputed.

Targeting:

  refresh                   every tracked repo
  refresh <repo>            every repo matching the name pattern
  refresh .                 the repo of the current directory
  refresh -t <tag>          every repo carrying the tag
  refresh <repo> -t <tag>   repos matching the name AND the tag

Unlike 'worktree add' and 'worktree remove', omitting the repo argument means
every repository, not the current one, and a name pattern matching several
repositories refreshes all of them.`,
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
	return runWorktreeRefresh(cfg, repos, stdout)
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
// A repository that fails is reported and skipped; the rest still run.
// Configuration is saved once, after every repository has been attempted: a
// save per repository would let a later save of stale data overwrite an
// earlier repository's result.
func runWorktreeRefresh(cfg *config.Config, repos []*config.Repository, stdout io.Writer) error {
	var failures []string

	for _, repo := range repos {
		if err := syncRepoWorktrees(repo); err != nil {
			failures = append(failures, fmt.Sprintf("  %s: %v", repo.Name, err))
		}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	if len(failures) > 0 {
		fmt.Fprintf(stdout, "\n%d %s:\n%s\n",
			len(failures), pluralize(len(failures), "error", "errors"),
			strings.Join(failures, "\n"))
		return errPartialFailure
	}

	return nil
}
