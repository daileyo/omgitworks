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

var worktreeRefreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-sync stored worktree data with git",
	Long: `Re-sync stored worktree data with what git reports.

For each repository, runs git worktree repair, then git worktree prune, then
git worktree list, and rebuilds the stored worktree entries from the result.
Worktrees created or deleted outside omgitworks are picked up, and whether
each worktree sits in the projects root is recomputed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := runWorktreeRefreshCommand(os.Stdout)
		if err != nil && !errors.Is(err, errPartialFailure) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return err
	},
}

func init() {
	// A partial failure prints its own report; other errors print themselves in RunE.
	worktreeRefreshCmd.SilenceErrors = true
	worktreeRefreshCmd.SilenceUsage = true
	worktreeCmd.AddCommand(worktreeRefreshCmd)
}

// runWorktreeRefreshCommand loads configuration and refreshes every tracked
// repository.
func runWorktreeRefreshCommand(stdout io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	return runWorktreeRefresh(cfg, allRepositories(cfg), stdout)
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
