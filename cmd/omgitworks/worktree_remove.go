package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
	"github.com/daileyo/omgitworks/internal/xdg"
)

var (
	flagWorktreeRemoveTags   []string
	flagWorktreeRemoveForce  bool
	flagWorktreeRemoveDryRun bool
	flagWorktreeRemoveYes    bool
)

// stdinIsTerminalFunc reports whether stdin is attached to a terminal. It is a
// variable so tests can force either answer, following stdoutIsTerminalFunc.
var stdinIsTerminalFunc = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

var worktreeRemoveCmd = &cobra.Command{
	Use:     "remove [repo] <branch>",
	Aliases: []string{"rm"},
	Short:   "Remove a worktree",
	Long: `Remove the worktree for a branch, in one repository or across a tagged group.

The branch itself is never deleted: removing a checkout is not the same as
discarding the work on it.

Uncommitted or untracked changes block removal, because git itself refuses.
Pass --force to override. Locked worktrees are always skipped, even with
--force.

A run targeting more than one worktree prints the full plan and asks for
confirmation first. Use --dry-run to preview without removing, or --yes to
skip the prompt in scripts.

Targeting follows the same rules as 'worktree add':

  remove <branch>                   the repo of the current directory
  remove -t <tag> <branch>          every repo carrying the tag
  remove <repo> <branch>            repos matching the name pattern
  remove <repo> <branch> -t <tag>   repos matching the name AND the tag

Examples:
  gws worktree remove my-repo feat-auth
  gws worktree rm feat-auth                  # Repo from the current directory
  gws worktree remove -t backend feat-auth   # Every repo tagged backend
  gws worktree remove -t backend feat-auth --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := runWorktreeRemoveCommand(args, os.Stdout, os.Stdin)
		if err != nil && !errors.Is(err, errPartialFailure) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return err
	},
}

func init() {
	worktreeRemoveCmd.Flags().StringArrayVarP(&flagWorktreeRemoveTags, "tag", "t", nil,
		"Select repositories by tag (single value; not repeatable)")
	worktreeRemoveCmd.Flags().BoolVar(&flagWorktreeRemoveForce, "force", false,
		"Remove even when the worktree has uncommitted or untracked changes")
	worktreeRemoveCmd.Flags().BoolVar(&flagWorktreeRemoveDryRun, "dry-run", false,
		"Preview what would be removed without removing anything")
	worktreeRemoveCmd.Flags().BoolVarP(&flagWorktreeRemoveYes, "yes", "y", false,
		"Skip the confirmation prompt")
	// The bulk path prints its own summary; other errors print themselves in RunE.
	worktreeRemoveCmd.SilenceErrors = true
	worktreeRemoveCmd.SilenceUsage = true
	worktreeRemoveCmd.ValidArgsFunction = completeRemovableBranches
	_ = worktreeRemoveCmd.RegisterFlagCompletionFunc("tag",
		func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeAllTags(toComplete)
		})
	worktreeCmd.AddCommand(worktreeRemoveCmd)
}

// removeOptions carries the flag state through the removal path.
type removeOptions struct {
	Force  bool
	DryRun bool
	Yes    bool
}

// worktreeNotFoundError reports that the repository has no worktree for the
// branch. Typed so a bulk run can classify it as a skip without matching text.
type worktreeNotFoundError struct{ Repo, Branch string }

func (e *worktreeNotFoundError) Error() string {
	return fmt.Sprintf("no worktree for branch '%s' in repository '%s'", e.Branch, e.Repo)
}

// worktreeLockedError reports a locked worktree, which is never removed.
type worktreeLockedError struct{ Repo, Branch, Reason string }

func (e *worktreeLockedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("worktree for branch '%s' is locked: %s", e.Branch, e.Reason)
	}
	return fmt.Sprintf("worktree for branch '%s' is locked", e.Branch)
}

func runWorktreeRemoveCommand(args []string, stdout io.Writer, stdin io.Reader) error {
	tag, err := singleTagValue(flagWorktreeRemoveTags, "--tag")
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

	repos, err := selectWorktreeTargets(cfg, pattern, tag)
	if err != nil {
		return err
	}

	opts := removeOptions{
		Force:  flagWorktreeRemoveForce,
		DryRun: flagWorktreeRemoveDryRun,
		Yes:    flagWorktreeRemoveYes,
	}
	// A tag makes this a bulk run whatever it matched; deriving this from the
	// number of selected repositories would turn a legitimate skip into a hard
	// error when a tag matches exactly one repository.
	single := tag == "" && pattern == ""
	if pattern != "" && tag == "" {
		single = true
	}

	return runWorktreeRemove(cfg, repos, branch, opts, single, stdout, stdin)
}

// findWorktreeForBranch returns the repository's worktree for an exactly
// matching branch. Partial matching would delete the wrong worktree.
func findWorktreeForBranch(repo *config.Repository, branch string) (config.Worktree, bool) {
	for _, wt := range repo.Worktrees {
		if wt.Branch == branch {
			return wt, true
		}
	}
	return config.Worktree{}, false
}

// removalPlan is one worktree's entry in a planned run. The same structure
// feeds the dry run, the confirmation prompt, and the removal itself, so the
// text a user approves is the text --dry-run shows.
type removalPlan struct {
	RepoName   string
	RepoPath   string
	Branch     string
	Path       string
	Locked     bool
	LockReason string
	Dirty      bool
}

// buildRemovalPlan computes the full plan before anything is removed.
func buildRemovalPlan(repos []*config.Repository, branch string, force bool) ([]removalPlan, []error) {
	var plans []removalPlan
	var missing []error

	for _, repo := range repos {
		wt, ok := findWorktreeForBranch(repo, branch)
		if !ok {
			missing = append(missing, &worktreeNotFoundError{Repo: repo.Name, Branch: branch})
			continue
		}

		p := removalPlan{
			RepoName: repo.Name,
			RepoPath: repo.Path,
			Branch:   branch,
			Path:     wt.Path,
		}
		// Checked up front so the preview can mark locks without a failed
		// subprocess, and so --force can never override one.
		p.Locked, p.LockReason = git.IsWorktreeLocked(repo.Path, wt.Path)
		if !p.Locked && !force {
			p.Dirty = worktreeIsDirty(wt.Path)
		}
		plans = append(plans, p)
	}

	return plans, missing
}

// worktreeIsDirty reports whether the worktree has uncommitted or untracked
// changes, so the preview can flag what would fail. The real removal still
// defers to git's own refusal.
func worktreeIsDirty(worktreePath string) bool {
	status, err := git.GetStatus(worktreePath)
	if err != nil {
		return false
	}
	return !status.IsClean
}

// renderRemovalPlan prints the plan shared by --dry-run and the confirmation
// prompt, following the shape of worktree align --dry-run.
//
// Both callers are previews of work not yet done, so both use the same
// conditional wording: the listing a user approves at the prompt is textually
// identical to the one --dry-run shows.
func renderRemovalPlan(plans []removalPlan, missing []error, opts removeOptions, w io.Writer) {
	const verb = "Would remove"
	if opts.DryRun {
		fmt.Fprintln(w, "Dry run — no changes will be made:")
		fmt.Fprintln(w)
	}
	if opts.Force {
		fmt.Fprintln(w, "--force is set: uncommitted changes will not block removal.")
	} else {
		fmt.Fprintln(w, "--force is not set: worktrees with uncommitted changes will fail.")
	}
	fmt.Fprintln(w)

	// Counted the way the real run's summary counts them, so the preview and
	// the outcome agree: locked worktrees and repositories without the branch
	// are skips, not removals.
	toRemove, skipped := 0, len(missing)

	for _, p := range plans {
		suffix := ""
		switch {
		case p.Locked:
			skipped++
			suffix = "  (locked: will be skipped)"
			if p.LockReason != "" {
				suffix = fmt.Sprintf("  (locked: %s — will be skipped)", p.LockReason)
			}
		case p.Dirty:
			toRemove++
			suffix = "  (has uncommitted changes — will fail without --force)"
		default:
			toRemove++
		}
		fmt.Fprintf(w, "%s [%s] %s\n  path: %s%s\n\n", verb, p.RepoName, p.Branch, p.Path, suffix)
	}

	// Repositories that will be skipped because they lack the branch are shown
	// too: whether the tag matched what the user expected is exactly what a
	// preview exists to reveal.
	for _, m := range missing {
		fmt.Fprintf(w, "Would skip — %v\n\n", m)
	}

	fmt.Fprintf(w, "Total: %d %s to remove", toRemove, pluralize(toRemove, "worktree", "worktrees"))
	if skipped > 0 {
		fmt.Fprintf(w, ", %d skipped", skipped)
	}
	fmt.Fprintln(w)
}

// confirmRemoval asks the user to approve the plan. Declining is an answer, not
// an error.
func confirmRemoval(stdout io.Writer, stdin io.Reader) (bool, error) {
	if !stdinIsTerminalFunc() {
		return false, errors.New("confirmation required but stdin is not a terminal; pass --yes to proceed")
	}

	fmt.Fprint(stdout, "Remove these worktrees? [y/N]: ")
	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

func runWorktreeRemove(cfg *config.Config, repos []*config.Repository, branch string,
	opts removeOptions, single bool, stdout io.Writer, stdin io.Reader) error {

	plans, missing := buildRemovalPlan(repos, branch, opts.Force)

	// In a single-repository run a missing worktree is a hard error, preserving
	// the individual-command contract. In a bulk run it is a reported skip.
	if single && len(missing) > 0 {
		return missing[0]
	}
	if len(plans) == 0 {
		if len(missing) > 0 {
			for _, m := range missing {
				fmt.Fprintf(stdout, "Skipping — %v\n", m)
			}
			fmt.Fprintf(stdout, "\nRemoved 0 worktrees, skipped %d\n", len(missing))
			return nil
		}
		return fmt.Errorf("no worktree for branch '%s' in the selected repositories", branch)
	}

	if opts.DryRun {
		renderRemovalPlan(plans, missing, opts, stdout)
		return nil
	}

	// More than one worktree is gated: the user sees the whole plan first.
	if len(plans) > 1 && !opts.Yes {
		renderRemovalPlan(plans, missing, opts, stdout)
		fmt.Fprintln(stdout)
		ok, err := confirmRemoval(stdout, stdin)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(stdout, "Aborted; nothing was removed.")
			return nil
		}
	}

	var (
		removed  int
		skipped  = len(missing)
		failures []string
	)

	for _, m := range missing {
		fmt.Fprintf(stdout, "Skipping — %v\n", m)
	}

	for _, p := range plans {
		if p.Locked {
			skipped++
			fmt.Fprintf(stdout, "Skipping [%s] %s — %v\n", p.RepoName, p.Branch,
				&worktreeLockedError{Repo: p.RepoName, Branch: p.Branch, Reason: p.LockReason})
			continue
		}

		if err := git.RemoveWorktree(p.RepoPath, p.Path, opts.Force); err != nil {
			if single {
				return err
			}
			failures = append(failures, fmt.Sprintf("  %s: %v", p.RepoName, err))
			continue
		}

		removed++
		fmt.Fprintf(stdout, "Removed worktree for branch '%s' from %s\n", p.Branch, p.RepoName)

		if err := cleanupEmptyWorktreeDirs(p.RepoName, p.Path); err != nil {
			fmt.Fprintf(stdout, "  (could not tidy empty directories: %v)\n", err)
		}
	}

	if removed > 0 {
		refreshWorktreeData(cfg, repos)
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}
	}

	if single {
		return nil
	}

	fmt.Fprintf(stdout, "\nRemoved %d %s", removed, pluralize(removed, "worktree", "worktrees"))
	if skipped > 0 {
		fmt.Fprintf(stdout, ", skipped %d", skipped)
	}
	if len(failures) > 0 {
		fmt.Fprintf(stdout, ", %d failed", len(failures))
	}
	fmt.Fprintln(stdout)

	if len(failures) > 0 {
		fmt.Fprintf(stdout, "\n%d %s:\n%s\n",
			len(failures), pluralize(len(failures), "error", "errors"),
			strings.Join(failures, "\n"))
		return errPartialFailure
	}

	return nil
}

// refreshWorktreeData re-discovers worktrees for the affected repositories so
// the single save at the end of the run records accurate state.
func refreshWorktreeData(cfg *config.Config, repos []*config.Repository) {
	for _, repo := range repos {
		entries, err := git.ListWorktrees(repo.Path)
		if err != nil {
			continue
		}
		wts := make([]config.Worktree, len(entries))
		for i, e := range entries {
			wts[i] = config.Worktree{
				Path:    e.Path,
				Branch:  e.Branch,
				Aligned: git.IsAligned(e.Path, repo.Name),
			}
		}
		repo.Worktrees = wts
	}
	_ = cfg
}

// cleanupEmptyWorktreeDirs removes directories left empty by a removal, walking
// upward from the removed worktree's parent. It stops at the repository's
// projects directory and never removes the projects root, following the
// conservative shape of removeEmptyLegacyDir: remove the husk, never discard
// something the user still has.
func cleanupEmptyWorktreeDirs(repoName, removedPath string) error {
	repoDir, err := xdg.RepoProjectsDir(repoName)
	if err != nil {
		return err
	}
	repoDir = git.ResolvePath(repoDir)

	dir := filepath.Dir(git.ResolvePath(removedPath))
	for {
		// Stay inside this repository's projects directory. A bare string
		// prefix is not containment: projects/svc-a-old has the prefix
		// projects/svc-a but belongs to another repository. Require the path
		// to be the directory itself or to continue past a separator, the same
		// test git.IsAligned uses.
		if dir != repoDir && !strings.HasPrefix(dir, repoDir+string(filepath.Separator)) {
			return nil
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return nil
		}
		if err := os.Remove(dir); err != nil {
			return nil
		}
		if dir == repoDir {
			return nil
		}
		dir = filepath.Dir(dir)
	}
}
