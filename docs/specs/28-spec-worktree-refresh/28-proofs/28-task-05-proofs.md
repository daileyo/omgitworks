# Task 05 Proofs - `--dry-run` previews a refresh without changing anything

## Task Summary

`omgw worktree refresh --dry-run` reports what a refresh would change, using the same
per-repository lines and summary as a real run, and changes nothing. Repair and prune
both modify git state, so a dry run skips them. It builds its report from
`git worktree list` plus a check that each path exists, and says so in its output.

## What This Task Proves

- `config.json` is byte-identical after a dry run that reports pending changes.
- Repair doesn't run (a broken `.git` link stays broken), and neither does prune (git
  still records a deleted worktree).
- Nothing under HOME, the workspace, or the external worktrees is created, modified, or
  removed. That includes git's own files, so `git worktree list` itself writes nothing.
- The output opens with the dry-run header and says repair and prune were not run.
- On a fixture with every kind of change, including a repairable worktree and a failing
  repository, the preview exactly matches the real run that follows, apart from the
  header and the summary's wording.

## Evidence Summary

- 6 dry-run tests pass.
- 5 of 5 mutants are caught. One initially survived because of coarse file timestamps.
  That exposed a real blind spot in the filesystem test, which was fixed and re-verified.
- CLI: the config checksum and git's records are unchanged by `--dry-run`, and the real
  run that follows reports exactly the same changes.
- `make ci` passes: vet, golangci-lint with 0 issues, race-detector tests.

## Artifact: Dry-run tests

**Command:**

~~~bash
go test ./cmd/omgitworks -count=1 -v -run 'TestWorktreeRefreshDryRun'
~~~

**Result summary:** All pass.

