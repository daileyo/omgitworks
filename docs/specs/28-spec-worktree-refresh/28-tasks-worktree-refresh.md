# 28-tasks-worktree-refresh.md

> Source spec: `docs/specs/28-spec-worktree-refresh/28-spec-worktree-refresh.md`
>
> Depends on: **spec 25** (`worktree-repo-context`) for `repocontext.ResolveCurrent` and the `worktreeScope` model; **spec 26** (`worktree-add-by-tag`) for the `--tag` AND-semantics precedent. This branch's base is `feat/worktree-remove` (`9bae1a6`), which carries specs 25, 26, and **27**.
>
> *Rebased during task 1.0:* planning assumed a base of `feat/worktree-add-by-tag` (`f4b0601`). Spec 27, developed in parallel on a sibling branch, added a fourth copy of the re-discovery sequence that this spec's consolidation could not see. The two branches merge with no conflict and no test failure, so nothing automated would have caught the duplicate. The branch was rebased onto spec 27 so task 1.0 could absorb that copy.

## Planning Notes (Codebase Assessment)

These findings come from reading the code on this branch and refine what the spec assumed. They are recorded here so implementation does not re-derive them.

- **There are four copies, not three.** `refresh.go:108` (`discoverWorktrees`) and `worktree_align.go:232` both run the full repair → prune → list → rebuild sequence. `worktree_add.go` runs **only** list → rebuild, with no repair and no prune. Spec 27 added `refreshWorktreeData` in `worktree_remove.go:396`, also list → rebuild only. A further site, `worktree_align.go:87`, runs repair + prune alone as a planning pass, with no rebuild, and is not a copy of the sequence.
- **The copies diverge in three observable ways.** `refresh.go` skips entries whose path is gone via `os.Stat` and sets `Worktrees = nil` when none remain; `worktree_align.go:232`, `worktree_add.go`, and `worktree_remove.go` do neither and assign a possibly-empty non-nil slice. `worktree_add.go` and `worktree_remove.go` also discard the `ListWorktrees` error silently.
- **Consolidation is therefore a behavior change on the `add` and `remove` paths**, which gain repair, prune, path-existence skipping, nil-on-empty, and error propagation. The spec calls this out and requires test coverage rather than assuming it is harmless.
- **Targeting needs two existing models reconciled.** `worktreeScope` / `worktreeScopeFor` handles `""`, `.`, and name patterns but carries no tag. Spec 27 moved spec 26's selection into `worktree.go` as `selectWorktreeTargets`, alongside `allRepositories`, `selectByName`, `filterByTag`, and a shared `singleTagValue(values, flagName)`.
- **`selectWorktreeTargets` must not be reused as-is for refresh.** Its precedence table was built for add and remove, and it differs from refresh's contract in two ways. With no pattern and no tag it resolves the *current* repository, where refresh must target *every* repository. And it rejects a name pattern matching more than one repository, where refresh refreshes them all. Refresh reuses the building blocks (`allRepositories`, `selectByName`, `filterByTag`, `singleTagValue`, `worktreeScopeFor`) but not the table.
- **There is no injectable git runner.** `internal/git.gitCommand` calls `exec` directly with no seam, so "repair did not run" cannot be asserted by mocking. Dry-run non-mutation is proved by observable side effects instead: a repairable worktree stays broken and a dead entry stays listed after a dry run.
- **Single-save has a behavioral test precedent.** `TestWorktreeAddBulk_SingleSave` proves it by asserting every repository retains its entry, since a per-repository save would leave only the last intact. Task 2.0 mirrors that rather than counting writes.
- **`worktree align` already ships `--dry-run`**, so this spec's flag matches its naming and output conventions.
- **Flag variables live in package scope.** `flagDryRun` (align), `flagWorktreeAddTags` (add), and `flagWorktreeRemoveTags` / `flagWorktreeRemoveDryRun` (remove) are package-level in `package main`, so refresh must use distinct names.
- **Never derive single-repository mode from `len(repos) == 1`.** Specs 26 and 27 both found this bug during validation: a tag that matches exactly one repository is still a bulk run. If refresh needs the distinction at all, derive it from the invocation shape.
- **`worktree.go` already anticipates this command** — `currentRepoArg`'s doc comment names "the future refresh" as a reason the argument position stays uniform.

