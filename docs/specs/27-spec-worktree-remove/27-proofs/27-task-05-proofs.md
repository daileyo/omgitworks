# Task 05 Proofs - Completion, documentation, and the full gate

> **Scope note:** spec 27 states no functional requirement for completion or
> documentation. Included on the repository's convention that `docs/site/` moves with
> behavior; flagged in the planning audit.

## Task Summary

`remove` completes to branches that actually have worktrees, the published reference
documents the command and its safety model, and the whole change clears `make ci`.

## What This Task Proves

- Completion suggests only removable branches, never a branch without a worktree.
- `commands-core.md` documents the signature, `rm` alias, safety model, preview and
  confirmation rules, and partial-failure behavior.
- `--help` shows the alias and matches the documentation.
- `make ci` passes on the complete change.

## Artifact: Removal completion

**What it proves:** The first argument completes to branches with worktrees in the
resolved repository, falls back to repository names outside one, and filters on prefix.

**Why it matters:** Completing to a branch with no worktree could only ever produce an
error, so the useful set here is narrower than "all branches".

**Command:**

```bash
go test -v -run TestCompleteWorktreeRemove ./cmd/omgitworks/
```

**Result summary:** Passes, including a case asserting `main` — a branch with no
worktree — is **not** offered.

## Artifact: Documentation

**Artifact path:** `docs/site/commands-core.md`

**Result summary:** A new *Remove Worktrees* section documents the signature and `rm`
alias, cross-references the shared precedence table rather than restating it, and sets out
the safety model as four explicit guarantees: git's refusal on dirty worktrees with
`--force` as the only override; locks never overridden by `--force`; branches never
deleted; only genuinely empty directories removed. It then covers `--dry-run`, `--yes`,
the more-than-one-worktree confirmation rule, the non-interactive refusal, and the
partial-failure contract, with a worked dry-run example.

## Artifact: Help output

**Result summary:** The usage line and alias match the documentation.

```text
Remove the worktree for a branch, in one repository or across a tagged group.

The branch itself is never deleted: removing a checkout is not the same as
discarding the work on it.

Uncommitted or untracked changes block removal, because git itself refuses.
```

## Artifact: Full repository gate

**Command:**

```bash
make ci
```

**Result summary:** Exit 0. **448 top-level tests pass, 0 fail**; `golangci-lint` reports
`0 issues`. The baseline before this spec was 416, so 32 top-level tests were added.

## Reviewer Conclusion

The most destructive command in the tool ships with its safety model documented, its
completion narrowed to valid targets, and the full CI gate green.
