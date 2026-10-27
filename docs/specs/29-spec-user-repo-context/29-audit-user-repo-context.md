# 29-audit-user-repo-context.md

## Executive Summary

- Overall Status: PASS
- Required Gate Failures: 0
- Flagged Risks: 0 (1 raised in run 1, remediated in run 2)

Run 1: all four REQUIRED gates passed. One FLAG finding raised, concerning a code
path that task 1.0 moves without a test on either side of the move.
Run 2: remediation approved and applied; the flag is closed.

## Gateboard

| Gate | Status | Why it failed (<=10 words) | Exact fix target |
| --- | --- | --- | --- |
| Requirement-to-test traceability | PASS | — | — |
| Proof artifact verifiability | PASS | — | — |
| Repository standards consistency | PASS | — | — |
| Open question resolution | PASS | — | — |
| Regression-risk blind spots | PASS (run 2) | run 1: `--use-subdirs` real run moved untested | `## Tasks > 1.0` (1.8a added) |
| Non-goal leakage | PASS | — | — |

## Standards Evidence Table

| Source File | Read | Standards Extracted | Conflicts |
| --- | --- | --- | --- |
| `README.md` | yes | `make ci` is the full gate; Conventional Commits drive Release Please; `make setup-hooks` installs hooks; `cmd/omgitworks/` + `internal/<pkg>/` layout | none |
| `Makefile` | yes | `ci` = vet + golangci-lint + `go test -race`; `build` emits `./build/omgitworks`; `use-dev` installs a dev binary | none |
| `.golangci.yml` | yes | v2 config; gosec, noctx, unparam, govet relaxed in `_test.go`; goimports local prefix `github.com/daileyo/omgitworks`; gofmt enforced | none |
| `.github/workflows/ci.yml` | yes | Go 1.24.x / 1.25.x matrix; tests run with `-race` and coverage; lint pinned to golangci-lint v2.13.2 | none |
| `.githooks/` | yes | `commit-msg` pads conventional types; `pre-push` runs vet, lint, tests | none |
| `AGENTS.md` | not found | — | — |
| `CLAUDE.md` | not found | — | — |
| `CONTRIBUTING.md` | not found | — | — |
| `.github/pull_request_template.md` | not found | — | — |
| `.pre-commit-config.yaml` | not found | — | — |

Five guideline sources were read, above the two-source minimum. Neither `AGENTS.md`
nor `CONTRIBUTING.md` exists; the root `README.md` was reviewed. No conflicts between
sources.

## Requirement-to-Test Traceability

Every functional requirement maps to at least one sub-task and one planned test or
observable artifact. Verified requirement by requirement, not by sampling.

| Spec requirement | Sub-tasks | Planned artifact |
| --- | --- | --- |
| U1-1 one-arg form applies to `ResolveCurrent` result | 2.2, 2.4, 2.6 | `TestUserAssign_OneArg_FromRepoRoot`, `_FromSubdirectory` |
| U1-2 two-arg form unchanged, multi-match error preserved | 1.5–1.8, 2.4 | `TestUserAssign_TwoArgs_AssignsNamedRepo`, `_MultiMatchError`, `_DryRunOutput` |
| U1-3 resolver consulted only when argument omitted | 2.4, 2.10 | `TestUserAssign_TwoArgs_ResolverNotConsulted` |
| U1-4 inside a worktree, write lands on owning repository | 2.8 | `TestUserAssign_OneArg_FromInsideWorktree` + `assign-from-worktree.md` |
| U1-5 `ErrNotTracked` returned unchanged | 2.4, 2.11 | `TestUserAssign_OneArg_NotTracked` |
| U1-6 repository resolved before profile looked up | 2.5, 2.12, 1.7 | `TestUserAssign_OneArg_UnknownProfile_ReportsProfileError`, `TestUserAssign_TwoArgs_UnknownProfileError` |
| U1-7 `--dry-run` names resolved repository | 2.14 | `TestUserAssign_OneArg_DryRunNamesResolvedRepo` |
| U1-8 same success output, no "inferred" line | 2.9 | `TestUserAssign_OneArg_OutputMatchesTwoArgForm` |
| U1-9 `--use-subdirs` refused before any change, incl. dry-run | 2.3, 2.13 | `TestUserAssign_OneArg_UseSubdirsRefused` (2 sub-cases) |
| U1-10 arity one or two; zero or three fail | 2.1, 2.15 | `TestUserAssign_ArgCount` |
| U2-1 first-arg completion: profiles when resolved | 3.1, 3.4 | `TestCompleteUserAssign_ProfilesWhenResolved` |
| U2-2 first-arg completion: repositories when not resolved | 3.1, 3.5 | `TestCompleteUserAssign_ReposWhenNotResolved` |
| U2-3 second-arg completion: profiles | 3.1, 3.6 | `TestCompleteUserAssign_ProfilesForSecondArg` |
| U2-4 `Use` line and `assign` `Long` | 4.1, 4.2, 4.4, 4.5 | `TestUserAssign_HelpTextUsesOmgw`, `_HelpDocumentsBothForms` |
| U2-5 `user` parent `Long` uses `omgw`, shows one-arg example | 4.3, 4.4, 4.5 | same two tests |
| U2-6 `commands-user.md` updated | 4.6, 4.8 | `docs-diff.md` + `grep` of the usage line |
| U2-7 `shellinit.go` needs no change | 4.7 | existing `shellinit_test.go` / `shellinit_exec_test.go` pass unmodified |

