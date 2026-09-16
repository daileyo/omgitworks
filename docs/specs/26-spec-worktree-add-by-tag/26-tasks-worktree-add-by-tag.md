# 26-tasks-worktree-add-by-tag.md

> Source spec: `26-spec-worktree-add-by-tag.md` in this directory.
> Clarification record: `../25-spec-worktree-repo-context/25-questions-1-...md` and `25-questions-2-...md`.

## Prerequisite

This spec **depends on spec 25**. The precedence table's one-positional-no-tag row is
spec 25's resolver, and every task below extends `addWorktreeForRepo`, the shared
creation path spec 25 introduced.

This branch (`feat/worktree-add-by-tag`) is currently based on `aa04200`, which predates
spec 25's implementation: `internal/repocontext` does not exist here and
`worktree add` is still `cobra.ExactArgs(2)`. **Before task 1.1, the branch must take
spec 25's commits** — either by rebasing onto `feat/worktree-repo-context` or by waiting
for spec 25 to merge and rebasing onto `main`. The tasks below are written against the
post-spec-25 tree.

## Planning Assumption (needs confirmation)

The spec is ambiguous about one interaction and the tasks below resolve it explicitly.

**Question:** when two positionals are combined with `-t`, and the name-and-tag
intersection matches *more than one* repository, is that an error or a bulk run?

- FR U2-6 says "the **two-positional form without `-t`** shall continue to reject an
  ambiguous name pattern", which scopes the ambiguity error to the no-tag form.
- Technical Considerations say "the ambiguity check must run **after** the tag filter is
  applied, not before", which only matters if the check still applies when `-t` is present.

**Assumption taken:** the user story "I want to narrow a tag to a subset by also giving a
name pattern so that I can **act on part of a tagged group**" implies acting on several
repositories, so `add <repo> <branch> -t <tag>` performs a **bulk run over the
intersection** and raises no ambiguity error. The ambiguity error is preserved only for
the two-positional form without `-t`, exactly as today.

Sub-tasks 1.8 and 1.9 implement this reading. If the opposite is intended, only those two
sub-tasks and `TestWorktreeAddSelection_AmbiguityAfterTagFilter` change.

## Relevant Files

| File | Why It Is Relevant |
| --- | --- |
| `cmd/omgitworks/worktree_add.go` | The whole feature lands here: the `-t` flag, the selection logic, the bulk loop, and the summary. The spec is explicit that this extends the existing command rather than adding a parallel one. |
| `cmd/omgitworks/worktree_add_test.go` | Five pre-existing single-repository cases that must keep passing; the two-positional ambiguity error lives here. |
| `cmd/omgitworks/worktree_add_current_test.go` | Spec 25's resolution tests, including `TestWorktreeAddCmd_Arity`, which must be updated for the new argument shapes. |
| `cmd/omgitworks/worktree_add_tag_test.go` | **New.** Tag selection, the four-row precedence table, combined AND filtering, and the empty-match errors. |
| `cmd/omgitworks/worktree_add_bulk_test.go` | **New.** Bulk creation, single-save persistence, partial failure, summary counts, and exit status. |
| `cmd/omgitworks/worktree.go` | Holds `worktreeScope` and the completion functions; `--tag` completion is registered here. |
| `internal/filter/filter.go` | `MatchesExact` (line 55) for tags and `MatchesPattern` (line 36) for name patterns. Read-only: the spec requires reuse, not new matching logic. |
| `cmd/omgitworks/worktree_align.go` | The reference implementation for bulk-run output: `errors []string` collection, counted summary, separately headed error block. |
| `cmd/omgitworks/init.go` | Supplies the `pluralize` helper (line 115) used by every summary line. |
| `cmd/omgitworks/list.go` | Line 169 defines `--tag`/`-t` for `list`; its stale "(repeatable for AND logic)" description is corrected in task 4.0. |
| `docs/site/commands-core.md` | Published reference for `worktree add`; gains the `-t` form and the precedence table. |

### Notes

- Tests live beside the code they cover. Run a package with `go test ./cmd/omgitworks/`,
  and the full gate with `make ci` (vet + golangci-lint + `go test -race`).
- `golangci-lint` must be on `PATH`; it is installed at `$(go env GOPATH)/bin` from
  spec 25 at the CI-pinned v2.13.2.