## Validation Notes

Items the validation phase must cover explicitly, beyond the spec's own requirements.

- **`worktree remove` behavior changed in this spec and was never validated under spec 27.**
  Spec 27 validated its original list → rebuild re-sync. Task 1.10 moved that re-sync onto
  the shared helper, so after a removal `worktree remove` now also:
  1. repairs, then prunes, the repository's other worktrees;
  2. drops stored entries whose directory no longer exists on disk;
  3. stores nil rather than an empty slice when no worktrees remain, omitting `worktrees`
     from `config.json`;
  4. leaves stored data untouched when the repository cannot be listed.

  Validate each as a changed behavior of `worktree remove`, with evidence. Points 1 and 2
  are covered by `TestRunWorktreeRemove_RefreshUsesSharedRules`. Points 3 and 4 are covered
  only at the helper level (`TestBuildWorktreeEntries_NilWhenNoneSurvive`,
  `TestSyncRepoWorktrees_PropagatesListError`), so validation must decide whether that is
  enough. Spec 27's validation report needs no change: its only reference, row U1-11
  ("`refreshWorktreeData` + one `config.Save`"), still holds.
- **`worktree add` behavior changed in the same way** (task 1.6), and so did
  **`worktree align`** (planning audit flag 1, which has no align-specific test). Validate
  both on the same basis.

## Relevant Files

| File | Why It Is Relevant |
| --- | --- |
| `cmd/omgitworks/worktree_discover.go` | New. Home of the extracted shared discovery helpers `buildWorktreeEntries` and `syncRepoWorktrees`. |
| `cmd/omgitworks/worktree_discover_test.go` | New. Unit tests for the extracted helpers: path skipping, nil-on-empty, error propagation. |
| `cmd/omgitworks/worktree_refresh.go` | New. The `omgw worktree refresh` command: declaration, flags, targeting, run loop, change reporting, dry run. |
| `cmd/omgitworks/worktree_refresh_test.go` | New. Core re-sync tests: external add/delete, repair ordering, alignment recomputation, empty case, scope containment, partial failure, single save. |
| `cmd/omgitworks/worktree_refresh_target_test.go` | New. Targeting tests: the four forms, AND semantics, repeated `--tag`, unmatched-filter errors, completion. |
| `cmd/omgitworks/worktree_refresh_report_test.go` | New. Change-reporting tests: added/removed/realigned lines, detached HEAD keying, silence when unchanged, summary counts. |
| `cmd/omgitworks/worktree_refresh_dryrun_test.go` | New. Dry-run tests: config byte-identical, repair/prune not run, caveat line, output parity. |
| `cmd/omgitworks/refresh.go` | Contains `discoverWorktrees` (line 108), the most complete existing copy; converted to call the shared helper. |
| `cmd/omgitworks/worktree_align.go` | Contains the repair+prune planning pass (line 87) and the full rebuild copy (line 232); converted to the shared helper. Also the `--dry-run` convention to match. |
| `cmd/omgitworks/worktree_add.go` | Contains a partial list-only copy; converted to the shared helper, gaining repair and prune. |
| `cmd/omgitworks/worktree_remove.go` | Spec 27. Contains `refreshWorktreeData` (line 396), a partial list-only copy; converted to the shared helper. Also the `--dry-run` and bulk-summary conventions of the newest worktree command. |
| `cmd/omgitworks/worktree_remove_test.go` | Spec 27's remove tests; gains the test pinning remove's behavior change. |
| `cmd/omgitworks/worktree.go` | Registers subcommands; holds `worktreeScope`, `worktreeScopeFor`, `currentRepoArg`, `completeWorktreeRepoOrDot`, `completeAllTags`, and, since spec 27, `selectWorktreeTargets`, `allRepositories`, `selectByName`, `filterByTag`, `singleTagValue`. Its `Long` subcommand list needs the new command. |
| `internal/git/worktree.go` | Provides `RepairWorktrees`, `PruneWorktrees`, `ListWorktrees`, `IsAligned`, `ResolvePath`. Consumed unchanged. |
| `internal/config/config.go` | Defines `Worktree{Path,Branch,Aligned}` and `Repository.Worktrees`, plus `Load`/`Save`. |
| `internal/filter/filter.go` | Provides `MatchesPattern` (name) and `MatchesExact` (tag). |
| `internal/repocontext/repocontext.go` | Provides `ResolveCurrent` for the `.` targeting form. |
| `docs/site/commands-core.md` | Command reference; gains a `worktree refresh` section beside `worktree list` and `worktree align`. |

