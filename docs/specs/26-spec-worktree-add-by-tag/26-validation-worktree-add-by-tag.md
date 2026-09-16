# 26-validation-worktree-add-by-tag.md

## 1) Executive Summary

- **Overall:** **PASS** (after remediation). The first validation run **FAILED GATE A**
  on one HIGH defect, which was fixed and re-verified within this phase.
- **Implementation Ready:** **Yes.** All 21 functional requirements verify against
  independently re-executed evidence, and the defect found during validation is fixed,
  regression-tested, and reproduced as corrected against the live binary.
- **Key metrics:**
  - Requirements verified: **21/21 (100%)**, 0 Unknown
  - Proof artifacts working: **4/4 documents, 16/16 artifacts**
  - Files changed: **13** — 4 core (all in Relevant Files), 9 supporting
  - Test suite: **416 top-level pass, 0 fail, 1 skip** (1075 pass including sub-tests)

Validation re-ran the gates and reproduced behavior in clean sandboxes rather than
reading the proof documents. That is how the defect below was found: it is not visible in
any test or transcript the implementation phase produced.

## 2) Coverage Matrix

### Functional Requirements

| Requirement | Status | Evidence |
| --- | --- | --- |
| U1-1 `-t`/`--tag` selects repositories by tag | Verified | `worktree_add.go:58` `StringArrayVarP`; `filterByTag`; `TestWorktreeAddSelection_PrecedenceTable` row 2; live run created 3 worktrees across `backend` |
| U1-2 Tag matched with `filter.MatchesExact` | Verified | `filterByTag` calls `filter.MatchesExact`; `TestWorktreeAddSelection_TagMatching` passes exact, case-insensitive, `back*`, `backen?` |
| U1-3 `--tag` given twice rejected | Verified | `worktreeAddTag()` returns an error for `len > 1`; `TestWorktreeAddCmd_TagFlag/repeated_rejected`; live: `Error: --tag accepts a single value, but was given 2 times`, exit 1 |
| U1-4 With `-t`, one positional is the branch | Verified | `runWorktreeAddResolved` takes `args[len(args)-1]` as branch; precedence row 2 |
| U1-5 Create per matched repo, existing behavior | Verified | `createWorktreeForRepo` retains the original `git.AddWorktree` call; `TestWorktreeAddBulk_CreatesForEachTagged` |
| U1-6 Parent dirs for slashed branch names | Verified | `os.MkdirAll(filepath.Dir(destPath))` preserved; `TestWorktreeAddBulk_BranchWithSlash` |
| U1-7 Existing worktree skipped, not failed | **Verified after fix** | Originally broken for a single-match tag — see Issue 1. Now `TestWorktreeAddBulk_SkipsExisting` and `TestWorktreeAddBulk_SingleMatchTagStillBulk`; live re-verification shows `Skipping [...]`, exit 0 |
| U1-8 Unmatched tag exits non-zero naming it | Verified | `TestWorktreeAddSelection_EmptyMatches/tag_alone`; live: `Error: no repository found tagged 'nosuchtag'`, exit 1 |
| U1-9 Configuration saved once | Verified | `grep config.Save` in the creation path returns **one** call site, outside the loop; `createWorktreeForRepo` contains **zero**; `TestWorktreeAddBulk_SingleSave` reloads from disk and asserts every repo recorded |
| U2-1 Four-row precedence table | Verified | `selectWorktreeAddTargets` switch; `TestWorktreeAddSelection_PrecedenceTable` has one passing sub-case per row |
| U2-2 Name plus tag requires both | Verified | `filterByTag(selectByName(...))`; `TestWorktreeAddSelection_CombinedAnd` asserts exclusion in both directions |
| U2-3 Name pattern uses `filter.MatchesPattern` | Verified | `selectByName` calls it; pre-existing single-repository tests still pass |
| U2-4 Resolver only when neither is supplied | Verified | `repocontext.ResolveCurrent` appears **only** in the `default:` branch of the switch, unreachable when a pattern or tag is set; `TestWorktreeAddSelection_ResolverNotConsulted` (3 sub-cases) |
| U2-5 Combined empty match names both filters | Verified | `TestWorktreeAddSelection_EmptyMatches/combined_names_both` |
| U2-6 Two-positional form still rejects ambiguity | Verified | `TestWorktreeAddSelection_AmbiguityAfterTagFilter`: `api` alone errors with the original wording; `api` + `-t backend` returns both (see Assumption) |
| U3-1 Continue past failures | Verified | The two `return err` statements in the loop are `single`-guarded and unreachable in a bulk run; `TestWorktreeAddBulk_ContinuesPastFailure` asserts the repo *after* the failure is created |
| U3-2 Errors collected and printed after the run | Verified | `failures []string` collected, printed after the loop in `worktree_align`'s format; live mixed run shows the headed error block |
| U3-3 Summary reports created/skipped/failed | Verified | `printWorktreeAddSummary`; `TestWorktreeAddBulk_SummaryCounts`; live: `Created 1 worktree, skipped 1, 1 failed` |
| U3-4 Each failure names repo and reason | Verified | `TestWorktreeAddBulk_NamesFailedRepos`; live: `svc-b: failed to create worktree: ... no such file or directory` |
| U3-5 Non-zero on failure, zero on skips only | **Verified after fix** | `TestWorktreeAddBulk_ExitStatus` (3 sub-cases); live: mixed run exit 1, skip-only run exit 0 |
| U3-6 Successful creations retained | Verified | No `Remove`/rollback call anywhere in `runWorktreeAddBulk` (grep count 0); `TestWorktreeAddBulk_NoRollback` breaks the last repo and asserts two survive on disk and in config |

