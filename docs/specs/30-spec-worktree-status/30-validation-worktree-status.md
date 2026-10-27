# 30-validation-worktree-status.md

## 1) Executive Summary

- **Overall:** PASS (no gates tripped)
- **Implementation Ready:** **Yes.** All 23 functional requirements are verified by tests re-run under the race detector and by independent CLI probes against a scratch fixture; the only findings are three LOW housekeeping items, two of which were fixed during validation.
- **Key metrics:** 23/23 requirements verified (100%); 20/20 proof artifacts working (100%); 7 core files changed, all mapped to tasks; 1 supporting test file changed outside the planning table, with linkage recorded.

Gate results:

| Gate | Result | Note |
| --- | --- | --- |
| A — no CRITICAL/HIGH issues | PASS | Three LOW findings only |
| B — no Unknown in coverage matrix | PASS | |
| C — proof artifacts accessible and functional | PASS | Every referenced test re-run; every CLI command re-executed |
| D — file integrity | PASS | D1 clean; one D2 supporting file linked via task 2 proof and commit `9c1c754` |
| E — repository standards | PASS | |
| F — no secrets in proofs | PASS | One scratch path with a local username scrubbed from the task 6 proof |

## 2) Coverage Matrix

### Functional Requirements

Requirement IDs number the "The system shall" lines in spec order within each unit.

| ID | Requirement (abridged) | Status | Evidence |
| --- | --- | --- | --- |
| U1-1 | Rename alignment column to `ALIGNED` with `yes`/`no` | Verified | `TestRunWorktreeList_AllWorktrees`, `_UnalignedIndicator` pass; probe P1 header reads `ALIGNED`, rows `yes` |
| U1-2 | `-s` bare appends compact icons after the branch, column widened, icons sub-column aligned | Verified | `TestWorktreeListStatus_Compact`, `_CompactAligned`; probe P2 |
| U1-3 | `-S` bare adds a `STATUS` column of icons only | Verified | `TestWorktreeListStatus_Full`; probe P3 |
| U1-4 | Both flags: `-S` wins, `BRANCH` plain | Verified | `TestWorktreeListStatus_ShowStatusWins`; probe P4 |
| U1-5 | Icons produced by the same formatter as `omgw list` | Verified | `worktree_list.go` calls `formatStatusIcons`; `TestWorktreeListStatus_Color` asserts byte equality with `formatStatusIcons(status, true)` |
| U1-6 | `--color auto|always|never` with `list`'s resolution rule | Verified | `resolveColorEnabled(mode)` shared, commit `19da74a`; `_Color`; probe P8 shows `36m`/`32m` with `always`, none with `never` |
| U1-7 | Unreadable status renders blank, command succeeds | Verified | `TestWorktreeListStatus_MissingWorktree`; probe P16 shows the `gone` row with no icons, exit 0 |
| U1-8 | Detached worktree shows `(detached)` with icons appended | Verified | `TestWorktreeListStatus_Detached`; probes P2 and P3 include a real detached worktree |
| U1-9 | Without a status flag: output unchanged except rename, no git runs | Verified | `TestWorktreeListStatus_OptIn` (cache file absent); probe P1 runs with `PATH=/nonexistent` and succeeds |
| U1-10 | Help and documentation describe flags, terms, renamed column; example output updated | Verified | `TestWorktreeListHelp_StatusFlags`; `docs/site/commands-core.md` diff in `789081f`; `mkdocs build --strict` exit 0 |
| U2-1 | `-s=`/`--status=`/`-S=`/`--show-status=` show and filter | Verified | `_FilterDirty`, `_ShowAndFilterAhead`; probe P5 (`-s=dirty`, `--status=ahead`) |
| U2-2 | Status text built by `list`'s helper, same partial pattern rule | Verified | `worktree_list.go` calls `statusFilterText` + `filter.MatchesPattern`; probe P16b shows `-s=clean` keeping `ahead` (text `clean ahead`) exactly as `list` would |
| U2-3 | Positional is always the repository; filter values attach with `=` (amended: `-sdirty` dropped) | Verified | `TestWorktreeListFlags_PositionalIsRepo` incl. the `-sdirty` rejection; probes P6, P7 |
| U2-4 | Help shows the attached form | Verified | `Help_StatusFlags` asserts `-s=dirty`; probe P12 |
| U2-5 | No readable status never matches a filter | Verified | `_NoStatusExcluded`; probe P16b drops `gone` |
| U2-6 | Filtering applies after scoping | Verified | `_ScopeThenFilter` (two repositories); probe P5 `fixture -S=behind` |
| U2-7 | Empty result prints the no-worktrees message, exit 0 | Verified | `_FilterEmpty`; probes P5, P6 |
| U3-1 | Pipeline extracted into one helper called by both commands | Verified | `status_fetch.go`; `grep` shows `FetchAll`/`PrefetchInBackground` only there; `runList` calls `fetchStatuses` (`19da74a`) |
| U3-2 | Same cache file and TTL, keyed by worktree path | Verified | `_CacheWritten`, `_CacheHit`; probe P9 served with `PATH=/nonexistent`; task 5 transcript lists worktree keys |
| U3-3 | `--workers` resolved flag → preference → 8 | Verified | `TestWorktreeListWorkers_Resolution`; probe P12 |
| U3-4 | Spinner renders on stderr only when it is a terminal | Verified | Unchanged `internal/git/progress.go`; `TestProgress_NonTTY`, `_StartStop`, `_Increment`, `_StopWithoutStart` pass under `-race` |
| U3-5 | `omgw list -s`/`-S` unchanged by the extraction | Verified | Full `list_test.go` passes with no assertion edits (`make ci`); probes P11, P14, P15 (`-s dirty`, `-s clean`, `-S`, `-o json -S`) |
| U3-6 | `status_workers` documentation names both commands | Verified | `docs/site/configuration.md` row in `789081f` |