### Notes

- Tests live beside the code in `cmd/omgitworks/`, named `*_test.go`, using `t.TempDir()` with real git repositories.
- Reuse the existing fixture helpers rather than writing new ones: `setupTaggedFixture`, `standardFixture`, `threeBackendFixture`, `setupScopeFixture`, `setupTwoWorktreeTestRepos`, `breakRepo`, `chdirForTest`, `captureStdoutStr`, `saveConfigForWorktreeTests`, `scopeFor`.
- Run `make ci` (vet + golangci-lint + `go test -race`) before considering any parent task complete.
- Commit messages follow Conventional Commits; the `.githooks/commit-msg` hook pads the type for aligned `git log` output. Use `feat(worktree):` for the new command and `refactor(worktree):` for the extraction.
- `goimports` local prefix is `github.com/daileyo/omgitworks`; group imports accordingly.

## Tasks

### [x] 1.0 Extract the shared worktree discovery helper and adopt it at every existing call site

Consolidate the divergent repair → prune → list → rebuild copies into one helper, and convert `refresh.go`, `worktree_align.go`, `worktree_add.go`, and `worktree_remove.go` to it. This lands before the new command so that `worktree refresh` consumes the helper rather than adding another copy. Success metric 5 depends on this task alone. The split into two functions is deliberate: dry run (task 5.0) needs the non-mutating half on its own.

#### 1.0 Proof Artifact(s)

- CLI: `grep -rn 'ListWorktrees(' cmd internal --include='*.go' | grep -v _test.go` returns one caller, inside `worktree_discover.go`, demonstrating the full repair → prune → list → rebuild sequence exists in one place, down from four. *Amended during implementation:* the plan originally grepped for `RepairWorktrees`/`PruneWorktrees` and expected them only in `worktree_discover.go`, but task 1.5 kept align's standalone repair + prune planning pass, which is not the full sequence, so that grep could never pass
- Test: `cmd/omgitworks/worktree_discover_test.go` passes, covering path-existence skipping, nil-on-empty, and `ListWorktrees` error propagation, demonstrating the helper preserves `refresh.go`'s stricter semantics
- Test: `TestWorktreeAdd_RepairsBeforeRebuild` in `worktree_add_test.go` passes, demonstrating the intended `add`-path behavior change is deliberate and covered
- Test: `TestRunWorktreeRemove_RefreshUsesSharedRules` in `worktree_remove_test.go` passes, demonstrating the equivalent `remove`-path behavior change is covered
- Test: existing `worktree_align_test.go`, `worktree_add_test.go`, `worktree_add_bulk_test.go`, `worktree_remove_test.go`, and `worktree_remove_bulk_test.go` pass without modification to their assertions, demonstrating no regression in the converted call sites
- CLI: `make ci` exits zero, demonstrating vet, lint, and race-detector tests all pass after the refactor

#### 1.0 Tasks

