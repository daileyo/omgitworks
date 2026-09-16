package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/filter"
	"github.com/daileyo/omgitworks/internal/git"
	"github.com/daileyo/omgitworks/internal/repocontext"
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
  gws worktree list [repo|.]            # List all worktrees (. means the current repo)
  gws worktree add [repo] <branch>      # Create a new worktree in the projects root
  gws worktree align [repo|.]           # Move unaligned worktrees into the projects root
  gws worktree remove [repo] <branch>   # Remove a worktree (alias: rm)`,
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

// currentRepoArg is the repository argument meaning "the repository I am
// standing in". It occupies the same argument position as a repository name so
// that list, align, and the future refresh stay uniform, and it is intercepted
// before any name matching: a tracked repository whose name contains a period
// must never be matched by it.
const currentRepoArg = "."

// worktreeScope describes which repositories a worktree subcommand acts on.
//
// The zero value means every tracked repository, preserving what an omitted
// argument has always meant for list and align. RepoPath and NamePattern are
// never both set: RepoPath is an already-resolved exact path, NamePattern is
// the legacy pattern.
type worktreeScope struct {
	NamePattern string
	RepoPath    string
	Label       string // what to call this scope in messages
}

// matches reports whether repo falls inside the scope.
func (s worktreeScope) matches(repo *config.Repository) bool {
	if s.RepoPath != "" {
		// Exact, never prefix: see repocontext.Resolve for why.
		return git.ResolvePath(repo.Path) == s.RepoPath
	}
	if s.NamePattern != "" {
		return filter.MatchesPattern(repo.Name, s.NamePattern)
	}
	return true
}

// worktreeScopeFor turns a repository argument into a scope. An empty argument
// means all repositories, "." resolves the current one, and anything else is a
// name pattern.
func worktreeScopeFor(cfg *config.Config, arg string) (worktreeScope, error) {
	switch arg {
	case "":
		return worktreeScope{}, nil
	case currentRepoArg:
		repo, err := repocontext.ResolveCurrent(cfg)
		if err != nil {
			return worktreeScope{}, err
		}
		return worktreeScope{RepoPath: git.ResolvePath(repo.Path), Label: repo.Name}, nil
	default:
		return worktreeScope{NamePattern: arg, Label: arg}, nil
	}
}

// completeBranchNames suggests local branch names from the repository at
// repoPath, filtered on the typed prefix the same way completeRepoNames filters
// repository names.
func completeBranchNames(repoPath, toComplete string) ([]string, cobra.ShellCompDirective) {
	all, err := git.ListBranches(repoPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var branches []string
	for _, b := range all {
		if strings.HasPrefix(strings.ToLower(b), strings.ToLower(toComplete)) {
			branches = append(branches, b)
		}
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}

// completeWorktreeAddArgs completes "worktree add [repo] <branch>".
//
// The first positional is ambiguous by design: it is a branch when the working
// directory resolves to a tracked repository, and a repository name otherwise.
// Completion follows the same rule the command itself uses, so the suggestions
// match what the argument will actually mean.
func completeWorktreeAddArgs(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	switch len(args) {
	case 0:
		if repo, err := repocontext.ResolveCurrent(cfg); err == nil {
			return completeBranchNames(repo.Path, toComplete)
		}
		return completeRepoNames(toComplete)
	case 1:
		// The first argument named a repository, so the second is its branch.
		repos := findRepositories(cfg, args[0])
		if len(repos) != 1 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeBranchNames(repos[0].Path, toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeWorktreeRepoOrDot completes the optional repository argument of list
// and align, offering "." only when the working directory actually resolves.
func completeWorktreeRepoOrDot(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names, directive := completeRepoNames(toComplete)

	cfg, err := config.Load()
	if err != nil {
		return names, directive
	}
	if _, err := repocontext.ResolveCurrent(cfg); err != nil {
		return names, directive
	}
	if !strings.HasPrefix(currentRepoArg, toComplete) {
		return names, directive
	}
	return append([]string{currentRepoArg}, names...), directive
}

// completeAllTags returns the deduplicated tags in use across every tracked
// repository, filtered on the typed prefix. Used for --tag on worktree add,
// where the tag is workspace-wide rather than scoped to one repository.
func completeAllTags(toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	seen := make(map[string]bool)
	var tags []string
	for _, repo := range cfg.Repositories {
		for _, tag := range repo.Tags {
			if seen[tag] {
				continue
			}
			if strings.HasPrefix(strings.ToLower(tag), strings.ToLower(toComplete)) {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags, cobra.ShellCompDirectiveNoFileComp
}

// selectWorktreeTargets resolves the precedence table to a repository set.
//
// Detection is the lowest-precedence source: the resolver is consulted only when
// neither a name pattern nor a tag narrows the selection.
func selectWorktreeTargets(cfg *config.Config, pattern, tag string) ([]*config.Repository, error) {
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

// singleTagValue returns the one value of a single-valued tag flag, rejecting
// repetition. The flag is declared as a string slice precisely so a repeat is
// detectable: a plain string would silently keep the last value.
func singleTagValue(values []string, flagName string) (string, error) {
	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return values[0], nil
	default:
		return "", fmt.Errorf("%s accepts a single value, but was given %d times", flagName, len(values))
	}
}

// completeRemovableBranches suggests only branches that actually have a
// worktree in the targeted repository. Completing to a branch without one could
// only ever produce an error.
func completeRemovableBranches(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var repos []*config.Repository
	switch len(args) {
	case 0:
		if repo, rErr := repocontext.ResolveCurrent(cfg); rErr == nil {
			repos = []*config.Repository{repo}
		} else {
			// Outside a tracked repository the first argument is a repo name.
			return completeRepoNames(toComplete)
		}
	case 1:
		repos = findRepositories(cfg, args[0])
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	seen := make(map[string]bool)
	var branches []string
	for _, repo := range repos {
		for _, wt := range repo.Worktrees {
			if wt.Branch == "" || seen[wt.Branch] {
				continue
			}
			if strings.HasPrefix(strings.ToLower(wt.Branch), strings.ToLower(toComplete)) {
				seen[wt.Branch] = true
				branches = append(branches, wt.Branch)
			}
		}
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}
