# 25-validation-worktree-repo-context.md

## 1) Executive Summary

- **Overall:** **PASS** — no gates tripped.
- **Implementation Ready:** **Yes.** All 23 functional requirements are verified against
  independently re-executed evidence, the full repository gate passes, and no core file
  changed outside the planned scope.
- **Key metrics:**
  - Requirements verified: **23/23 (100%)**, 0 Unknown
  - Proof artifacts working: **5/5 proof documents, 19/19 artifacts (100%)**
  - Files changed: **22** — 7 core (all planned), 15 supporting (all linked to a task)
  - Test suite: **397 top-level cases pass, 0 fail, 1 skip**; 1034 pass including
    sub-tests, 10 skips (9 pre-existing `pwsh not installed`)

Validation was performed by re-running the gates and reproducing the CLI transcripts
from scratch in a clean sandbox, not by reading the proof documents.

## 2) Coverage Matrix

### Functional Requirements

Requirement IDs follow the spec's three Demoable Units (U1 = resolver + add,
U2 = `.` targeting, U3 = completion + docs).

| Requirement | Status | Evidence |
| --- | --- | --- |
| U1-1 Reusable resolver callable from all worktree subcommands | Verified | `internal/repocontext/repocontext.go:41` `Resolve`, `:77` `ResolveCurrent`; called from `worktree_add.go`, `worktree.go`; `TestResolve` passes; commit `12a3b86` |
| U1-2 Determine checkout via `git rev-parse --show-toplevel` | Verified | `internal/git/toplevel.go:15`; `TestToplevel_RepoRoot`, `TestToplevel_Subdirectory`, `TestResolve_Subdirectory` pass |
| U1-3 Compare repository `Path`, then `Worktrees[].Path` | Verified | `repocontext.go:56-69` — two ordered passes; `TestResolve`, `TestResolve_InsideWorktree` pass |
| U1-4 Worktree match returns the owning repository | Verified | `repocontext.go:65-67` returns `&cfg.Repositories[i]`; `TestResolve_InsideWorktree` asserts `repo.Path == repoPath`; reproduced live: `add feat-2` from inside `feat-1` created under `alpha`, not the worktree |
| U1-5 Resolve symlinks on both sides | Verified | `repocontext.go:52,57,65` — `git.ResolvePath` applied to git output *and* both stored-path reads; `TestResolve_SymlinkedPath` passes |
| U1-6 Exact comparison, not prefix | Verified | `repocontext.go:90-95` `samePath`; no `HasPrefix` in the file (grep confirms); `TestResolve_NestedRepoNotPrefixMatched` passes |
| U1-7 `worktree add <branch>` uses the resolved repository | Verified | `worktree_add.go` `RunE` dispatches `len(args)==1` → `runWorktreeAddCurrent`; `TestRunWorktreeAdd_ResolvesCurrentRepo`, `_ResolvesFromSubdirectory`, `_ResolvesFromInsideWorktree` pass; reproduced live |
| U1-8 `worktree add <repo> <branch>` unchanged | Verified | `runWorktreeAdd` contains **zero** references to `repocontext` (grep count 0), so the resolver is not reachable on the explicit path; 5 pre-existing `TestRunWorktreeAdd_*` cases pass unmodified; `TestRunWorktreeAdd_ExplicitRepoIgnoresCwd` passes |
| U1-9 Failure exits non-zero naming both remedies | Verified | `repocontext.go:33` `ErrNotTracked`; `TestResolve_NotTracked` asserts all three phrases; reproduced live: `exit=1` with both remedies printed |
| U1-10 `rev-parse` failure yields the same error, not git's | Verified | `repocontext.go:47-50` discards git's error; `TestResolve_NotAGitRepo` asserts absence of `rev-parse`, `not a git repository`, `fatal` |
| U1-11 Resolver does not write config, create dirs, or prompt | Verified | `TestResolve_DoesNotMutateConfig` compares `config.json` bytes across one success and one failure; `TestRunWorktreeAddCurrent_NotTracked` asserts nothing created under the projects root |
| U2-1 `.` accepted on `list` and `align` | Verified | `worktree.go` `worktreeScopeFor` `case currentRepoArg`; both `RunE`s call it; reproduced live on both commands |
| U2-2 Bare `list` still lists all repositories | Verified | `matches()` returns `true` for the zero scope; `TestRunWorktreeList_NoArgListsAllRepos` passes; live: bare list showed `alpha` **and** `beta` |
| U2-3 Bare `align` still processes all repositories | Verified | Same zero-scope path; `TestRunWorktreeAlign_NoArgProcessesAllRepos` passes; live: bare `align --dry-run` planned **2** moves |
| U2-4 `list .` scopes to the resolved repository | Verified | `TestRunWorktreeList_DotScopesToCurrentRepo` passes; live: from `alpha`, `list .` returned only `alpha/*`; from `beta`, only `beta/feat-3` |
| U2-5 `align .` scopes and honors `--dry-run` | Verified | `TestRunWorktreeAlign_DotScopesToCurrentRepo` (asserts the out-of-scope worktree is not moved) and `_DotHonorsDryRun` pass; live: `align . --dry-run` planned **1** move vs **2** for bare |
| U2-6 `.` failure yields the U1 error | Verified | `worktreeScopeFor` returns the resolver error unchanged; `TestWorktreeDotResolutionFailure` runs both `RunE`s and asserts `errors.Is(..., ErrNotTracked)`; live: `list .` outside a repo → `exit=1` |
| U2-7 Literal `.` is never a name pattern | Verified | `worktreeScopeFor` returns from `case currentRepoArg` **before** the `default` branch that sets `NamePattern`; `TestRunWorktreeList_DotNotTreatedAsNamePattern` asserts empty `NamePattern` and that a tracked repo named `my.repo` is not matched |
| U3-1 `add` first-arg completion suggests branches when resolved | Verified | `completeWorktreeAddArgs` `case 0`; `TestCompleteWorktreeAdd_BranchesWhenResolved` passes; `__complete` transcript shows branch names |
| U3-2 `add` first-arg completion falls back to repo names | Verified | Same function's fallback on resolver error; `TestCompleteWorktreeAdd_ReposWhenUnresolved` passes; `__complete` from a temp dir shows repository names |
| U3-3 `list`/`align` completion includes `.` | Verified | `completeWorktreeRepoOrDot`; `TestCompleteWorktreeRepoOrDot` (3 sub-cases) and `_PrefixFiltersDot` pass; `__complete` shows `.` first inside a repo and absent outside |
| U3-4 `shellinit.go` requires no changes | Verified | `git diff --stat aa04200..HEAD -- cmd/omgitworks/shellinit.go` is **empty**; 7 `TestShellTemplates*` cases pass |
| U3-5 `commands-core.md` documents the new forms | Verified | All 7 required content markers found by grep: `add [repo] <branch>`, `list [repo\|.]`, `align [repo\|.]`, `Current-Repository Targeting`, the error text, the owning-worktree rule, and the `omgw refresh` caveat |