- Fixtures that create git repositories must keep using the `GIT_CONFIG_*` hooks
  isolation already present in `cmd/omgitworks`, or a global `core.hooksPath` will
  reject the scaffolding commit messages.
- Follow `worktree_align.go` for output shape: per-item lines during the run, then a
  counted summary, then a separately headed error block only when failures occurred.
- Commit as `feat(worktree): ...`.

## Tasks

### [x] 1.0 Repository Selection: the `-t` Flag and the Precedence Table

Decide *which repositories* an invocation targets, before anything is created. This task
delivers selection only; creation stays single-repository until task 2.0.

#### 1.0 Proof Artifact(s)

- Test: `go test -run TestWorktreeAddSelection_PrecedenceTable -v ./cmd/omgitworks/` passes with one sub-case per row of the spec's four-row table, demonstrates the full precedence contract (FR U2-1)
- Test: `go test -run TestWorktreeAddSelection_TagMatching -v ./cmd/omgitworks/` passes for an exact tag, a differently-cased tag, and a wildcard, demonstrates reuse of `filter.MatchesExact` (FR U1-2)
- Test: `go test -run TestWorktreeAddSelection_CombinedAnd -v ./cmd/omgitworks/` passes, asserting a repository carrying the tag but not matching the name is excluded and vice versa, demonstrates AND semantics (FR U2-2)
- Test: `go test -run TestWorktreeAddSelection_ResolverNotConsulted -v ./cmd/omgitworks/` passes from a directory inside repository A while `-t` selects repository B, and asserts the result is B, demonstrating detection is the lowest-precedence source (FR U2-4)
- Test: `go test -run TestWorktreeAddSelection_AmbiguityAfterTagFilter -v ./cmd/omgitworks/` passes, showing a pattern that is ambiguous alone becomes unambiguous once the tag narrows it, demonstrates ordering of the ambiguity check (FR U2-6)
- Test: `go test -run TestWorktreeAddSelection_EmptyMatches -v ./cmd/omgitworks/` passes asserting the tag-only error names the tag and the combined error names both pattern and tag (FR U1-8, U2-5)
- Test: `go test -run TestWorktreeAddCmd_TagFlag -v ./cmd/omgitworks/` passes asserting `--tag` given twice is rejected with an error rather than silently using the last value (FR U1-3)
- CLI: `omgw worktree add --help` transcript showing the `-t, --tag` flag with a description stating it takes a single value, demonstrates the constraint is discoverable

#### 1.0 Tasks

