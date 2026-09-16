# 27-audit-worktree-remove.md

## Executive Summary

- Overall Status: PASS
- Required Gate Failures: 0
- Flagged Risks: 2
- Audit runs: 1

All four REQUIRED gates pass. 27 functional requirements map to 5 parent tasks and 73
sub-tasks, with 27 planned test artifacts and 8 CLI/documentation artifacts. Two FLAG
findings are recorded; neither blocks implementation.

## Gateboard

| Gate | Status | Why it failed (<=10 words) | Exact fix target |
| --- | --- | --- | --- |
| Requirement-to-test traceability | PASS | — | — |
| Proof artifact verifiability | PASS | — | — |
| Repository standards consistency | PASS | — | — |
| Open question resolution | PASS | Both spec questions already resolved | — |
| Regression-risk blind spots | FLAG | Shared-selector move touches shipped spec 26 code | `## Tasks > 3.0 > 3.1` |
| Non-goal leakage | FLAG | Task 5.0 exceeds the spec's requirements | `## Tasks > 5.0` |

**Traceability evidence:** the spec's 27 `shall` statements (11 + 8 + 8 across three
units) each appear in the `## Requirement Coverage Map` mapped to a parent task carrying
at least one planned test artifact.

## Standards Evidence Table (Required)

| Source File | Read | Standards Extracted | Conflicts |
| --- | --- | --- | --- |
| `AGENTS.md` / `CONTRIBUTING.md` / PR template | not found | — | — |
| `README.md` | yes | Conventional Commits, `feat:` → minor; `make ci` is the gate; `cmd/` + `internal/` layout | none |
| `Makefile` | yes | `make ci` = vet + golangci-lint + `go test -race` | none |
| `.github/workflows/ci.yml` | yes | Go 1.24.x/1.25.x; ubuntu-only test job; lint pinned v2.13.2 | none |
| `.golangci.yml` | yes | gosec/noctx/unparam/misspell; goimports local prefix | none |
| `internal/git/worktree.go` | yes | `AddWorktree`, `MoveWorktree`, `IsWorktreeLocked(repoPath, worktreePath) (bool, string)`; **no** removal primitive | none |
| `cmd/omgitworks/worktree_align.go` | yes | `errors []string` collection, counted summary, dry-run shape; `removeEmptyLegacyDir` (line 282) conservative cleanup | none |
| `cmd/omgitworks/navigate.go` | yes | `runNavigate` (line 24) injects `stderr`/`stdout`/`stdin`; `bufio.NewScanner(stdin)` prompt (line 210); `isCharDevice` (line 233) | none |
| `cmd/omgitworks/list.go` | yes | `term.IsTerminal(int(os.Stdout.Fd()))` at line 112 | see note |
| `cmd/omgitworks/cd.go` | yes | `stdoutIsTerminalFunc` overridable-var pattern (line 17) for testable TTY checks | none |
| `cmd/omgitworks/main.go` | yes | Usage template already renders an `Aliases:` block (line 172) | see note |

**Two spec premises checked and qualified** (both verified directly against the tree,
neither is a conflict requiring a decision):

1. The spec says "the repository already performs TTY detection for `--color` handling in
   `list.go`". Correct — but it detects **stdout**. Unit 3 needs **stdin**. Sub-task 4.8
   applies the same call to `os.Stdin` and wraps it in an overridable variable following
   `cd.go`'s `stdoutIsTerminalFunc`, so the non-interactive path is testable. Two TTY
   approaches coexist in the tree (`term.IsTerminal` and `isCharDevice`); the plan picks
   the `x/term` one for consistency with `list.go`.
2. The spec says the `rm` alias uses "cobra's native `Aliases` field". Correct as an
   approach, but **no command in this repository sets that field today** — `remove` would
   be the first. The usage template already renders aliases, so display needs no work.
   Sub-task 2.18 asserts the alias resolves through cobra's command lookup rather than by
   reading the field, so the test proves behavior rather than restating configuration.

## Findings

### FLAG Findings

1. **Task 3.0 modifies code that has already shipped in an open PR.**
   - Risk: sub-task 3.1 moves `selectWorktreeAddTargets` out of `worktree_add.go` into
     `worktree.go` as `selectWorktreeTargets`, updating three call sites in spec 26's
     code. That code is under review in PR #101. If #101 changes during review, this
     branch must be rebased and the move re-applied; and if the move is done carelessly,
     it could alter `add` behavior that spec 26's validation already certified.
   - Mitigation already planned: the move is a pure rename and relocation, and sub-task
     3.2 requires spec 26's `TestWorktreeAddSelection_*` suite to pass **unmodified** as
     the behavior-preservation guard — if any assertion needs editing, the move was not
     behavior-preserving.
   - Suggested remediation: accept as planned, or defer task 3.0's extraction until #101
     merges and rebase onto `main` first. The second option removes the risk entirely at
     the cost of sequencing.

2. **Task 5.0 exceeds the spec's stated requirements.**
   - Risk: spec 27 defines three units, none covering tab completion or documentation.
     Task 5.0 adds both, as spec 26's task 4.0 did. Shipping the repository's most
     destructive command with no published documentation of its safety model would be a
     poor outcome, which is why it is included — but it is not a requirement of this spec.
   - Suggested remediation: user's choice, consistent with whatever was decided for spec
     26's task 4.0. Keep it, drop it, or keep documentation (5.4-5.8) and drop completion
     (5.1-5.3).

## Carried-Forward Defect Guard

Spec 26's validation found that deriving single-repository mode from `len(repos) == 1`
turns a legitimate skip into a hard error when a tag matches exactly one repository.
Removal has the identical shape — a tagged repository that lacks the branch must be a
skip, not a failure — so the same bug is available to be reintroduced here.

Sub-task 3.6 requires the flag to be derived from the invocation shape, and sub-task 3.13
(`TestWorktreeRemoveBulk_SingleMatchTagStillBulk`) is the explicit regression test. This
is recorded here so the connection is not lost between specs.

## User-Approved Remediation Plan

- Pending approval — no remediation is required for the REQUIRED gates. Awaiting the
  user's decision on the two FLAG findings.
