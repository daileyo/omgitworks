# 30-tasks-worktree-status.md

> Source spec: `docs/specs/30-spec-worktree-status/30-spec-worktree-status.md`
>
> Branch: `feat/worktree-status`, based on `main` at `78e3910` (release 3.5.0). Builds on **spec 19** (`status-compact`) for the `-s`/`-S` convention, **spec 20** (`worktree-management`) for `omgw worktree list`, and **spec 25** (`worktree-repo-context`) for `.` targeting. No sibling branch is known to touch `list.go` or `worktree_list.go`.

## Planning Notes (Codebase Assessment)

These findings come from reading the code on this branch and refine what the spec assumed. They are recorded here so implementation does not re-derive them.

- **The status pipeline in `runList` has no direct tests.** `list_test.go` exercises `displayTable`, `displayJSON`, and the flag parsing with a pre-built `map[string]*git.Status`; the cache, `FetchAll`, `GetStale`, and `PrefetchInBackground` are tested in `internal/git/status_test.go`. Nothing drives `runList` end to end against a cache file. The spec's "existing `list` tests are the regression net" holds for display and parsing but not for the two-phase fresh/stale/fetch/save/prefetch ordering, so the extracted helper (task 1.0) must carry its own tests for that ordering.
- **The cache file lives at `$HOME/.gws/status-cache.json`** via `os.UserHomeDir`, not under XDG. Tests already redirect `HOME` to a temp dir (`saveConfigForWorktreeTests` in `worktree_list_test.go`), so cache-sharing tests can assert on that file without new plumbing.
- **Everything to reuse is already in `package main`.** `formatStatusIcons`, `statusFilterText`, `displayWidth`, `stripANSI`, `colorize`, `resolveWorkers`, and `showColumnSentinel` are unexported in `list.go` and callable from `worktree_list.go` as is. Nothing needs exporting.
- **`resolveColorEnabled` reads the package-level `flagColor`.** Its `*cobra.Command` parameter is ignored. For the worktree list to have its own `--color` flag it either needs a value parameter or the worktree command must share the variable. Sharing a variable between two commands is the pattern this spec wants to avoid, so the sub-tasks should add a parameter.
- **`reassignTrailingArg` scans `os.Args` for `list`'s uppercase letters.** It is specific to `list`'s flag table (`shortToShowFlag`) and must not be called from `worktree list`, where the positional is the repository. This is the mechanism behind the spec's attached-value rule.
- **`worktreeCmd` has a persistent `-q/--quiet`.** None of `-s`, `-S`, `--workers`, or `--color` collide with it or with any flag on `worktree list`'s siblings.
- **Column widths in `runWorktreeList` use `len()`.** Branch and path are ASCII today, but icons are multi-byte and colored icons carry ANSI codes. The table must switch to `displayWidth` for any column that can hold icons.
- **Progress renders only on a TTY stderr** with a 200 ms delay, so tests never see it and the CLI transcript is the only place it is observable.

## Relevant Files