- [x] 1.1 Rebase this branch so spec 25's commits are present, then confirm `internal/repocontext` exists and `worktree_add.go` has `cobra.RangeArgs(1, 2)` and `addWorktreeForRepo`. Run `make ci` once on the rebased tree to establish a green baseline before any change.
- [x] 1.2 Declare `var flagWorktreeAddTags []string` in `worktree_add.go` and register it in `init()` as `worktreeAddCmd.Flags().StringArrayVarP(&flagWorktreeAddTags, "tag", "t", nil, "Select repositories by tag (single value; not repeatable)")`. `StringArrayVar` rather than `StringVar` is required: a plain string silently keeps the last value, which FR U1-3 forbids.
- [x] 1.3 Add `func worktreeAddTag() (string, error)`: return `""` for no occurrences, the single value for one, and an error naming the flag for two or more. Call it first in `RunE` so a duplicate `--tag` is rejected before any repository work.
- [x] 1.4 Widen `Args` to `cobra.RangeArgs(1, 2)` unchanged, and add a `PreRunE`-free arity note: with `-t` the one-positional form means branch, so no arity change is needed. Update `TestWorktreeAddCmd_Arity` only if a new bound is actually introduced.
- [x] 1.5 Add `func selectByTag(cfg *config.Config, tag string) []*config.Repository`, matching each repository's `Tags` entries with `filter.MatchesExact(t, tag)`. Return pointers into `cfg.Repositories` so callers can mutate and save.
- [x] 1.6 Add `func selectByName(cfg *config.Config, pattern string) []*config.Repository` wrapping the existing `filter.MatchesPattern` loop, so name matching stays in one place rather than being duplicated between the single and bulk paths.
- [x] 1.7 Add `func selectWorktreeAddTargets(cfg *config.Config, pattern, tag string) ([]*config.Repository, error)` implementing the precedence table: neither given returns the resolver's single repository via `repocontext.ResolveCurrent`; tag only returns `selectByTag`; pattern only returns `selectByName`; both returns the intersection, computed by applying `selectByName` then filtering that result by tag so the AND semantics are explicit.
- [x] 1.8 In `selectWorktreeAddTargets`, apply the ambiguity check **only** on the pattern-without-tag path, preserving `multiple repositories match '%s', narrow your query` verbatim. Per the Planning Assumption above, the pattern-plus-tag path returns the full intersection without an ambiguity error.
- [x] 1.9 Add the empty-selection errors: tag alone that matches nothing returns an error naming the tag; pattern alone keeps today's `no repository found matching '%s'`; pattern plus tag returns an error naming **both** the pattern and the tag, as FR U2-5 requires.
- [x] 1.10 Guarantee the resolver is only consulted when both `pattern` and `tag` are empty, by returning from the tag and pattern branches before the detection branch is reached. This is the lowest-precedence rule from spec 25's forward constraint.
- [x] 1.11 Rewrite `RunE` to read the branch from the last positional, derive the pattern from the first positional when two are present, and dispatch through `selectWorktreeAddTargets`. Keep `runWorktreeAdd` and `runWorktreeAddCurrent` as thin wrappers so spec 25's tests continue to call them directly.
- [x] 1.12 Create `cmd/omgitworks/worktree_add_tag_test.go` with a fixture building four tracked repositories: two tagged `backend`, one tagged `frontend`, one untagged, with names that let a pattern match across tag groups. Reuse the `GIT_CONFIG_*` hooks isolation already present in the package.
- [x] 1.13 Write `TestWorktreeAddSelection_PrecedenceTable` as a table test with one sub-case per row of the spec's four-row table, asserting the exact set of repository names returned for each invocation shape.
- [x] 1.14 Write `TestWorktreeAddSelection_TagMatching` covering an exact tag, a differently-cased tag, and a wildcard such as `back*`, asserting `filter.MatchesExact` semantics are what selection uses.
- [x] 1.15 Write `TestWorktreeAddSelection_CombinedAnd`, asserting a repository carrying the tag but not matching the name is excluded, and one matching the name but lacking the tag is excluded.
- [x] 1.16 Write `TestWorktreeAddSelection_ResolverNotConsulted`: chdir inside repository A, select with `-t` matching only repository B, and assert the result is B alone. Add a second sub-case asserting a pattern alone also ignores the working directory.
- [x] 1.17 Write `TestWorktreeAddSelection_AmbiguityAfterTagFilter` with two sub-cases: a pattern matching two repositories **without** a tag errors with the existing message, and the same pattern **with** a tag that narrows to two returns both without error, pinning the Planning Assumption.
- [x] 1.18 Write `TestWorktreeAddSelection_EmptyMatches` asserting the tag-only error contains the tag, and the combined error contains both the pattern and the tag.
- [x] 1.19 Write `TestWorktreeAddCmd_TagFlag` invoking the flag parser with `--tag a --tag b` and asserting `worktreeAddTag` returns an error mentioning `--tag`; add a passing sub-case for a single occurrence.
- [x] 1.20 Run `gofmt -l`, `go vet ./...`, and `golangci-lint run ./cmd/...`; resolve findings rather than adding exclusions.

### [x] 2.0 Bulk Creation Across the Selected Repositories

Create one worktree per selected repository, reusing the proven single-repository path,
and persist the result exactly once.

#### 2.0 Proof Artifact(s)

- CLI: transcript of `omgw worktree add -t backend feat-x` across three tagged repositories followed by `omgw worktree list`, showing three new worktrees and none in the untagged repository, demonstrates the feature end to end (FR U1-1, U1-5)
- Test: `go test -run TestWorktreeAddBulk_CreatesForEachTagged -v ./cmd/omgitworks/` passes, asserting a worktree exists in every tagged repository and none in untagged ones (FR U1-1, U1-5)
- Test: `go test -run TestWorktreeAddBulk_SingleSave -v ./cmd/omgitworks/` passes, asserting the reloaded configuration records the worktree for **every** repository in the run, demonstrating no repository's data was lost to an overwriting save (FR U1-9)
- Test: `go test -run TestWorktreeAddBulk_BranchWithSlash -v ./cmd/omgitworks/` passes, demonstrates parent directories are created for nested branch names (FR U1-6)
- Test: `go test -run TestWorktreeAddBulk_SkipsExisting -v ./cmd/omgitworks/` passes, asserting a repository that already has the branch's worktree is skipped, announced during the run, and not counted as a failure (FR U1-7)
- Test: `go test -run TestRunWorktreeAdd -v ./cmd/omgitworks/` shows the pre-existing single-repository cases still passing, demonstrates the bulk path did not disturb them (FR U2-3)

