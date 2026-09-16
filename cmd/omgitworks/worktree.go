package main

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
)

var flagWorktreeQuiet bool

// worktreeCmd is the parent command for worktree management subcommands.
// When called with a branch pattern argument (e.g., "gws worktree feat-auth"),
// it navigates to the matching worktree (shorthand for "gws worktree navigate").
var worktreeCmd = &cobra.Command{
	Use:   "worktree [branch-pattern]",
	Short: "Manage git worktrees",
	Long: `Manage git worktrees across your workspace.

When called with a branch pattern, navigates to the matching worktree:
  gws worktree feat-auth                # Same as: gws worktree navigate feat-auth
  gws worktree "feat-*"                 # Wildcard match with interactive selection

Subcommands:
  gws worktree navigate <branch>        # Navigate to a worktree by branch name
  gws worktree list [repo]              # List all worktrees (optionally filtered by repo)
  gws worktree add [repo] <branch>      # Create a new worktree in the projects root
  gws worktree align [repo]             # Move unaligned worktrees into the projects root`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return runWorktreeNavigateGlobal(args[0], flagWorktreeQuiet, cfg.Repositories, os.Stderr, os.Stdout, os.Stdin)
		}
		return cmd.Help()
	},
}

func init() {
	worktreeCmd.PersistentFlags().BoolVarP(&flagWorktreeQuiet, "quiet", "q", false, "Suppress verbose output, print only the path")

	// Tab completion: suggest worktree branch names across all repos
	worktreeCmd.ValidArgsFunction = completeWorktreeBranches
	rootCmd.AddCommand(worktreeCmd)
}

// completeWorktreeBranches returns unique worktree branch names for tab completion.
func completeWorktreeBranches(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	seen := make(map[string]bool)
	var branches []string
	for _, repo := range cfg.Repositories {
		for _, wt := range repo.Worktrees {
			if wt.Branch != "" && !seen[wt.Branch] && strings.HasPrefix(strings.ToLower(wt.Branch), strings.ToLower(toComplete)) {
				seen[wt.Branch] = true
				branches = append(branches, wt.Branch)
			}
		}
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
