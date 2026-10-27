# Task 02 Proofs - ALIGNED column and compact status on worktree list

## Task Summary

`omgw worktree list` now has `-s/--status` and `--color`. With `-s`, each row fetches git status through the shared `fetchStatuses` pipeline and shows the same compact icons `omgw list -s` uses, right-justified in a sub-column after the branch name. The alignment column was renamed from `STATUS` to `ALIGNED` with `yes`/`no` values so that "status" means git status in both commands. Without `-s` no git command runs.

## What This Task Proves

- The header reads `ALIGNED`; rows show `yes` or `no`.
- `-s` appends `✓`, `✗`, and `↑1 ✓` after `clean`, `dirty`, and `ahead` branches, with no `STATUS` column.
- Icon groups end at the same visible column across rows, measured with `displayWidth`.
- `--color always` emits exactly the ANSI sequences `formatStatusIcons(status, true)` produces; `--color never` emits none.
- A worktree whose directory is missing renders with a blank icon area and the command still exits zero.
- Without a status flag the status cache file is never created, so no fetch ran.

## Evidence Summary

- Five new `TestWorktreeListStatus_*` tests pass against a real-git fixture with a bare origin (clean, dirty, and ahead worktrees).
- The eight pre-existing `runWorktreeList` callers pass with the signature change; the two alignment assertions now check `yes`/`no`.
- `make ci` exits zero.
- A transcript from the real binary against a scratch fixture shows the plain, compact, and colored outputs.

## Artifact: Fixture-backed tests

**What it proves:** Rename, compact mode, sub-column alignment, color parity, missing-worktree handling, and opt-in behavior.

**Why it matters:** These are the Unit 1 requirements of the spec other than the `-S` column (task 3.0).

**Command:**

~~~bash
go test ./cmd/omgitworks -run 'TestWorktreeListStatus|TestRunWorktreeList|TestWorktreeScope|TestFetchStatuses' -v
~~~

**Result summary:** 19 tests pass, including the five new status tests and the pre-existing list and scope tests on the new signature.

~~~text
--- PASS: TestFetchStatuses_FreshEntryNeedsNoGit (0.02s)
--- PASS: TestFetchStatuses_StaleServedThenRefreshed (0.01s)
--- PASS: TestFetchStatuses_UncachedFetchedAndSavedBeforeBackground (0.01s)
--- PASS: TestFetchStatuses_UnreadablePathHasNoEntry (0.01s)
--- PASS: TestWorktreeListStatus_Compact (0.08s)
--- PASS: TestWorktreeListStatus_CompactAligned (0.07s)
--- PASS: TestWorktreeListStatus_Color (0.08s)
--- PASS: TestWorktreeListStatus_MissingWorktree (0.08s)
--- PASS: TestWorktreeListStatus_OptIn (0.06s)
--- PASS: TestRunWorktreeList_AllWorktrees (0.00s)
--- PASS: TestRunWorktreeList_FilterByRepo (0.00s)
--- PASS: TestRunWorktreeList_NoWorktrees (0.00s)
--- PASS: TestRunWorktreeList_NoWorktreesForRepo (0.00s)
--- PASS: TestRunWorktreeList_UnalignedIndicator (0.00s)
--- PASS: TestRunWorktreeList_NoArgListsAllRepos (0.03s)
--- PASS: TestRunWorktreeList_DotScopesToCurrentRepo (0.03s)
--- PASS: TestRunWorktreeList_DotNotTreatedAsNamePattern (0.03s)
--- PASS: TestWorktreeScopeFor_NamePattern (0.03s)
--- PASS: TestWorktreeScopeFor_Empty (0.03s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.570s
~~~

## Artifact: Real binary against a scratch fixture

**What it proves:** The rename and compact mode end to end, including color under `--color always`.

**Why it matters:** The tests drive `runWorktreeList` directly; this shows the cobra flag wiring and `.` scoping work from the command line.

**Command:** see each `$` line. The fixture is a scratch repository under a temporary `HOME`; paths are rewritten to `/home/user` after capture, which is why the `PATH` column's header and separator are wider than the rewritten paths below them. The binary sized the column for the original, longer paths. `cat -v` makes the ANSI codes visible.

**Result summary:** Plain output shows `ALIGNED` with `yes`; `-s` adds the icons after the branch; colored output wraps `✓` in green (`32m`), `✗` in red (`31m`), and `↑1` in cyan (`36m`).

~~~text
$ omgw worktree list .
REPO     BRANCH  PATH                                                                                                                                                   ALIGNED
-------  ------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean   /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty   /home/user/.local/share/gws/projects/fixture/dirty  yes

$ omgw worktree list . -s
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean      ✓  /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty      ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes

$ omgw worktree list fixture --color always -s | cat -v
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ^[[36mM-bM-^FM-^Q1^[[0m ^[[32mM-bM-^\M-^S^[[0m  /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean      ^[[32mM-bM-^\M-^S^[[0m  /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty      ^[[31mM-bM-^\M-^W^[[0m  /home/user/.local/share/gws/projects/fixture/dirty  yes
~~~

## Artifact: Full quality gate

**What it proves:** Vet, golangci-lint, and the race-detector run pass across every package.

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit zero.

~~~text
Tests complete
All CI checks passed!
~~~

## Reviewer Conclusion

`omgw worktree list -s` renders the same icons as `omgw list -s`, aligned in a sub-column after the branch, with color parity and a renamed `ALIGNED` column. Status stays opt-in, and a worktree with no readable status degrades to a blank cell rather than an error.
