# 29-validation-user-repo-context.md

## 1) Executive Summary

- **Overall:** **PASS**. No gate tripped.
  - Gate A (no CRITICAL/HIGH): PASS. Three LOW findings, none affecting functionality.
  - Gate B (no `Unknown` in the matrix): PASS. 17/17 functional requirements Verified.
  - Gate C (proof artifacts functional): PASS. Every referenced test re-run twice under `-race`; every transcript reproduced against a fresh binary.
  - Gate D (file integrity): PASS. Every changed core file is in the task list's Relevant Files table; every supporting file is linked by a commit `notes:` block naming its task.
  - Gate E (repository standards): PASS. `make ci` exits zero; commits follow the hook-enforced format with the `user` scope.
  - Gate F (no secrets in proofs): PASS. Only `example.com` and `company.com` placeholder addresses; temporary paths scrubbed to `<tmp>`. The scan found one absolute local path in `29-task-04-proofs.md`, echoed from MkDocs output; it was scrubbed to `<repo>/site` as part of this validation.
- **Implementation Ready:** **Yes.** The one-argument form works from the root, a subdirectory, and a worktree, always writing to the owning repository; every current invocation is unchanged; and the probes found no behavior the spec did not predict.
- **Key metrics:**
  - Requirements verified: 17 of 17 (100%)
  - Proof artifacts working: 31 of 31 (100%)
  - Files changed vs expected: 18 changed, all 18 listed in Relevant Files or under the spec directory; 11 planning-era entries correctly left unchanged
  - Test suite: 55 top-level results from the referenced suites, each passing on both `-count=2` runs

## 2) Coverage Matrix

### Functional Requirements

Unit 1, current-repository resolution:

| Requirement | Status | Evidence |
| --- | --- | --- |
| U1-1 one positional applies the profile to `ResolveCurrent`'s repository | Verified | `selectAssignTarget` returns `repocontext.ResolveCurrent(cfg)` when `repoArg == ""`; `TestUserAssign_OneArg_FromRepoRoot`, `_FromSubdirectory`; transcript `assign-current-repo.md`; probe P1, P7 |
| U1-2 two positionals behave exactly as today, including the multi-match error | Verified | `findRepositories` substring matcher retained verbatim; `TestUserAssign_TwoArgs_AssignsNamedRepo`, `_MultiMatchError`, `_DryRunOutput`, `_UseSubdirsStillWorks`; probe P6 reproduces the listing |
| U1-3 resolver consulted only when the argument is omitted | Verified | `TestUserAssign_TwoArgs_ResolverNotConsulted` (two sub-cases: inside another repo, in an untracked dir); probe P1 shows `work` the repository untouched when `work` the profile is meant |
| U1-4 inside a worktree, write lands on the owning repository's `.git/config` and its stored fields | Verified | `TestUserAssign_OneArg_FromInsideWorktree` reads `user.email` from the main checkout and the worktree and checks stored `Path`; transcript `assign-from-worktree.md`; probe P5 shows `user.signingkey` and `commit.gpgsign` in the main checkout and `user show secure` counting 1 repository |
| U1-5 `ErrNotTracked` returned unchanged on resolution failure | Verified | `TestUserAssign_OneArg_NotTracked` asserts `errors.Is`; transcript failure block and probes P3, P4 show the spec 25 message verbatim with exit 1 |
| U1-6 repository resolved before the profile is looked up | Verified | code order in `runUserAssign`; `TestUserAssign_OneArg_UnknownProfile_ReportsProfileError`, `TestUserAssign_TwoArgs_UnknownProfileError`; probe P2: `profile 'my-api' not found` |
| U1-7 `--dry-run` names the resolved repository | Verified | `TestUserAssign_OneArg_DryRunNamesResolvedRepo`; probe P5 dry run names `my-api` from inside its worktree; P7 through the shell function |
| U1-8 same success output as the two-argument form, no "inferred" line | Verified | `TestUserAssign_OneArg_OutputMatchesTwoArgForm` compares stdout byte for byte after restoring state between runs |
| U1-9 `--use-subdirs` with one argument refused before any change, including with `--dry-run` | Verified | refusal precedes `config.Load()` in `runUserAssign`; `TestUserAssign_OneArg_UseSubdirsRefused` (real run, dry run) asserts empty stdout, no move, no `.gitconfig`; transcript failure block shows both refusals |
| U1-10 one or two positionals accepted; zero or three fail with cobra's usage error | Verified | `cobra.RangeArgs(1, 2)`; `TestUserAssign_ArgCount`; probe P6: `accepts between 1 and 2 arg(s), received 0` / `received 3` |

Unit 2, completion, help, and docs:

| Requirement | Status | Evidence |
| --- | --- | --- |
| U2-1 first argument completes to profiles when resolved | Verified | `TestCompleteUserAssign_ProfilesWhenResolved`; `completion.md` inside `my-api` lists `work`, `personal`; probe P8 with `--dry-run` typed first |
| U2-2 first argument completes to repositories when not resolved | Verified | `TestCompleteUserAssign_ReposWhenNotResolved`; `completion.md` from `$HOME` lists `my-api`, `web-ui` |
| U2-3 second argument completes to profiles | Verified | `TestCompleteUserAssign_ProfilesForSecondArg` incl. prefix filter; `completion.md` with `my-api` typed |
| U2-4 `Use` is `assign [repository] <profile>`; `Long` documents both forms, detection, `omgw` examples | Verified | `TestUserAssign_HelpTextUsesOmgw`, `_HelpDocumentsBothForms`; `assign-help.md` shows `Usage: omgitworks user assign [repository] <profile> [flags]` |
| U2-5 `user` parent `Long` uses `omgw` and shows the one-argument example | Verified | same two tests; `assign-help.md` second block lists `omgw user assign work` |
| U2-6 `commands-user.md` documents the omitted form, worktree behavior, error, `--use-subdirs` restriction, usage and examples | Verified | `docs-diff.md`; `grep -n 'omgw user assign \[repository\] <profile>'` → line 125; `mkdocs build --strict` exit 0 |
| U2-7 `shellinit.go` needs no change | Verified | `git diff --quiet main..HEAD -- cmd/omgitworks/shellinit.go` clean; `TestShell*` pass unmodified; probe P7 runs `assign` through the evaluated `omgw` function |

Spec success metrics 1–6 are each covered by the rows above. Metric 2 (identical output for every current invocation) rests on the task 1.0 characterization tests, which were written against the pre-change code and pass unchanged after it.

### Repository Standards

| Standard Area | Status | Evidence & Compliance Notes |
| --- | --- | --- |
| Go layout and cobra structure | Verified | command in `cmd/omgitworks/user_assign.go`, registration kept in `user.go`'s `init()`; `cobra.RangeArgs(1, 2)` and `ValidArgsFunction` as spec 25 did for `worktree add` |
| Resolver reuse | Verified | one call to `repocontext.ResolveCurrent(cfg)`; no new `git rev-parse` path or path comparison added (`internal/` unchanged) |
| Completion helpers | Verified | `completeUserAssignArgs` reuses `completeProfileNames` and `completeRepoNames`; arity switch mirrors `completeWorktreeAddArgs` |
| Testing patterns | Verified | standard `testing`, `t.TempDir()`, `withTempHome`, `chdirForTest` with cleanup, real git repositories; fixture reuse of `setupWorktreeTestRepo`, `runWorktreeAdd`, `projectsPath`, `captureStdoutStr`, `configBytes` |
| Quality gates | Verified | `make ci` (vet, golangci-lint v2.13.2, `go test -race ./...`) exits zero at each parent task and at validation; `0 issues` from lint |
| Commit conventions | Verified | six commits, hook-normalized headers, `refactor(user):` / `feat(user):` / `docs(user):` / `docs:`; each `notes:` block names its task and spec |
| Documentation | Verified | `commands-user.md` follows the usage-block, flags-table, examples pattern; strict site build passes |

### Proof Artifacts