| File | Why It Is Relevant |
| --- | --- |
| `cmd/omgitworks/status_fetch.go` | New. Home of the extracted pipeline `fetchStatuses(paths []string, workers int) (map[string]*git.Status, func())`: cache load, fresh/stale/foreground phases, progress spinner, save, background prefetch. |
| `cmd/omgitworks/status_fetch_test.go` | New. Ordering tests for the helper: all-fresh means no fetch, stale served then refreshed, uncached fetched and saved, save precedes the background phase. |
| `cmd/omgitworks/list.go` | Owns `runList` (the pipeline to extract, lines 443-508), `resolveColorEnabled` (gains a value parameter), and the reusable unexported helpers `formatStatusIcons`, `statusFilterText`, `displayWidth`, `resolveWorkers`, `showColumnSentinel`. |
| `cmd/omgitworks/list_test.go` | Existing display and parsing tests; must pass with no assertion changes. Holds `captureStdout`. |
| `cmd/omgitworks/worktree_list.go` | The command being extended: new flags, `worktreeListOptions`, status fetch call, filtering, `ALIGNED` rename, compact and full rendering with `displayWidth`. |
| `cmd/omgitworks/worktree_list_test.go` | Five existing tests; their `runWorktreeList` calls gain an options argument and two assertions move from `aligned`/`(unaligned)` to `yes`/`no`. Holds `saveConfigForWorktreeTests`, which redirects `HOME`. |
| `cmd/omgitworks/worktree_list_status_test.go` | New. The real-git `statusFixture` (clean, dirty, ahead, and missing worktrees) and every status display, precedence, filter, positional, cache, and worker test. |
| `cmd/omgitworks/worktree.go` | `worktreeCmd.Long` subcommand block (line 32) describes `list`; persistent `-q` lives here. |
| `cmd/omgitworks/worktree_discover_test.go` | Provides `gitWorktreeAdd(t, repoDir, branch, wtPath)` for the fixture. |
| `cmd/omgitworks/worktree_refresh_test.go` | Provides `runGitIn(t, dir, args...)` for the fixture. |
| `cmd/omgitworks/main_test.go` | Provides `helpListedFlags(t, cmd)` for the help-text test. |
| `internal/git/status.go` | `GetStatus` and the `Status` struct. Consumed unchanged. |
| `internal/git/cache.go` | `NewCache`, `Get`, `GetStale`, `FetchAll`, `Save`, `Load`, `PrefetchInBackground`, `GetCachePath` (`$HOME/.gws/status-cache.json`), `DefaultTTL`. Consumed unchanged. |
| `internal/git/progress.go` | `NewProgress`, `Start`, `Increment`, `Stop`; TTY-only rendering already covered by `progress_test.go`. Consumed unchanged. |
| `internal/config/config.go` | `Preferences.StatusWorkers` for worker resolution; `Worktree{Path,Branch,Aligned}`. |
| `internal/filter/filter.go` | `MatchesPattern` for status filtering. |
| `docs/site/commands-core.md` | `List Worktrees` subsection (lines 459-490) gains flags, terms, attached-value rule, and new example output. |
| `docs/site/configuration.md` | `status_workers` row (line 123) gains `omgw worktree list --workers`. |
| `docs/specs/30-spec-worktree-status/30-proofs/` | New. One `30-task-0N-proofs.md` per parent task holding the CLI transcripts and test output named in the proof artifacts. |

### Notes

- Tests live beside the code in `cmd/omgitworks`. Run one file's worth with `go test ./cmd/omgitworks -run 'TestRunWorktreeList' -v`, and the full gate with `make ci`.
- Fixtures use real git repositories in `t.TempDir()`; a bare "origin" repository plus `git push -u` gives each worktree an upstream so ahead/behind counts resolve.
- `saveConfigForWorktreeTests` sets `HOME` to a temp dir, so the shared cache file lands under it; call it before anything that touches the cache.
- Commit once per parent task with the conventional type the task names; the `commit-msg` hook pads the type token.
- Proof transcripts are produced against scratch fixtures so they never contain private repository or branch names.

## Tasks

### [x] 1.0 Extract the shared status pipeline out of `runList`

Move the two-phase status fetch (load cache, serve fresh entries, fall back to stale, fetch the rest in parallel with a progress spinner, save, refresh stale entries in the background) into one helper that takes a list of paths and returns the status map plus a completion hook for the background prefetch. Convert `runList` to call it. This lands first so `worktree list` consumes the helper rather than copying sixty lines. Success metric 4 depends on this task alone.

#### 1.0 Proof Artifact(s)

- CLI: `grep -n 'PrefetchInBackground\|FetchAll(' cmd/omgitworks/*.go | grep -v _test.go` returns matches only inside the new helper's file, demonstrating the pipeline exists in exactly one place
- Test: new helper tests pass covering: all paths fresh in cache means no fetch; a stale entry is served immediately and refreshed in the background; an uncached path is fetched in the foreground and written to the cache file; the cache file is saved after the foreground fetch, demonstrating the ordering `runList` relied on is preserved
- Test: the full existing `cmd/omgitworks` and `internal/git` suites pass with no assertion changes, demonstrating `omgw list -s` and `-S` are unchanged
- CLI: `make ci` exits zero, demonstrating vet, lint, and race-detector tests pass after the refactor

#### 1.0 Tasks

