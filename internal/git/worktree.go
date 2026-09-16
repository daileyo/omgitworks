package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/daileyo/omgitworks/internal/xdg"
)

// WorktreeEntry represents a single git worktree discovered via git worktree list.
type WorktreeEntry struct {
	Path   string // Absolute filesystem path to the worktree
	Branch string // Branch checked out (without refs/heads/ prefix)
}

// ListWorktrees runs "git worktree list --porcelain" for the given repo and
// returns all worktrees except the main one (whose path matches repoPath).
func ListWorktrees(repoPath string) ([]WorktreeEntry, error) {
	output, err := gitCommand(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	// Resolve symlinks so we correctly match the main worktree path
	// (e.g., macOS /var → /private/var)
	resolved, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		resolved = repoPath
	}

	return parseWorktreeListPorcelain(output, resolved), nil
}

// parseWorktreeListPorcelain parses the porcelain output of git worktree list.
// Each entry is separated by a blank line. Lines of interest:
//
//	worktree <path>
//	branch refs/heads/<name>
func parseWorktreeListPorcelain(output, repoPath string) []WorktreeEntry {
	if output == "" {
		return nil
	}

	repoClean := filepath.Clean(repoPath)
	var entries []WorktreeEntry

	// Split into blocks separated by blank lines
	blocks := splitWorktreeBlocks(output)
	for _, block := range blocks {
		var path, branch string
		for _, line := range strings.Split(block, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "worktree ") {
				path = strings.TrimPrefix(line, "worktree ")
			} else if strings.HasPrefix(line, "branch ") {
				ref := strings.TrimPrefix(line, "branch ")
				branch = strings.TrimPrefix(ref, "refs/heads/")
			}
		}

		// Skip the main worktree (same path as the repo itself)
		if path == "" || filepath.Clean(path) == repoClean {
			continue
		}

		entries = append(entries, WorktreeEntry{
			Path:   path,
			Branch: branch,
		})
	}

	return entries
}

// splitWorktreeBlocks splits porcelain output into blocks separated by blank lines.
func splitWorktreeBlocks(output string) []string {
	var blocks []string
	var current strings.Builder

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Len() > 0 {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteString("\n")
		}
		current.WriteString(line)
	}
	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}

	return blocks
}

// AddWorktree creates a new git worktree at destPath for the given branch.
// If the branch already exists, it is checked out. If not, a new branch is created.
func AddWorktree(repoPath, branch, destPath string) error {
	// Try checking out existing branch first
	_, err := gitCommand(repoPath, "rev-parse", "--verify", branch)
	if err == nil {
		// Branch exists — check it out into the worktree
		_, err = gitCommand(repoPath, "worktree", "add", destPath, branch)
		return err
	}
	// Branch doesn't exist — create it
	_, err = gitCommand(repoPath, "worktree", "add", "-b", branch, destPath)
	return err
}

// RemoveWorktree removes the worktree at worktreePath from the repository at
// repoPath, using git worktree remove.
//
// Without force, git refuses to remove a worktree containing uncommitted
// changes, untracked files, or a submodule. That refusal is deliberate: the
// cleanliness check is git's rather than ours, so its message is wrapped with
// context rather than replaced, keeping the reason visible to the user.
//
// Locks are not checked here. Callers consult IsWorktreeLocked before calling,
// so a dry-run preview can mark locked entries without spending a failed
// subprocess on them.
//
// The branch the worktree had checked out is never touched.
func RemoveWorktree(repoPath, worktreePath string, force bool) error {
	args := []string{"worktree", "remove", worktreePath}
	if force {
		args = append(args, "--force")
	}

	if _, err := gitCommand(repoPath, args...); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}

