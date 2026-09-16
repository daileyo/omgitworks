# Task 03 Proofs - "." targeting for worktree list and align

## Task Summary

`omgw worktree list` and `omgw worktree align` now accept `.` as the repository
argument, meaning the repository resolved from the current directory. Omitting the
argument still means every tracked repository on both commands, so no existing
invocation changed meaning.

## What This Task Proves

- `list .` and `align .` scope to the current repository, while bare `list` and bare
  `align` still cover all repositories.
- `align .` honors the existing `--dry-run` flag and leaves out-of-scope worktrees
  untouched.
- A literal `.` is intercepted before name matching, so a tracked repository named
  `my.repo` is never matched by it.
- `.` that cannot be resolved produces the same error as task 1.0, identically on both
  subcommands.
- The 13 pre-existing call sites migrated to the new signature with **no assertion
  changed**, and all still pass.

## Evidence Summary

- CLI: bare `align --dry-run` plans 2 worktrees; `align . --dry-run` plans 1.
- CLI: bare `list` shows `other-repo`; `list .` does not.
- 10 new tests and 12 pre-existing `list`/`align` tests all pass.

## Artifact: list vs list . side by side

**What it proves:** The two forms differ in scope, and the difference is exactly the
current repository.

**Why it matters:** This is the core user-visible behavior of Unit 2, and it also shows
the bare form was not altered.

**Command:**

~~~bash
cd <workspace>/demo-repo
omgw worktree list
omgw worktree list .
~~~

**Result summary:** The bare form lists worktrees from both `demo-repo` and
`other-repo`. The `.` form lists only `demo-repo`, dropping `other-repo/feat-z`.

~~~text
$ omgw worktree list
REPO        BRANCH         PATH                                     STATUS
----------  -------------  ---------------------------------------  ------
demo-repo   feat-arity     <sandbox>/.../demo-repo/feat-arity       aligned
demo-repo   feat-arity-ok  <sandbox>/.../demo-repo/feat-arity-ok    aligned
demo-repo   feat-arity2    <sandbox>/.../demo-repo/feat-arity2      aligned
demo-repo   feat-x         <sandbox>/.../demo-repo/feat-x           aligned
demo-repo   feat-y         <sandbox>/.../demo-repo/feat-y           aligned
other-repo  feat-z         <sandbox>/.../other-repo/feat-z          aligned

$ omgw worktree list .
REPO       BRANCH         PATH                                      STATUS
---------  -------------  ----------------------------------------  ------
demo-repo  feat-arity     <sandbox>/.../demo-repo/feat-arity        aligned
demo-repo  feat-arity-ok  <sandbox>/.../demo-repo/feat-arity-ok     aligned
demo-repo  feat-arity2    <sandbox>/.../demo-repo/feat-arity2       aligned
demo-repo  feat-x         <sandbox>/.../demo-repo/feat-x            aligned
demo-repo  feat-y         <sandbox>/.../demo-repo/feat-y            aligned
~~~

## Artifact: Scoped align with --dry-run

**What it proves:** Scoping works on the mutating command too, and the existing
`--dry-run` flag still applies.

**Why it matters:** `align` moves directories. Getting its scope wrong would relocate
another repository's worktrees, so this is the highest-risk path in the spec. An
unaligned worktree was created in each repository so both forms have real work to plan.

**Command:**

~~~bash
cd <workspace>/demo-repo
omgw worktree align --dry-run      # all repos
omgw worktree align . --dry-run    # current repo only
~~~

**Result summary:** The bare form plans 2 moves, one per repository. The `.` form plans
1, for `demo-repo` only, and `other-repo`'s loose worktree is left out entirely.

~~~text
$ omgw worktree align --dry-run          # all repos
Dry run — no changes will be made:

Would move [demo-repo] loose-demo
  from: <sandbox>/loose/demo
  to:   <sandbox>/data/gws/projects/demo-repo/loose-demo

Would move [other-repo] loose-other
  from: <sandbox>/loose/other
  to:   <sandbox>/data/gws/projects/other-repo/loose-other

Total: 2 worktrees to align

