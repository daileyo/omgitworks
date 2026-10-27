package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
)

// Flag variables scoped to worktree list. They are deliberately separate from
// list's flagStatus, flagColor, and flagWorkers so the two commands can never
// clobber each other's state.
var (
	flagWTStatus     string
	flagWTShowStatus string
	flagWTColor      string
)

var worktreeListCmd = &cobra.Command{
	Use:   "list [repo|.]",
	Short: "List worktrees across tracked repositories",
	Long: `List all git worktrees across tracked repositories, optionally filtered to a
single repository.

Each entry shows the repository name, branch, path, and whether the worktree
is aligned (inside the projects root).

Passing "." lists only the repository the current directory belongs to. Omitting
the argument continues to list every tracked repository. The positional
argument is always the repository, so a status filter value must be attached
to its flag: -s=dirty, -sdirty, or --status=dirty.

Git status is opt-in and follows the same flag convention as "gws list":
  -s  (lowercase) = compact status icons after the branch name
  -S  (uppercase) = a separate STATUS column holding the icons
  Either flag with a value also filters: clean, dirty, ahead, behind.
  When both are given, -S wins.

Examples:
  gws worktree list                 # List all worktrees
  gws worktree list my-repo         # List worktrees for my-repo only
  gws worktree list .               # List worktrees for the current repo
  gws worktree list . -s            # Compact status after each branch
  gws worktree list my-repo -S      # Full STATUS column for my-repo
  gws worktree list -s=dirty        # Only worktrees with uncommitted changes
  gws worktree list my-repo -S=ahead  # Only my-repo worktrees ahead of origin
  gws worktree list -s my-repo      # my-repo is the repository, not a filter`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The positional is always the repository argument. It is never
		// reassigned to a status flag the way list does, so a filter value
		// must be attached: -s=dirty, -sdirty, or --status=dirty.
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
		return runWorktreeList(scope, parseWorktreeListFlags(cmd), os.Stdout)
	},
}

func init() {
	worktreeListCmd.Flags().StringVarP(&flagWTStatus, "status", "s", "",
		"Show compact status after the branch, or show and filter with -s=<pattern>")
	worktreeListCmd.Flags().Lookup("status").NoOptDefVal = showColumnSentinel
	worktreeListCmd.Flags().StringVarP(&flagWTShowStatus, "show-status", "S", "",
		"Show a STATUS column, or show and filter with -S=<pattern>")
	worktreeListCmd.Flags().Lookup("show-status").NoOptDefVal = showColumnSentinel
	worktreeListCmd.Flags().StringVar(&flagWTColor, "color", "auto", "Color output: auto, always, never")

	worktreeListCmd.ValidArgsFunction = completeWorktreeRepoOrDot
	worktreeCmd.AddCommand(worktreeListCmd)
}

// worktreeListOptions controls what runWorktreeList fetches and shows.
type worktreeListOptions struct {
	CompactStatus bool   // icons after the branch, inside the BRANCH column
	ShowStatus    bool   // a separate STATUS column holding the icons
	FilterStatus  string // keep only worktrees whose status text matches
	ColorEnabled  bool
	Workers       int
}

// needsStatus reports whether git status has to be fetched at all. Without a
// status flag no git command runs.
func (o worktreeListOptions) needsStatus() bool {
	return o.CompactStatus || o.ShowStatus
}

// parseWorktreeListFlags reads cobra flag state into options.
func parseWorktreeListFlags(cmd *cobra.Command) worktreeListOptions {
	var opts worktreeListOptions
	// Uppercase flag: a separate column. Its value, if any, is the filter.
	if cmd.Flags().Changed("show-status") {
		opts.ShowStatus = true
		if flagWTShowStatus != showColumnSentinel && flagWTShowStatus != "" {
			opts.FilterStatus = flagWTShowStatus
		}
	}
	// Lowercase flag: compact icons unless -S already claimed the display.
	// A non-empty value overrides the uppercase filter, as parsePair does
	// for list.
	if cmd.Flags().Changed("status") {
		opts.CompactStatus = !opts.ShowStatus
		if flagWTStatus != showColumnSentinel && flagWTStatus != "" {
			opts.FilterStatus = flagWTStatus
		}
	}
	opts.ColorEnabled = resolveColorEnabled(flagWTColor)
	return opts
}

// worktreeListEntry holds data for a single row in worktree list output.
type worktreeListEntry struct {
	Repo    string
	Branch  string
	Path    string
	Aligned bool
}

