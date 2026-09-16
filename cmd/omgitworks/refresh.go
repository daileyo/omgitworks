package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/discovery"
	"github.com/daileyo/omgitworks/internal/git"
)

// refreshCmd is the Cobra subcommand for refreshing repository metadata.
var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh repository metadata and git status cache",
	Long: `Re-scan the workspace for repositories, update metadata, and clear the git status cache.
New repositories are added, removed repositories are cleaned up, and existing tags are preserved.

Examples:
  gws refresh`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRefresh()
	},
}

func init() {
	rootCmd.AddCommand(refreshCmd)
}

// runRefresh handles the refresh logic.
func runRefresh() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Printf("Refreshing workspace at: %s\n", cfg.Workspace)

	// Phase 1: Validate existing repos using their stored real paths.
	validRepos, removedCount, updatedCount := validateExistingRepos(cfg)

	// Phase 2: Scan workspace for entries not yet tracked.
	newRepos, scanErrors := scanWorkspaceForNewRepos(cfg.Workspace, validRepos)
	if len(scanErrors) > 0 {
		fmt.Printf("\nWarning: %d errors occurred during workspace scan:\n", len(scanErrors))
		for i, e := range scanErrors {
			if i < 3 {
				fmt.Printf("  - %v\n", e)
			}
		}
		if len(scanErrors) > 3 {
			fmt.Printf("  ... and %d more errors\n", len(scanErrors)-3)
		}
	}

	allRepos := append(validRepos, newRepos...)

	fmt.Println("Detecting git user configuration...")
	userDetectedCount := detectUserForRepos(allRepos, cfg.Profiles)

	// Phase 3: Discover worktrees for all repos.
	worktreeRepoCount := discoverWorktrees(allRepos)

	cfg.Repositories = allRepos
	syncProfilesFromRepos(cfg)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	cachePath, err := git.GetCachePath()
	if err == nil {
		statusCache := git.NewCache(git.DefaultTTL)
		statusCache.Clear()
		_ = statusCache.Save(cachePath)
		fmt.Println("Cleared git status cache")
	}

	fmt.Printf("\nRefresh complete!\n")
	fmt.Printf("Total repositories: %d\n", len(allRepos))
	if removedCount > 0 {
		fmt.Printf("Removed %d %s (path no longer valid)\n", removedCount, pluralize(removedCount, "repository", "repositories"))
	}
	if len(newRepos) > 0 {
		fmt.Printf("Found %d new %s\n", len(newRepos), pluralize(len(newRepos), "repository", "repositories"))
	}
	if updatedCount > 0 {
		fmt.Printf("Updated %d %s\n", updatedCount, pluralize(updatedCount, "repository", "repositories"))
	}
	if userDetectedCount > 0 {
		fmt.Printf("Repositories with user configuration: %d\n", userDetectedCount)
	}
	warnMissingEmail(os.Stderr, allRepos)
	if worktreeRepoCount > 0 {
		fmt.Printf("Repositories with worktrees: %d\n", worktreeRepoCount)
	}

	return nil
}

// discoverWorktrees populates the Worktrees field on each repo by running
// git worktree list. Returns the number of repos that have worktrees.
func discoverWorktrees(repos []config.Repository) int {
	count := 0
	for i := range repos {
		if err := syncRepoWorktrees(&repos[i]); err != nil {
			continue
		}
		if len(repos[i].Worktrees) > 0 {
			count++
		}
	}
	return count
}