- [x] 1.1 Create `cmd/omgitworks/status_fetch.go` with `fetchStatuses(paths []string, workers int) (map[string]*git.Status, func())`. Move the body of `runList` from `statusCache = git.NewCache(...)` through the `PrefetchInBackground` call into it, keyed on `paths` instead of `filtered`. Return the status map and a `wait` function that blocks on the background `done` channel, or is a no-op when no background prefetch was started.
- [x] 1.2 Preserve the ordering in a comment at the top of the helper: fresh entries first, stale entries served immediately, uncached paths fetched in the foreground under a `git.Progress`, `Save` after the foreground fetch, and only then the background prefetch of stale entries.
- [x] 1.3 Replace the extracted block in `runList` with `statusMap, wait := fetchStatuses(repoPaths, workers)` followed by `defer wait()`. Keep `resolveWorkers` and the `repoPaths` loop in `runList`. Delete the now-unused local variables.
- [x] 1.4 Change `resolveColorEnabled(_ *cobra.Command) bool` to `resolveColorEnabled(mode string) bool` and update the one caller at `list.go:158` to pass `flagColor`. This removes the only package-global read that `worktree list` would otherwise inherit.
- [x] 1.5 Write `status_fetch_test.go` with a real git repository per case (`runGitIn` to init and commit) and `t.Setenv("HOME", t.TempDir())` so `git.GetCachePath()` lands in the temp dir. Case A: seed the cache file with a fresh entry for the path, set `PATH` to an empty directory, call the helper, assert the entry is returned, proving no git call was needed.
- [x] 1.6 Case B: seed a stale entry (`LastChecked` older than `git.DefaultTTL`). Assert the returned map holds the stale `LastChecked`, then call `wait()` and assert the cache file's entry now has a newer `LastChecked`, proving stale-then-refresh.
- [x] 1.7 Case C: no cache file. Assert the returned map holds a fetched status and that the cache file exists with that path before `wait()` is called, proving the foreground fetch and the save-before-background ordering.
- [x] 1.8 Run `go test ./cmd/omgitworks ./internal/git` and confirm every pre-existing test passes with no assertion edits. Run the grep from the proof artifact and confirm `FetchAll(` and `PrefetchInBackground` appear only in `status_fetch.go` outside tests.
- [x] 1.9 Run `make ci`. Commit as `refactor(list): extract the status fetch pipeline`.

### [x] 2.0 Rename the alignment column and add compact status with `-s` and `--color`

Rename the `STATUS` column of `omgw worktree list` to `ALIGNED` with `yes`/`no` values, register `-s/--status` (sentinel-valued) and `--color`, and when `-s` is present fetch status for every listed worktree through the task 1.0 helper and append the compact icons after the branch inside the `BRANCH` column, aligned in a sub-column. Worktrees with no readable status show a blank icon area. Without `-s`, no git command runs.

#### 2.0 Proof Artifact(s)

- Test: the header reads `ALIGNED` and rows show `yes` or `no`, with the two pre-existing `aligned`/`(unaligned)` assertions updated, demonstrating the rename
- Test: `-s` against a fixture with one clean, one dirty, and one ahead worktree renders `✓`, `✗`, and `↑1` after the respective branch names and no `STATUS` header, demonstrating compact mode
- Test: icons across rows start at the same visible column, computed with `displayWidth`, demonstrating the sub-column alignment
- Test: `--color always` wraps icons in the same ANSI sequences `formatStatusIcons(status, true)` produces and `--color never` leaves them plain, demonstrating color parity
- Test: a worktree whose directory was deleted renders a row with a blank icon area and the run exits zero, demonstrating graceful handling
- Test: with no status flag the helper is not invoked (asserted via an injected fetcher or an empty cache file remaining absent), demonstrating status stays opt-in
- CLI: transcript of `omgw worktree list .` and `omgw worktree list . -s` against a scratch fixture repository, saved in `30-proofs/30-task-02-proofs.md`, demonstrating the rename and compact mode end to end

#### 2.0 Tasks