$ omgw worktree align . --dry-run        # current repo only
Dry run — no changes will be made:

Would move [demo-repo] loose-demo
  from: <sandbox>/loose/demo
  to:   <sandbox>/data/gws/projects/demo-repo/loose-demo

Total: 1 worktree to align
~~~

## Artifact: Consistent error for an unresolvable "."

**What it proves:** `.` failure reuses the task 1.0 error contract rather than inventing
a second message.

**Command:**

~~~bash
cd $(mktemp -d) && omgw worktree list .
~~~

**Result summary:** Identical wording to `worktree add`'s failure, naming both remedies.

~~~text
Error: current directory is not inside a tracked repository
  Supply a repository argument, or run 'omgw add' to track this repository
~~~

## Artifact: Test suite, new and pre-existing

**What it proves:** Scoping, the preserved bare-argument behavior, the period-in-name
guard, and the shared error are all covered; and the signature migration broke nothing.

**Why it matters:** `runWorktreeList` and `runWorktreeAlign` changed signature, touching
13 existing call sites. Those tests were migrated mechanically with every assertion left
as it was, so their continued passing is meaningful evidence rather than a rewrite.

**Command:**

~~~bash
go test -v -run 'TestRunWorktreeList|TestRunWorktreeAlign|TestWorktreeDot|TestWorktreeScope' ./cmd/omgitworks/
~~~

**Result summary:** 22 cases pass, 12 of them pre-existing.

~~~text
--- PASS: TestRunWorktreeAlign_MovesUnaligned (0.03s)
--- PASS: TestRunWorktreeAlign_SkipsAligned (0.00s)
--- PASS: TestRunWorktreeAlign_DryRun (0.02s)
--- PASS: TestRunWorktreeAlign_DuplicateNames (0.03s)
--- PASS: TestRunWorktreeAlign_FilterByRepo (0.03s)
--- PASS: TestRunWorktreeAlign_MigratesLegacyWtLayout (0.03s)
--- PASS: TestRunWorktreeAlign_LegacyDirWithOtherContentKept (0.02s)
--- PASS: TestRunWorktreeList_AllWorktrees (0.00s)
--- PASS: TestRunWorktreeList_FilterByRepo (0.00s)
--- PASS: TestRunWorktreeList_NoWorktrees (0.00s)
--- PASS: TestRunWorktreeList_NoWorktreesForRepo (0.00s)
--- PASS: TestRunWorktreeList_UnalignedIndicator (0.00s)
--- PASS: TestRunWorktreeList_NoArgListsAllRepos (0.04s)
--- PASS: TestRunWorktreeList_DotScopesToCurrentRepo (0.04s)
--- PASS: TestRunWorktreeList_DotNotTreatedAsNamePattern (0.04s)
--- PASS: TestRunWorktreeAlign_NoArgProcessesAllRepos (0.04s)
--- PASS: TestRunWorktreeAlign_DotScopesToCurrentRepo (0.05s)
--- PASS: TestRunWorktreeAlign_DotHonorsDryRun (0.04s)
--- PASS: TestWorktreeDotResolutionFailure (0.04s)
    --- PASS: TestWorktreeDotResolutionFailure/list (0.00s)
    --- PASS: TestWorktreeDotResolutionFailure/align (0.00s)
--- PASS: TestWorktreeScopeFor_NamePattern (0.04s)
--- PASS: TestWorktreeScopeFor_Empty (0.04s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.531s
~~~

## Design note: why a scope type

`.` cannot be handled by translating it into the resolved repository's *name*, because
the name would re-enter `filter.MatchesPattern` and could match more than one
repository. `worktreeScope` therefore carries an already-resolved exact path, compared
with `git.ResolvePath` equality, and `worktreeScopeFor` returns before any pattern
matching for the `.` case. `TestRunWorktreeList_DotNotTreatedAsNamePattern` asserts both
halves of that: `NamePattern` is empty and a repository named `my.repo` is not matched.

## Reviewer Conclusion

Both subcommands scope correctly to the current repository, the mutating one respects
`--dry-run` and leaves other repositories alone, and the meaning of the bare form is
demonstrably unchanged.