| Unit/Task | Proof Artifact | Status | Verification Result |
| --- | --- | --- | --- |
| 1.0 | `TestUserAssign_TwoArgs_AssignsNamedRepo` | Verified | re-run: PASS ×2 under `-race` |
| 1.0 | `TestUserAssign_TwoArgs_MultiMatchError` | Verified | re-run: PASS ×2 |
| 1.0 | `TestUserAssign_TwoArgs_UnknownProfileError` | Verified | re-run: PASS ×2 |
| 1.0 | `TestUserAssign_TwoArgs_DryRunOutput` | Verified | re-run: PASS ×2 |
| 1.0 | `TestUserAssign_TwoArgs_UseSubdirsStillWorks` (accepted, canceled) | Verified | re-run: PASS ×2 |
| 1.0 | `git diff --stat` of the extraction; `go build`, `make ci` | Verified | commit `bab9663`: `user.go` −155, `user_assign.go` +191; CI re-run exit 0 |
| 1.0 | `TestUserSubcommandsRegistered` unmodified | Verified | `user_test.go` unchanged vs `main`; PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_FromRepoRoot` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_FromSubdirectory` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_FromInsideWorktree` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_OutputMatchesTwoArgForm` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_TwoArgs_ResolverNotConsulted` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_NotTracked` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_UnknownProfile_ReportsProfileError` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_UseSubdirsRefused` (real run, dry run) | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_OneArg_DryRunNamesResolvedRepo` | Verified | PASS ×2 |
| 2.0 | `TestUserAssign_ArgCount` | Verified | PASS ×2 |
| 2.0 | CLI transcript `assign-current-repo.md` | Verified | reproduced by probe P1/P7 with a fresh binary and sandboxed `HOME` |
| 2.0 | CLI transcript `assign-from-worktree.md` | Verified | reproduced by probe P5, extended to a signing profile |
| 2.0 | `make ci` | Verified | exit 0 |
| 3.0 | `TestCompleteUserAssign_ProfilesWhenResolved` | Verified | PASS ×2 |
| 3.0 | `TestCompleteUserAssign_ReposWhenNotResolved` | Verified | PASS ×2 |
| 3.0 | `TestCompleteUserAssign_ProfilesForSecondArg` | Verified | PASS ×2 |
| 3.0 | `TestCompleteUserAssign_NothingAfterTwoArgs` | Verified | PASS ×2 |
| 3.0 | `__complete` transcript `completion.md` | Verified | reproduced by probe P8 (with a flag typed first) |
| 4.0 | `TestUserAssign_HelpTextUsesOmgw` | Verified | PASS ×2 |
| 4.0 | `TestUserAssign_HelpDocumentsBothForms` | Verified | PASS ×2 |
| 4.0 | `shellinit_test.go` and `shellinit_exec_test.go` unmodified | Verified | unchanged vs `main`; all 9 `TestShell*` PASS ×2 |
| 4.0 | `deprecated_test.go` unmodified | Verified | unchanged vs `main`; all 14 `TestDeprecated*` PASS ×2 |
| 4.0 | `assign-help.md`; `docs-diff.md`; `grep` of the usage line | Verified | help re-captured from a fresh binary; grep → `125:omgw user assign [repository] <profile> [flags]`; `mkdocs build --strict` exit 0 |
| 4.0 | `make ci` | Verified | exit 0 |

Proof-document structure: all four `29-task-0N-proofs.md` files open with a descriptive title, `## Task Summary`, `## What This Task Proves`, and `## Evidence Summary`, give **What it proves** and **Why it matters** before each raw block, and end with `## Reviewer Conclusion`.

## 3) Validation Issues

| Severity | Issue | Impact | Recommendation |
| --- | --- | --- | --- |
| LOW | **Usage block follows every runtime error.** `userAssignCmd` does not set `SilenceUsage`, so the resolver error and the `--use-subdirs` refusal are each followed by cobra's usage text (see the failure block of `assign-from-worktree.md`). The worktree commands set `SilenceUsage` and print only the error. Pre-existing on `assign`; left unchanged to keep every existing error path's output identical, which success metric 2 requires. | Usability: the remedy lines of the spec 25 error are pushed up by ten lines of usage | Consider setting `SilenceUsage` (and `SilenceErrors` with an explicit print, as `worktree_add.go` does) in spec 30, which already changes `assign`'s error paths for bulk runs |
| LOW | **`commands-core.md` still introduces the resolver as a worktree-only feature.** Its Current-Repository Targeting section opens with "`omgw worktree add` lets you omit the repo argument …", and the user docs now link to it. The mechanism description is accurate for `user assign`. | Documentation: a reader following the link may wonder whether it applies to `user` | Add `omgw user assign` to that opening sentence in the spec 30 docs pass, which touches the same section for `-t` |
| LOW | **Known resolver limitation carried over from spec 25.** From a worktree created with plain `git worktree add` and not yet refreshed, `omgw user assign work` fails with advice to run `omgw add`, while the actual remedy is `omgw worktree refresh <repo>` (probe P4 shows the fix working). Spec 29 accepted this explicitly in its technical considerations, and spec 28's validation recorded the same follow-up. | Usability, bounded: the user docs added in this spec state the refresh remedy | Track as the same follow-up spec 28 recorded; the message lives in `internal/repocontext`, outside this spec's scope |

No MEDIUM, HIGH, or CRITICAL issues. No `Unknown` matrix entries. No unmapped core files.

## 4) Evidence Appendix

### Commits analyzed (`git log --stat main..HEAD`)

