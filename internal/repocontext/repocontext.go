// Package repocontext resolves which tracked repository a directory belongs to.
//
// Worktree subcommands that take an optional repository argument use it to fill
// that argument in from the working directory. It sits between internal/git and
// internal/config deliberately: internal/git stays free of configuration
// knowledge, and internal/config stays free of subprocess calls.
//
// Resolution is read-only. Nothing here writes configuration, creates
// directories, or prompts, so a failed resolution leaves no state behind.
package repocontext

import (
	"errors"
	"os"
	"runtime"
	"strings"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
)

// ErrNotTracked is returned when the directory is not inside a tracked
// repository, including when it is not inside any git repository at all. The
// two cases are deliberately indistinguishable to the user: both are fixed the
// same two ways, and git's own message ("not a git repository") is not
// actionable in terms of the workspace.
var ErrNotTracked = errors.New(
	"current directory is not inside a tracked repository\n" +
		"  Supply a repository argument, or run 'omgw add' to track this repository")

// Resolve returns the tracked repository that owns dir, or ErrNotTracked.
//
// The enclosing checkout is determined by git itself, so subdirectories,
// nested repositories, and submodule boundaries follow git's rules rather than
// a hand-rolled upward walk. The result is matched against tracked repository
// paths first and worktree paths second; a worktree hit returns the repository
// that owns it, never the worktree.
//
// The returned pointer aliases into cfg.Repositories, so callers may mutate it
// and save cfg, as worktree add does.
func Resolve(cfg *config.Config, dir string) (*config.Repository, error) {
	if cfg == nil {
		return nil, ErrNotTracked
	}

	toplevel, err := git.Toplevel(dir)
	if err != nil {
		// Replace git's message rather than surfacing it: the user needs the
		// workspace remedy, not git's view of the directory.
		return nil, ErrNotTracked
	}
	root := git.ResolvePath(toplevel)

	// Repository paths first. A worktree can never share a path with a tracked
	// repository root, so the order is a matter of cost, not correctness.
	for i := range cfg.Repositories {
		if samePath(git.ResolvePath(cfg.Repositories[i].Path), root) {
			return &cfg.Repositories[i], nil
		}
	}

	// Then worktree paths, returning the owning repository.
	for i := range cfg.Repositories {
		for _, wt := range cfg.Repositories[i].Worktrees {
			if samePath(git.ResolvePath(wt.Path), root) {
				return &cfg.Repositories[i], nil
			}
		}
	}

	return nil, ErrNotTracked
}

// ResolveCurrent is Resolve for the process's working directory.
func ResolveCurrent(cfg *config.Config) (*config.Repository, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, ErrNotTracked
	}
	return Resolve(cfg, dir)
}

// samePath compares two already-resolved paths.
//
// Comparison is exact, never by prefix: --show-toplevel returns a checkout root
// rather than an arbitrary directory, and prefix matching would wrongly resolve
// a repository nested inside another repository's tree. On Windows the compare
// is case-insensitive, because the filesystem is, and git reports the drive
// letter in a case that need not match what was stored in config.
func samePath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
