# 27-validation-worktree-remove.md

## 1) Executive Summary

- **Overall:** **PASS** (after remediation). The first validation run **FAILED GATE A** on
  one HIGH defect and also found one MEDIUM defect. Both were fixed, regression-tested,
  and re-verified live within this phase.
- **Implementation Ready:** **Yes.** All 27 functional requirements verify against
  independently re-executed evidence, and both defects are fixed with regression tests
  proven to fail on the original code.
- **Key metrics:**
  - Requirements verified: **27/27 (100%)**, 0 Unknown
  - Proof artifacts working: **5/5 documents**
  - Files changed: **11 non-spec files** — 4 core, 7 supporting
  - Test suite: **450 top-level pass, 0 fail, 1 skip** (448 + 2 regression tests)

Neither defect is visible in any implementation-phase test or transcript. Both surfaced
from reading the code for suspect patterns and then reproducing them against the real
binary.

## 2) Coverage Matrix

### Functional Requirements

| Requirement | Status | Evidence |
| --- | --- | --- |
| U1-1 `remove <repo> <branch>` with `rm` alias | Verified | `Aliases: []string{"rm"}`; `TestWorktreeRemoveCmd_Alias` resolves through cobra lookup; live `worktree rm` resolved |
| U1-2 Git-level `git worktree remove` | Verified | `git.RemoveWorktree`; `TestRemoveWorktree` |
| U1-3 One positional uses the resolved repo | Verified | `selectWorktreeTargets` default branch; `TestRunWorktreeRemove_CurrentRepo` |
| U1-4 Exact branch match, not partial | Verified | `findWorktreeForBranch` uses `==`; `TestRunWorktreeRemove_ExactBranchMatch` asserts `feat` does not match `feat-auth` |
| U1-5 Unmatched branch exits non-zero naming both | Verified | `worktreeNotFoundError`; `TestRunWorktreeRemove_NoSuchBranch`; live error names repo and branch |
| U1-6 No `--force` by default; git's refusal surfaced | Verified | `--force` appended only when set; `TestRemoveWorktree_DirtyRefused`, `_UntrackedRefused`; **live**: dirty without `--force` → exit 1, worktree survives |
| U1-7 `--force` passes through | Verified | **live**: dirty with `--force` → exit 0, removed |
| U1-8 Locks skipped even with `--force` | Verified | Lock checked in `buildRemovalPlan` before `git.RemoveWorktree` is reachable; `TestRunWorktreeRemove_LockedSkipped` (force on and off); **live** with a real `git worktree lock` and `--force` → skipped, worktree survives |
| U1-9 Branch never touched | Verified | No `branch -d`/`-D` anywhere in `internal/git/worktree.go`; `TestRemoveWorktree_BranchSurvives`; **live**: branch present after a forced removal |
| U1-10 Remove *the repository's* dir only if empty | **Verified after fix** | Originally deleted a sibling repository's directory — see Issue 1. Now `TestRunWorktreeRemove_CleansEmptyDirs` and `TestRunWorktreeRemove_CleanupRespectsSiblingRepo`; live re-run leaves `svc-a-old` intact |
| U1-11 Worktree data updated and saved | Verified | `refreshWorktreeData` + one `config.Save`; `TestRunWorktreeRemove_Individual` reloads from disk |
| U2-1 `-t`/`--tag`, `MatchesExact`, single value | Verified | `StringArrayVarP` + shared `singleTagValue`; selection via shared `filterByTag` |
| U2-2 Removes across every tagged repo | Verified | `TestWorktreeRemoveBulk_RemovesAcrossTag` asserts the untagged repo survives |
| U2-3 Spec 26's precedence table applies | Verified | One shared `selectWorktreeTargets`; `TestWorktreeRemoveTargets_PrecedenceTable`, 4/4 rows |
| U2-4 Missing branch is a skip, not a failure | Verified | `TestWorktreeRemoveBulk_SkipsMissingBranch`; `_SingleMatchTagStillBulk` guards spec 26's defect |
| U2-5 Continue past failures; counted summary | Verified | `TestWorktreeRemoveBulk_SummaryAndExit` |
| U2-6 Non-zero on failure, zero on skips only | Verified | Same test, both sub-cases |
| U2-7 No restore after a later failure | Verified | `TestWorktreeRemoveBulk_NoRestore` |
| U2-8 Configuration saved once | Verified | `TestWorktreeRemoveBulk_SingleSave` reloads from disk |
| U3-1 `--dry-run` changes nothing | Verified | `TestWorktreeRemove_DryRunChangesNothing` byte-compares `config.json` |
| U3-2 Preview lists repo/branch/path and a count | **Verified after fix** | Count originally disagreed with the real run — see Issue 2. Now `TestWorktreeRemove_PreviewShowsSkips` |
| U3-3 Preview marks locked and would-fail-dirty | Verified | `TestWorktreeRemove_DryRunAnnotations` |
| U3-4 Multi-worktree runs prompt | Verified | `TestWorktreeRemove_ConfirmationGate` |
| U3-5 `--yes` skips the prompt | Verified | `TestWorktreeRemove_YesSkipsPrompt` with a non-TTY stdin and empty reader |
| U3-6 Non-interactive without `--yes` exits non-zero | Verified | `TestWorktreeRemove_NonInteractiveRefuses`; live exit 1 with nothing removed |
| U3-7 Declining exits zero, removes nothing | Verified | `TestWorktreeRemove_ConfirmationGate/declining_...` |
| U3-8 `--dry-run` with `--yes` is a dry run | Verified | `TestWorktreeRemove_DryRunWithYes` |