#### 2.0 Tasks

- [x] 2.1 Split the persistence out of `addWorktreeForRepo`: extract everything from the duplicate check through the worktree re-discovery into `func createWorktreeForRepo(repo *config.Repository, branch string) (destPath string, err error)`, which performs **no** `config.Save` and **no** printing.
- [x] 2.2 Introduce a typed sentinel for the already-present case, `type worktreeExistsError struct{ Branch, Path string }` implementing `error` with today's exact wording `worktree for branch '%s' already exists at %s`, and return it from `createWorktreeForRepo`. A typed error lets the bulk path classify a skip without string matching.
- [x] 2.3 Reduce `addWorktreeForRepo` to: call `createWorktreeForRepo`, return the error unchanged on failure (so `TestRunWorktreeAdd_DuplicateBranch` still sees the same message), then `config.Save` and print the existing `Created worktree for branch '%s' at %s` line. Verify the five pre-existing single-repository tests pass before continuing.
- [x] 2.4 Add `func runWorktreeAddBulk(cfg *config.Config, repos []*config.Repository, branch string) error` that loops over the selection calling `createWorktreeForRepo`, classifying each result as created, skipped (`errors.As` against `worktreeExistsError`), or failed.
- [x] 2.5 Print one line per repository as the run proceeds — created and skipped alike — following the `Skipping [%s] %s — %s` shape already used by `runWorktreeAlign` for locked worktrees.
- [x] 2.6 Call `config.Save(cfg)` exactly **once**, after the loop completes, and only when at least one worktree was created. The loop must hold the single loaded `*config.Config` throughout; saving per repository would write stale data over earlier repositories, which is the failure mode the spec's Technical Considerations call out.
- [x] 2.7 Route the single-repository paths through the bulk function where the selection has exactly one repository, so there is one creation loop rather than two code paths that can drift. Keep the single-repository output wording unchanged.
- [x] 2.8 Create `cmd/omgitworks/worktree_add_bulk_test.go` with a fixture of three tagged and one untagged repository, each a real git repository with an initial commit.
- [x] 2.9 Write `TestWorktreeAddBulk_CreatesForEachTagged`, asserting a worktree directory exists under each tagged repository's projects path and none exists for the untagged one.
- [x] 2.10 Write `TestWorktreeAddBulk_SingleSave`: run the bulk creation, reload configuration from disk, and assert **every** tagged repository records the new worktree. This is the regression guard for the overwriting-save failure mode.
- [x] 2.11 Write `TestWorktreeAddBulk_BranchWithSlash` using a branch such as `hotfix/urgent`, asserting the nested directory is created in each repository.
- [x] 2.12 Write `TestWorktreeAddBulk_SkipsExisting`: pre-create the branch's worktree in one tagged repository, run the bulk add, and assert that repository is skipped and announced while the others are created, and that the run does not report a failure.
- [x] 2.13 Re-run `go test -run TestRunWorktreeAdd -v ./cmd/omgitworks/` and confirm all five pre-existing single-repository cases and spec 25's resolution cases still pass unchanged.

### [x] 3.0 Partial-Failure Reporting and Exit Status

Make a mixed-outcome run legible and correctly signalled, without aborting early and
without rolling back what already succeeded.

#### 3.0 Proof Artifact(s)

