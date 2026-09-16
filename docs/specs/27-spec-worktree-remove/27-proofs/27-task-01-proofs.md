# Task 01 Proofs - The git.RemoveWorktree primitive

## Task Summary

`internal/git` had `AddWorktree`, `MoveWorktree`, `RepairWorktrees`, `PruneWorktrees`, and
`IsWorktreeLocked`, but no removal. This task adds `RemoveWorktree` and the two safety
behaviors every higher form inherits.

## What This Task Proves

- Removal deletes the directory and git's own worktree entry.
- Without `--force`, git refuses when the worktree has uncommitted **or untracked**
  changes; with it, removal succeeds.
- The branch survives: removing a checkout never discards the work on it.
- Git's own explanation survives the error wrapping.

## Evidence Summary

6 primitive tests pass. The refusal is genuinely git's — no cleanliness check was
reimplemented — so the tests use real repositories and real worktrees.

## Artifact: Primitive test suite

**What it proves:** Each case pins one requirement from Unit 1: the operation itself, the
dirty and untracked refusals, branch survival, and error wrapping.

**Why it matters:** Question 5b chose to defer the cleanliness check to git. That makes
git's message the *only* explanation a user gets for a refusal, so
`TestRemoveWorktree_ErrorKeepsGitReason` asserts it is wrapped rather than replaced.

**Command:**

```bash
go test -v -run TestRemoveWorktree ./internal/git/
```

**Result summary:** All 6 pass.

```text
--- PASS: TestRemoveWorktree (0.02s)
--- PASS: TestRemoveWorktree_DirtyRefused (0.03s)
--- PASS: TestRemoveWorktree_UntrackedRefused (0.02s)
--- PASS: TestRemoveWorktree_BranchSurvives (0.02s)
--- PASS: TestRemoveWorktree_ErrorKeepsGitReason (0.02s)
--- PASS: TestRemoveWorktree_NotAWorktree (0.01s)
ok  	github.com/daileyo/omgitworks/internal/git	0.114s
```

## Design note: locks are not checked here

`RemoveWorktree` deliberately does **not** consult `IsWorktreeLocked`. Callers check it
before invoking, which lets the dry-run preview mark a locked entry accurately without
spending a failed subprocess, and makes it structurally impossible for `--force` to
override a lock — the force flag never reaches a locked worktree.

## Reviewer Conclusion

The primitive works, defers its safety judgement to git, and cannot delete a branch.