### Repository Standards

| Standard Area | Status | Evidence & Compliance Notes |
| --- | --- | --- |
| Go conventions and file placement | Verified | Changes confined to `cmd/omgitworks/{list,status_fetch,worktree_list,worktree}.go`; `gofmt -l` empty; goimports grouping with the local prefix |
| Cobra structure and sentinel reuse | Verified | Flags registered in `init()` on `worktreeListCmd`; `showColumnSentinel` reused; root usage template strips it from help (`TestHelpListsExactlyTheFlagsEachCommandAccepts` passes) |
| Separate flag variables | Verified | `grep 'flagWorkers\|flagColor\b\|flagStatus\b\|flagShowStatus\b' worktree_list.go` matches only the explanatory comment |
| Testing patterns | Verified | Real git repositories in `t.TempDir()`, `bytes.Buffer` capture, `t.Setenv("HOME")`, reuse of `runGitIn`, `gitWorktreeAdd`, `helpListedFlags`, `saveConfigForWorktreeTests` |
| Commit messages | Verified | Seven conventional commits, hook-formatted bodies, each citing its task and spec 30 in `notes:` |
| CI quality gates | Verified | `make ci` (vet, golangci-lint, `go test -race`) exit 0 at every task and again during validation |

### Proof Artifacts

| Task | Proof Artifact | Status | Verification Result |
| --- | --- | --- | --- |
| 1.0 | CLI: grep shows pipeline calls only in `status_fetch.go` | Verified | Re-run: two matches, both in `status_fetch.go` |
| 1.0 | Test: four `TestFetchStatuses_*` ordering cases | Verified | Pass under `-race` |
| 1.0 | Test: existing `cmd/omgitworks` and `internal/git` suites unchanged | Verified | `make ci` exit 0; no assertion edits in `list_test.go` or `internal/git` per `git log --name-only` |
| 1.0 | CLI: `make ci` | Verified | Exit 0 |
| 2.0 | Test: rename, compact, aligned sub-column, color, missing worktree, opt-in | Verified | All six pass under `-race` |
| 2.0 | CLI: transcript of `.` and `. -s` | Verified | Re-executed as probes P1, P2; output matches the proof |
| 3.0 | Test: full, precedence, detached, help | Verified | All four pass under `-race` |
| 3.0 | CLI: transcript of `. -S`, `. -s -S`, `--help` | Verified | Re-executed as probes P3, P4, P12 |
| 4.0 | Test: filter, show-and-filter, positional, scope-then-filter, no-status, empty | Verified | All six pass under `-race`, including the `-sdirty` rejection |
| 4.0 | CLI: two-repository `-s=dirty` and `-s fixture` transcript | Verified | Re-executed as probes P5, P6, P7 |
| 5.0 | Test: cache written, cache hit, worker resolution | Verified | All three pass under `-race` |
| 5.0 | CLI: timed cold/warm transcript and cache keys | Verified | Re-executed as probe P9 with `PATH=/nonexistent` on the warm run |
| 5.0 | CLI: grep for `list` flag variables | Verified | Re-run; comment-only match |
| 6.0 | Diff: `commands-core.md` List Worktrees section | Verified | Present in `789081f`; no `(unaligned)` or `-sdirty` remains in `docs/site` |
| 6.0 | Diff: `configuration.md` `status_workers` row | Verified | Present in `789081f` |
| 6.0 | CLI: `mkdocs build --strict` | Verified | Re-run, exit 0 |
| 6.0 | CLI: help beside documentation | Verified | Probe P12/P13 match the documented flags |

