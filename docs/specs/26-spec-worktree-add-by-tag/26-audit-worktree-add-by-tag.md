# 26-audit-worktree-add-by-tag.md

## Executive Summary

- Overall Status: PASS
- Required Gate Failures: 0
- Flagged Risks: 2
- Audit runs: 1

All four REQUIRED gates pass. Two FLAG findings and one explicit planning assumption are
recorded below; none blocks implementation, but the assumption and both flags are
decisions the user should confirm before task 1.1.

## Gateboard

| Gate | Status | Why it failed (<=10 words) | Exact fix target |
| --- | --- | --- | --- |
| Requirement-to-test traceability | PASS | — | — |
| Proof artifact verifiability | PASS | — | — |
| Repository standards consistency | PASS | — | — |
| Open question resolution | PASS | Resolved by documented assumption | — |
| Regression-risk blind spots | FLAG | Branch based on pre-spec-25 commit | `## Prerequisite` |
| Non-goal leakage | FLAG | Task 4.0 exceeds spec's requirements | `## Tasks > 4.0` |

**Traceability evidence:** all 21 `shall` requirements across the spec's three units
appear in the `## Requirement Coverage Map`, each mapped to a parent task carrying at
least one planned test artifact. 18 test artifacts and 8 CLI/documentation artifacts are
planned across 4 parent tasks and 54 sub-tasks.

## Standards Evidence Table (Required)

| Source File | Read | Standards Extracted | Conflicts |
| --- | --- | --- | --- |
| `AGENTS.md` | not found | — | — |
| `CONTRIBUTING.md` | not found | — | — |
| `.github/pull_request_template.md` | not found | — | — |
| `README.md` | yes | Conventional Commits, `feat:` → minor bump; `make ci` is the gate; `cmd/` + `internal/` layout | none |
| `Makefile` | yes | `make ci` = vet + golangci-lint + `go test -race`; `pluralize` pattern via `init.go` | none |
| `.github/workflows/ci.yml` | yes | Go 1.24.x/1.25.x; ubuntu-only test job; golangci-lint pinned v2.13.2 | none |
| `.golangci.yml` | yes | gosec/noctx/unparam/misspell enabled; goimports local prefix `github.com/daileyo/omgitworks` | none |
| `internal/filter/filter.go` | yes | `MatchesExact` (line 55, case-insensitive + glob) for tags; `MatchesPattern` (line 36, partial) for names — both exist as the spec assumes | none |
| `cmd/omgitworks/worktree_align.go` | yes | Bulk-run output shape: per-item lines, counted summary, separately headed error block, `errors []string` collection | none |
| `cmd/omgitworks/list.go` | yes | `--tag`/`-t` is `StringVarP`, single-valued | **see below** |

**Conflict noted, decision documented.** `cmd/omgitworks/list.go:169` declares `--tag` as
a single-valued `StringVarP` but describes it as "(repeatable for AND logic)". The spec's
Non-Goal 3 relies on that stale description, asserting that `omgw list -t` "is
repeatable" and that spec 26 therefore diverges from it. **The code contradicts the
spec's premise:** `list -t` is single-valued, so spec 26's single-valued `--tag` is
consistent with it, not a divergence. **Decision:** implement `--tag` as single-valued as
specified — the requirement is unaffected — and correct the stale `list.go` description
in sub-task 4.7 so code, help text, and published docs agree.

## Findings

### FLAG Findings

1. **The implementation branch is based on a commit that predates its own dependency.**
   - Risk: `feat/worktree-add-by-tag` is at `aa04200`. Verified directly: `internal/repocontext`
     does not exist on this branch and `worktree_add.go` still declares
     `cobra.ExactArgs(2)`. Spec 26 extends `addWorktreeForRepo` and relies on
     `repocontext.ResolveCurrent` for the one-positional-no-tag row of the precedence
     table. Beginning implementation on this base would either fail immediately or
     tempt a reimplementation of spec 25's resolver, which that spec's Technical
     Considerations explicitly forbid ("If specs are implemented out of order, that row
     must be deferred rather than reimplemented here").
   - Suggested remediation: sub-task 1.1 already requires rebasing onto
     `feat/worktree-repo-context` (or onto `main` after spec 25 merges) and re-running
     `make ci` for a green baseline. Confirm which base to use before starting, since
     spec 25 is not yet merged.

2. **Task 4.0 exceeds the spec's stated requirements.**
   - Risk: spec 26 defines three units and none of them covers tab completion or
     documentation, unlike spec 25's Unit 3. Task 4.0 adds `--tag` completion, four
     documentation sub-tasks, and sub-task 4.7, which edits `cmd/omgitworks/list.go` —
     a file outside this spec's scope entirely. Against that, the repository's
     convention is that `docs/site/` moves with behavior; spec 18 exists precisely
     because documentation drifted, and spec 25 shipped its docs in the same branch.
     Shipping a new user-facing flag with no published documentation would repeat the
     drift spec 18 was created to fix.
   - Suggested remediation: user's choice. Keep 4.0 as planned; or drop it and track
     documentation as a follow-up spec; or keep 4.4-4.6 (documentation only) and drop
     4.1-4.3 (completion) and 4.7 (the `list.go` fix). The task file already marks 4.0
     as out-of-spec in a scope note.

## Planning Assumption Requiring Confirmation

The spec is internally ambiguous about one interaction, and the plan resolves it
explicitly rather than leaving it open. This is recorded here because it changes
behavior, not just wording.

**Question:** with two positionals *and* `-t`, when the name-and-tag intersection matches
more than one repository — error, or bulk run?

| Source | Implication |
| --- | --- |
| FR U2-6: "the two-positional form **without `-t`** shall continue to reject an ambiguous name pattern" | Ambiguity error is scoped to the no-tag form |
| Technical Considerations: "the ambiguity check must run **after** the tag filter is applied, not before" | Implies the check still applies when `-t` is present |
| User story: "narrow a tag to a subset by also giving a name pattern so that I can **act on part of a tagged group**" | Implies acting on several repositories |

**Assumption taken:** the user story governs. `add <repo> <branch> -t <tag>` performs a
bulk run over the intersection with no ambiguity error; the ambiguity error is preserved
only for the two-positional form without `-t`. Implemented by sub-tasks 1.8 and 1.9 and
pinned by `TestWorktreeAddSelection_AmbiguityAfterTagFilter`.

**Blast radius if wrong:** sub-tasks 1.8, 1.9, and one test. No other task changes.

## User-Approved Remediation Plan

- Pending approval — no remediation edits required for the REQUIRED gates. Awaiting the
  user's decisions on the two FLAG findings and the planning assumption above.
