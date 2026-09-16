# Task 02 Proofs - `omgw worktree refresh` re-syncs stored worktree data

## Task Summary

`omgw worktree refresh` exists and does the core job from spec Unit 1. For every
tracked repository it runs the shared repair → prune → list → rebuild helper from task
1.0, reports and skips repositories that fail, saves configuration once, and exits
non-zero when anything failed. It touches worktree data only.

Targeting is task 3.0, so here the command takes no arguments and refreshes every
tracked repository. Change reporting (4.0) and `--dry-run` (5.0) are also still to come.

## What This Task Proves

- A worktree created with plain git is picked up, and one deleted by hand is dropped.
- A worktree whose `.git` link is broken but whose directory is intact survives, because
  repair runs before prune.
- `Aligned` is recomputed when a worktree moves into the projects root.
- A repository left with no worktrees ends up with no stored entries.
- Repository list, user fields, tags, and the status cache are untouched, and an untracked
  repository beside the tracked ones is not discovered.
- A failing repository is reported by name with the git step that failed. The rest
  still refresh, the failed repository's stored data is left alone, and the process
  exits 1.
- A stale per-repository save, the data-loss form of "saving more than once", would be
  caught.

## Evidence Summary

- 10 `TestWorktreeRefresh*` tests and 4 `TestSyncRepoWorktrees*` tests pass.
- 5 of 5 targeted mutants are caught. The first attempt at a save mutant survived because
  it wasn't actually harmful; it was replaced with the real hazard.
- CLI: a real binary against a scratch workspace shows stale data corrected, and a broken
  repository producing exit 1 while the healthy one is still refreshed.
- `make ci` passes: vet, golangci-lint with 0 issues, race-detector tests.

## Artifact: Core re-sync tests

**What it proves:** Each Unit 1 functional requirement, one test per requirement, with
assertions made against configuration reloaded from disk.

**Why it matters:** Reloading from disk means each test checks what was actually saved,
not just what the in-memory configuration looked like.

**Command:**

~~~bash
go test ./cmd/omgitworks -count=1 -v -run 'TestWorktreeRefresh|TestSyncRepoWorktrees'
~~~

**Result summary:** All 14 pass.