### Repository Standards

| Standard Area | Status | Evidence & Compliance Notes |
| --- | --- | --- |
| Extends rather than forks the command | Verified | All behavior lands in `worktree_add.go`; no parallel command added |
| Cobra structure | Verified | `-t` registered on `worktreeAddCmd` in its existing `init()` |
| Filtering reuse | Verified | `filter.MatchesExact` for tags, `filter.MatchesPattern` for names; no new matching logic |
| Bulk output format | Verified | `errors []string` collection, `pluralize` counts, separately headed error block — matches `runWorktreeAlign` |
| Testing patterns | Verified | `t.TempDir()`, `t.Setenv`, `t.Cleanup`, `captureStdoutStr`, table-driven sub-tests |
| Quality gates | Verified | `make ci` exits 0; `golangci-lint` reports `0 issues` with no new exclusions |
| Commit conventions | Verified | `feat(worktree):` and `docs:`, each naming its tasks and spec |
| Documentation | Verified | `docs/site/commands-core.md` updated in the same branch |

### Proof Artifacts

| Task | Proof Artifact | Status | Verification Result |
| --- | --- | --- | --- |
| 1.0 | Precedence table test | Verified | Re-run: 4/4 sub-cases pass |
| 1.0 | Tag matching / AND / resolver-precedence tests | Verified | Re-run: all pass incl. 4 matching forms |
| 1.0 | Ambiguity test pinning the assumption | Verified | Re-run: passes |
| 1.0 | CLI: `--tag` twice, unmatched tag | Verified | Reproduced: exit 1 each, single clean line |
| 2.0 | CLI: bulk creation across a tag | Verified | Reproduced: 3 created, untagged repo untouched |
| 2.0 | Test: single-save persistence | Verified | Re-run: passes; `config.Save` call sites counted independently |
| 2.0 | Test: slashed branch, skip, single-repo contract | Verified | Re-run: all pass |
| 3.0 | CLI: mixed-outcome run | Verified | Reproduced: skip + create + fail, correct counts, exit 1 |
| 3.0 | Test: exit-status table | Verified | Re-run: 3/3 sub-cases pass |
| 3.0 | Test: continue-past-failure, naming, no rollback | Verified | Re-run: all pass |
| 4.0 | CLI + test: `--tag` completion | Verified | Reproduced: `backend`, `frontend` returned deduplicated |
| 4.0 | Documentation diff | Verified | Precedence table, single-value note, partial-failure example all present |
| 4.0 | `list.go` description fix | Verified | Now `Filter by tag (single value)`, matching the single-valued `StringVarP` |
| 4.0 | CLI: `make ci` | Verified | Re-run: exit 0 |

## 3) Validation Issues

