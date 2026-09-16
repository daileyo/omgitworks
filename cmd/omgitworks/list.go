package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/filter"
	"github.com/daileyo/omgitworks/internal/git"
)

// showColumnSentinel is the NoOptDefVal used to distinguish "flag present
// without a value" (show column only) from "flag not present at all".
const showColumnSentinel = "\x00show"

// Flag variables scoped to the list subcommand.
//
// Lowercase flags (filter only): set a filter value without displaying the column.
// Uppercase flags (show + optional filter): display the column; with NoOptDefVal
// so bare usage shows the column, and a value both shows and filters.
var (
	// Lowercase filter-only flags
	flagType       string
	flagVisibility string
	flagTag        string
	flagPath       string
	flagStatus     string
	flagUser       string
	flagRemote     string
	flagRemoteRaw  string
	flagName       string

	// Uppercase show (+filter) flags
	flagShowType       string
	flagShowVisibility string
	flagShowTag        string
	flagShowPath       string
	flagShowStatus     string
	flagShowUser       string
	flagShowRemote     string
	flagShowRemoteRaw  string

	// Output/display flags
	outputFormat string
	verboseCount int
	flagWorkers  int
	flagColor    string
)

// ANSI color codes
const (
	ansiReset   = "\033[0m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiOrange  = "\033[38;5;208m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
)

// maxBranchDisplayLen is the maximum display length for branch names before truncation.
const maxBranchDisplayLen = 30

// ansiPattern matches ANSI escape sequences for stripping from display width calculations.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// colorize wraps text with an ANSI color code and reset sequence.
func colorize(text, code string) string {
	return code + text + ansiReset
}

// stripANSI removes all ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// displayWidth returns the visible character count of a string,
// ignoring ANSI escape codes and counting Unicode runes.
func displayWidth(s string) int {
	return utf8.RuneCountInString(stripANSI(s))
}

// truncateBranch truncates a branch name to maxLen characters,
// appending "..." if truncation occurs.
func truncateBranch(name string, maxLen int) string {
	if utf8.RuneCountInString(name) <= maxLen {
		return name
	}
	runes := []rune(name)
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// resolveColorEnabled determines if color output should be enabled
// based on the --color flag value and TTY detection.
func resolveColorEnabled(_ *cobra.Command) bool {
	switch flagColor {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}

// listCmd is the Cobra subcommand for listing repositories.
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked repositories",
	Long: `List all repositories tracked by gws with optional filtering.

By default, only repository names are shown in a compact multi-column layout.
Use flags to filter results and control which columns are displayed.

Flag convention:
  LOWERCASE (-t, -y, -s, etc.) = filter only (no column displayed)
  UPPERCASE (-T, -Y, -S, etc.) = show column (bare) or show + filter (with value)

Examples:
  gws list                          # Multi-column repo names
  gws list -s                       # Compact status icons in name column
  gws list -s dirty                 # Filter dirty repos with compact status
  gws list -v                       # Table with type, visibility, tags, path
  gws list -vv                      # Table with all columns
  gws list -YTSP                    # Show type, tags, status, path columns
  gws list -t ai                    # Filter by tag "ai" (no tag column shown)
  gws list -T ai                    # Filter by tag "ai" AND show tag column
  gws list -T                       # Show tag column (no filter)
  gws list -y github -T             # Filter by type, show tags column
  gws list -n "api-*"               # Filter by name pattern
  gws list -S                       # Full status column (branch + icons)
  gws list -SU                      # Show status and user columns
  gws list -R                       # Show formatted remote URL column
  gws list -B                       # Show raw remote URL column
  gws list -o json                  # Output as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle trailing arg: when an uppercase NoOptDefVal flag gets the
		// sentinel but a value follows as a positional arg, reassign it.
		// e.g., "-T ai" parses as T=sentinel + args=["ai"]
		if len(args) == 1 {
			reassigned := reassignTrailingArg(cmd, args[0])
			if !reassigned {
				return fmt.Errorf("unexpected argument: %s", args[0])
			}
		}
		opts := parseDualPurposeFlags(cmd)
		opts.ColorEnabled = resolveColorEnabled(cmd)
		return runList(opts)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// --- Lowercase flags: filter only (no column display) ---
	listCmd.Flags().StringVarP(&flagType, "type", "y", "", "Filter by type")
	listCmd.Flags().StringVarP(&flagVisibility, "visibility", "i", "", "Filter by visibility")
	listCmd.Flags().StringVarP(&flagTag, "tag", "t", "", "Filter by tag (single value)")
	listCmd.Flags().StringVarP(&flagPath, "path", "p", "", "Filter by path pattern")
	listCmd.Flags().StringVarP(&flagStatus, "status", "s", "", "Show compact status in name column, or show and filter by status pattern")
	listCmd.Flags().Lookup("status").NoOptDefVal = showColumnSentinel
	listCmd.Flags().StringVarP(&flagUser, "show-user", "u", "", "Filter by user name")
	listCmd.Flags().StringVarP(&flagRemote, "remote", "r", "", "Filter by remote URL pattern")
	listCmd.Flags().StringVarP(&flagRemoteRaw, "remote-raw", "b", "", "Filter by raw remote URL pattern")
	listCmd.Flags().StringVarP(&flagName, "name", "n", "", "Filter by repository name (partial match, supports wildcards)")

	// --- Uppercase flags: show column (+ optional filter) ---
	// NoOptDefVal allows bare usage (e.g., -T) to show column without value.
	// With a value (e.g., -T ai), shows column AND filters.
	listCmd.Flags().StringVarP(&flagShowType, "show-type", "Y", "", "Show type column, or show and filter by type")
	listCmd.Flags().Lookup("show-type").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowVisibility, "show-visibility", "I", "", "Show visibility column, or show and filter by visibility")
	listCmd.Flags().Lookup("show-visibility").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowTag, "show-tag", "T", "", "Show tags column, or show and filter by tag")
	listCmd.Flags().Lookup("show-tag").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowPath, "show-path", "P", "", "Show path column, or show and filter by path")
	listCmd.Flags().Lookup("show-path").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowStatus, "show-status", "S", "", "Show status column, or show and filter by status")
	listCmd.Flags().Lookup("show-status").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowUser, "show-user-col", "U", "", "Show user column, or show and filter by user")
	listCmd.Flags().Lookup("show-user-col").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowRemote, "show-remote", "R", "", "Show remote column, or show and filter by remote")
	listCmd.Flags().Lookup("show-remote").NoOptDefVal = showColumnSentinel

	listCmd.Flags().StringVarP(&flagShowRemoteRaw, "show-remote-raw", "B", "", "Show raw remote column, or show and filter by raw remote")
	listCmd.Flags().Lookup("show-remote-raw").NoOptDefVal = showColumnSentinel

	// --- Output/display flags ---
	listCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json")
	listCmd.Flags().CountVarP(&verboseCount, "verbose", "v", "Verbose output (-v stored data, -vv all columns)")
	listCmd.Flags().IntVar(&flagWorkers, "workers", 0, "Number of concurrent workers for status fetching (default: 8)")
	listCmd.Flags().StringVar(&flagColor, "color", "auto", "Color output: auto, always, never")

	// Custom help function that hides the showColumnSentinel NoOptDefVals.
	listCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStdout(), cmd.Long+"\n\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Usage:\n  %s\n\nFlags:\n%s", cmd.UseLine(), flagUsages(cmd.Flags()))
	})
}

// ListOptions holds all parameters for the list operation.
type ListOptions struct {
	// Filter values (empty = no filter)
	FilterType       string
	FilterVisibility string
	FilterTags       []string
	FilterName       string
	FilterPath       string
	FilterStatus     string
	FilterUser       string
	FilterRemote     string
	FilterRemoteRaw  string

	// Column display toggles
	ShowType       bool
	ShowVisibility bool
	ShowTags       bool
	ShowPath       bool
	ShowStatus     bool
	ShowUser       bool
	ShowRemote     bool
	ShowRemoteRaw  bool

	// Output
	OutputFormat string
	VerboseLevel int

	// Concurrency
	Workers int

	// Compact status display (icons in NAME column)
	CompactStatus bool

	// Color output
	ColorEnabled bool
}

// AnyColumnSelected returns true if any column display flag is set.
func (o ListOptions) AnyColumnSelected() bool {
	return o.ShowType || o.ShowVisibility || o.ShowTags || o.ShowPath ||
		o.ShowStatus || o.ShowUser || o.ShowRemote || o.ShowRemoteRaw
}

// shortToShowFlag maps uppercase short flag letters to their flag variables.
var shortToShowFlag = map[byte]*string{
	'Y': &flagShowType,
	'I': &flagShowVisibility,
	'T': &flagShowTag,
	'P': &flagShowPath,
	'S': &flagShowStatus,
	'U': &flagShowUser,
	'R': &flagShowRemote,
	'B': &flagShowRemoteRaw,
	's': &flagStatus,
}

// reassignTrailingArg reassigns a trailing positional argument to the last
// uppercase NoOptDefVal flag that received the sentinel value. This allows
// "-T ai" to work as show+filter (Cobra would otherwise treat "ai" as a
// positional arg since -T has NoOptDefVal).
//
// For stacked flags like "-YT ai", the LAST letter in the stack (T) gets
// the value, following POSIX conventions.
//
// Returns true if the arg was successfully reassigned.
func reassignTrailingArg(_ *cobra.Command, value string) bool {
	// Scan os.Args to find the last uppercase short flag letter that
	// was used. This correctly handles stacking order.
	var lastFlagVar *string
	for _, arg := range os.Args {
		if len(arg) < 2 || arg[0] != '-' || arg[1] == '-' {
			continue // skip non-short-flag args and long flags
		}
		// Walk letters in this short flag group (e.g., "-YT" → 'Y', 'T')
		for i := 1; i < len(arg); i++ {
			if fv, ok := shortToShowFlag[arg[i]]; ok {
				lastFlagVar = fv
			}
		}
	}
	if lastFlagVar == nil || *lastFlagVar != showColumnSentinel {
		return false
	}
	*lastFlagVar = value
	return true
}

// parseDualPurposeFlags reads cobra flag state and builds ListOptions.
//
// For each column, two flags are checked:
//   - Lowercase (e.g., -t): filter only, does not show column
//   - Uppercase (e.g., -T): shows column; bare = show only, with value = show + filter
//
// If both lowercase and uppercase are set for the same column, the column is
// shown (from uppercase) and the filter is applied (preferring lowercase value).
func parseDualPurposeFlags(cmd *cobra.Command) ListOptions {
	// Handle deprecated list-level flags before parsing
	handleDeprecatedListFlags(cmd)

	opts := ListOptions{
		OutputFormat: outputFormat,
		VerboseLevel: verboseCount,
	}

	// parsePair merges a lowercase filter flag and an uppercase show flag.
	parsePair := func(filterFlag, showFlag string, filterVal, showVal string) (filter string, show bool) {
		// Uppercase flag: controls column visibility
		if cmd.Flags().Changed(showFlag) {
			show = true
			if showVal != showColumnSentinel && showVal != "" {
				filter = showVal
			}
		}
		// Lowercase flag: filter only (overrides uppercase filter if both set)
		if cmd.Flags().Changed(filterFlag) {
			if filterVal != "" {
				filter = filterVal
			}
		}
		return filter, show
	}

	opts.FilterType, opts.ShowType = parsePair("type", "show-type", flagType, flagShowType)
	opts.FilterVisibility, opts.ShowVisibility = parsePair("visibility", "show-visibility", flagVisibility, flagShowVisibility)
	opts.FilterPath, opts.ShowPath = parsePair("path", "show-path", flagPath, flagShowPath)
	opts.FilterStatus, opts.ShowStatus = parsePair("status", "show-status", flagStatus, flagShowStatus)

	// Compact status: -s triggers compact display (icons in NAME column) unless -S overrides
	if cmd.Flags().Changed("status") && !opts.ShowStatus {
		opts.CompactStatus = true
	}
	// Clean sentinel from filter value (bare -s sets sentinel, not a filter)
	if opts.FilterStatus == showColumnSentinel {
		opts.FilterStatus = ""
	}

	opts.FilterUser, opts.ShowUser = parsePair("show-user", "show-user-col", flagUser, flagShowUser)
	opts.FilterRemote, opts.ShowRemote = parsePair("remote", "show-remote", flagRemote, flagShowRemote)
	opts.FilterRemoteRaw, opts.ShowRemoteRaw = parsePair("remote-raw", "show-remote-raw", flagRemoteRaw, flagShowRemoteRaw)

	// Tag: supports both lowercase (-t) and uppercase (-T).
	// Collect filter values from both; uppercase also enables column display.
	if cmd.Flags().Changed("show-tag") {
		opts.ShowTags = true
		if flagShowTag != showColumnSentinel && flagShowTag != "" {
			opts.FilterTags = append(opts.FilterTags, flagShowTag)
		}
	}
	if cmd.Flags().Changed("tag") {
		if flagTag != "" {
			opts.FilterTags = append(opts.FilterTags, flagTag)
		}
	}

	// Name is filter-only (always displayed)
	opts.FilterName = flagName

	// Apply verbose overrides
	if opts.VerboseLevel >= 1 {
		opts.ShowType = true
		opts.ShowVisibility = true
		opts.ShowTags = true
		opts.ShowPath = true
	}
	if opts.VerboseLevel >= 2 {
		opts.ShowStatus = true
		opts.ShowUser = true
		opts.ShowRemote = true
	}

	// Workers flag
	opts.Workers = flagWorkers

	return opts
}

// resolveWorkers returns the effective worker count from flag, config, or default.
func resolveWorkers(flagVal int, cfg *config.Config) int {
	if flagVal > 0 {
		return flagVal
	}
	if cfg.Preferences != nil && cfg.Preferences.StatusWorkers > 0 {
		return cfg.Preferences.StatusWorkers
	}
	return 8
}

// runList handles the list logic with explicit options.
func runList(opts ListOptions) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Repositories) == 0 {
		if opts.OutputFormat == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("No repositories found. Run 'gws init' to discover repositories.")
		}
		return nil
	}

	// Build filter criteria
	criteria := filter.Criteria{
		Type:       opts.FilterType,
		Visibility: opts.FilterVisibility,
		Tags:       opts.FilterTags,
		Name:       opts.FilterName,
		Path:       opts.FilterPath,
	}

	// Apply filters
	filtered := filter.Apply(cfg.Repositories, criteria)

	if len(filtered) == 0 {
		if opts.OutputFormat == "json" {
			fmt.Println("[]")
		} else {
			fmt.Println("No repositories match the specified filters.")
		}
		return nil
	}

	// Load git status cache and pre-fetch all statuses in parallel
	var statusCache *git.Cache
	var statusMap map[string]*git.Status
	var repoPaths []string
	var workers int
	var cachePath string
	needsStatus := opts.ShowStatus || opts.CompactStatus
	if needsStatus {
		statusCache = git.NewCache(git.DefaultTTL)
		if cp, err := git.GetCachePath(); err == nil {
			cachePath = cp
			_ = statusCache.Load(cachePath)
		}

		// Collect repo paths
		workers = resolveWorkers(opts.Workers, cfg)
		repoPaths = make([]string, len(filtered))
		for i, repo := range filtered {
			repoPaths[i] = repo.Path
		}

		// Two-phase approach:
		// Phase 1: Build display map from fresh cache + stale fallback
		statusMap = make(map[string]*git.Status, len(repoPaths))
		needsFetch := false
		for _, p := range repoPaths {
			if s := statusCache.Get(p); s != nil {
				statusMap[p] = s
			} else if s := statusCache.GetStale(p); s != nil {
				statusMap[p] = s
				needsFetch = true
			} else {
				needsFetch = true
			}
		}

		// If any repos need fresh data and we have no stale data for them,
		// we must fetch now to have something to display
		var uncached []string
		for _, p := range repoPaths {
			if statusMap[p] == nil {
				uncached = append(uncached, p)
			}
		}
		if len(uncached) > 0 {
			// Show progress spinner for foreground fetching
			progress := git.NewProgress(len(uncached))
			progress.Start()
			freshResults := statusCache.FetchAll(uncached, workers, progress)
			progress.Stop()
			for p, s := range freshResults {
				statusMap[p] = s
			}
		}

		// Save cache after foreground fetching
		if cachePath != "" {
			_ = statusCache.Save(cachePath)
		}

		// Phase 2: Background prefetch for stale entries
		if needsFetch {
			done := statusCache.PrefetchInBackground(repoPaths, workers, cachePath)
			defer func() { <-done }() // wait for background prefetch before exit
		}
	}

	// Apply status filter (post-fetch since status data isn't available during initial filtering)
	if opts.FilterStatus != "" && statusMap != nil {
		var statusFiltered []config.Repository
		for _, repo := range filtered {
			text := statusFilterText(statusMap[repo.Path])
			if filter.MatchesPattern(text, opts.FilterStatus) {
				statusFiltered = append(statusFiltered, repo)
			}
		}
		filtered = statusFiltered

		if len(filtered) == 0 {
			if opts.OutputFormat == "json" {
				fmt.Println("[]")
			} else {
				fmt.Println("No repositories match the specified filters.")
			}
			return nil
		}
	}

	// Display results based on format
	switch opts.OutputFormat {
	case "json":
		return displayJSON(filtered, statusMap, opts)
	default:
		displayTable(filtered, statusMap, opts)
	}

	return nil
}

// repoDisplayName returns the repo name with a colored (wt) suffix if it has worktrees.
func repoDisplayName(repo config.Repository, colorEnabled ...bool) string {
	if len(repo.Worktrees) > 0 {
		if len(colorEnabled) > 0 && colorEnabled[0] {
			return repo.Name + " " + ansiOrange + "(wt)" + ansiReset
		}
		return repo.Name + " (wt)"
	}
	return repo.Name
}

// displayTable shows repositories in a formatted table with selective columns.
func displayTable(repos []config.Repository, statusMap map[string]*git.Status, opts ListOptions) {
	fmt.Printf("Found %d %s:\n\n", len(repos), pluralize(len(repos), "repository", "repositories"))

	// If no columns selected and no verbose, show compact multi-column names
	if !opts.AnyColumnSelected() && opts.VerboseLevel == 0 {
		if opts.CompactStatus && statusMap != nil {
			// Build "name + right-justified icons" entries for multi-column display
			displayMultiColumnWithStatus(repos, statusMap, opts.ColorEnabled)
		} else {
			names := make([]string, len(repos))
			for i, repo := range repos {
				names[i] = repoDisplayName(repo, opts.ColorEnabled)
			}
			displayMultiColumn(names)
		}
		return
	}

	// Calculate column widths
	maxNameLen := 4   // "NAME"
	maxTypeLen := 4   // "TYPE"
	maxVisLen := 10   // "VISIBILITY"
	maxTagsLen := 4   // "TAGS"
	maxBranchLen := 0 // branch sub-column within STATUS
	maxIconsLen := 0  // icons sub-column within STATUS
	maxUserLen := 4   // "USER"
	maxEmailLen := 5  // "EMAIL"
	maxRemoteLen := 6 // "REMOTE"
	maxPathLen := 4   // "PATH"

	// Pre-compute user info and drift status for width calculation and display
	type repoUserInfo struct {
		userDisplay string
		email       string
		signStr     string
		hasDrift    bool
	}
	userInfoMap := make(map[string]repoUserInfo)

	// Pre-compute remote display strings
	// By default, format URLs to clean HTTPS. With ShowRemoteRaw, use raw URLs.
	remoteDisplayMap := make(map[string]string)
	if opts.ShowRemote {
		useRaw := opts.ShowRemoteRaw
		formatURL := func(rawURL string) string {
			if useRaw {
				return rawURL
			}
			return git.FormatRemoteURL(rawURL)
		}
		for _, repo := range repos {
			remoteInfo, err := git.GetRemoteInfo(repo.Path)
			if err != nil {
				// Graceful fallback: use stored URL without asterisk
				remoteDisplayMap[repo.Path] = formatURL(repo.RemoteURL)
				continue
			}
			switch {
			case remoteInfo.OriginURL != "" && remoteInfo.HasMultiple:
				remoteDisplayMap[repo.Path] = "* " + formatURL(remoteInfo.OriginURL)
			case remoteInfo.OriginURL != "":
				remoteDisplayMap[repo.Path] = formatURL(remoteInfo.OriginURL)
			case remoteInfo.HasMultiple:
				remoteDisplayMap[repo.Path] = "*"
			default:
				remoteDisplayMap[repo.Path] = formatURL(repo.RemoteURL)
			}
		}
	}

	// Get global default user for marking repos using the default identity
	var globalDefaultUser *git.GlobalUserConfig
	if opts.ShowUser {
		globalDefaultUser, _ = git.GetGlobalDefaultUser()
	}

	for _, repo := range repos {
		if n := len(repoDisplayName(repo)); n > maxNameLen {
			maxNameLen = n
		}
		if opts.ShowType {
			if len(repo.Type) > maxTypeLen {
				maxTypeLen = len(repo.Type)
			}
		}
		if opts.ShowVisibility {
			if len(repo.Visibility) > maxVisLen {
				maxVisLen = len(repo.Visibility)
			}
		}
		if opts.ShowTags {
			tags := strings.Join(repo.Tags, ", ")
			if len(tags) > maxTagsLen {
				maxTagsLen = len(tags)
			}
		}

		// Calculate path and remote column widths
		if opts.ShowPath || opts.ShowRemote {
			if len(repo.Path) > maxPathLen {
				maxPathLen = len(repo.Path)
			}
		}
		if opts.ShowRemote {
			if rd := remoteDisplayMap[repo.Path]; len(rd) > maxRemoteLen {
				maxRemoteLen = len(rd)
			}
		}

		// Calculate status sub-column widths (full STATUS column only)
		if opts.ShowStatus && statusMap != nil {
			if status := statusMap[repo.Path]; status != nil {
				branchStr := formatStatusBranch(status)
				iconsStr := formatStatusIcons(status, false)
				if bw := displayWidth(branchStr); bw > maxBranchLen {
					maxBranchLen = bw
				}
				if iw := displayWidth(iconsStr); iw > maxIconsLen {
					maxIconsLen = iw
				}
			}
		}

		// Calculate user column widths and pre-compute display values
		if opts.ShowUser {
			info := repoUserInfo{}

			// Start with stored config values
			displayUser := repo.User
			displayEmail := repo.Email
			displaySign := repo.SigningEnabled
			displaySource := repo.UserSource

			// Read effective config at display time to detect local overrides
			currentCfg, err := git.GetUserConfig(repo.Path)
			if err == nil && currentCfg != nil {
				if currentCfg.Source == config.UserSourceLocal {
					// Local config detected - show effective local values
					displayUser = currentCfg.Name
					displayEmail = currentCfg.Email
					displaySign = currentCfg.SignCommits
					displaySource = config.UserSourceLocal
				} else if (repo.User != "" && repo.User != currentCfg.Name) ||
					(repo.Email != "" && repo.Email != currentCfg.Email) {
					info.hasDrift = true
				}
			}

			// Format user display with source marker
			info.userDisplay = displayUser
			if info.userDisplay == "" {
				info.userDisplay = "-"
			} else if displaySource == config.UserSourceLocal {
				info.userDisplay += " (local)"
			} else if displaySource == config.UserSourceGlobal && globalDefaultUser != nil &&
				globalDefaultUser.Name != "" && displayUser == globalDefaultUser.Name {
				info.userDisplay += " *"
			}

			info.email = displayEmail
			if info.email == "" {
				info.email = "-"
			}

			if displaySign {
				info.signStr = "✓"
			} else {
				info.signStr = ""
			}

			userInfoMap[repo.Path] = info

			if len(info.userDisplay) > maxUserLen {
				maxUserLen = len(info.userDisplay)
			}
			if len(info.email) > maxEmailLen {
				maxEmailLen = len(info.email)
			}
		}
	}

	// Expand NAME column for compact status icons (right-justified in NAME field)
	maxCompactIconsLen := 0
	if opts.CompactStatus && statusMap != nil {
		for _, repo := range repos {
			if status := statusMap[repo.Path]; status != nil {
				icons := formatStatusIcons(status, false)
				if iw := displayWidth(icons); iw > maxCompactIconsLen {
					maxCompactIconsLen = iw
				}
			}
		}
		if maxCompactIconsLen > 0 {
			maxNameLen += 2 + maxCompactIconsLen // 2 = minimum padding
		}
	}

	// Build and print header
	var headerParts []string
	var separatorParts []string

	headerParts = append(headerParts, fmt.Sprintf("%-*s", maxNameLen, "NAME"))
	separatorParts = append(separatorParts, strings.Repeat("-", maxNameLen))

	if opts.ShowStatus && statusMap != nil {
		// STATUS column width = branch sub-column + space + icons sub-column
		// Ensure minimum width fits the "STATUS" header (6 chars)
		statusColWidth := maxBranchLen
		if maxIconsLen > 0 {
			statusColWidth += 1 + maxIconsLen // 1 for space separator
		}
		if statusColWidth < 6 {
			statusColWidth = 6
		}
		headerParts = append(headerParts, fmt.Sprintf("%-*s", statusColWidth, "STATUS"))
		separatorParts = append(separatorParts, strings.Repeat("-", statusColWidth))
	}

	if opts.ShowUser {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxUserLen, "USER"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxUserLen))
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxEmailLen, "EMAIL"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxEmailLen))
		headerParts = append(headerParts, fmt.Sprintf("%-4s", "SIGN"))
		separatorParts = append(separatorParts, strings.Repeat("-", 4))
	}

	if opts.ShowType {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxTypeLen, "TYPE"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxTypeLen))
	}

	if opts.ShowVisibility {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxVisLen, "VISIBILITY"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxVisLen))
	}

	if opts.ShowTags {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxTagsLen, "TAGS"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxTagsLen))
	}

	if opts.ShowPath && opts.ShowRemote {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxPathLen, "PATH"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxPathLen))
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxRemoteLen, "REMOTE"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxRemoteLen))
	} else if opts.ShowPath {
		headerParts = append(headerParts, "PATH")
		separatorParts = append(separatorParts, strings.Repeat("-", 4))
	} else if opts.ShowRemote {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", maxRemoteLen, "REMOTE"))
		separatorParts = append(separatorParts, strings.Repeat("-", maxRemoteLen))
	}

	fmt.Println(strings.Join(headerParts, "  "))
	fmt.Println(strings.Join(separatorParts, "  "))

	// Print repositories
	for _, repo := range repos {
		var rowParts []string

		// Name column - add worktree and drift indicators
		nameDisplay := repoDisplayName(repo, opts.ColorEnabled)
		if opts.ShowUser {
			if info, ok := userInfoMap[repo.Path]; ok && info.hasDrift {
				nameDisplay += " ⚠"
			}
		}

		// Compact status: right-justify icons within the NAME field
		nameVisWidth := displayWidth(nameDisplay)
		if opts.CompactStatus && statusMap != nil {
			status := statusMap[repo.Path]
			if status != nil && status.Branch != "" {
				icons := formatStatusIcons(status, opts.ColorEnabled)
				iconsVisWidth := displayWidth(icons)
				padding := maxNameLen - nameVisWidth - iconsVisWidth
				if padding < 2 {
					padding = 2
				}
				rowParts = append(rowParts, nameDisplay+strings.Repeat(" ", padding)+icons)
			} else {
				namePad := maxNameLen - nameVisWidth
				if namePad < 0 {
					namePad = 0
				}
				rowParts = append(rowParts, nameDisplay+strings.Repeat(" ", namePad))
			}
		} else {
			namePad := maxNameLen - nameVisWidth
			if namePad < 0 {
				namePad = 0
			}
			rowParts = append(rowParts, nameDisplay+strings.Repeat(" ", namePad))
		}

		// Status column (optional) - branch left-justified, icons right-justified
		if opts.ShowStatus && statusMap != nil {
			statusColWidth := maxBranchLen
			if maxIconsLen > 0 {
				statusColWidth += 1 + maxIconsLen
			}
			if statusColWidth < 6 {
				statusColWidth = 6
			}

			status := statusMap[repo.Path]
			if status == nil || status.Branch == "" {
				// No status or "no commits" — span full column left-justified
				display := "-"
				if status != nil {
					display = formatStatusBranch(status)
				}
				rowParts = append(rowParts, fmt.Sprintf("%-*s", statusColWidth, display))
			} else {
				branch := formatStatusBranch(status)
				icons := formatStatusIcons(status, opts.ColorEnabled)
				iconsVisWidth := displayWidth(icons)
				// Left-justify branch, right-justify icons
				// Padding = total width - branch width - icons visible width
				padding := statusColWidth - displayWidth(branch) - iconsVisWidth
				if padding < 1 {
					padding = 1
				}
				rowParts = append(rowParts, branch+strings.Repeat(" ", padding)+icons)
			}
		}

		// User columns (optional)
		if opts.ShowUser {
			info := userInfoMap[repo.Path]
			rowParts = append(rowParts, fmt.Sprintf("%-*s", maxUserLen, info.userDisplay))
			rowParts = append(rowParts, fmt.Sprintf("%-*s", maxEmailLen, info.email))
			rowParts = append(rowParts, fmt.Sprintf("%-4s", info.signStr))
		}

		// Type column (optional)
		if opts.ShowType {
			typeStr := string(repo.Type)
			if typeStr == "" {
				typeStr = "unknown"
			}
			rowParts = append(rowParts, fmt.Sprintf("%-*s", maxTypeLen, typeStr))
		}

		// Visibility column (optional)
		if opts.ShowVisibility {
			visStr := string(repo.Visibility)
			if visStr == "" {
				visStr = "unknown"
			}
			rowParts = append(rowParts, fmt.Sprintf("%-*s", maxVisLen, visStr))
		}

		// Tags column (optional)
		if opts.ShowTags {
			tags := strings.Join(repo.Tags, ", ")
			if tags == "" {
				tags = "-"
			}
			rowParts = append(rowParts, fmt.Sprintf("%-*s", maxTagsLen, tags))
		}

		// Path column (optional, padded when remote column follows)
		if opts.ShowPath && opts.ShowRemote {
			rowParts = append(rowParts, fmt.Sprintf("%-*s", maxPathLen, repo.Path))
			rowParts = append(rowParts, remoteDisplayMap[repo.Path])
		} else if opts.ShowPath {
			rowParts = append(rowParts, repo.Path)
		} else if opts.ShowRemote {
			rowParts = append(rowParts, remoteDisplayMap[repo.Path])
		}

		fmt.Println(strings.Join(rowParts, "  "))
	}

}

// getTerminalWidth returns the terminal width, or 80 if detection fails.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

// displayMultiColumn prints names in a compact multi-column layout like ls.
// Names are sorted alphabetically and fill top-to-bottom, then left-to-right
// (column-first order). Falls back to single-column when stdout is not a TTY.
func displayMultiColumn(names []string) {
	if len(names) == 0 {
		return
	}

	sort.Strings(names)

	// Non-TTY: one name per line
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		for _, name := range names {
			fmt.Println(name)
		}
		return
	}

	termWidth := getTerminalWidth()
	colGap := 2

	// Find the widest visible name (ignoring ANSI escape codes)
	maxLen := 0
	for _, name := range names {
		if w := displayWidth(name); w > maxLen {
			maxLen = w
		}
	}

	// Calculate number of columns that fit
	colWidth := maxLen + colGap
	numCols := termWidth / colWidth
	if numCols < 1 {
		numCols = 1
	}
	// Cap columns so we get at least 3 rows for readability,
	// unless there are very few items (≤3 always single column).
	if len(names) > 3 {
		maxCols := len(names) / 3
		if maxCols < 1 {
			maxCols = 1
		}
		if numCols > maxCols {
			numCols = maxCols
		}
	}

	// Calculate rows needed (column-first layout)
	numRows := (len(names) + numCols - 1) / numCols

	// Print in column-first order: row by row, picking items that belong
	// to each column. Item at column c, row r = names[c*numRows + r].
	for r := 0; r < numRows; r++ {
		for c := 0; c < numCols; c++ {
			idx := c*numRows + r
			if idx >= len(names) {
				break
			}
			// Last column in this row or last item: no trailing padding
			isLastCol := c == numCols-1 || (c+1)*numRows+r >= len(names)
			if isLastCol {
				fmt.Print(names[idx])
			} else {
				// Pad based on visible width to handle ANSI codes
				pad := colWidth - displayWidth(names[idx])
				if pad < 0 {
					pad = 0
				}
				fmt.Print(names[idx] + strings.Repeat(" ", pad))
			}
		}
		fmt.Println()
	}
}

// displayMultiColumnWithStatus prints repos in a multi-column layout with
// compact status icons right-justified within each entry. Each entry is
// formatted as "name <padding> icons" with a fixed entry width.
func displayMultiColumnWithStatus(repos []config.Repository, statusMap map[string]*git.Status, colorEnabled bool) {
	if len(repos) == 0 {
		return
	}

	// Compute the max name length and max icons visual width
	maxNameLen := 0
	maxIconsVisLen := 0
	for _, repo := range repos {
		if n := len(repoDisplayName(repo)); n > maxNameLen {
			maxNameLen = n
		}
		if status := statusMap[repo.Path]; status != nil {
			icons := formatStatusIcons(status, false)
			if iw := displayWidth(icons); iw > maxIconsVisLen {
				maxIconsVisLen = iw
			}
		}
	}

	// Entry width = name + gap + icons
	entryVisWidth := maxNameLen + 2 + maxIconsVisLen

	// Build display entries: each has a visual representation and fixed visible width
	type entry struct {
		display  string // the formatted string (may contain ANSI codes)
		visWidth int    // visible width (excluding ANSI)
	}
	entries := make([]entry, len(repos))
	for i, repo := range repos {
		name := repoDisplayName(repo, colorEnabled)
		status := statusMap[repo.Path]
		if status != nil && status.Branch != "" {
			icons := formatStatusIcons(status, colorEnabled)
			iconsVisWidth := displayWidth(icons)
			nameVisWidth := displayWidth(name)
			padding := entryVisWidth - nameVisWidth - iconsVisWidth
			if padding < 2 {
				padding = 2
			}
			entries[i] = entry{
				display:  name + strings.Repeat(" ", padding) + icons,
				visWidth: entryVisWidth,
			}
		} else {
			entries[i] = entry{
				display:  name,
				visWidth: displayWidth(name),
			}
		}
	}

	// Sort entries alphabetically by repo name
	sort.Slice(entries, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	// Non-TTY: one entry per line
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		for _, e := range entries {
			fmt.Println(e.display)
		}
		return
	}

	termWidth := getTerminalWidth()
	colGap := 2

	// Column width based on the widest entry
	colWidth := entryVisWidth + colGap
	numCols := termWidth / colWidth
	if numCols < 1 {
		numCols = 1
	}
	// Cap columns for readability (at least 3 rows)
	if len(entries) > 3 {
		maxCols := len(entries) / 3
		if maxCols < 1 {
			maxCols = 1
		}
		if numCols > maxCols {
			numCols = maxCols
		}
	}

	numRows := (len(entries) + numCols - 1) / numCols

	for r := 0; r < numRows; r++ {
		for c := 0; c < numCols; c++ {
			idx := c*numRows + r
			if idx >= len(entries) {
				break
			}
			isLastCol := c == numCols-1 || (c+1)*numRows+r >= len(entries)
			if isLastCol {
				fmt.Print(entries[idx].display)
			} else {
				// Pad based on visible width to handle ANSI codes
				pad := colWidth - entries[idx].visWidth
				if pad < 0 {
					pad = 0
				}
				fmt.Print(entries[idx].display + strings.Repeat(" ", pad))
			}
		}
		fmt.Println()
	}
}

// formatStatusBranch returns the branch name portion of the status,
// truncated if it exceeds maxBranchDisplayLen.
func formatStatusBranch(status *git.Status) string {
	if status.Branch == "" {
		return "no commits"
	}
	return truncateBranch(status.Branch, maxBranchDisplayLen)
}

// formatStatusIcons returns the status icons portion (behind, ahead, clean/dirty).
// When colorEnabled is true, icons are wrapped with ANSI color codes.
func formatStatusIcons(status *git.Status, colorEnabled bool) string {
	if status.Branch == "" {
		return ""
	}

	var parts []string

	// Behind indicator
	if status.Behind > 0 {
		behind := fmt.Sprintf("↓%d", status.Behind)
		if colorEnabled {
			parts = append(parts, colorize(behind, ansiMagenta))
		} else {
			parts = append(parts, behind)
		}
	}

	// Ahead indicator
	if status.Ahead > 0 {
		ahead := fmt.Sprintf("↑%d", status.Ahead)
		if colorEnabled {
			parts = append(parts, colorize(ahead, ansiCyan))
		} else {
			parts = append(parts, ahead)
		}
	}

	// Clean/dirty indicator
	if status.HasChanges {
		if colorEnabled {
			parts = append(parts, colorize("✗", ansiRed))
		} else {
			parts = append(parts, "✗")
		}
	} else {
		if colorEnabled {
			parts = append(parts, colorize("✓", ansiGreen))
		} else {
			parts = append(parts, "✓")
		}
	}

	return strings.Join(parts, " ")
}

// formatStatusShort returns a short status string with visual indicators.
// Used by displayJSON and other callers that need a single combined string without color.
func formatStatusShort(status *git.Status) string {
	branch := formatStatusBranch(status)
	icons := formatStatusIcons(status, false)
	if icons == "" {
		return branch
	}
	return branch + " " + icons
}

// statusFilterText returns searchable text for status filtering.
// Produces terms like "clean", "dirty", "ahead", "behind" for pattern matching.
func statusFilterText(status *git.Status) string {
	if status == nil || status.Branch == "" {
		return ""
	}
	var parts []string
	if status.HasChanges {
		parts = append(parts, "dirty")
	} else {
		parts = append(parts, "clean")
	}
	if status.Ahead > 0 {
		parts = append(parts, "ahead")
	}
	if status.Behind > 0 {
		parts = append(parts, "behind")
	}
	return strings.Join(parts, " ")
}

// displayJSON outputs repositories in JSON format respecting column selection.
// Only fields for displayed columns are included; "name" is always present.
func displayJSON(repos []config.Repository, statusMap map[string]*git.Status, opts ListOptions) error {
	entries := make([]map[string]interface{}, len(repos))

	for i, repo := range repos {
		entry := map[string]interface{}{
			"name": repo.Name,
		}
		if len(repo.Worktrees) > 0 {
			entry["has_worktrees"] = true
		}

		if opts.ShowType {
			t := string(repo.Type)
			if t == "" {
				t = "unknown"
			}
			entry["type"] = t
		}
		if opts.ShowVisibility {
			v := string(repo.Visibility)
			if v == "" {
				v = "unknown"
			}
			entry["visibility"] = v
		}
		if opts.ShowTags {
			if repo.Tags != nil {
				entry["tags"] = repo.Tags
			} else {
				entry["tags"] = []string{}
			}
		}
		if opts.ShowPath {
			entry["path"] = repo.Path
		}
		if opts.ShowStatus && statusMap != nil {
			if status := statusMap[repo.Path]; status != nil {
				entry["status"] = formatStatusShort(status)
			} else {
				entry["status"] = "-"
			}
		}
		if opts.ShowUser {
			entry["user"] = repo.User
			entry["email"] = repo.Email
			entry["signing_enabled"] = repo.SigningEnabled
		}
		if opts.ShowRemote {
			jsonFormatURL := func(rawURL string) string {
				if opts.ShowRemoteRaw {
					return rawURL
				}
				return git.FormatRemoteURL(rawURL)
			}
			remoteInfo, err := git.GetRemoteInfo(repo.Path)
			if err == nil {
				if remoteInfo.OriginURL != "" {
					entry["remote_url"] = jsonFormatURL(remoteInfo.OriginURL)
				} else {
					entry["remote_url"] = jsonFormatURL(repo.RemoteURL)
				}
				entry["has_multiple_remotes"] = remoteInfo.HasMultiple
			} else {
				entry["remote_url"] = jsonFormatURL(repo.RemoteURL)
				entry["has_multiple_remotes"] = false
			}
		}

		entries[i] = entry
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
