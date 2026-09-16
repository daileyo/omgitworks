# Task 01 Proofs - One shared home for worktree re-discovery

## Task Summary

The repair → prune → list → rebuild sequence that re-syncs a repository's stored
worktrees now lives in one place, `cmd/omgitworks/worktree_discover.go`, and every
command that re-discovers worktrees calls it: `refresh`, `worktree align`,
`worktree add`, and spec 27's `worktree remove`. This lands before `worktree refresh`
exists so the new command consumes the helper instead of adding another copy.

The helper is split in two on purpose. `buildWorktreeEntries` only reads git state;
`syncRepoWorktrees` repairs, prunes, then calls it. Dry run (task 5.0) needs the
read-only half by itself.

## What This Task Proves

- The full sequence has exactly one caller of `git.ListWorktrees`, down from four.
- The helper keeps `refresh.go`'s stricter semantics: a missing directory is skipped,
  an empty result is nil (in memory only: on disk `omitempty` makes nil and empty
  identical), and a list
  failure is returned while stored data is left untouched.
- Repair runs before prune, and a test fails if the order is reversed.
- The `add` path's behavior change — it now repairs, prunes, skips missing paths, and
  reports list errors — is deliberate and covered by a test that fails on the old code.
- The same change on spec 27's `remove` path is covered by a test that fails on the old code.
- All 65 existing align, add, remove, and refresh tests pass with no assertion changed.

## Evidence Summary

- `grep` finds one non-test `ListWorktrees(` caller, in `worktree_discover.go`.
- 8 new tests pass: 6 for the helper, 1 for the `add` path, 1 for the `remove` path.
- Two mutation checks: reversing repair and prune fails the ordering test; restoring the
  original `worktree_add.go` or `worktree_remove.go` fails all three assertions of the
  matching path test.
- `make ci` passes: `go vet`, golangci-lint with 0 issues, and the full race-detector suite.

## Rebase onto spec 27 during this task

**What happened:** Planning and the first version of this task were based on
`feat/worktree-add-by-tag` (`f4b0601`). Spec 27 (`worktree remove`) was being built at the
same time on a sibling branch from the same base, and it added `refreshWorktreeData` in
`worktree_remove.go`, a fourth list → rebuild copy. The session that built spec 27 merged
the two branches in a throwaway worktree and got no conflict, a clean build, and passing
tests — yet the combined tree had two `ListWorktrees` callers. It reported this to this
session.

**Why it matters:** Nothing automated would have caught it. Merged naively, the branches
would silently break success metric 5.

**Resolution:** With the user's approval, `feat/worktree-refresh` was rebased onto
`feat/worktree-remove` (`9bae1a6`, carrying specs 25–27). The rebase was clean. Sub-task
1.10 was added to convert the fourth copy. Before the real branch was touched, the whole
sequence was first run in a throwaway detached worktree.

## Artifact: The sequence exists in one place

**What it proves:** Success metric 5 — the repair → prune → list → rebuild sequence
exists in exactly one place.

**Why it matters:** Before this task, `refresh.go`, `worktree_align.go`, and
`worktree_add.go` each rebuilt stored worktrees their own way, and spec 27's
`worktree_remove.go` added a fourth. The copies disagreed.

**Command:**

~~~bash
grep -rn 'ListWorktrees(' cmd internal --include='*.go' | grep -v _test.go | grep -v 'func ListWorktrees'
~~~

**Result summary:** One caller remains, inside the shared helper.

~~~text
cmd/omgitworks/worktree_discover.go:18:	entries, err := git.ListWorktrees(repoPath)
~~~

**Deviation from plan:** The planned grep searched for `RepairWorktrees` / `PruneWorktrees`
and expected hits only in `worktree_discover.go`. That could never pass once task 1.5
decided to keep align's standalone repair + prune planning pass
(`worktree_align.go:90-91`). That pass is not the full sequence: it fixes broken
`.git` links before align issues `git worktree move` calls based on stored data, and
removing it would make those moves fail (planning audit flag 2). The proof artifact in
the task file was amended to grep for `ListWorktrees(`, the step unique to the full
sequence.

## Artifact: Shared helper tests

**What it proves:** The extracted helper enforces path skipping, nil-on-empty, error
propagation, `Aligned` recomputation, repair-before-prune, and pruning of dead entries.