- [x] 1.1 Create `cmd/omgitworks/worktree_discover.go` with `buildWorktreeEntries(repoPath, repoName string) ([]config.Worktree, error)`: call `git.ListWorktrees`, return the error unwrapped on failure, skip any entry whose `Path` fails `os.Stat`, set `Aligned` via `git.IsAligned(e.Path, repoName)`, and return a nil slice when no entries survive.
- [x] 1.2 In the same file add `syncRepoWorktrees(repo *config.Repository) error`: run `git.RepairWorktrees` then `git.PruneWorktrees` (ignoring their errors as all current call sites do), then call `buildWorktreeEntries` and assign the result to `repo.Worktrees`, returning any error. Document in a comment that repair must precede prune because prune discards recoverable entries.
- [x] 1.3 Rewrite `discoverWorktrees` in `refresh.go` to loop over repos calling `syncRepoWorktrees`, skipping repos that error, and preserving its existing `int` return of repos that ended with worktrees.
- [x] 1.4 Replace the rebuild loop at `worktree_align.go:232` with a `syncRepoWorktrees` call per affected repo, keeping the existing `continue`-on-error behavior.
- [x] 1.5 Remove the now-redundant standalone repair+prune planning pass at `worktree_align.go:87`, or leave it with a comment explaining why the planning pass still needs it before `repo.Worktrees` is read. Pick one and record the reason in the commit message. *Decision:* kept, with a comment — the planned moves fail on a worktree whose `.git` link is broken, so the pass is not redundant (audit flag 2).
- [x] 1.6 Replace the list-only rebuild in `worktree_add.go` with `syncRepoWorktrees`. Decide how to surface the error that was previously discarded — propagate it, since the enclosing function already returns `(string, error)` — and note the `add` path now also repairs, prunes, skips missing paths, and nils on empty.
- [x] 1.7 Write `worktree_discover_test.go` with three cases: an entry whose directory was deleted is excluded; a repo whose entries are all missing yields a nil `Worktrees`; a `ListWorktrees` failure (use `breakRepo`) is returned rather than swallowed.
- [x] 1.8 Add `TestWorktreeAdd_RepairsBeforeRebuild` to `worktree_add_test.go`: create a repo with a worktree whose git pointer is broken but whose directory exists, run `worktree add`, and assert the entry survives — proving repair now runs on this path.
- [x] 1.9 Run `make ci` and confirm the pre-existing align and add tests pass with their assertions unchanged.
- [x] 1.10 *Added after rebasing onto spec 27.* Replace the body of `refreshWorktreeData` in `worktree_remove.go` with `syncRepoWorktrees` per repository, dropping its unused `cfg` parameter. Add `TestRunWorktreeRemove_RefreshUsesSharedRules`, which must fail against the previous `worktree_remove.go`, and re-run spec 27's `TestRunWorktreeRemove_*`, `TestWorktreeRemoveBulk_*`, and `TestWorktreeRemove_*` suites.

### [ ] 2.0 Add `omgw worktree refresh` with scoped metadata re-sync

Create `cmd/omgitworks/worktree_refresh.go` and implement spec Unit 1: for each targeted repository run the shared helper, rebuild stored `Worktrees` with `Aligned` recomputed, skip and report per-repository failures, save configuration exactly once, and exit non-zero when any repository failed. Targeting is stubbed to "all repositories" here and completed in task 3.0.

#### 2.0 Proof Artifact(s)

- Test: `TestWorktreeRefresh_DiscoversExternalAdd` passes — a worktree created with plain `git worktree add` appears in stored data after refresh, demonstrating discovery of external additions
- Test: `TestWorktreeRefresh_ClearsDeletedWorktree` passes — a worktree whose directory was deleted by hand is dropped from stored data, demonstrating stale-entry clearing
- Test: `TestWorktreeRefresh_RepairBeforePrune` passes — a worktree with a broken git pointer but a live directory survives the run, demonstrating the load-bearing ordering requirement
- Test: `TestWorktreeRefresh_RecomputesAligned` passes — a worktree moved into the XDG projects root reports `Aligned: true` afterwards, demonstrating correct recomputation
- Test: `TestWorktreeRefresh_ClearsWhenEmpty` passes — a repository ending with zero worktrees has `Worktrees` set to nil, demonstrating the empty case
- Test: `TestWorktreeRefresh_LeavesOtherDataUntouched` passes — the repository list, `User`/`Email`/`Tags` fields, and the status cache file are unchanged across a refresh, demonstrating scope containment per success metric 3
- Test: `TestWorktreeRefresh_PartialFailure` passes — with the first of two repositories broken, the second is still refreshed and the command returns a non-nil error, demonstrating partial-failure handling and the exit contract
- Test: `TestWorktreeRefresh_SingleSave` passes — every refreshed repository retains its rebuilt entries, which a per-repository save of stale data would not produce, demonstrating the single-save requirement the way `TestWorktreeAddBulk_SingleSave` does
- CLI: transcript against a scratch fixture workspace showing `omgw worktree list`, a hand-made worktree change, `omgw worktree refresh`, and `omgw worktree list` reflecting it, demonstrating the feature end to end

#### 2.0 Tasks