- [x] 2.1 In `worktree_list.go` add `type worktreeListOptions struct { CompactStatus, ShowStatus bool; FilterStatus string; ColorEnabled bool; Workers int }` and change `runWorktreeList(scope worktreeScope, stdout io.Writer)` to `runWorktreeList(scope worktreeScope, opts worktreeListOptions, stdout io.Writer)`. Update the five existing callers in `worktree_list_test.go` to pass `worktreeListOptions{}`.
- [x] 2.2 Rename the fourth column: header `ALIGNED`, separator width 7, row value `yes` when `e.Aligned` and `no` otherwise. Update `TestRunWorktreeList_AllWorktrees` and `TestRunWorktreeList_UnalignedIndicator` to assert `yes`/`no` and the `ALIGNED` header.
- [x] 2.3 Declare package-level `flagWTStatus string` and `flagWTColor string`. In `init()` register `--status`/`-s` with `NoOptDefVal = showColumnSentinel` and help text `Show compact status after the branch, or show and filter with -s=<pattern>`, and `--color` with default `auto`. Register them on `worktreeListCmd`, not the parent.
- [x] 2.4 Add `parseWorktreeListFlags(cmd *cobra.Command) worktreeListOptions`: when `--status` changed, set `CompactStatus = true` and, if the value is not the sentinel, `FilterStatus`. Set `ColorEnabled = resolveColorEnabled(flagWTColor)`. Wire it into `RunE`. Do not call `reassignTrailingArg`.
- [x] 2.5 In `runWorktreeList`, after building `entries`, when `opts.CompactStatus || opts.ShowStatus` collect every entry's `Path`, call `fetchStatuses(paths, resolveWorkers(opts.Workers, cfg))`, `defer wait()`, and keep the returned map. Otherwise leave the map nil so no git command runs.
- [x] 2.6 Compact rendering: compute `maxBranch` with `displayWidth`; when compact and the map is non-nil, add `2 + maxIcons` where `maxIcons` is the widest `formatStatusIcons(status, false)`. Per row, when a status exists and `status.Branch != ""`, emit `branch + padding + formatStatusIcons(status, opts.ColorEnabled)` right-justified in the widened column, mirroring `list.go` lines 826-840; otherwise pad the branch to the full width. Use `displayWidth` for all padding math so colored icons do not shift columns.
- [x] 2.7 Create `worktree_list_status_test.go` with `statusFixture(t) (repoPath string, wt map[string]string)`: init a repo with a `main` commit and local `user.name`/`user.email`; create a bare origin in another temp dir, `git remote add origin`, `git push -u origin main`; then `gitWorktreeAdd` branches `clean`, `dirty`, `ahead` and `git push -u origin <branch>` from each; write an untracked file in `dirty`; commit once in `ahead`. Save the config with `saveConfigForWorktreeTests` listing the three worktrees under repository `fixture`, and return the paths. Call `saveConfigForWorktreeTests` before any cache-touching step.
- [x] 2.8 Tests: `TestWorktreeListStatus_Compact` asserts `✓` follows `clean`, `✗` follows `dirty`, `↑1` follows `ahead`, and no `STATUS` header appears. `TestWorktreeListStatus_CompactAligned` strips ANSI and asserts the icon start column (index of the first icon rune after `displayWidth` of the prefix) is equal across the three rows.
- [x] 2.9 `TestWorktreeListStatus_Color`: run with `ColorEnabled: true` and assert each row contains exactly `formatStatusIcons(status, true)` for that worktree's status (obtain the status via `git.GetStatus(path)`); run with `ColorEnabled: false` and assert no `\x1b[` in the output.
- [x] 2.10 `TestWorktreeListStatus_MissingWorktree`: add a fourth config entry whose path does not exist; assert its row prints with a blank icon area, the other rows keep their icons, and the error is nil.
- [x] 2.11 `TestWorktreeListStatus_OptIn`: run with `worktreeListOptions{}` and assert `$HOME/.gws/status-cache.json` does not exist afterward, proving the pipeline was not invoked.
- [x] 2.12 Produce `30-proofs/30-task-02-proofs.md` with transcripts of `omgw worktree list .` and `omgw worktree list . -s` from a scratch fixture built the same way, plus the `go test` output. Run `make ci`. Commit as `feat(worktree): show compact status in worktree list`.

### [ ] 3.0 Add the full `STATUS` column with `-S` and define precedence

Register `-S/--show-status` (sentinel-valued). When present, add a `STATUS` column holding icons only, after `ALIGNED`. When both `-s` and `-S` are given, `-S` wins and `BRANCH` stays plain. A detached worktree keeps `(detached)` in `BRANCH` with icons in whichever place the active mode puts them. Update the subcommand's `Long` help with the flag convention paragraph and attached-value examples.