// validateExistingRepos iterates cfg.Repositories and validates each repo by
// checking its stored real path. Returns the set of valid repos, a count of
// repos removed (path gone), and a count whose metadata changed.
func validateExistingRepos(cfg *config.Config) ([]config.Repository, int, int) {
	var validRepos []config.Repository
	removedCount := 0
	updatedCount := 0

	// trackedPaths guards against adding the same real path twice.
	trackedPaths := make(map[string]bool, len(cfg.Repositories))
	for _, repo := range cfg.Repositories {
		trackedPaths[repo.Path] = true
	}

	for _, repo := range cfg.Repositories {
		gitDir := filepath.Join(repo.Path, ".git")
		if _, err := os.Stat(gitDir); err != nil {
			// Repo no longer exists at its stored path.
			removedCount++
			removeWorkspaceSymlink(cfg.Workspace, repo.Name, repo.Path)
			delete(trackedPaths, repo.Path)

			// Scan the parent directory to pick up any repos added at the same level.
			parentDir := filepath.Dir(repo.Path)
			if scanResult, scanErr := discovery.Scan(parentDir); scanErr == nil {
				for _, found := range scanResult.Repositories {
					if !trackedPaths[found.Path] {
						trackedPaths[found.Path] = true
						ensureWorkspaceSymlink(cfg, found.Path, found.Name)
						validRepos = append(validRepos, found)
					}
				}
			}
			continue
		}

		// Re-read metadata so remote URL, type, visibility are current.
		updated, buildErr := buildRepository(repo.Path)
		if buildErr != nil {
			// Can't rebuild — keep existing entry unchanged.
			validRepos = append(validRepos, repo)
			continue
		}

		// Preserve user-set tags.
		updated.Tags = repo.Tags

		if updated.RemoteURL != repo.RemoteURL || updated.Name != repo.Name ||
			updated.Type != repo.Type || updated.Visibility != repo.Visibility {
			updatedCount++
		}

		ensureWorkspaceSymlink(cfg, repo.Path, repo.Name)
		validRepos = append(validRepos, *updated)
	}

	return validRepos, removedCount, updatedCount
}

// removeWorkspaceSymlink removes workspace/name only when it is a symlink
// whose target matches repoPath. Leaves any non-symlink or differently-targeted
// entry untouched.
func removeWorkspaceSymlink(workspace, name, repoPath string) {
	symlinkPath := filepath.Join(workspace, name)
	fi, err := os.Lstat(symlinkPath)
	if err != nil {
		return
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return
	}
	target, err := os.Readlink(symlinkPath)
	if err != nil || target != repoPath {
		return
	}
	_ = os.Remove(symlinkPath)
}

// ensureWorkspaceSymlink creates or corrects workspace/name → repoPath for
// repos that live outside the workspace directory. No-op for repos inside the
// workspace.
func ensureWorkspaceSymlink(cfg *config.Config, repoPath, repoName string) {
	workspacePath := filepath.Clean(cfg.Workspace)
	repoClean := filepath.Clean(repoPath)
	if repoClean == workspacePath || strings.HasPrefix(repoClean, workspacePath+string(filepath.Separator)) {
		return
	}

	symlinkPath := filepath.Join(cfg.Workspace, repoName)
	fi, err := os.Lstat(symlinkPath)
	if err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(symlinkPath)
			if readErr == nil && target == repoPath {
				return // already correct
			}
			// Remove incorrect symlink so we can recreate it.
			_ = os.Remove(symlinkPath)
		} else {
			return // non-symlink; leave it alone
		}
	}

	_ = os.Symlink(repoPath, symlinkPath)
}

// scanWorkspaceForNewRepos reads the workspace directory one level deep,
// resolves symlinks, and returns repos not already present in existingRepos.
func scanWorkspaceForNewRepos(workspacePath string, existingRepos []config.Repository) ([]config.Repository, []error) {
	trackedPaths := make(map[string]bool, len(existingRepos))
	for _, repo := range existingRepos {
		trackedPaths[repo.Path] = true
	}

	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read workspace directory: %w", err)}
	}

	var newRepos []config.Repository
	var errs []error

	for _, entry := range entries {
		entryPath := filepath.Join(workspacePath, entry.Name())

		// Resolve to real path — follows symlinks.
		realPath, err := filepath.EvalSymlinks(entryPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("broken symlink or inaccessible entry %s: %w", entryPath, err))
			continue
		}

		fi, err := os.Stat(realPath)
		if err != nil || !fi.IsDir() {
			continue
		}

		if trackedPaths[realPath] {
			continue
		}

		// Must contain a .git directory to qualify as a repo.
		if _, err := os.Stat(filepath.Join(realPath, ".git")); err != nil {
			continue
		}

		repo, err := buildRepository(realPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read repository at %s: %w", realPath, err))
			continue
		}

		trackedPaths[realPath] = true
		newRepos = append(newRepos, *repo)
	}

	return newRepos, errs
}
