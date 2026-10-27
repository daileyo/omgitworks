# Task 01 Proofs - The status fetch pipeline lives in one helper

## Task Summary

`runList` carried a sixty-line, two-phase status fetch (fresh cache, stale fallback, foreground fetch under a spinner, save, background refresh). This task moved it into `fetchStatuses(paths, workers)` in `cmd/omgitworks/status_fetch.go` so `omgw worktree list` can call the same pipeline in task 2.0 instead of copying it. `resolveColorEnabled` now takes the color mode as a value, removing the only package-global read the worktree command would otherwise inherit.

## What This Task Proves

- The fetch pipeline exists in exactly one place; `FetchAll` and `PrefetchInBackground` are called only from `status_fetch.go`.
- The ordering `runList` depended on is now pinned by tests that did not exist before: fresh entries need no git, stale entries are served then refreshed, uncached paths are fetched and saved before the background phase.
- `omgw list -s` and `-S` are unchanged: every pre-existing test in `cmd/omgitworks` and `internal/git` passes with no assertion edits.
- Vet, lint, and race-detector tests pass.

## Evidence Summary

- A grep over non-test Go files finds the two pipeline calls only in the new helper.
- Four new `TestFetchStatuses_*` tests pass against real git repositories with `HOME` redirected so the shared cache file lands in a temp dir.
- `make ci` exits zero.

## Artifact: Pipeline exists in one place

**What it proves:** The full fetch sequence was moved, not duplicated.

**Why it matters:** Success metric 4 of the spec requires one pipeline shared by both commands.

**Command:**

~~~bash
grep -n 'PrefetchInBackground\|FetchAll(' cmd/omgitworks/*.go | grep -v _test.go
~~~

**Result summary:** Both calls resolve to `status_fetch.go`; `list.go` no longer references either.

~~~text
cmd/omgitworks/status_fetch.go:58:		fresh := statusCache.FetchAll(uncached, workers, progress)
cmd/omgitworks/status_fetch.go:73:		done := statusCache.PrefetchInBackground(paths, workers, cachePath)
~~~

## Artifact: Ordering tests for the helper

**What it proves:** Fresh-first, stale-served-then-refreshed, foreground-fetch-then-save-then-background, and no-entry-for-unreadable-paths all hold.

**Why it matters:** The planning audit found that `runList`'s ordering had no direct tests; these close that gap before the worktree command depends on it.

**Command:**

~~~bash
go test ./cmd/omgitworks -run 'TestFetchStatuses' -v
~~~

**Result summary:** All four cases pass. The fresh case runs with an empty `PATH` so no `git` binary is reachable; the stale case checks the served timestamp equals the seeded stale one and the saved file is newer after `wait()`; the uncached case reads the cache file before `wait()`.

~~~text
=== RUN   TestFetchStatuses_FreshEntryNeedsNoGit
--- PASS: TestFetchStatuses_FreshEntryNeedsNoGit (0.01s)
=== RUN   TestFetchStatuses_StaleServedThenRefreshed
--- PASS: TestFetchStatuses_StaleServedThenRefreshed (0.01s)
=== RUN   TestFetchStatuses_UncachedFetchedAndSavedBeforeBackground
--- PASS: TestFetchStatuses_UncachedFetchedAndSavedBeforeBackground (0.01s)
=== RUN   TestFetchStatuses_UnreadablePathHasNoEntry
--- PASS: TestFetchStatuses_UnreadablePathHasNoEntry (0.01s)
PASS
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.033s
~~~

## Artifact: Full quality gate

**What it proves:** The refactor passes vet, golangci-lint, and the race-detector test run across every package.

**Why it matters:** This is the same gate CI runs; pre-existing `list` tests pass with no assertion changes.

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit zero. Tail of the output:

~~~text
ok  	github.com/daileyo/omgitworks/internal/xdg	1.030s
Tests complete
All CI checks passed!
~~~

## Reviewer Conclusion

The status pipeline is now a single helper with its own ordering tests, `runList` is a thin caller of it, and nothing observable about `omgw list` changed.
