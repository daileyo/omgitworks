package main

import (
	"github.com/daileyo/omgitworks/internal/git"
)

// fetchStatuses resolves the git status of every path through the shared
// status cache and returns the statuses alongside a wait function.
//
// The ordering is load-bearing and must be preserved by every caller:
//
//  1. Entries that are fresh in the cache are served as they are.
//  2. Entries that are stale are served immediately, so the user sees
//     something, and are marked for a background refresh.
//  3. Paths with no entry at all are fetched in the foreground under a
//     progress spinner, because there is nothing else to show for them.
//  4. The cache is saved after the foreground fetch, before any background
//     work starts, so a foreground result is never lost to a crash or an
//     early exit.
//  5. Stale entries are refreshed in the background. The returned wait
//     function blocks until that refresh has finished and been saved;
//     callers defer it so the process does not exit mid-write.
//
// The map is keyed by path, so repository paths and worktree paths share one
// cache without colliding. A path whose status cannot be read has no entry.
func fetchStatuses(paths []string, workers int) (map[string]*git.Status, func()) {
	statusCache := git.NewCache(git.DefaultTTL)
	var cachePath string
	if cp, err := git.GetCachePath(); err == nil {
		cachePath = cp
		_ = statusCache.Load(cachePath)
	}

	// Phase 1: serve fresh entries, fall back to stale ones.
	statusMap := make(map[string]*git.Status, len(paths))
	needsFetch := false
	for _, p := range paths {
		if s := statusCache.Get(p); s != nil {
			statusMap[p] = s
		} else if s := statusCache.GetStale(p); s != nil {
			statusMap[p] = s
			needsFetch = true
		} else {
			needsFetch = true
		}
	}

	// Phase 2: paths with nothing to show are fetched now.
	var uncached []string
	for _, p := range paths {
		if statusMap[p] == nil {
			uncached = append(uncached, p)
		}
	}
	if len(uncached) > 0 {
		progress := git.NewProgress(len(uncached))
		progress.Start()
		fresh := statusCache.FetchAll(uncached, workers, progress)
		progress.Stop()
		for p, s := range fresh {
			statusMap[p] = s
		}
	}

	// Save before any background work so foreground results persist.
	if cachePath != "" {
		_ = statusCache.Save(cachePath)
	}

	// Phase 3: refresh stale entries in the background.
	wait := func() {}
	if needsFetch {
		done := statusCache.PrefetchInBackground(paths, workers, cachePath)
		wait = func() { <-done }
	}
	return statusMap, wait
}
