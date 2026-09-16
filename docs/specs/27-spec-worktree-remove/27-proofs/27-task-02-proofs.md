# Task 02 Proofs - Individual removal

## Task Summary

`omgw worktree remove` (alias `rm`) in its single-repository forms: explicit repo,
current-directory resolution, exact branch matching, lock handling, empty-directory
cleanup, and persistence.

## What This Task Proves

- Removal clears the worktree from disk, from git, and from configuration.
- One positional resolves the repository from the current directory (spec 25).
- Branch matching is **exact**: `feat` does not match a worktree on `feat-auth`.
- A locked worktree is skipped and reported **with and without** `--force`.
- The branch survives.
- Empty directories are tidied; directories still holding a worktree are not.

## Evidence Summary

- 12 test cases pass, including both lock sub-cases and both cleanup sub-cases.
- CLI: removal followed by `git branch --list` showing the branch intact.

## Artifact: End-to-end CLI

**What it proves:** The command works against a real binary, and the branch outlives its
worktree.

**Result summary:** The worktree is removed; `git branch --list` still reports the branch.

```text
$ omgw worktree remove web-ui feat-auth
Removed worktree for branch 'feat-auth' from web-ui
$ git -C web-ui branch --list feat-auth
  feat-auth
```

## Artifact: Individual removal test suite

**What it proves:** Every Unit 1 command-level requirement, including the two that are
easy to get wrong — partial branch matching, and `--force` overriding a lock.

**Why it matters:** `--force` covers dirty worktrees **only**. The lock test runs the same
assertions with force on and off, so a future change that lets force override a lock fails
the suite.

**Command:**

```bash
go test -v -run 'TestRunWorktreeRemove|TestWorktreeRemoveCmd|TestCompleteWorktreeRemove' ./cmd/omgitworks/
```

**Result summary:** All pass.

```text
--- PASS: TestRunWorktreeRemove_Individual (0.03s)
--- PASS: TestRunWorktreeRemove_CurrentRepo (0.03s)
--- PASS: TestRunWorktreeRemove_ExactBranchMatch (0.02s)
--- PASS: TestRunWorktreeRemove_NoSuchBranch (0.01s)
--- PASS: TestRunWorktreeRemove_LockedSkipped (0.03s)
    --- PASS: TestRunWorktreeRemove_LockedSkipped/without_force (0.02s)
    --- PASS: TestRunWorktreeRemove_LockedSkipped/with_force (0.01s)
--- PASS: TestRunWorktreeRemove_BranchSurvives (0.03s)
--- PASS: TestRunWorktreeRemove_DirtyNeedsForce (0.02s)
--- PASS: TestRunWorktreeRemove_CleansEmptyDirs (0.06s)
    --- PASS: TestRunWorktreeRemove_CleansEmptyDirs/nested_branch_leaves_no_empty_parent (0.02s)
    --- PASS: TestRunWorktreeRemove_CleansEmptyDirs/directory_holding_another_worktree_is_kept (0.03s)
--- PASS: TestWorktreeRemoveCmd_Alias (0.00s)
--- PASS: TestCompleteWorktreeRemove (0.03s)
    --- PASS: TestCompleteWorktreeRemove/suggests_only_branches_that_have_worktrees (0.00s)
    --- PASS: TestCompleteWorktreeRemove/filters_on_the_typed_prefix (0.00s)
    --- PASS: TestCompleteWorktreeRemove/falls_back_to_repo_names_outside_a_tracked_repo (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.263s
```

## Reviewer Conclusion

Individual removal is correct and conservative: exact matching, locks respected
regardless of force, branches untouched, and only genuinely empty directories removed.
