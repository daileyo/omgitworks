package main

import (
	"os"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
)

// buildWorktreeEntries lists a repository's linked worktrees and converts them
// to stored entries, recomputing Aligned for each. It reads git state but never
// changes it, which is what lets a dry run use it on its own.
//
// Entries whose directory no longer exists are skipped, and a repository with
// no surviving entries yields nil rather than an empty slice. On disk the two
// are indistinguishable — omitempty drops both and both reload as nil — so this
// only keeps in-memory state consistent with what a reload would produce.
func buildWorktreeEntries(repoPath, repoName string) ([]config.Worktree, error) {
	entries, err := git.ListWorktrees(repoPath)
	if err != nil {
		return nil, err
	}

	var wts []config.Worktree
	for _, e := range entries {
		if _, err := os.Stat(e.Path); err != nil {
			continue
		}
		wts = append(wts, config.Worktree{
			Path:    e.Path,
			Branch:  e.Branch,
			Aligned: git.IsAligned(e.Path, repoName),
		})
	}
	return wts, nil
}

// syncRepoWorktrees brings a repository's stored worktree data back in line
// with git: repair, then prune, then rebuild from the fresh list. It is the
// single home of that sequence; every command that re-discovers worktrees
// calls it rather than repeating the steps.
//
// Repair must run before prune. Prune discards any entry whose linked .git
// file cannot be found, and repair is what restores that file when the
// worktree directory is still intact, so reversing the order throws away
// worktrees that were recoverable.
//
// The first failing step stops the sync and leaves stored data untouched; the
// error names the git command that failed. Stopping on a repair error is safe:
// git worktree repair exits zero on every kind of damage it inspects, including
// the damage it fixes, and fails only when the repository itself cannot be read
// — in which case prune and list would fail too.
func syncRepoWorktrees(repo *config.Repository) error {
	if err := git.RepairWorktrees(repo.Path); err != nil {
		return err
	}
	if err := git.PruneWorktrees(repo.Path); err != nil {
		return err
	}

	wts, err := buildWorktreeEntries(repo.Path, repo.Name)
	if err != nil {
		return err
	}
	repo.Worktrees = wts
	return nil
}
