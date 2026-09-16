# Task 02 Proofs - Bulk creation with a single configuration save

## Task Summary

One branch, many repositories. `addWorktreeForRepo` was split so creation no longer
saves configuration, and a single bulk loop now serves both the one-repository and
many-repository paths.

## What This Task Proves

- `-t <tag> <branch>` creates a worktree in every tagged repository and none elsewhere.
- Configuration is saved **once**, after the whole run, so no repository's data is
  overwritten by a later save of stale state.
- Branch names with slashes still nest correctly in every repository.
- A repository already holding the branch is skipped and announced, not failed.
- The pre-existing single-repository behavior, including its duplicate-branch error, is
  unchanged.

## Evidence Summary

- CLI: three tagged repositories created in one command; the untagged one untouched.
- 6 bulk tests pass, plus the pre-existing single-repository suite.

## Artifact: End-to-end bulk creation

**What it proves:** The headline capability works against a real binary and real git
repositories.

**Why it matters:** This is the feature the spec exists for. The sandbox used isolated
`XDG_CONFIG_HOME` and `XDG_DATA_HOME`; paths are shown relative to `<sandbox>`.

**Command:**

~~~bash
omgw worktree add -t backend feat-auth
omgw worktree list
~~~

**Result summary:** Three creations, a summary line, exit 0. `web-ui`, tagged
`frontend`, receives nothing.

~~~text
$ omgw worktree add -t backend feat-auth
Created worktree for branch 'feat-auth' at <sandbox>/data/gws/projects/svc-a/feat-auth
Created worktree for branch 'feat-auth' at <sandbox>/data/gws/projects/svc-b/feat-auth
Created worktree for branch 'feat-auth' at <sandbox>/data/gws/projects/svc-c/feat-auth

Created 3 worktrees
$ echo $?  -> 0

$ omgw worktree list
svc-a feat-auth
svc-b feat-auth
svc-c feat-auth
~~~

Combined filtering works the same way:

~~~text
$ omgw worktree add svc-a feat-narrow -t backend
Created worktree for branch 'feat-narrow' at <sandbox>/data/gws/projects/svc-a/feat-narrow
~~~

## Artifact: Single-save persistence

**What it proves:** After a bulk run, the configuration reloaded **from disk** records the
new worktree for every repository in the run.

**Why it matters:** This is the specific failure the spec's Technical Considerations warn
about. `createWorktreeForRepo` deliberately performs no `config.Save`; the single save
happens after the loop. Saving per repository would write a stale snapshot over earlier
repositories, silently losing their data — a bug that would not show up on disk, only in
config.

**Command:**

~~~bash
go test -v -run TestWorktreeAddBulk_SingleSave ./cmd/omgitworks/
~~~

**Result summary:** Passes; all three tagged repositories are recorded after reload.

~~~text
--- PASS: TestWorktreeAddBulk_SingleSave (0.07s)
~~~

## Artifact: Bulk behavior suite

**What it proves:** Selection-to-creation, nested branch names, skip handling, and
preservation of the single-repository contract.

**Command:**

~~~bash
go test -v -run 'TestWorktreeAddBulk_(CreatesForEachTagged|BranchWithSlash|SkipsExisting|SingleRepoKeepsExistingErrors)' ./cmd/omgitworks/
~~~

**Result summary:** All pass. `SingleRepoKeepsExistingErrors` is the guard that routing
the single-repository path through the bulk loop did not turn its duplicate-branch
**error** into a **skip**.

~~~text
--- PASS: TestWorktreeAddBulk_CreatesForEachTagged (0.08s)
--- PASS: TestWorktreeAddBulk_BranchWithSlash (0.07s)
--- PASS: TestWorktreeAddBulk_SkipsExisting (0.06s)
--- PASS: TestWorktreeAddBulk_SingleRepoKeepsExistingErrors (0.05s)
~~~

## Design note: one loop, two contracts

The single-repository and bulk paths share `runWorktreeAddBulk` so there is one creation
loop rather than two that can drift. They differ in exactly one respect, guarded by a
`single` flag: with one repository, an "already exists" condition returns the original
error (preserving today's behavior and its test), while in a bulk run it is a reported
skip, as FR U1-7 requires.

## Reviewer Conclusion

Bulk creation works across a tag, persists correctly with a single save, and leaves the
established single-repository behavior untouched.