- CLI: transcript of a mixed-outcome run across four repositories — some created, one skipped as already present, one failed — showing per-item lines, the counted summary, and the error block, demonstrates the reporting format (FR U3-2, U3-3)
- Test: `go test -run TestWorktreeAddBulk_ContinuesPastFailure -v ./cmd/omgitworks/` passes, asserting repositories after the failing one still get worktrees (FR U3-1)
- Test: `go test -run TestWorktreeAddBulk_SummaryCounts -v ./cmd/omgitworks/` passes, asserting created, skipped, and failed counts are each correct in a run containing all three outcomes (FR U3-3)
- Test: `go test -run TestWorktreeAddBulk_NamesFailedRepos -v ./cmd/omgitworks/` passes, asserting each failure line contains the repository name and the underlying reason (FR U3-4)
- Test: `go test -run TestWorktreeAddBulk_ExitStatus -v ./cmd/omgitworks/` passes with sub-cases for all-succeeded (nil error), skips-only (nil error), and any-failure (non-nil), demonstrates the exit-status contract (FR U3-5)
- Test: `go test -run TestWorktreeAddBulk_NoRollback -v ./cmd/omgitworks/` passes, asserting worktrees created before a later failure still exist afterwards (FR U3-6)
- CLI: `omgw worktree add -t <tag> <branch>; echo $?` transcript of a mixed run printing a non-zero status **once**, with no cobra usage block or duplicate error line over the summary, demonstrates the exit path is clean

#### 3.0 Tasks

- [x] 3.1 In `runWorktreeAddBulk`, collect failures into an `errors []string` slice formatted `  %s: %v` with the repository name and the underlying reason, mirroring `runWorktreeAlign`'s collection exactly. Never abort the loop on a failure.
- [x] 3.2 Print the counted summary after the loop: created, skipped, and failed counts, each through `pluralize` from `init.go:115`. Suppress a count line that is zero so a clean run stays quiet.
- [x] 3.3 Print the error block only when failures occurred, under its own heading, following `runWorktreeAlign`'s `\n%d %s:\n%s\n` shape.
- [x] 3.4 Define a package-level sentinel `var errPartialFailure = errors.New("")` and return it from `runWorktreeAddBulk` when the failure count is greater than zero, after the summary has been printed. Return `nil` when the run contained only creations and skips.
- [x] 3.5 Set `worktreeAddCmd.SilenceErrors = true` and `worktreeAddCmd.SilenceUsage = true` in `init()` so the sentinel produces a non-zero exit without cobra printing an empty error line or a usage block over the summary.
- [x] 3.6 Because those two flags also suppress cobra's output for **every** other `worktree add` error — including `no repository found matching '%s'` and the spec 25 resolution error, which previously printed a usage block — print those errors explicitly. Add a small wrapper in `RunE` that writes `Error: %v` to stderr for any non-sentinel error before returning it, so no error path becomes silent.
- [x] 3.7 Write `TestWorktreeAddBulk_ContinuesPastFailure`: make the middle repository of three fail, and assert the third still receives its worktree.
- [x] 3.8 Write `TestWorktreeAddBulk_SummaryCounts` over a run containing one creation, one skip, and one failure, asserting all three counts appear correctly in captured output via `captureStdoutStr`.
- [x] 3.9 Write `TestWorktreeAddBulk_NamesFailedRepos`, asserting the error block contains the failing repository's name and a recognizable fragment of the underlying reason.
- [x] 3.10 Write `TestWorktreeAddBulk_ExitStatus` with three sub-cases: all created returns `nil`; creations plus skips returns `nil`; any failure returns non-`nil` matching `errPartialFailure` via `errors.Is`.
- [x] 3.11 Write `TestWorktreeAddBulk_NoRollback`, asserting worktrees created before the failing repository still exist on disk and are recorded in the saved configuration afterwards.
- [x] 3.12 Write `TestWorktreeAddCmd_ErrorsStillPrinted`, asserting that with `SilenceErrors` set, a non-sentinel error such as an unmatched pattern still produces an `Error:` line on stderr. This guards the side effect introduced in 3.6.
- [x] 3.13 Verify the exit path manually: build the binary, run a mixed-outcome bulk add, and confirm `echo $?` is non-zero with exactly one summary, no duplicate error line, and no usage block.

### [x] 4.0 Completion, Documentation, and the Stale `list --tag` Description

> **Scope note:** spec 26 states no functional requirement for completion or
> documentation, unlike spec 25's Unit 3. This task is included on the repository's
> established convention that `docs/site/` is updated alongside behavior, which is what
> spec 18 exists to enforce. It is flagged in the planning audit as exceeding the spec's
> stated requirements; drop it if you would rather handle docs as a follow-up.

#### 4.0 Proof Artifact(s)