**Why it matters:** Two of the three original copies had none of the first three
behaviors. These tests fix the consolidated contract so later edits can't quietly weaken it.

**Command:**

~~~bash
go test ./cmd/omgitworks -count=1 -v -run 'TestBuildWorktreeEntries|TestSyncRepoWorktrees|TestWorktreeAdd_RepairsBeforeRebuild'
~~~

**Result summary:** All seven pass.

~~~text
--- PASS: TestWorktreeAdd_RepairsBeforeRebuild (0.03s)
--- PASS: TestBuildWorktreeEntries_SkipsMissingDirectories (0.02s)
--- PASS: TestBuildWorktreeEntries_NilWhenNoneSurvive (0.01s)
--- PASS: TestBuildWorktreeEntries_RecomputesAligned (0.02s)
--- PASS: TestSyncRepoWorktrees_PropagatesListError (0.01s)
--- PASS: TestSyncRepoWorktrees_RepairsBeforePruning (0.01s)
--- PASS: TestSyncRepoWorktrees_PrunesDeadEntries (0.01s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.112s
~~~

## Artifact: The ordering test fails when the order is wrong

**What it proves:** `TestSyncRepoWorktrees_RepairsBeforePruning` really tells the two
orders apart, rather than passing either way.

**Why it matters:** A test of ordering that can't fail proves nothing. The fixture was
found by experiment: of four kinds of worktree damage tried against git 2.34.1, only a
deleted linked `.git` file is both reported prunable ("gitdir file points to non-existent
location") and restored by `git worktree repair` run from the main repository. A
hand-moved worktree directory, a moved main repository, and a corrupt `gitdir` admin
file don't separate the two orders. The fixture is the `breakWorktreeLink` helper.

**Command:** The two calls in `syncRepoWorktrees` were swapped temporarily, the test
run, and the file restored and checked byte-identical with `cmp`.

**Result summary:** With prune first, the recoverable worktree is discarded and the test
fails. With the file restored, it passes.

~~~text
MUTANT (prune before repair):
--- FAIL: TestSyncRepoWorktrees_RepairsBeforePruning (0.01s)
    worktree_discover_test.go:163: recoverable worktree should survive, got []
FAIL
restored
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.017s
~~~

## Artifact: The add-path behavior change is real and covered

**What it proves:** `worktree add` now repairs and prunes, and
`TestWorktreeAdd_RepairsBeforeRebuild` would catch a regression to the old behavior.

**Why it matters:** The spec requires this change to be "covered by tests rather than
assumed harmless". A surviving entry on its own can't show it: listing without repair
still reports a worktree whose `.git` file is missing. So the test checks what only
repair and prune produce — the restored `.git` file, and the dead entry gone from git's
own records.

**Command:** `worktree_add.go` was temporarily replaced with its version from `HEAD`
(`git show HEAD:cmd/omgitworks/worktree_add.go`), the test run, and the file restored.

**Result summary:** On the original code all three assertions fail, and the old path is
shown storing the dead `gone` entry. On the new code the test passes.

~~~text
ORIGINAL add.go from HEAD:
--- FAIL: TestWorktreeAdd_RepairsBeforeRebuild (0.02s)
    worktree_add_test.go:193: repair should have restored the mislinked worktree's .git file: stat .../mislinked/.git: no such file or directory
    worktree_add_test.go:197: prune should have removed git's record of the deleted worktree
    worktree_add_test.go:209: expected mislinked and feature-x stored without gone, got map[feature-x:true gone:true mislinked:true]
FAIL
restored
~~~

**Error-handling note (task 1.6):** The old code silently dropped a `ListWorktrees`
failure. That error is now returned, but by then the worktree has already been created,
so the message says so and points to the recovery command:
`created worktree at <path> but could not refresh stored worktree data (run 'omgw worktree refresh')`.

## Artifact: The remove-path behavior change is real and covered

**What it proves:** After a removal, `worktree remove` now repairs and prunes the rest of the
repository's worktrees before saving, and `TestRunWorktreeRemove_RefreshUsesSharedRules`
would catch a regression.

**Why it matters:** Spec 27's `refreshWorktreeData` was the weakest copy: no repair, no
prune, no skipping of missing paths, no nil-on-empty, and list errors ignored. Its unused
`cfg` parameter (`_ = cfg`) was dropped in the conversion.

**Command:** `worktree_remove.go` was temporarily replaced with its version from `HEAD`, the
test run, and the file restored and checked byte-identical with `cmp`.

**Result summary:** On the original code all three assertions fail, and a deleted sibling
worktree (`gone`) is shown still stored after an unrelated removal. On the new code the test
passes.

~~~text
--- PASS: TestRunWorktreeRemove_RefreshUsesSharedRules (0.04s)

ORIGINAL worktree_remove.go from HEAD:
--- FAIL: TestRunWorktreeRemove_RefreshUsesSharedRules (0.03s)
    worktree_remove_test.go:397: repair should have restored the mislinked worktree's .git file: stat .../mislinked/.git: no such file or directory
    worktree_remove_test.go:400: prune should have removed git's record of the deleted worktree
    worktree_remove_test.go:412: expected only mislinked stored, got map[gone:true mislinked:true]
FAIL
restored
~~~

## Artifact: Existing call sites show no regression

**What it proves:** Converting `refresh.go`, `worktree_align.go`, and `worktree_add.go`
broke none of their existing tests.

**Why it matters:** This refactor touches four commands that spec 28 doesn't otherwise
change.

**Command:**

~~~bash
git diff --stat 9bae1a6 -- cmd/omgitworks/*_test.go
go test ./cmd/omgitworks -count=1 -v -run 'TestRunWorktreeRemove_|TestWorktreeRemoveBulk_|TestWorktreeRemove_|TestWorktreeRemoveTargets_|TestRunWorktreeAlign|TestWorktreeAddBulk|TestRunWorktreeAdd|TestRefresh|TestRunRefresh'
~~~

**Result summary:** Compared with the spec 27 base, test files only gained lines, with no
deletions: one new file and one appended test in each of `worktree_add_test.go` and
`worktree_remove_test.go`. All 66 matching tests pass: 65 pre-existing, plus the new
remove-path test, whose name matches the `TestRunWorktreeRemove_` pattern.

~~~text
 cmd/omgitworks/worktree_add_test.go      |  45 ++++++++
 cmd/omgitworks/worktree_discover_test.go | 189 +++++++++++++++++++++++++++++++
 cmd/omgitworks/worktree_remove_test.go   |  47 ++++++++
 3 files changed, 281 insertions(+)

     66 PASS:
~~~

**Known gap (planning audit flag 1, accepted without remediation):** `worktree align`'s
rebuild also picks up path skipping and nil-on-empty through the shared helper. Unlike
`add` and `remove`, align did not get its own path test. No
align-specific test checks those new behaviors. They are covered only at the helper level
(`TestBuildWorktreeEntries_*`).

## Artifact: Repository quality gate

**What it proves:** The refactor meets the repository's full CI gate.

**Why it matters:** `make ci` is the documented pre-push gate: vet, golangci-lint, and
race-detector tests.

**Command:**

~~~bash
PATH=$HOME/go/bin:$PATH make ci
~~~

**Result summary:** Vet is clean, lint reports 0 issues, and every package passes under
`-race`.

Run on the rebased tree, after the remove-path conversion. The `Error:` lines in the
full log are expected output from negative-path tests, not failures; `make` exited 0.

~~~text
Running go vet...
go vet ./...
Running linter...
0 issues.
...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.413s
ok  	github.com/daileyo/omgitworks/internal/classifier	(cached)
ok  	github.com/daileyo/omgitworks/internal/config	(cached)
ok  	github.com/daileyo/omgitworks/internal/discovery	(cached)
ok  	github.com/daileyo/omgitworks/internal/filter	(cached)
ok  	github.com/daileyo/omgitworks/internal/git	1.452s
ok  	github.com/daileyo/omgitworks/internal/repocontext	1.140s
ok  	github.com/daileyo/omgitworks/internal/user	(cached)
ok  	github.com/daileyo/omgitworks/internal/xdg	(cached)
All CI checks passed!
~~~

## Reviewer Conclusion

Worktree re-discovery now has one implementation, with the stricter of the old
behaviors, including the fourth copy that spec 27 added on a parallel branch. The
ordering requirement and the `add` and `remove` path changes are pinned by tests shown
to fail against the wrong code, and the four converted commands keep their existing
test suites green. One gap remains, as accepted at planning: align's inherited behavior
changes are tested only at the helper level.