#### 3.0 Proof Artifact(s)

- Test: `-S` adds a `STATUS` header and per-row icons only, with `BRANCH` unchanged, demonstrating full mode
- Test: `-s -S` together produce the `STATUS` column and a plain `BRANCH` column, demonstrating precedence matches `omgw list`
- Test: a detached worktree shows `(detached)` in `BRANCH` and its icons in the `STATUS` column under `-S`, demonstrating detached handling
- Test: `omgw worktree list --help` output contains both flags, the `clean`/`dirty`/`ahead`/`behind` terms, and an `-s=dirty` example, demonstrating the help contract
- CLI: transcript of `omgw worktree list . -S` and `omgw worktree list . -s -S` against the fixture, saved in `30-proofs/30-task-03-proofs.md`, demonstrating full mode and precedence

#### 3.0 Tasks

- [ ] 3.1 Declare `flagWTShowStatus string`; register `--show-status`/`-S` with `NoOptDefVal = showColumnSentinel` and help text `Show a STATUS column, or show and filter with -S=<pattern>`.
- [ ] 3.2 Extend `parseWorktreeListFlags`: when `--show-status` changed set `ShowStatus = true` and `CompactStatus = false`; take `FilterStatus` from the non-sentinel value of `-S`, letting a non-empty `-s` value override it, mirroring `parsePair` in `list.go`.
- [ ] 3.3 Full rendering: when `opts.ShowStatus`, append a `STATUS` header after `ALIGNED` with width `max(6, maxIcons)`, and per row emit `formatStatusIcons(status, opts.ColorEnabled)` or blank when no status. Keep `BRANCH` plain in this mode.
- [ ] 3.4 Detached handling: keep the `(detached)` substitution for an empty `Branch`; icons still append (compact) or populate `STATUS` (full) from the status fetched at the worktree path. Add a detached worktree to `statusFixture` via `git checkout --detach` inside a fourth worktree, or build it in the test that needs it.
- [ ] 3.5 Rewrite `worktreeListCmd.Long`: keep the scoping paragraph, add the flag-convention paragraph from `listCmd.Long` adapted to two flags, state that the positional is always the repository, and add examples `omgw worktree list . -s`, `omgw worktree list my-repo -S`, `omgw worktree list -s=dirty`, `omgw worktree list my-repo -S=ahead`, and `omgw worktree list -s my-repo`.
- [ ] 3.6 Tests: `TestWorktreeListStatus_Full` asserts the `STATUS` header, icons on each row, and that the `BRANCH` cell for `dirty` is exactly `dirty` padded. `TestWorktreeListStatus_ShowStatusWins` runs with both flags set and asserts the `STATUS` header exists and no icon appears within the `BRANCH` column span. `TestWorktreeListStatus_Detached` asserts `(detached)` in `BRANCH` and an icon in `STATUS` under `ShowStatus`.
- [ ] 3.7 `TestWorktreeListHelp_StatusFlags`: use `helpListedFlags(t, worktreeListCmd)` to assert `status`, `show-status`, `color` are listed, and assert `worktreeListCmd.Long` contains `clean`, `dirty`, `ahead`, `behind`, and `-s=dirty`.
- [ ] 3.8 Produce `30-proofs/30-task-03-proofs.md` with transcripts of `omgw worktree list . -S`, `omgw worktree list . -s -S`, and `omgw worktree list --help`. Run `make ci`. Commit as `feat(worktree): add the STATUS column to worktree list`.

### [ ] 4.0 Filter worktrees by status and keep the positional as the repository

Give `-s=<pattern>`, `--status=<pattern>`, `-S=<pattern>`, and `--show-status=<pattern>` show-and-filter semantics using `statusFilterText` and `filter.MatchesPattern`. Filtering runs after repository scoping. Worktrees with no readable status never match. The positional argument is always the repository: `omgw worktree list -s my-repo` scopes to `my-repo` with compact status. An empty result prints the existing no-worktrees message with the scope label and exits zero.

#### 4.0 Proof Artifact(s)

