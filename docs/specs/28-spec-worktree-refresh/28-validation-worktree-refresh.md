# 28-validation-worktree-refresh.md

## 1) Executive Summary

- **Overall:** PASS. No gates tripped. Gate A: 0 CRITICAL, 0 HIGH. Gates B–F pass.
- **Implementation Ready:** **Yes.** All 23 functional requirements and the shared-helper standard are verified by passing tests and re-run CLI evidence. The one usability gap found is outside spec 28's requirements.
- **Key metrics:**
  - Requirements verified: **23 / 23 (100%)**, plus the shared-helper repository standard.
  - Proof artifacts working: **6 / 6 proof files**. Of 62 test names and `-run` prefixes referenced by the task list and proofs, all 43 full test names exist except one renamed name that is recorded (LOW-4). 17 of 18 prefixes match real tests; the remaining one is a harmless regex alternative (LOW-4).
  - Files changed vs expected: **29 changed**. 14 are in *Relevant Files*. 15 are unlisted: 6 proof files, the audit, the task list, `worktree_completion_test.go`, and 6 task 6.6 files. All 15 are linked through task notes and commits.
  - Tests: **250 top-level passes, 0 failures** across two `-race` runs of every spec-28 and affected-command test. 30 PowerShell subtests skipped (`pwsh` not installed).
  - Quality gate: `make ci` exits 0 with lint at 0 issues. `mkdocs build --strict` succeeds.