- CLI: `omgw __complete worktree add --tag ""` transcript listing tags in use across tracked repositories, demonstrates tag completion
- Test: `go test -run TestCompleteWorktreeAddTag -v ./cmd/omgitworks/` passes, asserting known tags are suggested, deduplicated, and prefix-filtered
- Documentation: diff of `docs/site/commands-core.md` showing the `-t/--tag` flag, the four-row precedence table, the single-value constraint, and the partial-failure exit behavior, demonstrates the published reference matches shipped behavior
- Diff: `cmd/omgitworks/list.go:169` flag description corrected from "(repeatable for AND logic)" to a single-value wording, demonstrates the stale help text no longer contradicts the code and the published docs
- CLI: `make ci` exits 0, demonstrates vet, lint, and the race-enabled suite pass on the complete change

#### 4.0 Tasks

- [x] 4.1 Add `func completeAllTags(toComplete string) ([]string, cobra.ShellCompDirective)` in `worktree.go`, returning the deduplicated, prefix-filtered set of tags across all tracked repositories, following the shape of `completeRepoTags` in `tag.go:214`.
- [x] 4.2 Register it with `worktreeAddCmd.RegisterFlagCompletionFunc("tag", ...)` in `init()`, so the flag value completes rather than falling back to file completion.
- [x] 4.3 Write `TestCompleteWorktreeAddTag`, asserting tags from several repositories are suggested once each, filtered on the typed prefix, and that the directive is `ShellCompDirectiveNoFileComp`.
- [x] 4.4 Update the `omgw worktree add` section of `docs/site/commands-core.md`: signature `omgw worktree add [repo] <branch> [-t <tag>]`, a bulk example, and a statement that `--tag` takes a single value.
- [x] 4.5 Add the four-row precedence table to that section as a markdown table, matching the spec's table exactly so the documented behavior and the implemented behavior are checkable against one another.
- [x] 4.6 Document the partial-failure contract in the same section: the run continues past failures, prints created/skipped/failed counts, retains successful creations, and exits non-zero if anything failed.
- [x] 4.7 Correct `cmd/omgitworks/list.go:169`: the `--tag` description currently reads "Filter by tag (repeatable for AND logic)" while the flag is a single-valued `StringVarP`. Change it to describe a single value, matching both the code and what `docs/site/` already documents.
- [x] 4.8 Run `make ci` and confirm exit 0, then capture the CLI transcripts named in the 1.0-3.0 proof artifacts using a sandbox workspace with isolated `XDG_CONFIG_HOME` and `XDG_DATA_HOME`. Confirm no absolute home paths or identifying values appear in the captured output.

## Requirement Coverage Map

| Unit | Functional Requirement | Covered By |
| --- | --- | --- |
| 1 | `-t` / `--tag` flag selects repositories by tag | 1.0 |
| 1 | Tag matched with `filter.MatchesExact` | 1.0 |
| 1 | `--tag` supplied twice is rejected | 1.0 |
| 1 | With `-t` and one positional, the positional is the branch | 1.0 |
| 1 | Create per matched repo using single-repository behavior | 2.0 |
| 1 | Parent directories created for slashed branch names | 2.0 |
| 1 | Repo already holding the branch is skipped and reported | 2.0 |
| 1 | Unmatched tag exits non-zero naming the tag | 1.0 |
| 1 | Worktree data updated and configuration saved once | 2.0 |
| 2 | Four-row precedence table resolves targeting | 1.0 |
| 2 | Name pattern plus tag requires both (AND) | 1.0 |
| 2 | Name pattern still matched with `filter.MatchesPattern` | 1.0 |
| 2 | Resolver invoked only when neither pattern nor tag is given | 1.0 |
| 2 | Combined filter matching nothing names both filters | 1.0 |
| 2 | Two-positional form still rejects an ambiguous pattern | 1.0 |
| 3 | Creation attempted for every matched repo, continuing past failures | 3.0 |
| 3 | Per-repository errors collected and printed after the run | 3.0 |
| 3 | Summary reports created, skipped, and failed counts | 3.0 |
| 3 | Each failure names the repository and the reason | 3.0 |
| 3 | Non-zero exit on any failure, zero when only skips occurred | 3.0 |
| 3 | Successful creations retained, not rolled back | 3.0 |