- [ ] 2.1 Create `cmd/omgitworks/worktree_refresh.go` declaring `worktreeRefreshCmd` with `Use: "refresh [repo|.]"`, a `Short`, and `Args: cobra.MaximumNArgs(1)`.
- [ ] 2.2 Add an `init()` that registers the command with `worktreeCmd.AddCommand(worktreeRefreshCmd)`, following the pattern in `worktree_align.go`.
- [ ] 2.3 Declare package-scope flag variables with refresh-specific names — `flagWorktreeRefreshDryRun` and `flagWorktreeRefreshTags` — so they do not collide with align's `flagDryRun` or add's `flagWorktreeAddTags`.
- [ ] 2.4 Implement `runWorktreeRefresh(cfg *config.Config, repos []*config.Repository, dryRun bool) error` that iterates the targeted repositories calling `syncRepoWorktrees`.
- [ ] 2.5 Accumulate per-repository failures into a slice, print each as it happens, and continue to the next repository rather than aborting the run.
- [ ] 2.6 Call `config.Save(cfg)` exactly once after the loop completes, never inside it.
- [ ] 2.7 Return a non-nil error when the failure slice is non-empty so the process exits non-zero, and nil otherwise.
- [ ] 2.8 Confirm by inspection that the run path calls no repository-discovery, user-detection, or status-cache function, and add a comment stating that those belong to the full `omgw refresh`.
- [ ] 2.9 Write `worktree_refresh_test.go` covering the eight test proof artifacts above, reusing `setupTwoWorktreeTestRepos`, `breakRepo`, and `saveConfigForWorktreeTests`.
- [ ] 2.10 Capture the end-to-end CLI transcript against a scratch fixture workspace and save it under `docs/specs/28-spec-worktree-refresh/28-proofs/`.

### [ ] 3.0 Implement the four targeting forms and their unmatched-filter errors

Deliver spec Unit 2's selection half by reconciling `worktreeScope` (spec 25) with the tag AND-semantics of spec 26, now shared in `worktree.go`: no argument means every tracked repository, a name pattern uses `filter.MatchesPattern`, `.` resolves through `repocontext.ResolveCurrent`, `-t <tag>` uses `filter.MatchesExact` and rejects repetition, and a name plus a tag applies AND.

#### 3.0 Proof Artifact(s)

- Test: `TestWorktreeRefreshTargets_EachForm` passes — no-argument, name-pattern, `.`, and `-t` each select exactly the expected repository set from a fixture config, with the no-argument case run from *inside* a tracked repository to prove it still selects every repository, and a name pattern matching several repositories selecting all of them, demonstrating the targeting contract and guarding against reuse of add/remove's precedence table
- Test: `TestWorktreeRefreshTargets_SingleMatchTagIsBulk` passes — a tag matching exactly one repository takes the same code path and output as a multi-repository run, demonstrating the single-repository-mode lesson from specs 26 and 27
- Test: `TestWorktreeRefreshTargets_NameAndTagAreAnded` passes — a repository matching the name but not the tag is excluded, demonstrating AND rather than OR and consistency with spec 26
- Test: `TestWorktreeRefreshTargets_RepeatedTagRejected` passes — `-t a -t b` returns the single-value error, demonstrating the non-repeatable tag decision from round 1 question 2a
- Test: `TestWorktreeRefreshTargets_UnmatchedFilters` passes — an unmatched name, an unmatched tag, and an unmatched combination each return an error whose message contains the filter values supplied, demonstrating the error contract
- Test: `TestWorktreeRefreshTargets_DotOutsideRepo` passes — `.` outside any tracked repository returns the `repocontext.ResolveCurrent` error rather than falling back to refreshing everything, demonstrating safe failure of current-repo targeting
- Test: `TestWorktreeRefreshCompletion_OffersDot` passes — completion offers `.` only when the working directory resolves, mirroring `worktree_completion_test.go`, demonstrating completion parity with list and align
- CLI: transcript of `omgw worktree refresh -t <tag>` over a tagged fixture set showing only the tagged repositories processed, demonstrating scoped bulk refresh

#### 3.0 Tasks