- Test: `-s=dirty` keeps only the dirty worktree of the fixture, demonstrating filtering
- Test: `-S=ahead` keeps only the ahead worktree and shows the `STATUS` column, demonstrating show-and-filter
- Test: `-s my-repo` (space-separated) scopes to `my-repo` with compact icons and treats nothing as a filter, demonstrating the positional contract
- Test: `my-repo -s=dirty` lists only dirty worktrees of `my-repo`, demonstrating scope-then-filter ordering
- Test: a worktree with no status is omitted under `-s=clean`, demonstrating the no-status rule
- Test: `-s=behind` against an all-clean fixture prints `No worktrees found` (with the scope label when given) and exits zero, demonstrating the empty case
- CLI: transcript of `omgw worktree list -s=dirty` across a two-repository scratch workspace, saved in `30-proofs/30-task-04-proofs.md`, demonstrating cross-repository filtering

#### 4.0 Tasks

- [ ] 4.1 In `runWorktreeList`, after the status fetch and before width calculation, when `opts.FilterStatus != ""` keep only entries where `filter.MatchesPattern(statusFilterText(statusMap[e.Path]), opts.FilterStatus)`; `statusFilterText(nil)` returns `""`, which excludes worktrees with no status.
- [ ] 4.2 Move the existing empty-result check after the filter so a filtered-to-nothing result prints `No worktrees found for '<label>'` or `No worktrees found` and returns nil.
- [ ] 4.3 Confirm in `RunE` that `args[0]` is passed to `worktreeScopeFor` unchanged and that nothing inspects it as a filter value. Add a comment pointing at the spec's attached-value rule.
- [ ] 4.4 Tests: `TestWorktreeListStatus_FilterDirty` with `FilterStatus: "dirty", CompactStatus: true` asserts only the `dirty` row. `TestWorktreeListStatus_ShowAndFilterAhead` with `FilterStatus: "ahead", ShowStatus: true` asserts only the `ahead` row and the `STATUS` header.
- [ ] 4.5 `TestWorktreeListFlags_PositionalIsRepo`: set `os.Args` or call `worktreeListCmd.ParseFlags([]string{"-s", "fixture"})`, then assert `parseWorktreeListFlags` yields `CompactStatus: true, FilterStatus: ""` and the remaining positional is `fixture`. Also assert `ParseFlags([]string{"-s=dirty"})` yields `FilterStatus: "dirty"`, and `ParseFlags([]string{"-sdirty"})` the same.
- [ ] 4.6 `TestWorktreeListStatus_ScopeThenFilter`: two repositories in config, each with a dirty worktree; scope `worktreeScope{NamePattern: "fixture"}` plus `FilterStatus: "dirty"` lists only the first repository's dirty worktree.
- [ ] 4.7 `TestWorktreeListStatus_NoStatusExcluded`: with the missing-path entry from 2.10 and `FilterStatus: "clean"`, assert its row is absent while `clean` remains.
- [ ] 4.8 `TestWorktreeListStatus_FilterEmpty`: `FilterStatus: "behind"` against the fixture prints `No worktrees found for 'fixture'` when scoped and `No worktrees found` when not, with nil error.
- [ ] 4.9 Produce `30-proofs/30-task-04-proofs.md` with a transcript of `omgw worktree list -s=dirty` across a two-repository scratch workspace and `omgw worktree list -s fixture`. Run `make ci`. Commit as `feat(worktree): filter worktree list by status`.

### [ ] 5.0 Share the cache and honor `--workers`

Register `--workers N` on `worktree list`, resolved by `resolveWorkers` (flag, then `status_workers` preference, then 8). Confirm by test that worktree status is written to and read from the same cache file `omgw list` uses, keyed by worktree path, and that a second run within the TTL reads without invoking git.

#### 5.0 Proof Artifact(s)

- Test: after `worktree list -s`, `$HOME/.gws/status-cache.json` contains an entry keyed by the worktree path, demonstrating cache sharing
- Test: a second `worktree list -s` with a fresh cache entry does not call `git` (asserted via the injected fetcher or a `PATH` with no `git`), demonstrating the cache hit
- Test: `--workers 3` wins over `status_workers: 5`, `status_workers: 5` wins over the default, and the default is 8, demonstrating worker resolution
- CLI: transcript of `time omgw worktree list -s` run twice against the fixture, the second completing without a spinner, saved in `30-proofs/30-task-05-proofs.md`, demonstrating the cache hit end to end

