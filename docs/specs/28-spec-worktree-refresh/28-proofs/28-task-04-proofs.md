# Task 04 Proofs - Refresh reports what changed, and stays quiet otherwise

## Task Summary

`omgw worktree refresh` now reports what it changed. For each repository whose stored
worktrees differ after the sync, it prints a block of added, removed, realigned, and
re-branched entries. Every run ends with a summary of repositories refreshed, changed,
and failed. A repository whose data didn't change prints nothing.

The comparison is keyed on worktree path, as the spec requires, because a detached HEAD
has no branch to key on.

## What This Task Proves

- Each kind of change is reported under its own label, with branch and path.
- Two detached worktrees, both with an empty branch, stay distinct, and removing one
  reports exactly that one.
- An unchanged repository contributes no lines. A run where nothing changed prints only
  the summary.
- The summary counts are right across a mix of changed, unchanged, and failed
  repositories.
- Report and summary appear together in a real run of the binary, alongside a failure.

## Evidence Summary

- 4 report tests and 5 pure diff subtests pass.
- 6 of 6 mutants are caught, including a diff keyed on branch and a snapshot taken after
  the sync.
- CLI: one real run shows all four kinds of change, a silent unchanged repository, a
  failure, and exit 1.
- `make ci` passes: vet, golangci-lint with 0 issues, race-detector tests.

## Artifact: Report tests

**Command:**

~~~bash
go test ./cmd/omgitworks -count=1 -v -run 'TestWorktreeRefreshReport|TestDiffWorktrees'
~~~

**Result summary:** All pass.