| Severity | Issue | Impact | Recommendation |
| --- | --- | --- | --- |
| **HIGH** (fixed) | **A tag matching exactly one repository took the single-repository code path.** `runWorktreeAddBulk` derived its mode as `single := len(repos) == 1`, conflating "one repository selected" with "invoked in single-repository mode". Evidence: with one repo tagged `solo` already holding the branch, `omgw worktree add -t solo feat-x` printed `Error: worktree for branch 'feat-x' already exists ...` and exited **1**, with no summary. FR U1-7 requires a reported skip and FR U3-5 requires exit **0** when every matched repository succeeded or was skipped. Not caught by any implementation-phase test, all of which used multi-repository tags. | Functionality: a realistic invocation (narrow tag, or a tag narrowed by name) fails where the spec requires success | **Fixed during validation.** `single` is now passed by the caller from the invocation shape — `tag == ""` — so a tag is always a bulk run regardless of match count. Regression test `TestWorktreeAddBulk_SingleMatchTagStillBulk` added. Re-verified live: skip reported, exit 0; and the no-tag single-repository contract still errors with exit 1 |
| LOW | Task 4.0 remains out of the spec's stated scope, as the planning audit flagged. It adds `--tag` completion, documentation, and a one-line change to `cmd/omgitworks/list.go`, none of which corresponds to a functional requirement in spec 26. | Traceability: a core file outside the spec's subject area was modified | Accept, or split the `list.go` correction into its own commit. It is justified in the task file, the audit, and the task 4.0 proof, and it fixes a description that contradicted both the code and the published docs |
| LOW | The planning assumption about ambiguity plus `-t` is implemented but still unconfirmed by the user. Evidence: `TestWorktreeAddSelection_AmbiguityAfterTagFilter` pins "bulk over the intersection". | Behavior: `add <repo> <branch> -t <tag>` semantics | Confirm the reading. Reverting affects one test and two sub-tasks only |

## 4) Evidence Appendix

### Gates applied

| Gate | Run 1 | Run 2 (after fix) | Basis |
| --- | --- | --- | --- |
| A — no CRITICAL/HIGH | **FAIL** | **PASS** | One HIGH defect found, fixed, regression-tested, re-verified live |
| B — no `Unknown` in matrix | PASS | PASS | 21/21 Verified |
| C — proof artifacts functional | PASS | PASS | 16/16 re-executed or reproduced |
| D — file integrity | PASS | PASS | 4 core files changed, all in Relevant Files; 9 supporting files linked via commits |
| E — repository standards | PASS | PASS | 8/8 areas Verified |
| F — no credentials | PASS | PASS | No keys, tokens, emails, or absolute home paths in proofs |

### Commits analyzed

| Commit | Scope | Core files |
| --- | --- | --- |
| `fb6ead3` | Planning (tasks + audit) | none |
| `ed18256` | T1.0-T4.0 implementation | `worktree_add.go`, `worktree.go`, `list.go`, `commands-core.md` |

Baseline: `a435e09` (spec 25 validation). Total: 13 files, +1737 / -32. The validation
fix is a follow-up commit on top.

### Commands executed during validation

~~~bash
git log --oneline a435e09..HEAD && git diff --stat a435e09..HEAD
make ci                                            # exit 0, 416 top-level pass
grep -n "config.Save" cmd/omgitworks/worktree_add.go   # one call site, outside the loop
sed -n '/func createWorktreeForRepo/,/^}/p' ... | grep -c config.Save   # 0
sed -n '/func selectWorktreeAddTargets/,/^}/p' ... | grep ResolveCurrent  # default branch only
sed -n '/func runWorktreeAddBulk/,/^}/p' ... | grep -cE "RemoveAll|rollback"  # 0
# clean sandbox: 4 repos, 3 tagged backend; then a 2-repo sandbox with a single-match tag
omgw worktree add -t backend feat-auth / feat-mix
omgw worktree add -t solo feat-x        # the defect, and its fix
omgw __complete worktree add --tag ""
~~~

### The defect, before and after

~~~text
BEFORE — tag matching one repo, branch already present:
$ omgw worktree add -t solo feat-x
Error: worktree for branch 'feat-x' already exists at <s>/data/gws/projects/only-one/feat-x
  exit=1                                   <-- FR U1-7 and U3-5 both violated

AFTER:
$ omgw worktree add -t solo feat-x
Skipping [only-one] feat-x — worktree for branch 'feat-x' already exists at <s>/...

Created 0 worktrees, skipped 1
  exit=0                                   <-- correct

AND the single-repository contract is unchanged:
$ omgw worktree add only-one feat-x
Error: worktree for branch 'feat-x' already exists at <s>/...
  exit=1                                   <-- pre-existing behavior preserved
~~~

---

**Validation Completed:** 2026-09-15
**Validation Performed By:** Claude Opus 5