#### 5.0 Tasks

- [ ] 5.1 Declare `flagWTWorkers int`; register `--workers` with default 0 and the same help text `list` uses. In `parseWorktreeListFlags` set `opts.Workers = flagWTWorkers`. `runWorktreeList` already calls `resolveWorkers(opts.Workers, cfg)` from 2.5.
- [ ] 5.2 `TestWorktreeListStatus_CacheWritten`: after a compact run, read `$HOME/.gws/status-cache.json`, unmarshal into the `statuses` map, and assert a key equal to each worktree path with a non-zero `LastChecked`.
- [ ] 5.3 `TestWorktreeListStatus_CacheHit`: run once to populate the cache, then set `t.Setenv("PATH", t.TempDir())` so no `git` binary is reachable, run again, and assert the same icons render. A cache miss would produce blank icons, so this proves no git call was made.
- [ ] 5.4 `TestWorktreeListWorkers_Resolution`: three sub-cases through `resolveWorkers`: flag 3 with preference 5 gives 3; flag 0 with preference 5 gives 5; flag 0 and nil preferences gives 8. Also assert `helpListedFlags(t, worktreeListCmd)` includes `workers`.
- [ ] 5.5 Verify the `list` flags are untouched: assert `flagWorkers` and `flagColor` in `list.go` are not read by any code path in `worktree_list.go` (grep in the proof file), so the two commands cannot clobber each other.
- [ ] 5.6 Produce `30-proofs/30-task-05-proofs.md` with the transcript of `time omgw worktree list -s` run twice against the fixture, noting the absence of the spinner on the second run, plus the grep and test output. Run `make ci`. Commit as `feat(worktree): share the status cache and workers with worktree list`.

### [ ] 6.0 Document worktree status

Update the worktree management section of `docs/site/commands-core.md`: the `List Worktrees` subsection gains the flags, the filter terms, the attached-value rule, the renamed column, and new example output for `-s` and `-S`. Update the `status_workers` row of `docs/site/configuration.md` to mention `omgw worktree list --workers`. Update the `Subcommands` block in `worktree.go`'s `Long` if its one-line description of `list` needs it.

#### 6.0 Proof Artifact(s)

- Diff: `docs/site/commands-core.md` shows the updated `List Worktrees` subsection with `ALIGNED` in the example output and `-s`, `-S`, `-s=dirty` examples, demonstrating the documentation matches the implementation
- Diff: `docs/site/configuration.md` `status_workers` row names both commands, demonstrating the preference is documented for both
- CLI: `mkdocs build --strict` exits zero, demonstrating the documentation site still builds
- CLI: `omgw worktree list --help` output pasted beside the documentation examples in `30-proofs/30-task-06-proofs.md`, demonstrating the two agree

#### 6.0 Tasks

- [ ] 6.1 In `docs/site/commands-core.md`, replace the `List Worktrees` subsection: usage line `omgw worktree list [repo|.] [flags]`, a short flags table (`--status`/`-s`, `--show-status`/`-S`, `--color`, `--workers`), the filter terms, a note that the positional is always the repository so filter values attach with `=`, and three example blocks for plain, `-s`, and `-S` output using `ALIGNED` with `yes`/`no`.
- [ ] 6.2 Add a cross-reference from the `omgw list` `--status` row to the worktree section, one sentence, so readers of either page find the other.
- [ ] 6.3 In `docs/site/configuration.md` line 123, extend the `status_workers` description to `omgw list --workers` or `omgw worktree list --workers`.
- [ ] 6.4 In `worktree.go` line 32, extend the `list` line's comment to `List all worktrees, optionally with git status (-s/-S)` if it still fits the column; otherwise leave it and rely on the subcommand help.
- [ ] 6.5 Run `mkdocs build --strict` from the repository root (install `docs/requirements.txt` into the existing `.venv` if needed) and confirm exit zero.
- [ ] 6.6 Produce `30-proofs/30-task-06-proofs.md` with the `omgw worktree list --help` output beside the documentation examples and the `mkdocs` result. Commit as `docs(worktree): document worktree status`.