~~~text
--- PASS: TestWorktreeRefreshReport_ListsAllThreeChangeKinds (0.04s)
--- PASS: TestWorktreeRefreshReport_KeysOnPath (0.02s)
--- PASS: TestWorktreeRefreshReport_SilentWhenUnchanged (0.04s)
--- PASS: TestWorktreeRefreshReport_SummaryCounts (0.04s)
--- PASS: TestDiffWorktrees (0.00s)
    --- PASS: TestDiffWorktrees/no_change (0.00s)
    --- PASS: TestDiffWorktrees/nil_and_empty_are_the_same (0.00s)
    --- PASS: TestDiffWorktrees/moved_worktree_is_a_removal_plus_an_addition (0.00s)
    --- PASS: TestDiffWorktrees/realignment_and_branch_change_on_one_entry (0.00s)
    --- PASS: TestDiffWorktrees/order_follows_the_input (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.142s
~~~

| Spec Unit 2 reporting requirement | Test |
| --- | --- |
| Report, per changed repository, worktrees added and removed and entries whose `Aligned` changed | `ListsAllThreeChangeKinds`, `KeysOnPath`, `TestDiffWorktrees` |
| Final summary: refreshed, changed, failed | `SummaryCounts`, `SilentWhenUnchanged` (summary-only output) |
| An unchanged repository is not listed individually | `SilentWhenUnchanged`, `SummaryCounts` |

**Fixture note (realigned):** A worktree only changes `Aligned` at the same path when the
projects root moves. `ListsAllThreeChangeKinds` simulates that by storing a wrong
`Aligned` value, as configuration written by an older omgitworks would have one. The CLI
transcript below produces the same change for real, by moving `XDG_DATA_HOME`.

## Artifact: The tests fail against wrong implementations

**Method:** Each mutant was applied to `worktree_refresh.go` in turn, the refresh and diff
tests run, and the file restored and verified byte-identical with `cmp`.

| Mutant | Caught by |
| --- | --- |
| Diff keyed on `Branch` instead of `Path` | `KeysOnPath`, `ListsAllThreeChangeKinds`, `TestDiffWorktrees` |
| Every repository printed, changed or not | `SilentWhenUnchanged`, `SummaryCounts` |
| Failed repositories counted as refreshed | `SummaryCounts`, `SingleMatchTagIsBulk` |
| Branch changes not detected | `ListsAllThreeChangeKinds`, `TestDiffWorktrees` |
| Realignments not detected | `ListsAllThreeChangeKinds`, `TestDiffWorktrees` |
| "Before" snapshot taken after the sync, so nothing ever differs | `ListsAllThreeChangeKinds`, `KeysOnPath`, `SilentWhenUnchanged`, `SummaryCounts` |

## Artifact: One real run with every kind of change

**What it proves:** The report format, quiet behavior, summary, and failure list work
together in the built binary on a realistic mix of external changes.

**Why it matters:** This is what a user actually sees, and the realignment here is
genuine, not simulated.

**Method:** Built with `go build` and run in a scratch workspace with an isolated `HOME`.
Before the final run, four things happened outside omgitworks: a worktree in `svc-a` was
deleted by hand and another created with plain git, `svc-c`'s directory was deleted,
`svc-d`'s worktree checked out a new branch, and `XDG_DATA_HOME` was pointed elsewhere,
moving the projects root away from `svc-a`'s `feat-stay` without moving the worktree. The
scratch path is replaced by `$DEMO`.

**Result summary:** The first run records the two external worktrees. The second prints
only `Refreshed 4 repositories, 0 changed`. The final run reports each change under its
own label, prints nothing for `svc-b`, counts `svc-c` as failed and names its failing git
step, and exits 1.

~~~text
# Setup (output hidden): four repositories. svc-a has two worktrees in the projects root;
# svc-b and svc-d each have one outside it. First, bring stored data in line:
$ omgitworks worktree refresh
[svc-b]
  added      feat-ok  $DEMO/elsewhere/svc-b  unaligned
[svc-d]
  added      feat-switch  $DEMO/elsewhere/svc-d  unaligned

Refreshed 4 repositories, 2 changed
(exit 0)

# Nothing else changes, so a second run prints only the summary
$ omgitworks worktree refresh

Refreshed 4 repositories, 0 changed
(exit 0)

# Now, outside omgitworks:
#   svc-a: delete feat-gone by hand, and add feat-ext with plain git
#   svc-b: nothing
#   svc-c: the repository directory is deleted
#   svc-d: check out a different branch inside its worktree
#   and XDG_DATA_HOME now points elsewhere, moving the projects root away from feat-stay
$ export XDG_DATA_HOME=$DEMO/newdata

$ omgitworks worktree refresh
[svc-a]
  added      feat-ext  $DEMO/elsewhere/svc-a  unaligned
  removed    feat-gone  $DEMO/home/.local/share/gws/projects/svc-a/feat-gone
  realigned  feat-stay  $DEMO/home/.local/share/gws/projects/svc-a/feat-stay  now unaligned
[svc-d]
  branch     feat-switched  $DEMO/elsewhere/svc-d  was feat-switch

Refreshed 3 repositories, 2 changed, 1 failed

1 error:
  svc-c: git worktree repair failed: chdir $DEMO/ws/svc-c: no such file or directory
(exit 1)
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
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.724s
All CI checks passed!
~~~

## Deviations From the Task Plan

- **Branch changes are reported (4.1):** A worktree at an unchanged path that now has a
  different branch checked out changes stored data, but fits none of the spec's three
  kinds. Reporting nothing would show a changed repository as unchanged, against the
  spec's goal to "report what actually changed". Such an entry prints as a `branch … was …`
  line. A worktree that moved paths is still reported as a removal plus an addition, as
  path keying implies.
- **Summary wording (4.5):** `Refreshed N repositories, M changed[, K failed]`. `Refreshed`
  counts successful repositories only, following `worktree add` and `worktree remove`.
  The summary always prints, and the error list follows it.
- **Test harness (4.6):** Output is captured from the `io.Writer` parameter rather than
  `captureStdoutStr`. `TestDiffWorktrees` was added, and
  `TestWorktreeRefreshTargets_SingleMatchTagIsBulk` was updated to require the new
  summary-then-errors shape for both single-match and multi-match runs.

## Reviewer Conclusion

Refresh reports each change to stored worktree data under a distinct label, stays silent
for repositories it didn't change, and ends with an accurate count. Path keying,
quiet-when-unchanged, and the counts each have a test shown to fail when that property
is broken.