**Validated commit:** `8c0e329`, on `feat/worktree-refresh`, based on `13db8d0` (spec 27, PR #102).

This validation re-ran evidence independently rather than trusting the proof files. It added four end-to-end probes against the real binary and the real `omgw` shell function (Appendix C), including the validation-note items the task list asked for. One probe found a MEDIUM usability issue (MEDIUM-1).

## 2) Coverage Matrix

### Functional Requirements

#### Unit 1 — Scoped worktree metadata re-sync

| Requirement | Status | Evidence |
| --- | --- | --- |
| U1-1 Provide `omgw worktree refresh [repo]` | Verified | `TestWorktreeRefreshCmd_Registered`; `worktree --help` lists `refresh` (proof 06); commit `160bb63` |
| U1-2 Run repair, then prune, then list, in that order | Verified | `TestSyncRepoWorktrees_RepairsBeforePruning` and `TestWorktreeRefresh_RepairBeforePrune`. Prune-before-repair mutant fails (proof 01). `worktree_discover.go` `syncRepoWorktrees` |
| U1-3 Rebuild stored entries, recomputing `Aligned` | Verified | `TestWorktreeRefresh_RecomputesAligned`, `TestBuildWorktreeEntries_RecomputesAligned`; real `XDG_DATA_HOME` realignment in the proof 04 CLI run |
| U1-4 Exclude worktrees whose path no longer exists | Verified | `TestBuildWorktreeEntries_SkipsMissingDirectories`, `TestWorktreeRefresh_ClearsDeletedWorktree` |
| U1-5 Clear stored data when none remain | Verified | `TestWorktreeRefresh_ClearsWhenEmpty`, `TestBuildWorktreeEntries_NilWhenNoneSurvive`. Nil and empty are identical on disk (`omitempty`), as corrected in `cb27041` |
| U1-6 Save configuration once at the end | Verified | `TestWorktreeRefresh_SingleSave` catches stale per-repository saves (mutant, proof 02). Single `config.Save` after the loop confirmed by inspection, which proof 02 states as the limit of what a test can show |
| U1-7 No repository re-scan, user detection, or cache clearing | Verified | `TestWorktreeRefresh_LeavesOtherDataUntouched`, including an untracked repository and a status-cache byte comparison. Cache-clearing mutant fails (proof 02) |
| U1-8 Failing repository reported and skipped; run continues | Verified | `TestWorktreeRefresh_PartialFailure`; abort-on-failure and ignore-errors mutants fail; CLI in proofs 02 and 04 |
| U1-9 Exit non-zero on any failure, zero otherwise | Verified | `TestWorktreeRefresh_PartialFailure`, `TestWorktreeRefresh_SucceedsWithZeroExit`; real process exit 1 and exit 0 (proof 02 CLI; Appendix C, P1 and P2) |

#### Unit 2 — Targeting and change reporting

| Requirement | Status | Evidence |
| --- | --- | --- |
| U2-1 No argument and no tag refreshes every repository | Verified | `TestWorktreeRefreshTargets_EachForm/no_argument…`, run from *inside* a repository. Reuse-of-precedence-table mutant fails (proof 03) |
| U2-2 `<repo>` uses `filter.MatchesPattern` | Verified | `EachForm/name_pattern…` (multi-match, no ambiguity error); `worktreeScope.matches` calls `filter.MatchesPattern` |
| U2-3 `.` uses the current-repository resolver | Verified | `EachForm/dot…`, `DotOutsideRepo` (resolver error unchanged). See MEDIUM-1 for a resolver limitation |
| U2-4 `-t <tag>` uses `filter.MatchesExact`, single value | Verified | `EachForm/tag…`, `RepeatedTagRejected`; `filterByTag` calls `filter.MatchesExact`; repeated-tag CLI exits 1 (proof 03) |
| U2-5 `<repo> -t <tag>` applies AND | Verified | `NameAndTagAreAnded`, `DotAndTagAreAnded`; OR mutant fails |
| U2-6 Unmatched filter exits non-zero, naming the filters | Verified | `UnmatchedFilters` (4 subtests); CLI exits 1 with named filters (proof 03) |
| U2-7 Report added, removed, and realigned entries per changed repository | Verified | `ListsAllThreeChangeKinds`, `KeysOnPath`, `TestDiffWorktrees`; branch-keyed, missing-kind, and snapshot-after-sync mutants fail (proof 04) |
| U2-8 Final summary: refreshed, changed, failed | Verified | `SummaryCounts`; failed-counted-as-refreshed mutant fails; CLI `Refreshed 3 repositories, 2 changed, 1 failed` (proof 04) |
| U2-9 Unchanged repository not listed individually | Verified | `SilentWhenUnchanged` (summary-only output); print-unchanged mutant fails |

#### Unit 3 — Dry run

| Requirement | Status | Evidence |
| --- | --- | --- |
| U3-1 `--dry-run` reports changes and exits without saving | Verified | `TestWorktreeRefreshDryRun_ConfigByteIdentical`, `FilesystemUntouched`; CLI config checksum unchanged (proof 05) |
| U3-2 No repair or prune; report built from the list plus path existence | Verified | `DoesNotRepair`, `DoesNotPrune`, `FilesystemUntouched`; syncs and prune-only mutants fail. `refreshedWorktrees` calls only `buildWorktreeEntries` in a dry run |
| U3-3 Output states that repair and prune were not run | Verified | `StatesCaveat`; no-caveat mutant fails; shown through the real shell function (Appendix C, P1) |
| U3-4 Same change format and summary, framed as would-change | Verified | `OutputParity` against a fixture with every change kind and a failure; CLI preview matches the real run line for line (proof 05) |
| U3-5 Configuration, git state, and filesystem untouched | Verified | `FilesystemUntouched` compares size, mode, backdated modification time, and SHA-256 under HOME, the workspace, and the worktrees. The save mutant is caught at `config.json` (proof 05) |

### Repository Standards

| Standard Area | Status | Evidence & Compliance Notes |
| --- | --- | --- |
| Shared discovery logic (spec: "exists in exactly one place", success metric 5) | Verified | `git grep 'git.ListWorktrees('` finds one caller, `worktree_discover.go:19`, down from four. Spec 27's `refreshWorktreeData` was absorbed after the rebase (task 1.10). Align's standalone repair + prune planning pass remains by documented decision (task 1.5): it is not the full sequence |
| Go conventions: new command file, cobra registration via `init()` | Verified | `cmd/omgitworks/worktree_refresh.go`; `worktreeCmd.AddCommand(worktreeRefreshCmd)`; `SilenceErrors`, `SilenceUsage`, and `errPartialFailure` follow `worktree remove` |
| Testing: Go `testing`, `t.TempDir()` real git repositories, buffer output capture | Verified | Every new test uses real repositories through existing fixtures (`setupTaggedFixture`, `standardFixture`, `addWorktreeTo`, `breakRepo`, `chdirForTest`, `configBytes`); output captured via `io.Writer` |
| Quality gates | Verified | `make ci` exit 0 at `8c0e329`: `go vet`, golangci-lint 0 issues (gofmt and goimports formatters, `local-prefixes`), `go test -race` all packages. `gofmt -l` clean |
| Commit conventions | Verified | All 9 commits use `type(scope):` with summaries of 25–44 characters (hook limit 50), a bullet body, notes, `Related to T<n> in spec 28`, and the co-author trailer |
| Documentation | Verified | Help text states the worktree-data-only scope (the spec's naming-overlap consideration). `commands-core.md` section; `mkdocs build --strict` passes; linked anchors exist in the rendered HTML |
| Security | Verified | Proof scan found no credential patterns, emails, home paths, or usernames; transcripts use `$DEMO` placeholders |

### Proof Artifacts

| Unit/Task | Proof Artifact | Status | Verification Result |
| --- | --- | --- | --- |
| T1.0 | `28-proofs/28-task-01-proofs.md`: single `ListWorktrees` caller; helper, add-path, and remove-path tests; ordering mutant | Verified | Grep re-run: 1 caller. Referenced tests pass under `-race` ×2. See LOW-4 for one historical test name in raw output |
| T2.0 | `28-proofs/28-task-02-proofs.md`: 10 core tests, 5 mutants, git exit-code table, end-to-end and partial-failure CLI | Verified | Tests pass. Independent probe P3 re-confirms the refactored full `omgw refresh` still drops dead and adds external worktrees |
| T3.0 | `28-proofs/28-task-03-proofs.md`: targeting tests, 6 mutants, `-t` CLI | Verified | Tests pass. Probe P2 re-confirms name targeting from outside the repository |
| T4.0 | `28-proofs/28-task-04-proofs.md`: report tests, 6 mutants, mixed CLI run | Verified | Tests pass |
| T5.0 | `28-proofs/28-task-05-proofs.md`: dry-run tests, 5 mutants, preview-then-real CLI | Verified | Tests pass. Probe P1 re-confirms dry-run output through the real shell function |
| T6.0 | `28-proofs/28-task-06-proofs.md`: help output, strict docs build, anchors, wrapper routing tests, rebase union | Verified | Wrapper passthrough tests pass for `list`/`align`/`add`/`remove`/`rm`/`refresh` in zsh and bash. Strict docs build re-run succeeds. Probe P1 uses the real wrapper and binary |

## 3) Validation Issues

No CRITICAL or HIGH issues.

| Severity | Issue | Impact | Recommendation |
| --- | --- | --- | --- |
| MEDIUM | **MEDIUM-1: `refresh .` from inside an unrecorded worktree fails with misleading advice.** Spec 28's first user story is a developer who created a worktree with plain git. From inside that worktree, `omgw worktree refresh .` exits 1 with `current directory is not inside a tracked repository … run 'omgw add' to track this repository`, but the repository *is* tracked; only the worktree is unrecorded. Spec 25's resolver matches tracked repositories and *recorded* worktrees only. `refresh svc-b` works, after which `.` works (Appendix C, P2). Spec 28 satisfies U2-3 as written ("per spec 25"), and `commands-core.md` already notes that unrecorded worktrees don't resolve. | Usability: the most natural command for this user story fails, and the hint points the wrong way. Functionality is available through the repository name. | (a) In `commands-core.md` *Current-Repository Targeting*, say to run `omgw worktree refresh <repo>` by name from an unrecorded worktree, since `.` won't resolve there. (b) As a follow-up, not a blocker: have the resolver fall back to `git rev-parse --git-common-dir` to find the owning tracked repository, and word the hint for the unrecorded-worktree case. That would be a spec 25 change. |
| LOW | **LOW-1: `worktree align`'s inherited re-sync behavior has no align-specific automated test** (planning audit flag 1, accepted without remediation). Covered by helper tests, and now verified end to end by probe P4: align drops a dead entry and keeps live ones. | Regression protection for align relies on helper-level tests. | Add an align test mirroring `TestWorktreeAdd_RepairsBeforeRebuild` / `TestRunWorktreeRemove_RefreshUsesSharedRules`. |
| LOW | **LOW-2: `remove`'s "stored data untouched on list failure" behavior is only helper-tested.** In practice it's unreachable through `remove`: a removal that succeeded implies a readable repository. | None observable. | None required. Recorded to close the task list's validation note item 4. |
| LOW | **LOW-3: PowerShell wrapper routing is verified by template string test only.** `pwsh` isn't installed, so 30 PowerShell exec subtests skip (as for all pre-existing PowerShell cases). | The PowerShell `-in` list edit is checked textually, not behaviorally. | Run `go test -run TestShellWrapper ./cmd/omgitworks` once on a machine with `pwsh`, or in a CI job that has it. |
| LOW | **LOW-4: Minor proof traceability residue.** `28-task-01-proofs.md` raw output shows `TestSyncRepoWorktrees_PropagatesListError`, which was renamed in task 2.0 (recorded in `28-task-02-proofs.md`). Its regression `-run` regex includes a `TestRefresh` alternative that matches no test (the other alternatives cover the refresh tests). | None: current names are traceable through the proof 02 note. | Optional: add a one-line rename note in proof 01. |
| LOW | **LOW-5: `refresh --help` lists the inherited `-q, --quiet` global flag** ("print only the path"), which has no effect on refresh. The same is true of `list` and `align`. | Cosmetic. | Out of scope; consider scoping `--quiet` to navigation in a later change. |

## 4) Evidence Appendix

### A. Git commits analyzed (`13db8d0..8c0e329`)

| Commit | Subject | Maps to | Core files |
| --- | --- | --- | --- |
| `76732cd` | docs: plan the worktree refresh feature | Planning | none |
| `77847b3` | refactor(worktree): share one worktree re-discovery helper | T1.0 (incl. 1.10) | `worktree_discover.go`, `refresh.go`, `worktree_add.go`, `worktree_align.go`, `worktree_remove.go` |
| `a2b060d` | docs: flag changed remove behavior for validation | Validation notes | none |
| `cb27041` | docs: correct the nil-versus-empty worktrees claim | T1.0 correction | `worktree_discover.go` (comment only) |
| `160bb63` | feat(worktree): add worktree refresh to re-sync worktrees | T2.0 (incl. 2.11) | `worktree_refresh.go`, `worktree_discover.go` |
| `3c29ae9` | feat(worktree): target worktree refresh by repo, dot, or tag | T3.0 | `worktree_refresh.go` |
| `092631d` | feat(worktree): report changes made by worktree refresh | T4.0 | `worktree_refresh.go` |
| `a69484a` | feat(worktree): add --dry-run to worktree refresh | T5.0 | `worktree_refresh.go` |
| `8c0e329` | docs(worktree): document worktree refresh | T6.0 (incl. 6.6) | `worktree.go`, `worktree_refresh.go` (help), `shellinit.go` |

**File integrity (Gate D):**

- **D1 (core):** 8 core files changed, all mapped to tasks. `shellinit.go` isn't in *Relevant Files* but is linked to added sub-task 6.6, commit `8c0e329`, and proof 06. No unmapped core changes.
- **D2 (supporting):** The unlisted supporting files are linked:
  - `shellinit_test.go`, `shellinit_exec_test.go`, `shell-integration.md`, and `configuration.md` link to task 6.6 or 6.4.
  - `worktree_completion_test.go` links to task 3.8.
  - The proof files, audit, and task list are SDD artifacts.
- **D3:** No missing linkage.

The spec document itself is unchanged since planning.

### B. Commands executed at validation time

| Command | Result |
| --- | --- |
| `go test ./cmd/omgitworks -race -count=2 -v -run '<all spec-28 and affected-command tests>'` | exit 0; 250 top-level PASS, 0 FAIL; 30 PowerShell subtests SKIP |
| Test-name check across the task list and proof files | 62 names and prefixes referenced: 43 defined test names, 18 `-run` prefixes (17 match at least one test; `TestRefresh` matches none), and 1 renamed name (`…_PropagatesListError`, LOW-4) |
| `git grep -n 'git.ListWorktrees(' -- cmd` | 1 caller: `cmd/omgitworks/worktree_discover.go:19` |
| SHA references in spec docs | Only `f4b0601`, `9bae1a6`, and `13db8d0`, all ancestors of HEAD; no stale pre-rebase SHAs |
| Security scan of `docs/specs/28-spec-worktree-refresh/` (credential patterns, emails, home paths, usernames) | No matches |
| Commit subject lengths against the commit-msg hook (≤ 50) | 25–44 characters; 9/9 carry the co-author trailer |
| `gofmt -l cmd internal` | clean |
| `PATH=$HOME/go/bin:$PATH make ci` | exit 0; `0 issues.`; 9 packages `ok`; `All CI checks passed!` |
| `mkdocs build --strict` (pinned `docs/requirements.txt`, scratch virtualenv) | Built with no warnings |

### C. Independent end-to-end probes (real binary, isolated `HOME`, scratch workspace as `$VAL`)

Two tracked repositories: `svc-a` with an omgitworks worktree, and `svc-b` with a worktree created by plain git and not yet recorded.

**P1: refresh through the real `omgw` shell function (bash).** Output reaches the terminal, and the working directory is unchanged. This confirms the task 6.6 routing fix with the real binary rather than the test stub.

~~~text
Dry run — no changes will be made:
git worktree repair and prune were not run, so a worktree that repair would fix is shown as it currently stands.

[svc-b]
  added  feat-ext  $VAL/elsewhere/feat-ext  unaligned

Would refresh 2 repositories, 1 would change
(exit 0)
cwd after: $VAL/ws
~~~

**P2: `refresh .` from inside the unrecorded worktree (MEDIUM-1).**

~~~text
Error: current directory is not inside a tracked repository
  Supply a repository argument, or run 'omgw add' to track this repository
(exit 1)
-- and naming its repository instead:
[svc-b]
  added  feat-ext  $VAL/elsewhere/feat-ext  unaligned

Refreshed 1 repository, 1 changed
(exit 0)
-- then '.' from the same place, now that it is recorded:

Refreshed 1 repository, 0 changed
(exit 0)
~~~

**P3: full `omgw refresh` after the `discoverWorktrees` refactor.** A hand-deleted worktree is dropped and an external one is recorded.

~~~text
Refresh complete!
Total repositories: 2
Repositories with worktrees: 2
(exit 0)
REPO  BRANCH  PATH  STATUS
svc-a  feat-new  $VAL/elsewhere/feat-new  (unaligned)
svc-b  feat-ext  $VAL/elsewhere/feat-ext  (unaligned)
~~~

**P4: `worktree align` drops a dead stored entry through the shared helper (LOW-1, validation notes).**

~~~text
-- stored before align:
"branch": "feat-new" "branch": "feat-ext" "branch": "feat-dead"
Aligned 2 worktrees
(exit 0)
-- stored after align:
"branch": "feat-new" "branch": "feat-ext"
~~~

### D. Validation notes from the task list — disposition

| Note | Disposition |
| --- | --- |
| `worktree remove` changed behavior (1) repair then prune, (2) skip missing paths, (3) nil when empty, (4) untouched on list failure | (1)(2) Verified by `TestRunWorktreeRemove_RefreshUsesSharedRules`, which fails on spec 27's original code. (3) No on-disk effect: nil and empty serialize identically, so nothing to validate. (4) Helper-level only and unreachable through `remove` (LOW-2). All spec 27 remove tests still pass: the `TestRunWorktreeRemove_` (10), `TestWorktreeRemoveBulk_` (6), `TestWorktreeRemoveTargets_` (1), and `TestWorktreeRemove_` (10) prefixes are all included in the 250 passes |
| `worktree add` changed behavior | Verified by `TestWorktreeAdd_RepairsBeforeRebuild`, which fails on the original `worktree_add.go`; all `TestRunWorktreeAdd*` and `TestWorktreeAddBulk*` pass |
| `worktree align` changed behavior | Helper-level tests plus probe P4 (LOW-1); all `TestRunWorktreeAlign*` pass |
| New non-navigating `worktree` subcommands need the shell passthrough lists | `refresh` is in all three wrapper lists; exec tests pass in zsh and bash; PowerShell is string-tested (LOW-3); P1 confirms with the real wrapper |

**Before merging:** do a final human code review of the implementation (`13db8d0..8c0e329`) and this report. MEDIUM-1 is worth a documentation tweak, or a spec 25 follow-up, before users rely on `refresh .` from inside new worktrees. Merge order follows the stacked-PR procedure agreed with the spec 27 session: this branch's PR targets `feat/worktree-remove`.

**Validation Completed:** 2026-09-16 10:33 PDT
**Validation Performed By:** Claude Opus 5 (`claude-opus-5`)
