# Task 04 Proofs - Tag completion, documentation, and a stale flag description

> **Scope note:** spec 26 states no functional requirement for completion or
> documentation. This task was included on the repository's convention that
> `docs/site/` moves with behavior, and was flagged as out-of-spec in the planning audit.

## Task Summary

`--tag` now completes to the tags actually in use, the published reference documents the
flag and the precedence table, and a stale flag description in `list.go` that contradicted
both the code and the docs is corrected.

## What This Task Proves

- `--tag` completes to the deduplicated set of tags across tracked repositories.
- `commands-core.md` documents the `-t` form, the four-row precedence table, the
  single-value constraint, and the partial-failure contract.
- `list.go`'s `--tag` description no longer claims to be repeatable when it is not.
- The complete change passes `make ci`.

## Artifact: Tag completion

**What it proves:** The flag value completes to real tags rather than falling back to
file completion.

**Command:**

~~~bash
omgw __complete worktree add --tag ""
go test -v -run TestCompleteWorktreeAddTag ./cmd/omgitworks/
~~~

**Result summary:** Both tags in the sandbox are offered, deduplicated across the three
repositories that carry `backend`.

~~~text
$ omgw __complete worktree add --tag ""
backend
frontend
:4

--- PASS: TestCompleteWorktreeAddTag (0.03s)
    --- PASS: .../suggests_every_tag_once (0.00s)
    --- PASS: .../filters_on_the_typed_prefix (0.00s)
    --- PASS: .../registered_on_the_tag_flag (0.00s)
~~~

## Artifact: Documentation

**What it proves:** The published reference matches shipped behavior.

**Artifact path:** `docs/site/commands-core.md`

**Result summary:** The `worktree add` signature becomes
`omgw worktree add [repo] <branch> [-t <tag>]`, and a new
*Creating across many repositories* subsection adds bulk examples, the single-value
constraint on `--tag`, the four-row precedence table reproduced from the spec, the
ambiguity rule (rejected without `-t`, bulk with it), and a worked partial-failure
example stating that skips are not failures, nothing is rolled back, and the command
exits non-zero if anything failed.

## Artifact: Correcting the stale `list --tag` description

**What it proves:** Code, help text, and published documentation now agree.

**Why it matters:** `cmd/omgitworks/list.go:169` declared `--tag` as a single-valued
`StringVarP` while describing it as "(repeatable for AND logic)". Spec 26's Non-Goal 3
relied on that description, asserting `omgw list -t` "is repeatable" and that this spec
therefore diverges from it. **The premise was false** — `list -t` has always been
single-valued, so spec 26's `--tag` is consistent with it, not a divergence.

**Result summary:** The description now reads `Filter by tag (single value)`, matching
the code and what `docs/site/` already documented.

~~~diff
-  listCmd.Flags().StringVarP(&flagTag, "tag", "t", "", "Filter by tag (repeatable for AND logic)")
+  listCmd.Flags().StringVarP(&flagTag, "tag", "t", "", "Filter by tag (single value)")
~~~

## Artifact: Full repository gate

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit 0. 415 top-level tests pass, 0 fail; `golangci-lint` reports
`0 issues`. The baseline before this spec was 397, so 18 top-level tests were added.

~~~text
make ci EXIT=0
PASS=415 FAIL=0
0 issues.
~~~

## Reviewer Conclusion

The new flag is discoverable through completion and documented alongside its precedence
rules, and a long-standing contradiction between `list`'s flag and its help text is gone.
