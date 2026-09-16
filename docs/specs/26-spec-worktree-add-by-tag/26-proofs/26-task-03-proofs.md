# Task 03 Proofs - Partial-failure reporting and exit status

## Task Summary

A bulk run now attempts every selected repository, reports created / skipped / failed
counts, names each failure, retains successful work, and exits non-zero if anything
failed — without cobra printing over the summary.

## What This Task Proves

- The run continues past a failing repository; later repositories are still created.
- The summary counts all three outcomes accurately in a single run containing all three.
- Each failure names the repository and the underlying reason.
- Exit status is non-zero on any failure and zero when the run held only creations and
  skips.
- Successful creations before a failure are retained on disk and in configuration.
- Non-sentinel errors still print, despite `SilenceErrors`.

## Evidence Summary

- CLI: a mixed run showing one skip, one creation, one failure, `exit=1`.
- 8 tests pass, including a three-case exit-status table.

## Artifact: Mixed-outcome run

**What it proves:** All three outcomes in one invocation, reported in the shape the spec
requires and the one `worktree align` already uses.

**Why it matters:** Bulk operations are only trustworthy if a partial result is legible.
The failure was produced genuinely, by removing `svc-b`'s repository directory, not by
stubbing anything.

**Command:**

~~~bash
omgw worktree add -t backend feat-mix
echo $?
~~~

**Result summary:** `svc-a` skipped (already had it), `svc-c` created, `svc-b` failed with
git's reason. Counts are correct, the error block is separately headed, and the exit
status is 1 with no usage block and no duplicate error line.

~~~text
$ omgw worktree add -t backend feat-mix
Skipping [svc-a] feat-mix — worktree for branch 'feat-mix' already exists at <sandbox>/data/gws/projects/svc-a/feat-mix
Created worktree for branch 'feat-mix' at <sandbox>/data/gws/projects/svc-c/feat-mix

Created 1 worktree, skipped 1, 1 failed

1 error:
  svc-b: failed to create worktree: git worktree add -b feat-mix <sandbox>/data/gws/projects/svc-b/feat-mix failed: chdir <sandbox>/ws/svc-b: no such file or directory
   exit=1
~~~

## Artifact: Exit-status contract

**What it proves:** The three outcome shapes map to the right statuses.

**Command:**

~~~bash
go test -v -run TestWorktreeAddBulk_ExitStatus ./cmd/omgitworks/
~~~

**Result summary:** All three sub-cases pass. Skips are explicitly **not** failures.

~~~text
--- PASS: TestWorktreeAddBulk_ExitStatus (0.19s)
    --- PASS: .../all_created_returns_nil (0.06s)
    --- PASS: .../creations_plus_skips_returns_nil (0.07s)
    --- PASS: .../any_failure_returns_the_sentinel (0.06s)
~~~

## Artifact: Continue-past-failure, diagnosability, and no rollback

**Command:**

~~~bash
go test -v -run 'TestWorktreeAddBulk_(ContinuesPastFailure|SummaryCounts|NamesFailedRepos|NoRollback)' ./cmd/omgitworks/
~~~

**Result summary:** All pass. `NoRollback` breaks the **last** repository and asserts the
two earlier worktrees survive both on disk and in the reloaded configuration.

~~~text
--- PASS: TestWorktreeAddBulk_ContinuesPastFailure (0.06s)
--- PASS: TestWorktreeAddBulk_SummaryCounts (0.06s)
--- PASS: TestWorktreeAddBulk_NamesFailedRepos (0.06s)
--- PASS: TestWorktreeAddBulk_NoRollback (0.06s)
~~~

## Behavior change worth reviewing: SilenceErrors and SilenceUsage

The spec calls for a sentinel error plus `SilenceErrors`/`SilenceUsage` so the non-zero
exit does not print over the summary. Neither flag was set anywhere in this codebase
before, so enabling them on `worktreeAddCmd` also suppresses cobra's output for **every
other** `worktree add` error — including `no repository found matching '%s'` and spec 25's
resolution error, both of which previously printed an error line and a usage block.

To avoid silently losing those messages, `runWorktreeAddCommand` prints `Error: %v` to
stderr for any error that is not the partial-failure sentinel. The visible effect is that
those errors keep their message and **lose the usage block** — which is the tidier
outcome, and is what spec 25's validation flagged as noise.

Confirmed live: both `--tag` repetition and an unmatched tag print one clean line and
exit 1, with no usage dump.

## Reviewer Conclusion

Partial failure is reported completely and exits correctly, successful work is never
discarded, and no error path was made silent by the cobra configuration this required.