~~~text
--- PASS: TestWorktreeRefreshDryRun_ConfigByteIdentical (0.05s)
--- PASS: TestWorktreeRefreshDryRun_DoesNotRepair (0.04s)
--- PASS: TestWorktreeRefreshDryRun_DoesNotPrune (0.05s)
--- PASS: TestWorktreeRefreshDryRun_FilesystemUntouched (0.05s)
--- PASS: TestWorktreeRefreshDryRun_StatesCaveat (0.01s)
--- PASS: TestWorktreeRefreshDryRun_OutputParity (0.05s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.251s
~~~

| Spec Unit 3 requirement | Test |
| --- | --- |
| `--dry-run` reports changes and exits without saving | `ConfigByteIdentical`, `FilesystemUntouched`, CLI checksum |
| No repair or prune; report built from the list plus path existence | `DoesNotRepair`, `DoesNotPrune`, `FilesystemUntouched` |
| Output states that repair and prune were not run | `StatesCaveat` |
| Same change format and summary, framed as what would change | `OutputParity`, `StatesCaveat` |
| Configuration, git state, and filesystem untouched | `ConfigByteIdentical`, `DoesNotRepair`, `DoesNotPrune`, `FilesystemUntouched` |

## Artifact: The tests fail against wrong implementations

**Method:** Each mutant was applied to `worktree_refresh.go` in turn, the refresh tests
run, and the file restored and verified byte-identical with `cmp`.

| Mutant | Caught by |
| --- | --- |
| Dry run performs the full sync (repair, prune, list) | `DoesNotRepair`, `DoesNotPrune`, `FilesystemUntouched` |
| Dry run runs prune but not repair | `DoesNotPrune`, `FilesystemUntouched` |
| Dry run saves configuration | `FilesystemUntouched` (after the fix below) |
| Caveat line omitted | `StatesCaveat`, `OutputParity` |
| Dry run inflates the "would change" count | `StatesCaveat` |

**A mutant that survived, and what it revealed:** the first `FilesystemUntouched` compared
size, mode, and modification time. A dry run that called `config.Save` passed it. Because
the configuration was unchanged in memory, the rewrite produced identical bytes, so
`ConfigByteIdentical` couldn't see it either. That was expected. A timestamp comparison
should have seen it, though, and a direct check showed why it didn't: the file was
rewritten and its modification time came back identical to the nanosecond. File
timestamps come from a coarse kernel clock, and the save landed in the same tick as the
fixture's last write.

~~~text
cfg in snapshot: "853 -rw------- 1789579124340639224"
cfg after save:  "853 -rw------- 1789579124340639224"
~~~

The fix is deterministic, with no sleeping. Before the run, every entry under the
snapshot roots is backdated to 2000-01-01, so any write gets a visibly different time.
Regular files are also compared by SHA-256 of their content. With the fix, the save
mutant is caught at the exact file:

~~~text
## MUTANT: dry-run-saves
    worktree_refresh_dryrun_test.go:169: modified during dry run: .../001/.config/gws/config.json
--- FAIL: TestWorktreeRefreshDryRun_FilesystemUntouched (0.04s)
~~~

The unmutated code passes the stricter test, so a real dry run writes nothing anywhere
under the roots, including git's own files.

**A side result that supports repair-before-prune:** the prune-only mutant deleted git's
administrative record for the *mislinked* worktree as well as the dead one (a trimmed
excerpt follows). Pruning without repairing first discards recoverable worktrees, which is
exactly what the spec's ordering requirement guards against.

~~~text
## MUTANT: dry-run-prunes
    removed during dry run: .../svc-a/.git/worktrees/feat-gone
    removed during dry run: .../svc-a/.git/worktrees/mislinked/HEAD
~~~

## Artifact: Preview, then the real run

**What it proves:** In the built binary, a dry run changes nothing and predicts the real
run exactly, including on a repairable worktree, the case the caveat warns about.

**Method:** Built with `go build` and run in a scratch workspace with an isolated `HOME`.
Before the runs, one worktree was deleted by hand, one had its `.git` link deleted, one
was created with plain git, a branch was checked out inside another worktree, and one
repository directory was deleted. The config is shown by the first 16 hex digits of its
SHA-256. The scratch path is replaced by `$DEMO`.

**Result summary:** The dry run and the real run print identical change blocks and
counts. After the dry run, the checksum, git's worktree records, and the broken link are
all unchanged. After the real run, the checksum changes and repair has restored the
link. The only difference in the error line is the first git step named: `list` in the
dry run, `repair` in the real run.

~~~text
# Setup (output hidden): three repositories with omgitworks-created worktrees. Then, outside omgitworks:
#   svc-a: feat-gone deleted by hand; feat-mislinked's .git link deleted (repairable); feat-ext added with plain git
#   svc-b: a new branch checked out inside feat-switch
#   svc-c: repository directory deleted
$ sha256sum config.json
0e9419e453e34ae3

$ omgitworks worktree refresh --dry-run
Dry run — no changes will be made:
git worktree repair and prune were not run, so a worktree that repair would fix is shown as it currently stands.

[svc-a]
  added      feat-ext  $DEMO/elsewhere/feat-ext  unaligned
  removed    feat-gone  $DEMO/home/.local/share/gws/projects/svc-a/feat-gone
[svc-b]
  branch     feat-switched  $DEMO/home/.local/share/gws/projects/svc-b/feat-switch  was feat-switch

Would refresh 2 repositories, 2 would change, 1 failed

1 error:
  svc-c: git worktree list --porcelain failed: chdir $DEMO/ws/svc-c: no such file or directory
(exit 1)

$ sha256sum config.json   # unchanged by the dry run
0e9419e453e34ae3

$ ls .git/worktrees/ in svc-a   # git's records untouched: no prune
feat-ext
feat-gone
feat-mislinked

$ test -e feat-mislinked/.git   # still missing: no repair
(exit 1)

$ omgitworks worktree refresh
[svc-a]
  added      feat-ext  $DEMO/elsewhere/feat-ext  unaligned
  removed    feat-gone  $DEMO/home/.local/share/gws/projects/svc-a/feat-gone
[svc-b]
  branch     feat-switched  $DEMO/home/.local/share/gws/projects/svc-b/feat-switch  was feat-switch

Refreshed 2 repositories, 2 changed, 1 failed

1 error:
  svc-c: git worktree repair failed: chdir $DEMO/ws/svc-c: no such file or directory
(exit 1)

$ sha256sum config.json   # the real run saved
b5bd8588604a43bb

$ test -e feat-mislinked/.git   # restored by repair
(exit 0)
~~~

**On the caveat:** In this run and in `OutputParity`, the repairable worktree gave the same
result with and without repair. Its directory exists and git still lists it, so it is kept
either way. The caveat is still stated as the spec requires, and it holds where repair
would change what `git worktree list` reports.

## Artifact: Repository quality gate

**Command:**

~~~bash
PATH=$HOME/go/bin:$PATH make ci
~~~

**Result summary:** Vet is clean, lint reports 0 issues, and every package passes under
`-race`.

~~~text
Running linter...
0 issues.
...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.865s
All CI checks passed!
~~~

## Deviations From the Task Plan

- **5.2:** The dry-run branch is a small `refreshedWorktrees` helper, so the loop, diff, and
  reporting are shared with the real run.
- **5.4:** Output opens with the header `Dry run — no changes will be made:`, already used
  by `worktree align` and `worktree remove`, followed by the caveat.
- **5.5:** Summary: `Would refresh N repositories, M would change[, K failed]`.
- **5.6:** Reuses spec 27's `configBytes`. Adds `FilesystemUntouched`, with the backdating
  fix described above. `DoesNotRepair` checks the missing `.git` file directly.

## Reviewer Conclusion

`--dry-run` is a faithful, side-effect-free preview. It changes no configuration, git
state, or file, and on every kind of change tested it predicts the real run exactly. The
one test gap found along the way, coarse timestamps hiding a same-content rewrite, was
fixed and verified against the mutant that exposed it.