- [ ] 3.1 Resolve the tag with the shared `singleTagValue(flagWorktreeRefreshTags, "--tag")`, which returns `""` for none, the value for one, and an error naming the count for more than one. Do not add a refresh-specific copy.
- [ ] 3.2 Register the `--tag`/`-t` flag as a `StringArrayVarP` bound to `flagWorktreeRefreshTags`, with the help text "Select repositories by tag (single value; not repeatable)" matching `worktree add`.
- [ ] 3.3 Implement `selectWorktreeRefreshTargets(cfg *config.Config, arg, tag string) ([]*config.Repository, error)` handling all four combinations: build the base set from `worktreeScopeFor(cfg, arg)` so `""` means every repository, `.` the current one, and a name pattern every match, then narrow with `filterByTag` when a tag is present. Do **not** call `selectWorktreeTargets`: its empty case resolves the current repository and it rejects multi-match patterns, both wrong for refresh (see planning notes).
- [ ] 3.4 Return an error naming both filters when the combined selection is empty, following the message shapes already used in `selectWorktreeTargets` (`"no repository found matching '%s' and tagged '%s'"`).
- [ ] 3.5 Propagate the `repocontext.ResolveCurrent` error unchanged when `arg` is `.` and the working directory is not inside a tracked repository.
- [ ] 3.6 Wire the command's `RunE` to load config, resolve the tag, call `selectWorktreeRefreshTargets`, and hand the result to `runWorktreeRefresh`.
- [ ] 3.7 Set `worktreeRefreshCmd.ValidArgsFunction = completeWorktreeRepoOrDot` and register tag completion with `completeAllTags`, matching `worktree add`.
- [ ] 3.8 Write `worktree_refresh_target_test.go` covering the six test proof artifacts above, reusing `setupTaggedFixture`, `standardFixture`, and `chdirForTest`.
- [ ] 3.9 Capture the `-t <tag>` CLI transcript into `28-proofs/`.

### [ ] 4.0 Report per-repository changes and a final summary

Deliver spec Unit 2's reporting half: snapshot each repository's stored worktrees before the rebuild, diff against the result keyed on path, and print the added, removed, and realigned entries for each repository that changed, followed by a summary of repositories refreshed, changed, and failed. A repository whose data did not change prints nothing.

#### 4.0 Proof Artifact(s)

- Test: `TestWorktreeRefreshReport_ListsAllThreeChangeKinds` passes — a repository with one added, one removed, and one realigned worktree produces all three labelled in its output, demonstrating change reporting
- Test: `TestWorktreeRefreshReport_KeysOnPath` passes — two worktrees on detached HEADs, both with an empty `Branch`, are tracked as distinct entries rather than collapsing, demonstrating the stated comparison key
- Test: `TestWorktreeRefreshReport_SilentWhenUnchanged` passes — a repository whose stored data is already accurate contributes no lines to stdout, demonstrating quiet-when-nothing-happened behavior
- Test: `TestWorktreeRefreshReport_SummaryCounts` passes — across a fixture with changed, unchanged, and failed repositories, the summary reports each count correctly, demonstrating the summary contract
- CLI: captured stdout of a mixed run showing per-repository change lines followed by the summary, demonstrating the format matches the repository's report-only-what-changed convention

#### 4.0 Tasks

- [ ] 4.1 Define a `worktreeChanges` struct holding `Added`, `Removed`, and `Realigned` slices of `config.Worktree`, plus a `changed() bool` method.
- [ ] 4.2 Implement `diffWorktrees(before, after []config.Worktree) worktreeChanges`, building maps keyed on `Path` — never on `Branch`, which is empty for a detached HEAD — and classifying an entry present in both whose `Aligned` differs as realigned.
- [ ] 4.3 In `runWorktreeRefresh`, copy each repository's `Worktrees` slice before calling `syncRepoWorktrees`, since the helper reassigns the field in place.
- [ ] 4.4 Print a per-repository block only when `changed()` is true, listing added, removed, and realigned entries with their paths and branches.
- [ ] 4.5 Print a final summary line giving repositories refreshed, changed, and failed, using the existing `pluralize` helper for agreement.
- [ ] 4.6 Write `worktree_refresh_report_test.go` covering the four test proof artifacts above, using `captureStdoutStr` for output assertions.
- [ ] 4.7 Capture the mixed-run stdout transcript into `28-proofs/`.

### [ ] 5.0 Add `--dry-run` preview

Deliver spec Unit 3: `--dry-run` reports what a refresh would change and exits without saving. Because `repair` and `prune` mutate git state, dry run calls `buildWorktreeEntries` alone — the non-mutating half split out in task 1.1 — and states in its output that repair and prune were not run, so the user understands a repairable entry may appear as it currently stands.