### Repository Standards

| Standard Area | Status | Evidence & Compliance Notes |
| --- | --- | --- |
| New command file + git primitive placement | Verified | `cmd/omgitworks/worktree_remove.go`; `RemoveWorktree` beside `AddWorktree`/`MoveWorktree` |
| Cobra structure and native `Aliases` | Verified | Registered in `init()`; first command in the repo to set `Aliases` |
| Real git fixtures, not stubs | Verified | Every removal test builds real repositories and worktrees in `t.TempDir()` |
| Injected I/O for the prompt | Verified | `runWorktreeRemove` takes `io.Writer`/`io.Reader`, following `runNavigate` |
| Testable TTY check | Verified | `stdinIsTerminalFunc` overridable variable, following `cd.go`'s `stdoutIsTerminalFunc` |
| Conservative directory cleanup | **Verified after fix** | Now uses exact containment, the same test `git.IsAligned` uses |
| Error summary format | Verified | `errors []string`, `pluralize`, separately headed block, matching `runWorktreeAlign` |
| Shared targeting, no duplication | Verified | Spec 26's selector extracted with a two-line test diff and zero assertion changes |
| Quality gates | Verified | `make ci` exit 0, `golangci-lint` `0 issues` |

### Proof Artifacts

| Task | Artifact | Status | Result |
| --- | --- | --- | --- |
| 1.0 | `RemoveWorktree` test suite | Verified | Re-run: 6/6 pass |
| 2.0 | CLI: removal with branch surviving | Verified | Reproduced live |
| 2.0 | Individual removal suite | Verified | Re-run: all pass |
| 3.0 | Extraction diff + spec 26 suite | Verified | Two-line diff confirmed; suite green |
| 3.0 | Bulk removal suite | Verified | Re-run: all pass |
| 4.0 | CLI: dry run, refusal, `--yes` run | Verified | Reproduced live |
| 4.0 | Dry-run and confirmation suite | Verified | Re-run: all pass |
| 5.0 | Completion test, docs, help, `make ci` | Verified | Re-run: exit 0 |

## 3) Validation Issues

| Severity | Issue | Impact | Recommendation |
| --- | --- | --- | --- |
| **HIGH** (fixed) | **Directory cleanup deleted another repository's projects directory.** `cleanupEmptyWorktreeDirs` bounded its upward walk with `strings.HasPrefix(dir, repoDir)`. A string prefix is not path containment: `projects/svc-a-old` has the prefix `projects/svc-a`. Evidence: with tracked repos `svc-a` and `svc-a-old`, removing a `svc-a` worktree located inside `projects/svc-a-old/` deleted `projects/svc-a-old`. This violates FR U1-10, which limits cleanup to *the repository's* directory, and is exactly the prefix-matching bug class spec 25 explicitly prohibits. **Blast radius is bounded:** only empty directories can be removed, so no file content was ever at risk — which is why this is HIGH rather than CRITICAL. | Functionality: the most destructive command acted on a resource outside the requested repository | **Fixed.** Containment now requires `dir == repoDir` or a prefix continuing past a path separator, the same idiom `git.IsAligned` already uses. Regression test `TestRunWorktreeRemove_CleanupRespectsSiblingRepo` **fails on the original code** and passes on the fix. Re-verified live |
| **MEDIUM** (fixed) | **The dry-run preview did not match the real run.** The preview omitted repositories that would be skipped for lacking the branch, while the real run reported them; and it counted locked worktrees in `Total` while the real run counted them as skipped. Evidence: with `svc-b` lacking the branch, `--dry-run` never mentioned `svc-b`, while the real run printed `skipped 1`. This breaks success metric 4 ("`--dry-run` output matches the subsequent real run exactly") and hides the very signal the user story wants from a preview — whether the tag matched what was expected. The confirmation prompt shared the same gap. | Verification: a user approving a preview could not see which repositories would be skipped | **Fixed.** Skipped repositories are rendered as `Would skip — ...`, and the footer counts the way the real summary does: `Total: N to remove, M skipped`. Regression test `TestWorktreeRemove_PreviewShowsSkips` asserts the preview count equals the real run's summary, and **fails on the original code**. Docs example updated |
| LOW | Task 5.0 remains out of the spec's stated scope, as flagged in planning. | Traceability | Accept, consistent with spec 26's task 4.0 |
| LOW | The first `make ci` run during validation exited 2 with `parallel golangci-lint is running`. No lint process was running when inspected, and an immediate re-run passed. Environmental lock contention, not a code defect. | None | No action. Noted so the transient failure is not mistaken for a regression |