// branchLabel is what the BRANCH column shows for the entry.
func (e worktreeListEntry) branchLabel() string {
	if e.Branch == "" {
		return "(detached)"
	}
	return e.Branch
}

// alignedLabel is what the ALIGNED column shows for the entry.
func (e worktreeListEntry) alignedLabel() string {
	if e.Aligned {
		return "yes"
	}
	return "no"
}

// padRight pads s with spaces to the given visible width, measuring with
// displayWidth so multi-byte icons and ANSI codes do not shift columns.
func padRight(s string, width int) string {
	pad := width - displayWidth(s)
	if pad < 0 {
		pad = 0
	}
	return s + strings.Repeat(" ", pad)
}

func runWorktreeList(scope worktreeScope, opts worktreeListOptions, stdout io.Writer) error {
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
		printNoWorktrees(scope, stdout)
		return nil
	}

	// Fetch status through the pipeline shared with list. A path whose
	// status cannot be read simply has no entry and renders blank.
	var statusMap map[string]*git.Status
	if opts.needsStatus() {
		paths := make([]string, len(entries))
		for i, e := range entries {
			paths[i] = e.Path
		}
		var wait func()
		statusMap, wait = fetchStatuses(paths, resolveWorkers(opts.Workers, cfg))
		defer wait()
	}

	// Calculate column widths on visible text.
	maxRepo := 4   // "REPO"
	maxBranch := 6 // "BRANCH"
	maxPath := 4   // "PATH"
	maxIcons := 0
	for _, e := range entries {
		if w := displayWidth(e.Repo); w > maxRepo {
			maxRepo = w
		}
		if w := displayWidth(e.branchLabel()); w > maxBranch {
			maxBranch = w
		}
		if w := displayWidth(e.Path); w > maxPath {
			maxPath = w
		}
		if s := statusMap[e.Path]; s != nil && s.Branch != "" {
			if w := displayWidth(formatStatusIcons(s, false)); w > maxIcons {
				maxIcons = w
			}
		}
	}
	// -S wins over -s: the icons go in their own column and BRANCH stays
	// plain. Compact mode instead widens BRANCH so the icons sit in a
	// sub-column after the branch name.
	compact := opts.CompactStatus && !opts.ShowStatus
	if compact && maxIcons > 0 {
		maxBranch += 2 + maxIcons
	}
	statusWidth := 6 // "STATUS"
	if maxIcons > statusWidth {
		statusWidth = maxIcons
	}

	// Print header
	header := []string{padRight("REPO", maxRepo), padRight("BRANCH", maxBranch), padRight("PATH", maxPath), "ALIGNED"}
	separator := []string{strings.Repeat("-", maxRepo), strings.Repeat("-", maxBranch), strings.Repeat("-", maxPath), strings.Repeat("-", 7)}
	if opts.ShowStatus {
		header = append(header, "STATUS")
		separator = append(separator, strings.Repeat("-", statusWidth))
	}
	fmt.Fprintln(stdout, strings.Join(header, "  "))
	fmt.Fprintln(stdout, strings.Join(separator, "  "))

	// Print rows
	for _, e := range entries {
		var icons string
		if s := statusMap[e.Path]; s != nil && s.Branch != "" {
			icons = formatStatusIcons(s, opts.ColorEnabled)
		}
		branchCell := e.branchLabel()
		if compact && icons != "" {
			padding := maxBranch - displayWidth(branchCell) - displayWidth(icons)
			if padding < 2 {
				padding = 2
			}
			branchCell += strings.Repeat(" ", padding) + icons
		}
		row := []string{padRight(e.Repo, maxRepo), padRight(branchCell, maxBranch), padRight(e.Path, maxPath), e.alignedLabel()}
		if opts.ShowStatus {
			row[3] = padRight(row[3], 7)
			row = append(row, icons)
		}
		fmt.Fprintln(stdout, strings.TrimRight(strings.Join(row, "  "), " "))
	}

	return nil
}

// printNoWorktrees prints the empty-result message, naming the scope when
// there is one.
func printNoWorktrees(scope worktreeScope, stdout io.Writer) {
	if scope.Label != "" {
		fmt.Fprintf(stdout, "No worktrees found for '%s'\n", scope.Label)
	} else {
		fmt.Fprintln(stdout, "No worktrees found")
	}
}