## 3) Validation Issues

| Severity | Issue | Impact | Recommendation |
| --- | --- | --- | --- |
| LOW | A scratch build path containing the local username appeared in `30-proofs/30-task-06-proofs.md` (the `mkdocs` output line). Evidence: validation grep for private paths. | Traceability only; no credential | **Fixed during validation:** the path was replaced with `site`. |
| LOW | The `make ci` artifact in the task 3 and task 4 proofs lacked the per-artifact "What it proves" line the proof format requires. Evidence: proof-structure count (2 of 3 artifacts annotated). | Slower review | **Fixed during validation:** lines added. |
| LOW | `cmd/omgitworks/worktree_scope_test.go` changed (three `runWorktreeList` call sites) but is not in the planning Relevant Files table. Evidence: changed-file comparison. Linkage exists in the task 2 proof ("eight pre-existing callers") and commit `9c1c754`. | None; supporting file with linkage (GATE D2) | No action. |

Resolved deviation, recorded for the reader rather than as an issue: the spec originally listed `-sdirty` as an attached filter spelling. pflag rejects it (`unknown shorthand flag: 'd' in -dirty`), so the spec, help, task 4.5, and docs were amended during task 4.0 and a test pins the rejection. Supported forms are `-s=dirty` and `--status=dirty`. Separately, the transcripts in the proof files show an over-wide `PATH` column because scratch paths were rewritten after capture; each proof says so.

## 4) Evidence Appendix

### Commits analyzed (base `78e3910`, release 3.5.0)

| Commit | Files |
| --- | --- |
| `ad827a7` docs: plan the worktree status feature | spec, questions, tasks, audit |
| `19da74a` refactor(list): extract the status fetch pipeline | `list.go`, `status_fetch.go`, `status_fetch_test.go`, task 1 proof, tasks |
| `9c1c754` feat(worktree): show compact status in worktree list | `worktree_list.go`, `worktree_list_status_test.go`, `worktree_list_test.go`, `worktree_scope_test.go`, task 2 proof, tasks |
| `3b2e7ca` feat(worktree): add the STATUS column to worktree list | `worktree_list.go`, status tests, task 3 proof, tasks |
| `a5a6b51` feat(worktree): filter worktree list by status | `worktree_list.go`, status tests, spec (amendment), task 4 proof, tasks |
| `ca7297d` feat(worktree): share the status cache and workers | `worktree_list.go`, status tests, task 5 proof, tasks |
| `789081f` docs(worktree): document worktree status | `worktree.go`, `commands-core.md`, `configuration.md`, task 6 proof, tasks |

Core files changed: `list.go`, `status_fetch.go`, `worktree_list.go`, `worktree.go`. All four are in the Relevant Files table and each maps to a task above. Relevant Files entries that remained unchanged (`internal/git/*`, `internal/config`, `internal/filter`, `list_test.go`, `main_test.go`, two helper test files) were consumed without modification, as the table said they would be.

### Commands executed