**Why the implementation phase missed both defects.** Each test exercised only the
favourable shape: every cleanup fixture used a uniquely named repository with no
similarly named sibling, and `TestWorktreeRemove_PreviewMatchesRun`'s fixture had no
skipped or locked repositories. This repeats the pattern behind spec 26's validation
defect. Both new regression tests were checked against the original code to confirm they
are genuine guards rather than tests written to pass.

## 4) Evidence Appendix

### Gates applied

| Gate | Run 1 | Run 2 (after fixes) | Basis |
| --- | --- | --- | --- |
| A — no CRITICAL/HIGH | **FAIL** | **PASS** | One HIGH and one MEDIUM, both fixed and re-verified |
| B — no `Unknown` in matrix | PASS | PASS | 27/27 Verified |
| C — proof artifacts functional | PASS | PASS | All re-executed or reproduced |
| D — file integrity | PASS | PASS | 4 core files, all in Relevant Files; the two spec 26 test files changed by one line each for the planned rename |
| E — repository standards | **FAIL** | **PASS** | Cleanup violated spec 25's exact-matching standard until fixed |
| F — no credentials | PASS | PASS | No keys, tokens, or absolute home paths in proofs |

### Commits analyzed

| Commit | Scope |
| --- | --- |
| `cb02b74` | Planning: task list and audit |
| `dea2b15` | T1.0-T5.0 implementation |

Baseline `f4b0601` (spec 26's validation fix). The validation fixes land as a follow-up
commit on top.

### Regression tests proven against the original code

~~~text
=== against the ORIGINAL (buggy) implementation — both must FAIL ===
--- FAIL: TestWorktreeRemove_PreviewShowsSkips
    preview omits svc-b, which the real run will skip
    preview count does not match the real run's accounting
--- FAIL: TestRunWorktreeRemove_CleanupRespectsSiblingRepo
    cleanup deleted another repository's projects directory .../projects/svc-a-old

=== restored FIXED implementation — both must PASS ===
ok  	github.com/daileyo/omgitworks/cmd/omgitworks
~~~

### Defect 1, before and after

~~~text
BEFORE: projects/svc-a-old exists: yes
        Removed worktree for branch 'feat-stray' from svc-a
AFTER:  projects/svc-a-old exists: NO   -> another repository's directory deleted

FIXED:  projects/svc-a-old exists: yes  -> not affected
~~~

### Defect 2, after the fix — preview and real run agree

~~~text
$ omgw worktree remove -t backend feat-x --dry-run
Would remove [svc-a] feat-x
Would remove [svc-c] feat-x
Would skip — no worktree for branch 'feat-x' in repository 'svc-b'
Total: 2 worktrees to remove, 1 skipped

$ omgw worktree remove -t backend feat-x --yes
Skipping — no worktree for branch 'feat-x' in repository 'svc-b'
Removed worktree for branch 'feat-x' from svc-a
Removed worktree for branch 'feat-x' from svc-c
Removed 2 worktrees, skipped 1
~~~

### Live verification through real flag parsing

~~~text
dirty, no --force          exit=1  worktree survives
dirty, --force             exit=0  removed
branch after forced remove feat-dirty still present
locked (git worktree lock), WITH --force
  Skipping [svc] feat-locked — worktree for branch 'feat-locked' is locked: in review
  worktree survives
rm alias                   resolves to remove
~~~

---

**Validation Completed:** 2026-09-16
**Validation Performed By:** Claude Opus 5
