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
// Repair and prune errors are deliberately ignored: both are best-effort
// cleanup, and the list that follows is the authoritative check on whether the
// repository can be read at all.
func syncRepoWorktrees(repo *config.Repository) error {
	_ = git.RepairWorktrees(repo.Path)
	_ = git.PruneWorktrees(repo.Path)

	wts, err := buildWorktreeEntries(repo.Path, repo.Name)
	if err != nil {
		return err
	}
	repo.Worktrees = wts
	return nil
}
