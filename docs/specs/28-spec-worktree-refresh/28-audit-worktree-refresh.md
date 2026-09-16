# 28-audit-worktree-refresh.md

## Executive Summary

- Overall Status: PASS
- Required Gate Failures: 0
- Flagged Risks: 2

Run 1: all four REQUIRED gates passed. Two FLAG findings raised, both concerning
regression risk introduced by the task 1.0 refactor on call sites that spec 28
does not otherwise touch.

## Gateboard

| Gate | Status | Why it failed (<=10 words) | Exact fix target |
| --- | --- | --- | --- |
| Requirement-to-test traceability | PASS | — | — |
| Proof artifact verifiability | PASS | — | — |
| Repository standards consistency | PASS | — | — |
| Open question resolution | PASS | — | — |
| Regression-risk blind spots | FLAG | Align path change untested | `## Tasks > 1.0` |
| Non-goal leakage | PASS | Refactor sanctioned by spec metric 5 | — |

## Standards Evidence Table

| Source File | Read | Standards Extracted | Conflicts |
| --- | --- | --- | --- |
| `README.md` | yes | Conventional Commits drive versioning; `make ci` is the full gate; `cmd/omgitworks/` + `internal/<pkg>/` layout | none |
| `Makefile` | yes | `ci` = vet + lint + test-race; `test` = `go test -v ./...`; `setup-hooks` installs the pre-push and commit-msg hooks | none |
| `.golangci.yml` | yes | v2 config; gosec/noctx/unparam enabled and relaxed in `_test.go`; goimports local-prefix `github.com/daileyo/omgitworks` | none |
| `.github/workflows/ci.yml` | yes | Go 1.24.x/1.25.x matrix; tests run with `-race` and coverage; lint is a separate job | none |
| `AGENTS.md` | not found | — | — |
| `CONTRIBUTING.md` | not found | — | — |
| `.github/pull_request_template.md` | not found | — | — |

Four guideline sources were read, above the two-source minimum. `AGENTS.md` does
not exist in this repository; the root `README.md` was reviewed.

## Requirement-to-Test Traceability

Every functional requirement in the spec maps to at least one task and one
planned test artifact. Summary by unit:

| Spec Unit | Requirements | Mapped Tasks | Coverage |
| --- | --- | --- | --- |
| Unit 1 — Scoped re-sync | 9 | 1.1, 1.2, 2.1, 2.4–2.8 | 9/9 |
| Unit 2 — Targeting and reporting | 9 | 3.1–3.5, 4.2, 4.4, 4.5 | 9/9 |
| Unit 3 — Dry run | 5 | 5.1–5.5 | 5/5 |
| Repository Standards — single helper | 1 | 1.1–1.6 | 1/1 |

## Findings

### FLAG Findings

1. **The task 1.0 refactor silently changes `worktree align` behavior, and only `add` has a new test for it.**
   - Risk: converting `worktree_align.go:232` to `syncRepoWorktrees` gives the
     align path three behaviors it does not have today — skipping worktrees
     whose directory is missing, setting `Worktrees` to nil rather than an empty
     slice when none remain, and the stricter list-error handling. Because
     `Repository.Worktrees` is tagged `omitempty`, the nil-versus-empty change is
     observable in `config.json`. Task 1.0 currently asserts only that the
     *existing* align tests still pass, which cannot detect a change those tests
     never covered. Task 1.8 adds a new test for the equivalent change on the
     `add` path but nothing does so for `align`.
   - Suggested remediation: add a sub-task under 1.0 mirroring 1.8 for align —
     a test that an align run over a repository with one missing-directory
     worktree drops that entry and writes no empty array — and add a matching
     proof artifact under 1.0.

2. **Task 1.5 offers a choice whose riskier branch has no test.**
   - Risk: the repair + prune pass at `worktree_align.go:87` is not redundant
     with the rebuild at line 232. It runs before the loop that reads the
     *stored* `repo.Worktrees` and issues `git worktree move` calls, so it exists
     to fix broken git pointers before those moves are attempted, not to refresh
     stored data. Removing it — the first branch task 1.5 offers — would make
     align fail to move worktrees whose pointers are broken but whose
     directories are intact, which is exactly the case the repair-before-prune
     requirement is meant to protect. No planned test would catch this.
   - Suggested remediation: narrow task 1.5 to keeping the pass and documenting
     why, dropping the removal option; or, if removal stays on the table, add a
     test that align successfully moves a worktree with a broken pointer.

## User-Approved Remediation Plan

- Pending approval

## Chain-of-Verification

1. Initial assessment: four REQUIRED gates evaluated against the spec, the task
   file, and the four standards sources; draft findings recorded.
2. Self-questioning: each REQUIRED gate was re-checked for explicit evidence.
   Traceability was verified requirement by requirement rather than by sampling,
   giving the 24/24 total in the table above.
3. Fact-checking: both FLAG findings were verified against the code rather than
   inferred. `worktree_align.go:86-125` was read to confirm the line 87 pass
   precedes a loop over stored `repo.Worktrees` and `git.IsWorktreeLocked` /
   move calls, confirming finding 2. `internal/config/config.go:86` was read to
   confirm the `omitempty` tag that makes the nil-versus-empty change
   observable, confirming finding 1.
4. Inconsistency resolution: an earlier draft proof artifact for task 5.0
   proposed asserting dry-run non-mutation "against a recording git runner".
   `internal/git` was checked and has no injectable seam — `gitCommand` calls
   `exec` directly — so that artifact was unachievable as written and was
   replaced with two observable side-effect assertions before this audit ran.
   An earlier draft also repeated the spec's claim that the discovery sequence
   exists in three identical copies; the code shows two full copies plus one
   partial, recorded in the task file's planning notes.
5. Final synthesis: PASS with zero REQUIRED failures and two FLAG findings. Both
   flags concern regression risk on call sites outside spec 28's feature
   surface, introduced by the spec-sanctioned consolidation. Neither blocks
   implementation.