// MoveWorktree moves a worktree from currentPath to newPath using git worktree move.
// If the move fails, it attempts to detect and recover from partial moves.
func MoveWorktree(repoPath, currentPath, newPath string) error {
	_, err := gitCommand(repoPath, "worktree", "move", currentPath, newPath)
	if err == nil {
		return nil
	}

	// Check for partial move: destination exists but source is gone
	_, srcErr := os.Stat(currentPath)
	_, dstErr := os.Stat(newPath)
	srcGone := os.IsNotExist(srcErr)
	dstExists := dstErr == nil

	if srcGone && dstExists {
		// Directory was moved but git internals weren't updated.
		// Move the directory back so the worktree isn't left in a broken state.
		if mvErr := os.Rename(newPath, currentPath); mvErr != nil {
			return fmt.Errorf("%w (rollback also failed: %v)", err, mvErr)
		}
		// Repair git's worktree tracking to match the restored filesystem state
		_, _ = gitCommand(repoPath, "worktree", "repair")
		return fmt.Errorf("%w (rolled back to original location)", err)
	}

	if !srcGone && dstExists {
		// Destination already exists (likely from a prior failed attempt).
		// Clean up the stale destination so a retry can succeed.
		if rmErr := os.RemoveAll(newPath); rmErr != nil {
			return fmt.Errorf("%w (destination %s already exists and cleanup failed: %v)", err, newPath, rmErr)
		}
		// Prune stale worktree entries left by prior failed attempts
		_, _ = gitCommand(repoPath, "worktree", "prune")
		// Retry the move after cleanup
		_, retryErr := gitCommand(repoPath, "worktree", "move", currentPath, newPath)
		if retryErr != nil {
			return fmt.Errorf("%w (retry after cleanup also failed: %v)", err, retryErr)
		}
		return nil
	}

	// A cross-device move is now plausible: the projects root lives under
	// $XDG_DATA_HOME while the repo may sit on a different mount. git worktree
	// move is ultimately a rename, so it cannot cross filesystems. Say so,
	// rather than surfacing git's raw error.
	if isCrossDeviceErr(err) {
		return fmt.Errorf(
			"cannot move worktree across filesystems:\n  from: %s\n  to:   %s\n"+
				"The projects root is on a different mount than the repository. "+
				"Set XDG_DATA_HOME to a location on the same filesystem, or move the worktree manually",
			currentPath, newPath)
	}

	// For any other failure, attempt repair to keep git state consistent
	_, _ = gitCommand(repoPath, "worktree", "repair")
	return err
}

// isCrossDeviceErr reports whether err looks like a rename across filesystems.
// git surfaces this as text rather than a typed error, so both the wrapped
// syscall error and git's message are checked.
func isCrossDeviceErr(err error) bool {
	if errors.Is(err, syscall.EXDEV) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "invalid cross-device link") ||
		strings.Contains(msg, "cross-device")
}

// RepairWorktrees runs "git worktree repair" to fix broken internal links
// without removing entries. This is safer than prune — it preserves worktrees
// whose directories still exist but have broken git pointers.
func RepairWorktrees(repoPath string) error {
	_, err := gitCommand(repoPath, "worktree", "repair")
	return err
}

// PruneWorktrees removes worktree entries whose directories no longer exist on disk.
// Call RepairWorktrees first to salvage fixable entries before pruning truly dead ones.
func PruneWorktrees(repoPath string) error {
	_, err := gitCommand(repoPath, "worktree", "prune")
	return err
}

// IsWorktreeLocked checks whether a worktree is locked by looking for a "locked"
// file in git's internal worktree directory.
func IsWorktreeLocked(repoPath, worktreePath string) (bool, string) {
	// Git stores worktree state in .git/worktrees/<name>/
	// A "locked" file in that directory means the worktree is locked.
	// The file contents (if any) are the lock reason.
	gitDir := filepath.Join(repoPath, ".git", "worktrees")
	entries, err := os.ReadDir(gitDir)
	if err != nil {
		return false, ""
	}

	// Find the worktree entry matching this path
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		gitdirFile := filepath.Join(gitDir, entry.Name(), "gitdir")
		content, err := os.ReadFile(gitdirFile)
		if err != nil {
			continue
		}
		// gitdir file contains the path to the worktree's .git file
		linkedPath := strings.TrimSpace(string(content))
		// The gitdir points to <worktreePath>/.git
		wtPath := filepath.Dir(linkedPath)
		if filepath.Clean(wtPath) == filepath.Clean(worktreePath) {
			lockFile := filepath.Join(gitDir, entry.Name(), "locked")
			if reason, err := os.ReadFile(lockFile); err == nil {
				return true, strings.TrimSpace(string(reason))
			}
			return false, ""
		}
	}
	return false, ""
}

// IsAligned reports whether a worktree lives under the repository's directory
// in the XDG projects root.
//
// This is keyed by repository name rather than repository path: the projects
// root is a fixed location, no longer derived from wherever the repo happens to
// sit on disk. A worktree in a legacy <repo>.wt/ directory is therefore no
// longer aligned, which is what makes `gws worktree align` pick it up for
// migration without any special-casing.
func IsAligned(worktreePath, repoName string) bool {
	projectsDir, err := xdg.RepoProjectsDir(repoName)
	if err != nil {
		return false
	}

	root := ResolvePath(projectsDir)
	wt := ResolvePath(worktreePath)

	return wt == root || strings.HasPrefix(wt, root+string(filepath.Separator))
}

// ResolvePath resolves symlinks where it can, falling back to a lexical clean
// for paths that do not exist yet. Both are needed: ~/.local/share is a symlink
// on some setups, while the projects root may not have been created yet.
func ResolvePath(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
}
