# Task 03 Proofs - Tag-scoped removal on the shared targeting model

## Task Summary

Spec 26's precedence table is now shared rather than duplicated, and `remove` adopts it
along with the partial-failure contract.

## What This Task Proves

- `selectWorktreeAddTargets` became `selectWorktreeTargets` in `worktree.go`, used by both
  `add` and `remove`, with **no behavior change**.
- `-t <tag> <branch>` removes across every tagged repository and none outside the tag.
- A tagged repository without that branch is a **skip**, not a failure.
- Mixed runs report removed/skipped/failed and exit non-zero; skip-only runs exit zero.
- Removals before a later failure are not restored.
- Configuration is saved once.

## Evidence Summary

- The extraction changed spec 26's tests by exactly **two lines** — the function name —
  with zero assertion changes, and its suite stays green.
- 11 bulk cases pass, including the spec 26 regression guard.

## Artifact: The extraction was behavior-preserving

**What it proves:** Moving the selector did not alter `add`.

**Why it matters:** Spec 26 is under review in PR #101 and its validation already
certified this behavior. The guard is that its own tests still pass without their
assertions being touched.

**Command:**

```bash
git diff --stat HEAD -- cmd/omgitworks/worktree_add_tag_test.go cmd/omgitworks/worktree_add_bulk_test.go
go test -run 'TestWorktreeAddSelection|TestWorktreeAddBulk|TestRunWorktreeAdd' ./cmd/omgitworks/
```

**Result summary:** Two files, one line each — `selectWorktreeAddTargets(` →
`selectWorktreeTargets(`. Every assertion unchanged, suite green.

```text
 cmd/omgitworks/worktree_add_bulk_test.go | 2 +-
 cmd/omgitworks/worktree_add_tag_test.go  | 2 +-
 2 files changed, 2 insertions(+), 2 deletions(-)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks
```

The tag-validation helper was shared the same way: `singleTagValue` now backs both
commands' `--tag`, so the repetition-rejection message cannot drift between them.

## Artifact: Bulk removal suite

**What it proves:** Tag selection, the shared precedence table, skip-versus-fail,
summary counts, exit status, no-restore, and single-save persistence.

**Command:**

```bash
go test -v -run 'TestWorktreeRemoveBulk|TestWorktreeRemoveTargets' ./cmd/omgitworks/
```

**Result summary:** All pass. `TestWorktreeRemoveTargets_PrecedenceTable` asserts
`remove` selects the same sets `add` does, row for row.

```text
--- PASS: TestWorktreeRemoveBulk_RemovesAcrossTag (0.09s)
--- PASS: TestWorktreeRemoveTargets_PrecedenceTable (0.03s)
    --- PASS: TestWorktreeRemoveTargets_PrecedenceTable/1_positional,_no_tag_->_current_repo (0.00s)
    --- PASS: TestWorktreeRemoveTargets_PrecedenceTable/1_positional_with_tag_->_all_tagged (0.00s)
    --- PASS: TestWorktreeRemoveTargets_PrecedenceTable/2_positionals,_no_tag_->_name_pattern (0.00s)
    --- PASS: TestWorktreeRemoveTargets_PrecedenceTable/2_positionals_with_tag_->_name_AND_tag (0.00s)
--- PASS: TestWorktreeRemoveBulk_SkipsMissingBranch (0.05s)
--- PASS: TestWorktreeRemoveBulk_SingleMatchTagStillBulk (0.02s)
--- PASS: TestWorktreeRemoveBulk_SummaryAndExit (0.06s)
    --- PASS: TestWorktreeRemoveBulk_SummaryAndExit/mixed_run_reports_counts_and_fails (0.05s)
    --- PASS: TestWorktreeRemoveBulk_SummaryAndExit/skips_only_returns_nil (0.01s)
--- PASS: TestWorktreeRemoveBulk_NoRestore (0.06s)
--- PASS: TestWorktreeRemoveBulk_SingleSave (0.09s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.414s
```

## Carried-forward defect guard

Spec 26's validation found that deriving single-repository mode from `len(repos) == 1`
turns a legitimate skip into a hard error when a tag matches exactly one repository.
Removal has the identical shape, so the bug was available to be reintroduced.

It is not: mode is derived from the invocation, and
`TestWorktreeRemoveBulk_SingleMatchTagStillBulk` asserts a single-match tag run with a
missing branch skips and returns nil.

## Reviewer Conclusion

One targeting table now serves both commands, and removal inherits spec 26's
partial-failure contract along with the guard against its known defect.