### Repository Standards

| Standard Area | Status | Evidence & Compliance Notes |
| --- | --- | --- |
| Coding standards / layout | Verified | New package under `internal/`, commands under `cmd/omgitworks/`, matching the documented structure. `gofmt -l` clean across `cmd/` and `internal/` |
| Git invocation | Verified | The resolver calls `git.Toplevel`, which wraps the existing unexported `gitCommand`. No `os/exec` in `internal/repocontext` (grep confirms), as the spec requires |
| Path comparison | Verified | `resolvePath` was exported as `git.ResolvePath` and reused rather than reimplemented; both original call sites in `IsAligned` updated |
| Testing patterns | Verified | `t.TempDir()`, `t.Setenv`, `t.Cleanup`, `bytes.Buffer`, and the existing `captureStdoutStr` helper are all reused. The `os.Chdir` + `t.Cleanup` pattern follows `init_test.go:72` |
| Test isolation | Verified | `internal/repocontext/hooks_isolation_test.go` mirrors the existing `internal/git/hooks_isolation_test.go` rather than introducing a new mechanism |
| Quality gates | Verified | `make ci` (vet + golangci-lint v2.13.2 + `go test -race`) exits **0**; linter reports `0 issues` with no new exclusions added |
| Cobra structure | Verified | Subcommands still register via `init()` with `worktreeCmd.AddCommand`; `ValidArgsFunction` assigned in the same `init` |
| Commit conventions | Verified | 4 `feat(worktree):` commits and 1 `docs:` commit, each naming its task (`Related to T<n>.0 in spec 25`). `feat` correctly triggers a minor bump |
| Documentation | Verified | `docs/site/commands-core.md` updated in the same branch as the behavior, per the spec 18 precedent |