~~~text
--- PASS: TestSyncRepoWorktrees_FailureLeavesStoredData (0.01s)
--- PASS: TestSyncRepoWorktrees_RepairsBeforePruning (0.02s)
--- PASS: TestSyncRepoWorktrees_PrunesDeadEntries (0.02s)
--- PASS: TestSyncRepoWorktrees_ReportsFailingStep (0.01s)
--- PASS: TestWorktreeRefresh_DiscoversExternalAdd (0.02s)
--- PASS: TestWorktreeRefresh_ClearsDeletedWorktree (0.03s)
--- PASS: TestWorktreeRefresh_RepairBeforePrune (0.02s)
--- PASS: TestWorktreeRefresh_RecomputesAligned (0.02s)
--- PASS: TestWorktreeRefresh_ClearsWhenEmpty (0.02s)
--- PASS: TestWorktreeRefresh_LeavesOtherDataUntouched (0.03s)
--- PASS: TestWorktreeRefresh_PartialFailure (0.03s)
--- PASS: TestWorktreeRefresh_SucceedsWithZeroExit (0.01s)
--- PASS: TestWorktreeRefresh_SingleSave (0.05s)
--- PASS: TestWorktreeRefreshCmd_Registered (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.289s
~~~

| Spec Unit 1 requirement | Test |
| --- | --- |
| Provide `omgw worktree refresh` | `TestWorktreeRefreshCmd_Registered` |
| Repair, then prune, then list | `TestWorktreeRefresh_RepairBeforePrune`, `TestWorktreeRefresh_ClearsDeletedWorktree` (asserts git's own record was pruned) |
| Rebuild entries, recomputing `Aligned` | `TestWorktreeRefresh_DiscoversExternalAdd`, `TestWorktreeRefresh_RecomputesAligned` |
| Exclude worktrees whose path is gone | `TestWorktreeRefresh_ClearsDeletedWorktree` |
| Clear stored data when none remain | `TestWorktreeRefresh_ClearsWhenEmpty` |
| Save once at the end | `TestWorktreeRefresh_SingleSave`, plus inspection (below) |
| No repository discovery, user detection, or cache clearing | `TestWorktreeRefresh_LeavesOtherDataUntouched` |
| Failed repository reported and skipped; run continues | `TestWorktreeRefresh_PartialFailure` |
| Non-zero exit on failure, zero otherwise | `TestWorktreeRefresh_PartialFailure`, `TestWorktreeRefresh_SucceedsWithZeroExit`, CLI transcript |

**On "clear stored data":** `TestWorktreeRefresh_ClearsWhenEmpty` asserts the reloaded
repository has zero worktrees. It doesn't distinguish nil from empty, because
`omitempty` writes both as an omitted field and both reload as nil.

## Artifact: The tests fail against wrong implementations

**What it proves:** The tests detect the failures they are named for.

**Why it matters:** Every test passed on its first run. Without mutation checks, that
can't be told apart from tests too weak to fail.

**Method:** Each mutant was applied to `worktree_refresh.go` in turn, the refresh tests
run, and the file restored and verified byte-identical with `cmp`.

| Mutant | Caught by |
| --- | --- |
| Return on the first failing repository | `TestWorktreeRefresh_PartialFailure` |
| Ignore sync errors entirely | `TestWorktreeRefresh_PartialFailure` |
| Clear the status cache during the run | `TestWorktreeRefresh_LeavesOtherDataUntouched` |
| Each repository saves its own snapshot, loaded before the run | `TestWorktreeRefresh_SingleSave` (svc-a and svc-b lost, only svc-c survived) |
| Prune before repair (in the helper, task 1.0) | `TestSyncRepoWorktrees_RepairsBeforePruning`; the same fixture drives `TestWorktreeRefresh_RepairBeforePrune` |

**A mutant that survived, and why that is correct:** the first save mutant loaded a fresh
configuration, synced one repository, and saved it, for each repository in turn. It
survived. Each fresh load already contained the previous save, so no data was lost. That
isn't the hazard the spec describes ("repositories processed early are lost to a later
save of stale data"). Data is only lost when a snapshot taken *before* earlier saves is
saved later, and that mutant was caught.

~~~text
## MUTANT: stale per-repository snapshots
    worktree_refresh_test.go:291: svc-a has no stored worktree; an earlier result was overwritten
    worktree_refresh_test.go:291: svc-b has no stored worktree; an earlier result was overwritten
--- FAIL: TestWorktreeRefresh_SingleSave (0.04s)
RESTORED
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.047s
~~~

**Limit:** no test can count writes, so a harmless extra save of the same shared
configuration would pass. The single save is confirmed by inspection: there is one
`config.Save` call in `worktree_refresh.go`, after the loop.

~~~text
$ grep -n 'config.Save' cmd/omgitworks/worktree_refresh.go
70:	if err := config.Save(cfg); err != nil {
~~~

## Artifact: Repair and prune failures are now reported

**What it proves:** The shared helper now stops at the first failing git step and names
it, as the spec requires ("a repository whose repair, prune, or list step fails shall be
reported and skipped").

**Why it matters:** Task 1.0 kept the old call sites' habit of ignoring repair and prune
errors. Stopping on a repair error would be harmful if `git worktree repair` exited
non-zero on the damage it fixes, because the sync would then abandon exactly the
worktrees it exists to salvage. That was measured before the change.

**Method:** Against git 2.34.1, `repair`, `prune`, and `list` exit codes were recorded for
a healthy repository and for each kind of damage.

**Result summary:** `repair` exits 0 on every kind of damage, including the kinds it
repairs. All three commands fail only together, when the directory is not a readable
repository.

| Repository state | repair | prune | list |
| --- | --- | --- | --- |
| Healthy | 0 | 0 | — |
| Linked `.git` file deleted (repairable) | 0, prints "`.git` file broken" | 0 | — |
| Linked directory deleted (dead) | 0 | 0 | — |
| Linked `.git` file overwritten with garbage | 0 | — | 0 |
| Admin `gitdir` file pointing nowhere | 0 | 0 | 0 |
| Directory is not a git repository | 128 | 128 | 128 |

`TestSyncRepoWorktrees_ReportsFailingStep` pins the new behavior. The rename of
`TestSyncRepoWorktrees_PropagatesListError` to `TestSyncRepoWorktrees_FailureLeavesStoredData`
reflects that a deleted repository now fails at repair, not list; its assertions are
unchanged.

**Effect on other call sites:** none observable. `refresh`, `align`, and `remove`
already skipped a repository on error, and `add` already reported one. Because the three
steps only fail together, no repository that used to reach a successful list now stops
earlier.

## Artifact: End-to-end CLI transcript

**What it proves:** The built binary corrects stale worktree data in a real workspace.

**Why it matters:** This is the user-visible scenario from the spec's user stories: a
worktree created with plain git and another deleted by hand.

**Method:** Built with `go build`, run in a scratch workspace with an isolated `HOME`.
Two repositories were registered with `omgitworks init`. The scratch path is replaced by
`$DEMO`, and table padding is compacted after that substitution.

**Result summary:** Before refresh, `worktree list` still shows the deleted
`feat-tracked` and doesn't show `feat-external`. After refresh, it shows exactly what
git has.

~~~text
$ omgitworks init $DEMO/ws
(exit 0)

$ omgitworks worktree add svc-a feat-tracked
(exit 0)

# Starting state
$ omgitworks worktree list
REPO  BRANCH  PATH  STATUS
-----  --------  --------  ------
svc-a  feat-tracked  $DEMO/home/.local/share/gws/projects/svc-a/feat-tracked  aligned
(exit 0)

# Outside omgitworks: create a worktree with plain git, and delete the tracked one by hand
$ git -C $DEMO/ws/svc-b worktree add -q -b feat-external $DEMO/elsewhere/feat-external
(exit 0)

$ rm -rf $DEMO/home/.local/share/gws/projects/svc-a/feat-tracked
(exit 0)

# Stored data is now stale
$ omgitworks worktree list
REPO  BRANCH  PATH  STATUS
-----  --------  --------  ------
svc-a  feat-tracked  $DEMO/home/.local/share/gws/projects/svc-a/feat-tracked  aligned
(exit 0)

$ omgitworks worktree refresh
(exit 0)

# Stored data matches git again
$ omgitworks worktree list
REPO  BRANCH  PATH  STATUS
-----  --------  --------  ------
svc-b  feat-external  $DEMO/elsewhere/feat-external  (unaligned)
(exit 0)
~~~

## Artifact: Partial failure exits non-zero

**What it proves:** A failing repository makes the real process exit 1, is reported with
the failing git step, and doesn't stop the healthy repository from refreshing.

**Command:** Continues the workspace above after deleting `svc-a`'s directory.

**Result summary:** Exit 1, with `svc-a` reported as failing at `git worktree repair`.
`svc-b` still refreshed.

~~~text
# Break svc-a, then refresh everything
$ rm -rf $DEMO/ws/svc-a
$ omgitworks worktree refresh

1 error:
  svc-a: git worktree repair failed: chdir $DEMO/ws/svc-a: no such file or directory
(exit 1)

# svc-b was still refreshed
$ omgitworks worktree list
REPO  BRANCH  PATH  STATUS
-----  --------  --------  ------
svc-b  feat-external  $DEMO/elsewhere/feat-external  (unaligned)
(exit 0)
~~~

## Artifact: Repository quality gate

**Command:**

~~~bash
PATH=$HOME/go/bin:$PATH make ci
~~~

**Result summary:** Vet is clean, lint reports 0 issues, and every package passes under
`-race`.

~~~text
Running linter...
0 issues.
...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.713s
All CI checks passed!
~~~

## Deviations From the Task Plan

- **2.1:** `Use: "refresh"` with `cobra.NoArgs`, not `refresh [repo|.]`, until task 3.0 adds
  targeting. A stub that accepted and ignored an argument would be misleading.
- **2.3:** Flag variables are declared in the tasks that register them (3.2, 5.1), so no
  unused variable is committed.
- **2.4:** `runWorktreeRefresh` takes an `io.Writer`, following `runWorktreeRemove`.
  `dryRun` is added in 5.0.
- **2.5:** Failures are listed together after the run, matching `worktree remove`'s bulk
  output, instead of as they happen.
- **2.11 (added):** The helper change above.

## Reviewer Conclusion

`omgw worktree refresh` corrects stored worktree data for external additions, hand
deletions, broken links, and moves, without touching anything else in the configuration
or the status cache. Failures are isolated per repository and reflected in the exit
status. The tests were shown to fail against the wrong implementations they guard
against. The one property no test can observe, the literal save count, was confirmed by
inspection.