~~~bash
go test -race ./cmd/omgitworks -run 'TestFetchStatuses|TestWorktreeListStatus|TestWorktreeListFlags|TestWorktreeListWorkers|TestWorktreeListHelp|TestRunWorktreeList|TestWorktreeScope|TestHelpListsExactly' -v
# 33 PASS, 0 FAIL
go test -race ./internal/git -run 'TestProgress|TestCache|TestFetchAll|TestGetStale|TestPrefetch'
# ok
make ci
# All CI checks passed!
.venv/bin/mkdocs build --strict
# exit 0
grep -n 'PrefetchInBackground\|FetchAll(' cmd/omgitworks/*.go | grep -v _test.go
# cmd/omgitworks/status_fetch.go:58 and :73 only
grep -n -i -E 'token|secret|password|api[_-]?key|ghp_|AKIA|claude-1000' docs/specs/30-spec-worktree-status/30-proofs/*.md
# no matches after the scrub
~~~

### Independent CLI probes

Run with the built binary against a scratch fixture (`fixture`: clean, dirty, ahead, and a detached worktree; `other`: one dirty worktree). Paths rewritten to `/home/user` after capture.

~~~text
--- P1 no flag, no git on PATH (status opt-in)
REPO     BRANCH      PATH                                                                                                                                                      ALIGNED
-------  ----------  --------------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead       /home/user/.local/share/gws/projects/fixture/ahead     yes
ls: cannot access '~/.gws/status-cache.json': No such file or directory
--- P2 -s (incl. detached)
REPO     BRANCH            PATH                                                                                                                                                      ALIGNED
-------  ----------------  --------------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead       ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead     yes
fixture  clean          ✓  /home/user/.local/share/gws/projects/fixture/clean     yes
fixture  (detached)     ✓  /home/user/.local/share/gws/projects/fixture/detached  yes
fixture  dirty          ✗  /home/user/.local/share/gws/projects/fixture/dirty     yes
--- P3 -S
REPO     BRANCH      PATH                                                                                                                                                      ALIGNED  STATUS
-------  ----------  --------------------------------------------------------------------------------------------------------------------------------------------------------  -------  ------
fixture  ahead       /home/user/.local/share/gws/projects/fixture/ahead     yes      ↑1 ✓
fixture  clean       /home/user/.local/share/gws/projects/fixture/clean     yes      ✓
fixture  (detached)  /home/user/.local/share/gws/projects/fixture/detached  yes      ✓
fixture  dirty       /home/user/.local/share/gws/projects/fixture/dirty     yes      ✗
--- P4 -s -S precedence
fixture  ahead       /home/user/.local/share/gws/projects/fixture/ahead     yes      ↑1 ✓
--- P5 filters
REPO     BRANCH     PATH                                                                                                                                                   ALIGNED
-------  ---------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  dirty   ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes
other    wip     ✗  /home/user/.local/share/gws/projects/other/wip      yes
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead  yes
No worktrees found for 'fixture'
--- P6 positional
6
No worktrees found for 'nosuchrepo'
--- P7 -sdirty rejected
Error: unknown shorthand flag: 'd' in -dirty
--- P8 color
fixture  ahead       /home/user/.local/share/gws/projects/fixture/ahead     yes      ^[[36mM-bM-^FM-^Q1^[[0m ^[[32mM-bM-^\M-^S^[[0m
fixture  ahead       /home/user/.local/share/gws/projects/fixture/ahead     yes      M-bM-^FM-^Q1 M-bM-^\M-^S
--- P9 cache hit without git
fixture  ahead       ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead     yes
fixture  clean          ✓  /home/user/.local/share/gws/projects/fixture/clean     yes
--- P10 missing path row
config files: ['~/.config/gws/config.json']
--- P11 omgw list regression (space-separated filter, json)
No repositories match the specified filters.
[
  {
    "has_worktrees": true,
    "name": "fixture",
    "status": "main ✓"
  },
  {
    "has_worktrees": true,
--- P12 --workers accepted + help
fixture  ahead       ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead     yes
5
--- P13 omgw worktree help line
  gws worktree list [repo|.]            # List all worktrees, with git status via -s/-S

--- P14 list -s clean (space-separated reassignment on list)
Found 2 repositories:

fixture (wt)  ✓
other (wt)    ✓
--- P15 list -S table
Found 2 repositories:

NAME          STATUS
------------  ------
fixture (wt)  main ✓
--- P16 missing worktree path via CLI
REPO     BRANCH            PATH                                                                                                                                                      ALIGNED
-------  ----------------  --------------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead       ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead     yes
fixture  clean          ✓  /home/user/.local/share/gws/projects/fixture/clean     yes
fixture  (detached)     ✓  /home/user/.local/share/gws/projects/fixture/detached  yes
fixture  dirty          ✗  /home/user/.local/share/gws/projects/fixture/dirty     yes
fixture  gone              /home/user/.local/share/gws/projects/fixture/gone      no
--- P16b -s=clean drops the missing one
REPO BRANCH
------- ----------------
fixture ahead
fixture clean
fixture (detached)
--- P16c refresh restores config
Refreshed 1 repository, 1 changed
~~~

**Validation Completed:** 2026-10-27
**Validation Performed By:** Claude Fable 5.1 (claude-fable-5-1)