### Proof Artifacts

| Unit/Task | Proof Artifact | Status | Verification Result |
| --- | --- | --- | --- |
| 1.0 | Test: resolver suite `./internal/repocontext/` | Verified | Re-run: 11 pass, 1 skip (Windows-only, by design), 0 fail |
| 1.0 | Test: `Toplevel` / `ListBranches` helpers | Verified | Re-run: 5 pass |
| 1.0 | CLI: `golangci-lint run ./internal/...`, `go vet ./...` | Verified | Re-run: `0 issues.`; vet silent |
| 2.0 | CLI: `cd <repo> && omgw worktree add feat-x` + `worktree list` | Verified | Reproduced in a clean sandbox: worktree created and listed |
| 2.0 | CLI: `worktree add` from inside an existing worktree | Verified | Reproduced: `feat-2` created under the **owning** repo `alpha` |
| 2.0 | CLI: unresolvable directory, non-zero exit | Verified | Reproduced: two-remedy error, `exit=1` |
| 2.0 | Test: `_ResolvesCurrentRepo`, `_ExplicitRepoIgnoresCwd`, 5 pre-existing | Verified | Re-run: all pass |
| 2.0 | CLI: arity bounds | Verified | Reproduced: `exit=1` for 0 and 3 args; `exit=0` for 1 and 2 |
| 3.0 | CLI: `list` vs `list .` side by side | Verified | Reproduced and strengthened: from `alpha` → `alpha/*` only; from `beta` → `beta/feat-3` only; bare → all three |
| 3.0 | CLI: `align . --dry-run` scoped | Verified | Reproduced: bare plans **2**, `.` plans **1** |
| 3.0 | Test: 10 new + 12 pre-existing list/align cases | Verified | Re-run: all pass |
| 3.0 | Test: `.` failure on both subcommands | Verified | Re-run: both sub-cases pass; live `exit=1` |
| 4.0 | CLI: `__complete worktree add ""` inside / outside a repo | Verified | Reproduced: branches inside, repository names outside |
| 4.0 | CLI: `__complete worktree list/align ""` | Verified | Reproduced: `.` present inside, absent outside |
| 4.0 | Test: 9 completion tests incl. registration guard | Verified | Re-run: all pass |
| 5.0 | Documentation: `commands-core.md` diff | Verified | All 7 required content markers present |
| 5.0 | Test: `shellinit.go` unchanged + 7 template tests | Verified | Diff empty; all 7 pass |
| 5.0 | CLI: `--help` cross-check on 3 subcommands | Verified | Usage lines match documented signatures exactly |
| 5.0 | CLI: `make ci` | Verified | Re-run: **exit 0** |

## 3) Validation Issues

No CRITICAL or HIGH issues. GATE A is not tripped.

| Severity | Issue | Impact | Recommendation |
| --- | --- | --- | --- |
| LOW | Imprecise test tally in a proof document. `25-task-05-proofs.md` states "397 test cases pass, 1 skips". Evidence: re-running `make ci` gives **397 top-level** pass / **1** top-level skip, but **1034** pass / **10** skips when sub-tests are counted. The 9 additional skips are `pwsh not installed` sub-cases in `TestShellWrapperWorktreePassthrough`, pre-existing and in a file this branch never touched (`git diff --stat aa04200..HEAD -- cmd/omgitworks/shellinit_exec_test.go` is empty). | Verification unaffected; a reviewer could misread the skip count as complete | Reword to "397 top-level tests (1034 including sub-tests); 1 skip is the Windows-only case, 9 are pre-existing `pwsh not installed` skips" |
| LOW | Two test files landed outside the planned "Relevant Files" list. `cmd/omgitworks/worktree_add_current_test.go` and `cmd/omgitworks/worktree_scope_test.go` were created, whereas sub-tasks 2.7-2.11 and 3.9-3.14 said the tests would be added to the existing `worktree_add_test.go` / `worktree_list_test.go` / `worktree_align_test.go`. Evidence: changed-file list vs the task file's Relevant Files table. | Traceability only; both are supporting files and both are named in their parent task's commit message | Acceptable as-is (separating new fixtures from existing ones is reasonable). Optionally note the split in the task file's Relevant Files table |
| LOW | `internal/repocontext/hooks_isolation_test.go` is an unplanned supporting file. Evidence: not in Relevant Files; created to neutralize a global `core.hooksPath` that broke fixture setup. | None; linkage is explicit | Already documented in `25-task-01-proofs.md` under "Note on test isolation" and in commit `12a3b86`. No action needed |