#### 5.0 Proof Artifact(s)

- Test: `TestWorktreeRefreshDryRun_ConfigByteIdentical` passes — the `config.json` bytes are compared before and after and are equal while the run still reports pending changes, demonstrating preview safety per success metric 4
- Test: `TestWorktreeRefreshDryRun_DoesNotRepair` passes — a worktree with a broken git pointer is still broken after a dry run, observable because a following `worktree list` still reports it broken, demonstrating that repair did not run without needing a mock
- Test: `TestWorktreeRefreshDryRun_DoesNotPrune` passes — a worktree whose directory was deleted is still present in `git worktree list --porcelain` output after a dry run, demonstrating that prune did not run
- Test: `TestWorktreeRefreshDryRun_StatesCaveat` passes — dry-run stdout contains the sentence stating repair and prune were not run, demonstrating the honesty requirement
- Test: `TestWorktreeRefreshDryRun_OutputParity` passes — the change lines and summary of a dry run match those of the subsequent real run on an unchanged fixture, demonstrating output parity
- CLI: transcript of `omgw worktree refresh --dry-run` followed by the real run against the same fixture, showing the preview matched the outcome, demonstrating preview fidelity

#### 5.0 Tasks

- [ ] 5.1 Register the `--dry-run` flag as a `BoolVar` bound to `flagWorktreeRefreshDryRun`, with help text "Preview changes without writing them" following align's wording.
- [ ] 5.2 In `runWorktreeRefresh`, branch on `dryRun`: call `buildWorktreeEntries(repo.Path, repo.Name)` and diff its result against the stored slice **without** assigning it, instead of calling `syncRepoWorktrees`.
- [ ] 5.3 Skip the `config.Save` call entirely when `dryRun` is set, and confirm no other write path runs.
- [ ] 5.4 Print the caveat sentence stating that `git worktree repair` and `git worktree prune` were not run and that a repairable worktree therefore appears as it currently stands.
- [ ] 5.5 Reuse the same per-repository block and summary rendering as the real run, wording the counts as what would change.
- [ ] 5.6 Write `worktree_refresh_dryrun_test.go` covering the five test proof artifacts above, reading the config file bytes directly with `os.ReadFile` for the byte-identity assertion.
- [ ] 5.7 Capture the dry-run-then-real-run CLI transcript into `28-proofs/`.

### [ ] 6.0 Document the command and clarify its scope in help text

Add `omgw worktree refresh` to `docs/site/commands-core.md` beside the existing `worktree list` and `worktree align` sections, and update `worktreeCmd`'s subcommand list. Per the spec's technical considerations, the help text must state explicitly that `worktree refresh` updates worktree data only, so users do not expect repository discovery from it.

#### 6.0 Proof Artifact(s)

- CLI: `omgw worktree refresh --help` output showing all four targeting forms, `--dry-run`, and a sentence stating that only worktree data is updated, demonstrating the naming-overlap clarification is delivered
- CLI: `omgw worktree --help` lists `refresh` among the subcommands, demonstrating registration and discoverability
- Diff: `docs/site/commands-core.md` gains a `worktree refresh` section structured like the existing `worktree align` section at line 572, demonstrating documentation parity
- CLI: `make docs` starts the MkDocs server with no warnings referencing the new section, demonstrating the documentation is well-formed

#### 6.0 Tasks

- [ ] 6.1 Write `worktreeRefreshCmd.Long` covering what the command does, the four targeting forms, and `--dry-run`, with an Examples block matching the style of `worktreeAlignCmd.Long`.
- [ ] 6.2 State explicitly in `Long` that `worktree refresh` updates worktree metadata only, and that repository discovery, user detection, and status-cache clearing remain with `omgw refresh`.
- [ ] 6.3 Add `gws worktree refresh [repo|.]` to the `Subcommands:` list in `worktreeCmd.Long` in `worktree.go`.
- [ ] 6.4 Add a `worktree refresh` section to `docs/site/commands-core.md` following the `worktree align` section's structure, with a usage line, the targeting forms, and examples.
- [ ] 6.5 Run `make docs` and confirm the new section renders without warnings, then capture the two `--help` transcripts into `28-proofs/`.
