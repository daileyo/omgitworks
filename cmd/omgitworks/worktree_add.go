package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/filter"
	"github.com/daileyo/omgitworks/internal/git"
	"github.com/daileyo/omgitworks/internal/repocontext"
	"github.com/daileyo/omgitworks/internal/xdg"
)

var worktreeAddCmd = &cobra.Command{
	Use:   "add [repo] <branch>",
	Short: "Create a new worktree in the projects root",
	Long: `Create a new git worktree for the given branch.

Worktrees live under the XDG projects root, one directory per repository:

  $XDG_DATA_HOME/gws/projects/<repo>/<branch>
  (default: ~/.local/share/gws/projects/<repo>/<branch>)

The directory is created automatically if it does not exist.

When the repo argument is omitted, the repository is determined from the
current directory. This also works from inside one of that repository's
worktrees, where the worktree's owning repository is used. Naming a repo
explicitly always takes precedence over the current directory.

With -t, the worktree is created in every repository carrying the tag. A repo
name and a tag together select repositories satisfying both.

  Invocation                        Repositories targeted
  add <branch>                      the repo of the current directory
  add -t <tag> <branch>             all repos carrying the tag
  add <repo> <branch>               repos matching the name pattern
  add <repo> <branch> -t <tag>      repos matching the name AND the tag

Examples:
  gws worktree add my-repo feature-auth
  gws worktree add my-repo feature/new-api
  gws worktree add feature-auth              # Repo from the current directory
  gws worktree add -t backend feat-auth      # Every repo tagged backend
  gws worktree add api feat-auth -t backend  # Repos matching both`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWorktreeAddCommand(args)
	},
}

func init() {
	worktreeAddCmd.Flags().StringArrayVarP(&flagWorktreeAddTags, "tag", "t", nil,
		"Select repositories by tag (single value; not repeatable)")
	worktreeAddCmd.ValidArgsFunction = completeWorktreeAddArgs
	_ = worktreeAddCmd.RegisterFlagCompletionFunc("tag",
		func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeAllTags(toComplete)
		})
	// The bulk path prints its own summary and signals partial failure with a
	// sentinel, so cobra must not print over it. Every other error path prints
	// itself explicitly in runWorktreeAddCommand.
	worktreeAddCmd.SilenceErrors = true
	worktreeAddCmd.SilenceUsage = true
	worktreeCmd.AddCommand(worktreeAddCmd)
}

// flagWorktreeAddTags backs --tag. It is a slice rather than a string so a
// repeated flag can be rejected: a plain string silently keeps the last value.
var flagWorktreeAddTags []string

// worktreeAddTag returns the single --tag value, rejecting repetition.
func worktreeAddTag() (string, error) {
	switch len(flagWorktreeAddTags) {
	case 0:
		return "", nil
	case 1:
		return flagWorktreeAddTags[0], nil
	default:
		return "", fmt.Errorf("--tag accepts a single value, but was given %d times", len(flagWorktreeAddTags))
	}
}

// runWorktreeAddCommand implements the targeting precedence table: the branch is
// always the last positional, a first positional is a repository name pattern,
// and -t narrows by tag.
func runWorktreeAddCommand(args []string) error {
	err := runWorktreeAddResolved(args)
	// SilenceErrors keeps the bulk summary clean, so anything that is not the
	// partial-failure sentinel has to print itself here.
	if err != nil && !errors.Is(err, errPartialFailure) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	return err
}