**Planned files intentionally left unchanged** (not issues, per the GATE D clarification
that Relevant Files are planning guidance): `cmd/omgitworks/shellinit.go` and
`shellinit_test.go` were listed as regression guards and their being unchanged *is* the
evidence for U3-4; `internal/git/worktree_test.go` needed no edit because the
`resolvePath` → `ResolvePath` rename had only two internal call sites.

## 4) Evidence Appendix

### Gates applied

| Gate | Result | Basis |
| --- | --- | --- |
| A — no CRITICAL/HIGH | **PASS** | 3 LOW issues only |
| B — no `Unknown` in coverage matrix | **PASS** | 23/23 Verified |
| C — proof artifacts accessible and functional | **PASS** | 19/19 re-executed or reproduced |
| D — file integrity | **PASS** | D1: 7 core files changed, all in Relevant Files, zero unmapped. D2/D3: 15 supporting files, each linked via commit message or proof doc |
| E — repository standards | **PASS** | 9/9 standard areas Verified |
| F — no credentials in proof artifacts | **PASS** | Scanned the full branch diff; only match was the substring `sk-` inside the filename `25-task-01-...`, a regex false positive. No emails, tokens, or keys |

### Commits analyzed

| Commit | Task | Core files touched |
| --- | --- | --- |
| `12a3b86` | T1.0 | `internal/repocontext/repocontext.go`, `internal/git/toplevel.go`, `internal/git/worktree.go` |
| `05dd754` | T2.0 | `cmd/omgitworks/worktree_add.go`, `cmd/omgitworks/worktree.go` |
| `014bd08` | T3.0 | `cmd/omgitworks/worktree.go`, `worktree_list.go`, `worktree_align.go` |
| `5ebae00` | T4.0 | `cmd/omgitworks/worktree.go`, and `init()` registration in the three subcommand files |
| `612c4ac` | T5.0 | `docs/site/commands-core.md` |

Baseline for all diffs: `aa04200` (the spec commit). Total: 22 files, +2433 / -113.

### Commands executed during validation

~~~bash
git log --oneline aa04200..HEAD
git diff --stat aa04200..HEAD
git diff --stat aa04200..HEAD -- cmd/omgitworks/shellinit.go        # empty
make ci                                                              # exit 0
go test -v ./internal/repocontext/
grep -c repocontext <runWorktreeAdd body>                            # 0
# clean-sandbox reproduction with isolated XDG_CONFIG_HOME / XDG_DATA_HOME:
omgw worktree add feat-1 / feat-2 / beta feat-3
omgw worktree list  |  omgw worktree list .   (from alpha, then from beta)
omgw worktree align --dry-run  |  omgw worktree align . --dry-run
omgw __complete worktree add ""  |  omgw __complete worktree list ""
~~~

### Independent reproduction result

A fresh two-repository workspace (`alpha`, `beta`) was built from scratch with isolated
XDG directories. Every headline claim reproduced:

~~~text
1) add, repo omitted:      Created worktree for branch 'feat-1' at <v>/data/gws/projects/alpha/feat-1
2) add from inside wt:     Created worktree for branch 'feat-2' at <v>/data/gws/projects/alpha/feat-2
3) list all ->             alpha/feat-1 alpha/feat-2 beta/feat-3
4) list . (from alpha) ->  alpha/feat-1 alpha/feat-2
5) list . (from beta)  ->  beta/feat-3
6) align --dry-run   -> 2 planned
7) align . --dry-run -> 1 planned
8) list . outside repo exit=1
~~~

---

**Validation Completed:** 2026-09-15
**Validation Performed By:** Claude Opus 5