Spec success metrics 1 through 6 are each covered by the rows above; metric 2
(identical output for every current invocation) rests on the task 1.0
characterization tests plus 2.9.

## Findings

### FLAG Findings

1. **The `--use-subdirs` real-run path is moved by task 1.0 with no test before or after.**
   - Risk: task 1.0 relocates the entire `RunE` body, including the branch that
     prompts, calls `user.AssignWithSubdirs`, moves the repository, and edits
     `~/.gitconfig`. The characterization tests in 1.5–1.8 cover local assignment,
     the multi-match error, the unknown-profile error, and dry run, but not this
     branch. Task 2.3 then inserts a refusal immediately ahead of it. A transcription
     slip in the move, or a refusal condition written as `opts.useSubdirs` without
     the `repoArg == ""` guard, would break the two-argument `--use-subdirs` form,
     and nothing planned would catch it. The path is reachable in tests: the fixture
     already uses a temporary `HOME`, so the `~/.gitconfig` edit is sandboxed, and
     the prompt reads `os.Stdin`, which `worktree remove`'s tests already redirect
     via `removeWithInput`.
   - Suggested remediation: add sub-task 1.8a, `TestUserAssign_TwoArgs_UseSubdirsStillWorks`,
     which pipes `y\n` to stdin, runs `[]string{"alpha", "work"}` with
     `useSubdirs: true`, and asserts the repository now lives at
     `<workspace>/work/alpha`, the stored `Path` was updated, and
     `$HOME/.gitconfig` contains an `includeIf` line. Add a matching proof artifact
     under 1.0. Alternatively, accept the flag and verify the path by hand once
     after task 2.0, recording the transcript in `29-proofs/`.

## User-Approved Remediation Plan

- Completed. The user approved adding the characterization test for the
  `--use-subdirs` path. Applied as sub-task 1.8a with a matching proof artifact
  under task 1.0. Because the prompt read `os.Stdin` directly, sub-task 1.2 was
  amended so `runUserAssign` takes an `io.Reader`, the seam `runWorktreeRemove`
  already uses, and sub-task 1.5's example call was updated to match.

## Chain-of-Verification

1. Initial assessment: four REQUIRED gates evaluated against the spec, the task
   file, and five standards sources; draft findings recorded.
2. Self-questioning: each REQUIRED gate was re-checked for explicit evidence.
   Traceability was verified requirement by requirement, giving the 17/17 table
   above.
3. Fact-checking: helper and constant names used by the sub-tasks were checked
   against the code rather than assumed. `configBytes` exists at
   `worktree_remove_confirm_test.go:51`; `config.UserSourceLocal` exists at
   `config.go:53`; the `### Current-Repository Targeting` anchor the docs task links
   to exists at `commands-core.md:746`; `chdirForTest`, `setupWorktreeTestRepo`,
   `projectsPath`, `runWorktreeAdd`, and `captureStdoutStr` were all read in source.
   The FLAG finding was verified by reading `userAssignCmd`'s `RunE`: the
   `--use-subdirs` branch spans the prompt, `AssignWithSubdirs`, and
   `config.Save`, and no planned test exercises it.
4. Inconsistency resolution: an earlier draft of the planning notes said the
   spec's "name-pattern lookup" meant `filter.MatchesPattern`. Reading
   `findRepositories` (`tag.go:398`) showed it is a case-insensitive substring
   matcher, so the task file now forbids swapping it for `selectByName`, which
   would change behavior that the spec requires to stay identical. A second
   draft claim, that `deprecated_test.go` reads `userCmd.Long`, was checked and
   found wrong: its `gws user list` string comes from `deprecated.go:113`. The
   planning note was corrected to say so.
5. Final synthesis: PASS with zero REQUIRED failures and one FLAG finding. The
   flag concerns an existing feature the spec does not change, moved by a
   refactor the spec sanctions. It does not block implementation.

## Re-Audit Delta (Run 2)

Triggered by approved remediation of run 1's flag 1.

- **Changed gate statuses:** Regression-risk blind spots FLAG → PASS. Sub-task
  1.8a adds `TestUserAssign_TwoArgs_UseSubdirsStillWorks` (accept and cancel
  sub-cases), so the `--use-subdirs` branch is pinned before task 1.0 moves it
  and before task 2.3 inserts the refusal ahead of it.
- **Still-failing REQUIRED gates:** none.
- **Newly introduced findings:** none. The `io.Reader` parameter added to
  `runUserAssign` by the amended sub-task 1.2 was checked against the code:
  `userAssignCmd`'s prompt is the only `os.Stdin` read in the command, and
  `runWorktreeRemove` already takes a reader the same way, so the change
  introduces no new pattern. Traceability stays 17/17; the new test strengthens
  U1-2 and U1-9 rather than mapping a new requirement.
- **Fact-check of the new test's assertions:** `AssignWithSubdirs`
  (`internal/user/assign.go:230`) moves the repository to
  `<workspace>/<profile>/<repo>`, writes `includeIf` into
  `$HOME/.gitconfig`, and sets `UserSource` to `config.UserSourceIncludeIf`,
  which is what 1.8a asserts. The fixture's temporary `HOME` keeps the
  gitconfig write sandboxed.