| Commit | Files | Task linkage |
| --- | --- | --- |
| `da26cad` docs: plan the user repo-context feature | spec, questions, tasks, audit | planning |
| `bab9663` refactor(user): extract assign into its own file | `user.go`, `user_assign.go`, `user_assign_test.go`, task-01 proofs, tasks | T1.0 |
| `848c70c` feat(user): assign a profile to the current repository | `user_assign.go`, `user_assign_test.go`, task-02 proofs, two transcripts, tasks | T2.0 |
| `34744ee` feat(user): complete assign arguments by context | `user_assign.go`, `user_assign_completion_test.go`, task-03 proofs, completion transcript, tasks | T3.0 |
| `5bd239d` feat(user): document assign's current-repository form | `user.go`, `user_assign.go`, `user_assign_test.go` | T4.0 |
| `d226956` docs(user): document assign from the current repository | `commands-user.md`, task-04 proofs, help and docs-diff transcripts, tasks | T4.0 |

### File classification (Gate D)

| File | Class | Mapping |
| --- | --- | --- |
| `cmd/omgitworks/user_assign.go` | core | Relevant Files; T1.0–T4.0 |
| `cmd/omgitworks/user.go` | core | Relevant Files; T1.0, T4.0 |
| `docs/site/commands-user.md` | supporting (docs) | Relevant Files; U2-6 |
| `cmd/omgitworks/user_assign_test.go` | supporting (tests) | Relevant Files; T1.0, T2.0, T4.0 |
| `cmd/omgitworks/user_assign_completion_test.go` | supporting (tests) | Relevant Files; T3.0 |
| `docs/specs/29-spec-user-repo-context/**` (13 files) | supporting (spec artifacts) | SDD lifecycle |

Unchanged planning-era entries, each verified as correctly untouched: `user_test.go`, `tag.go`, `worktree.go`, `worktree_add_current_test.go`, `worktree_add_test.go`, `worktree_align_test.go`, `internal/repocontext/repocontext.go`, `internal/user/assign.go`, `internal/user/profile.go`, `internal/config/config.go`, `docs/site/commands-core.md`. Confirmed also unchanged: `shellinit.go`, `deprecated.go`, all of `internal/`.

### Commands executed

```bash
# Referenced suites, twice under the race detector: 55 results, all PASS
go test ./cmd/omgitworks/ -race -count=2 -run \
  'TestUserAssign|TestCompleteUserAssign|TestUserSubcommandsRegistered|TestUserProfileCompletion|TestShell|TestDeprecated'
# ok  	github.com/daileyo/omgitworks/cmd/omgitworks	5.065s

# Full gate
PATH=$HOME/go/bin:$PATH make ci        # 0 issues; All CI checks passed!

# Docs
.venv/bin/python -m mkdocs build --strict   # exit 0
grep -n 'omgw user assign \[repository\] <profile>' docs/site/commands-user.md   # 125

# Scope
git diff --stat main..HEAD              # 18 files
git diff --quiet main..HEAD -- cmd/omgitworks/shellinit.go cmd/omgitworks/deprecated.go internal/   # clean

# Secrets scan over 29-proofs/: only example.com / company.com addresses; no tokens, keys, home or temp paths
```

### Independent probe of the built binary (sandboxed `HOME`, placeholder identities)

Run after rebuilding from `d226956`; paths scrubbed to `<tmp>`.

| Probe | Case | Observed | Verdict |
| --- | --- | --- | --- |
| P1 | repository `work` exists alongside profile `work`; `omgw user assign work` inside `my-api` | profile assigned to `my-api`; repository `work` untouched | as specified: one argument is always the profile |
| P2 | `omgw user assign my-api` inside `my-api` (repository name, not a profile) | `Error: profile 'my-api' not found` | as specified: lookup order, clear message |
| P3 | untracked git repository nested under a tracked one | `ErrNotTracked` message, exit 1 | as specified: exact-path match, no prefix leakage |
| P4 | worktree made with plain `git worktree add`, before refresh / after `omgw worktree refresh my-api` | before: `ErrNotTracked`; after: assigned to `my-api` | documented limitation; remedy works (LOW 3) |
| P5 | signing profile from inside an omgw worktree, dry run then real | dry run lists `user.signingkey` and `commit.gpgsign` changes; real run writes all four keys to the **main checkout**; `user show secure` reports 1 repository | as specified: stored `signing_enabled` and git keys both land on the owner |
| P6 | `assign my work` (ambiguous), zero args, three args | multi-match listing; `accepts between 1 and 2 arg(s), received 0` / `received 3` | as specified |
| P7 | `eval "$(omgitworks shell-init bash)"` then `omgw user assign work` and `--dry-run` inside `my-web` | passthrough; assigned; dry run names `my-web` | U2-7 confirmed through the shell function |
| P8 | `__complete user assign --dry-run ""` | profile names | completion unaffected by a flag typed first |

**Validation Completed:** 2026-10-26 21:39 PDT
**Validation Performed By:** Claude Fable 5.1