func runWorktreeAddResolved(args []string) error {
	tag, err := worktreeAddTag()
	if err != nil {
		return err
	}

	var pattern string
	branch := args[len(args)-1]
	if len(args) == 2 {
		pattern = args[0]
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	repos, err := selectWorktreeAddTargets(cfg, pattern, tag)
	if err != nil {
		return err
	}

	// A tag makes this a bulk run whatever it matched; without one, the
	// selection is a single repository by construction.
	return runWorktreeAddBulk(cfg, repos, branch, tag == "")
}

// selectWorktreeAddTargets resolves the precedence table to a repository set.
//
// Detection is the lowest-precedence source: the resolver is consulted only when
// neither a name pattern nor a tag narrows the selection.
func selectWorktreeAddTargets(cfg *config.Config, pattern, tag string) ([]*config.Repository, error) {
	switch {
	case pattern != "" && tag != "":
		// AND semantics, matching 'omgw tag add --repo X --path Y'. The
		// ambiguity check deliberately does not apply here: narrowing a tagged
		// group by name is a bulk operation over the intersection.
		repos := filterByTag(selectByName(cfg, pattern), tag)
		if len(repos) == 0 {
			return nil, fmt.Errorf("no repository found matching '%s' and tagged '%s'", pattern, tag)
		}
		return repos, nil

	case tag != "":
		repos := filterByTag(allRepositories(cfg), tag)
		if len(repos) == 0 {
			return nil, fmt.Errorf("no repository found tagged '%s'", tag)
		}
		return repos, nil

	case pattern != "":
		repos := selectByName(cfg, pattern)
		if len(repos) == 0 {
			return nil, fmt.Errorf("no repository found matching '%s'", pattern)
		}
		// Without a tag to narrow it, an ambiguous pattern is still rejected.
		if len(repos) > 1 {
			return nil, fmt.Errorf("multiple repositories match '%s', narrow your query", pattern)
		}
		return repos, nil

	default:
		repo, err := repocontext.ResolveCurrent(cfg)
		if err != nil {
			return nil, err
		}
		return []*config.Repository{repo}, nil
	}
}

// allRepositories returns pointers to every tracked repository.
func allRepositories(cfg *config.Config) []*config.Repository {
	repos := make([]*config.Repository, 0, len(cfg.Repositories))
	for i := range cfg.Repositories {
		repos = append(repos, &cfg.Repositories[i])
	}
	return repos
}

// selectByName returns repositories whose name matches the pattern, using the
// partial case-insensitive matching the command has always used.
func selectByName(cfg *config.Config, pattern string) []*config.Repository {
	var repos []*config.Repository
	for i := range cfg.Repositories {
		if filter.MatchesPattern(cfg.Repositories[i].Name, pattern) {
			repos = append(repos, &cfg.Repositories[i])
		}
	}
	return repos
}

// filterByTag keeps repositories carrying the tag, matched with the exact,
// case-insensitive, wildcard-aware rule used for tags everywhere else.
func filterByTag(repos []*config.Repository, tag string) []*config.Repository {
	var kept []*config.Repository
	for _, repo := range repos {
		for _, t := range repo.Tags {
			if filter.MatchesExact(t, tag) {
				kept = append(kept, repo)
				break
			}
		}
	}
	return kept
}

func runWorktreeAdd(repoName, branch string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	repos, err := selectWorktreeAddTargets(cfg, repoName, "")
	if err != nil {
		return err
	}

	return runWorktreeAddBulk(cfg, repos, branch, true)
}

// runWorktreeAddCurrent creates a worktree in the repository owning the current
// directory. The resolver is consulted only here: when a repo is named
// explicitly, the working directory is never inspected.
func runWorktreeAddCurrent(branch string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	repo, err := repocontext.ResolveCurrent(cfg)
	if err != nil {
		return err
	}

	return runWorktreeAddBulk(cfg, []*config.Repository{repo}, branch, true)
}

// errPartialFailure signals that at least one repository in a bulk run failed.
// It carries no message: the summary has already been printed, and cobra is
// configured not to print over it.
var errPartialFailure = errors.New("")

// worktreeExistsError reports that the repository already has a worktree for
// the branch. It is typed so the bulk path can classify it as a skip rather
// than a failure without matching on message text.
type worktreeExistsError struct {
	Branch string
	Path   string
}

func (e *worktreeExistsError) Error() string {
	return fmt.Sprintf("worktree for branch '%s' already exists at %s", e.Branch, e.Path)
}

// createWorktreeForRepo creates one worktree and refreshes that repository's
// stored worktree data. It deliberately does not save configuration or print:
// a bulk run must save once at the end, or an early repository's data is
// overwritten by a later save of a stale configuration.
func createWorktreeForRepo(repo *config.Repository, branch string) (string, error) {
	for _, wt := range repo.Worktrees {
		if wt.Branch == branch {
			return "", &worktreeExistsError{Branch: branch, Path: wt.Path}
		}
	}

	// Compute destination path under the XDG projects root
	wtDir, err := xdg.RepoProjectsDir(repo.Name)
	if err != nil {
		return "", err
	}
	destPath := filepath.Join(wtDir, branch)

	// Create the repo's projects directory if needed. A branch name containing
	// slashes nests, so create the destination's parent rather than wtDir.
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create projects directory: %w", err)
	}

	if err := git.AddWorktree(repo.Path, branch, destPath); err != nil {
		return "", fmt.Errorf("failed to create worktree: %w", err)
	}

	// Re-discover worktrees for this repo so the saved config is accurate.
	entries, err := git.ListWorktrees(repo.Path)
	if err == nil {
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

	return destPath, nil
}

// runWorktreeAddBulk creates the branch's worktree in each selected repository,
// continuing past failures and reporting a summary. Successful creations are
// retained when a later repository fails; there is no rollback.
//
// single selects the historical one-repository contract: an existing worktree is
// an error and no summary is printed. It is a property of the invocation, not of
// how many repositories were selected — a tag that happens to match one
// repository is still a bulk run, where an existing worktree is a reported skip.
func runWorktreeAddBulk(cfg *config.Config, repos []*config.Repository, branch string, single bool) error {

	var (
		created  int
		skipped  int
		failures []string
	)

	for _, repo := range repos {
		destPath, err := createWorktreeForRepo(repo, branch)

		var exists *worktreeExistsError
		switch {
		case errors.As(err, &exists):
			// Already present is not a failure: the desired state holds.
			if single {
				return err
			}
			skipped++
			fmt.Printf("Skipping [%s] %s — %v\n", repo.Name, branch, err)

		case err != nil:
			if single {
				return err
			}
			failures = append(failures, fmt.Sprintf("  %s: %v", repo.Name, err))

		default:
			created++
			fmt.Printf("Created worktree for branch '%s' at %s\n", branch, destPath)
		}
	}

	// Save once, after every repository has been attempted.
	if created > 0 {
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	}

	if single {
		return nil
	}

	printWorktreeAddSummary(created, skipped, len(failures))

	if len(failures) > 0 {
		fmt.Printf("\n%d %s:\n%s\n",
			len(failures), pluralize(len(failures), "error", "errors"),
			strings.Join(failures, "\n"))
		return errPartialFailure
	}

	return nil
}

// printWorktreeAddSummary reports the counted outcome of a bulk run, omitting
// counts that are zero so a clean run stays quiet.
func printWorktreeAddSummary(created, skipped, failed int) {
	fmt.Printf("\nCreated %d %s", created, pluralize(created, "worktree", "worktrees"))
	if skipped > 0 {
		fmt.Printf(", skipped %d", skipped)
	}
	if failed > 0 {
		fmt.Printf(", %d failed", failed)
	}
	fmt.Println()
}
