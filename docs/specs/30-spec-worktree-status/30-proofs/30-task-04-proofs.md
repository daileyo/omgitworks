# Task 04 Proofs - Status filtering with the positional kept as the repository

## Task Summary

`-s=<pattern>`, `--status=<pattern>`, `-S=<pattern>`, and `--show-status=<pattern>` now show status and keep only the worktrees whose status text matches, using the same `statusFilterText` and `filter.MatchesPattern` that `omgw list` uses. Filtering runs after repository scoping. A worktree with no readable status never matches. The positional argument is always the repository, so `omgw worktree list -s fixture` scopes to `fixture` with compact status. A filtered-to-nothing result prints the existing no-worktrees message and exits zero.

One planned form was dropped: the spec listed `-sdirty` as a third spelling, but pflag parses the letters after a sentinel-valued short flag as more short flags and rejects it. The spec, help, and task list were amended, and a test pins the rejection so the docs stay honest.

## What This Task Proves

- `-s=dirty` keeps only the dirty worktree; `-S=ahead` keeps only the ahead worktree and shows the STATUS column.
- `-s fixture` treats `fixture` as the repository and nothing as a filter; `-s=dirty` and `--status=dirty` set the filter; `-sdirty` is rejected.
- With two repositories, scoping to `fixture` plus `dirty` lists only that repository's dirty worktree, while the unscoped filter lists both.
- A worktree with a missing directory is dropped by any filter.
- An empty result prints `No worktrees found for 'fixture'` when scoped and `No worktrees found` otherwise, with a nil error.

## Evidence Summary

- Six new tests pass, including the flag-parsing cases through cobra's `ParseFlags`.
- `make ci` exits zero.
- A transcript against a two-repository scratch workspace shows cross-repository filtering, the positional contract, the empty case, and the rejected joined form.

## Artifact: Filtering and parsing tests

**What it proves:** Every Unit 2 requirement of the spec.

**Command:**

~~~bash
go test ./cmd/omgitworks -run 'TestWorktreeListStatus_(Filter|ShowAndFilter|ScopeThen|NoStatus)|TestWorktreeListFlags' -v
~~~

**Result summary:** All pass. `TestWorktreeListFlags_PositionalIsRepo` drives `worktreeListCmd.ParseFlags` directly and checks both the parsed options and the remaining positional.

~~~text
--- PASS: TestWorktreeListStatus_FilterDirty (0.07s)
--- PASS: TestWorktreeListStatus_ShowAndFilterAhead (0.07s)
--- PASS: TestWorktreeListFlags_PositionalIsRepo (0.00s)
--- PASS: TestWorktreeListStatus_ScopeThenFilter (0.08s)
--- PASS: TestWorktreeListStatus_NoStatusExcluded (0.07s)
--- PASS: TestWorktreeListStatus_FilterEmpty (0.07s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.368s
~~~

## Artifact: Two-repository transcript

**What it proves:** Cross-repository filtering, scope-then-filter, the positional contract, the empty case, and the `-sdirty` rejection, all from the command line.

**Why it matters:** The scope and filter interplay is only visible with more than one repository; the fixture has `fixture` (clean, dirty, ahead) and `other` (one dirty worktree). Paths are rewritten to `/home/user` after capture, so the PATH column is sized for the original, longer paths.

**Command:** see each `$` line.

**Result summary:** `-s=dirty` returns the dirty worktree of each repository; `-s fixture` lists all three `fixture` worktrees with compact icons; `fixture -S=behind` prints the scoped empty message; `-sdirty` fails with pflag's unknown-shorthand error.

~~~text
$ omgw worktree list -s
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean      ✓  /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty      ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes
other    wip        ✗  /home/user/.local/share/gws/projects/other/wip      yes

$ omgw worktree list -s=dirty
REPO     BRANCH     PATH                                                                                                                                                   ALIGNED
-------  ---------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  dirty   ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes
other    wip     ✗  /home/user/.local/share/gws/projects/other/wip      yes

$ omgw worktree list -s fixture
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean      ✓  /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty      ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes

$ omgw worktree list fixture -S=behind
No worktrees found for 'fixture'

$ omgw worktree list -sdirty
Error: unknown shorthand flag: 'd' in -dirty
~~~

## Artifact: Full quality gate

**What it proves:** Vet, golangci-lint, and the race-detector test run pass across every package after this task's changes.

**Why it matters:** This is the same gate CI runs.

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit zero.

~~~text
Tests complete
All CI checks passed!
~~~

## Reviewer Conclusion

Status filtering on worktrees uses the same vocabulary and matcher as `omgw list`, composes correctly with repository scoping, and never consumes the positional argument. The one deviation from the plan, dropping `-sdirty`, is recorded in the spec and pinned by a test.
