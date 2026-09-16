# 25-audit-worktree-repo-context.md

## Executive Summary

- Overall Status: PASS
- Required Gate Failures: 0
- Flagged Risks: 0 open (2 raised in run 1, both resolved in run 2)
- Audit runs: 2

Run 1: all four REQUIRED gates passed; two FLAG findings raised. Run 2: both flags
resolved by user-approved remediation. The plan is cleared for the implementation phase.

## Gateboard

| Gate | Status | Why it failed (<=10 words) | Exact fix target |
| --- | --- | --- | --- |
| Requirement-to-test traceability | PASS | — | — |
| Proof artifact verifiability | PASS | — | — |
| Repository standards consistency | PASS | — | — |
| Open question resolution | PASS | — | — |
| Regression-risk blind spots | PASS | Resolved in run 2 | — |
| Non-goal leakage | PASS | Resolved in run 2 | — |

**Traceability evidence:** all 23 functional requirements across the spec's three units
appear in the `## Requirement Coverage Map`, each with at least one named sub-task and at
least one planned test artifact. No requirement maps to documentation alone except
FR U3-5, which is additionally verified by the `--help` cross-check in sub-task 5.6.

## Standards Evidence Table (Required)

| Source File | Read | Standards Extracted | Conflicts |
| --- | --- | --- | --- |
| `AGENTS.md` | not found | — | — |
| `CONTRIBUTING.md` | not found | — | — |
| `.github/pull_request_template.md` | not found | — | — |
| `README.md` | yes | Conventional Commits, `feat:` triggers a minor bump; `make ci` is the full gate; `cmd/omgitworks/` + `internal/<pkg>/` layout | see note below |
| `Makefile` | yes | `make test` = `go test -v ./...`; `make ci` = vet + lint + test-race; `make setup-hooks` installs the pre-push gate | none |
| `.github/workflows/ci.yml` | yes | Go 1.24.x and 1.25.x test matrix on ubuntu only; cross-platform job builds and runs `--help` but does not test | none |
| `.golangci.yml` | yes | v2 config; `gosec`, `noctx`, `unparam`, `misspell` enabled; `goimports` local prefix `github.com/daileyo/omgitworks`; test files exempt from `gosec`/`noctx`/`unparam` | none |
| `.githooks/commit-msg` | yes | Pads unscoped `type:` and `type!:` subjects to a fixed column; warns and passes through anything else | see note below |
| `docs/site/commands-core.md` | yes | Published reference is updated alongside command changes, per the spec 18 precedent | none |

**Conflict noted, precedence documented.** `README.md` prescribes Conventional Commits with
an optional scope, but `.githooks/commit-msg` only recognizes unscoped `type:` / `type!:`
subjects and leaves scoped subjects unpadded. Recent history contains both forms
(`feat(worktree): relocate worktrees to the projects root` unpadded,
`fix(user):          run git config with a context` padded by hand). **Decision:** commit
this work as `feat(worktree): ...`, matching the established scope for worktree changes,
and accept that the hook will not pad it. Recorded in the task file's `### Notes`. The
padding is cosmetic and does not affect Release Please version resolution.

## Findings (Run 1, both now resolved)

### FLAG Findings

1. **Windows path comparison is untested where the plan relies on exact string equality.** — RESOLVED
   - Risk: `git rev-parse --show-toplevel` returns forward-slash paths on Windows
     (`C:/Users/...`) while stored `Repository.Path` values use backslashes. Sub-task 1.5
     routes the git output through `git.ResolvePath`, which normalizes separators via
     `filepath.Clean`/`EvalSymlinks`, so the separator case is handled. Drive-letter case
     (`C:` versus `c:`) is **not** normalized by either function, and exact comparison is
     case-sensitive, so a config written with one casing and a git result with the other
     would fail to resolve. CI's test job runs on ubuntu only; the Windows job builds the
     binary and runs `--help` without executing the suite, so no existing gate would catch
     this.
   - Suggested remediation: add a sub-task under 1.0 for a `runtime.GOOS == "windows"`
     guarded test asserting resolution succeeds across drive-letter case differences, or
     an explicit `strings.EqualFold` comparison on Windows in `Resolve`. Alternatively,
     accept the risk and record it in the spec's Technical Considerations.

2. **Sub-task 2.5 instructs rewriting stale `gws` examples to `omgw`, which is outside the
   spec's goals.** — RESOLVED
   - Risk: `gws` appears on 164 lines across 18 non-test files in `cmd/omgitworks/`. Changing
     only the `worktree add` help text leaves the CLI internally inconsistent, and changing
     all of it is a repo-wide rebrand cleanup that belongs to its own spec, not to spec 25.
     The spec's Non-Goals do not authorize it, and no functional requirement covers it.
   - Suggested remediation: strike the `omgw`-prefix clause from sub-task 2.5, leaving it
     to add only the omitted-repository form and the worktree-relative note in the existing
     `gws`-prefixed style. Track the rebrand cleanup as a separate spec.

## User-Approved Remediation Plan

- Completed

| # | Finding | User decision | Edit applied |
| --- | --- | --- | --- |
| 1 | Windows drive-letter case untested | "add a `runtime.GOOS == \"windows\"` guarded test" | Sub-task 1.6 now specifies a `samePath` helper using `strings.EqualFold` on Windows; 1.7 reuses it; new sub-task 1.16 is the guarded test; old 1.16 renumbered to 1.17 |
| 2 | Sub-task 2.5 rebrands help text | "don't do the change in 2. that is going to have to be a dedicated documentation/help text fix later" | The `omgw`-prefix clause is struck from 2.5, which now directs keeping the existing `gws` examples and notes the rebrand is a separate change |

## Re-Audit Delta (Run 2)

Changed gate statuses since run 1:

- Regression-risk blind spots: FLAG -> PASS. A `runtime.GOOS == "windows"` guarded test
  (`TestResolve_WindowsDriveLetterCase`) is now planned in sub-task 1.16, and the
  comparison rule it pins is specified in sub-task 1.6, so the risk is covered by a
  planned test artifact rather than accepted silently.
- Non-goal leakage: FLAG -> PASS. Sub-task 2.5 no longer edits branding text, so no task
  exceeds the spec's goals/non-goals boundary.

Still-failing REQUIRED gates: none.

Newly introduced findings: none. The four REQUIRED gates were re-checked against the
edited task file:

- Traceability: still 23 requirements to 23 coverage rows; the exact-comparison row now
  cites `TestResolve_WindowsDriveLetterCase` in addition to
  `TestResolve_NestedRepoNotPrefixMatched`.
- Proof artifact verifiability: the added artifact names an exact command and states the
  observable result on both Windows (pass) and other platforms (skip).
- Standards consistency: unchanged; the documented commit-message precedence still holds.
- Open questions: unchanged; none introduced by the remediation.

Sub-task count: 60 -> 61.
