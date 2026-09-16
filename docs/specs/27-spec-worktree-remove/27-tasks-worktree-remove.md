# 27-tasks-worktree-remove.md

> Source spec: `27-spec-worktree-remove.md` in this directory.
> Clarification record: `../25-spec-worktree-repo-context/25-questions-1-...md` and `25-questions-2-...md`.

## Prerequisite

This spec depends on **spec 25** for the current-repository resolver and shares the
targeting precedence model of **spec 26**.

This branch (`feat/worktree-remove`) is at `aa04200`: no `internal/repocontext`, and none
of spec 26's selection code. **Before task 1.1 the branch must be rebased onto
`feat/worktree-add-by-tag`**, which carries both. The tasks below are written against that
tree. When PRs #100 and #101 merge, rebase onto `main` instead.

## Planning Note: generalizing spec 26's selector

FR U2-3 requires `remove` to apply "the same targeting precedence table defined in spec
26". Spec 26 implemented that as `selectWorktreeAddTargets` in `worktree_add.go` — a name
and a home that are both specific to `add`.

Rather than duplicating the table, task 3.0 moves it to `worktree.go` as
`selectWorktreeTargets`, which `add` and `remove` both call. This is a rename plus a file
move with no behavior change, guarded by spec 26's existing precedence tests. Duplicating
it instead would leave two copies of a table that must stay identical across specs 26, 27,
and 28 — the exact drift the shared model exists to prevent.

## Relevant Files

| File | Why It Is Relevant |
| --- | --- |
| `internal/git/worktree.go` | Gains `RemoveWorktree`. Already holds `AddWorktree`, `MoveWorktree`, `IsWorktreeLocked` (line 207) and `removeEmptyLegacyDir`'s conservative-cleanup precedent. |
| `internal/git/worktree_test.go` | Tests for the new primitive against real git fixtures, including git's own dirty-worktree refusal. |
| `cmd/omgitworks/worktree_remove.go` | **New.** The command: flags, targeting, the removal loop, dry run, confirmation, and summary. |
| `cmd/omgitworks/worktree_remove_test.go` | **New.** Individual removal, force, locks, branch preservation, directory cleanup. |
| `cmd/omgitworks/worktree_remove_bulk_test.go` | **New.** Tag-scoped removal, skip-versus-fail, partial failure, exit status, single-save persistence. |
| `cmd/omgitworks/worktree_remove_confirm_test.go` | **New.** Dry run, the confirmation gate, `--yes`, and the non-interactive refusal. |
| `cmd/omgitworks/worktree.go` | Gains `selectWorktreeTargets` (moved from `worktree_add.go`) and the branch/tag completion for `remove`. Its `Long` text lists the subcommands. |
| `cmd/omgitworks/worktree_add.go` | Loses `selectWorktreeAddTargets` to `worktree.go` and calls the shared function; otherwise unchanged. |
| `cmd/omgitworks/worktree_align.go` | The reference for `errors []string` collection, counted summaries, and the `--dry-run` output shape that Unit 3 must match. |
| `cmd/omgitworks/navigate.go` | `runNavigate` (line 24) is the I/O-injection precedent — `stderr`, `stdout`, `stdin` as parameters — which the confirmation prompt requires to be testable. `isCharDevice` (line 233) is one of the two TTY approaches in the tree. |
| `cmd/omgitworks/list.go` | Line 112 uses `term.IsTerminal(int(os.Stdout.Fd()))`, the other TTY approach. Task 4.0 applies the same idea to **stdin**. |
| `cmd/omgitworks/init.go` | `pluralize` (line 115) for summary counts. |
| `docs/site/commands-core.md` | Published reference; gains the `remove`/`rm` section. |

### Notes

- Tests live beside the code they cover. `make ci` (vet + golangci-lint + `go test -race`)
  is the gate; `golangci-lint` is at `$(go env GOPATH)/bin` at the CI-pinned v2.13.2.
- Removal behavior around dirty worktrees is **git's**, not omgitworks'. Fixtures must be
  real repositories with real worktrees in `t.TempDir()`; stubbing the git layer would
  test nothing. Keep the `GIT_CONFIG_*` hooks isolation already in these packages.
- Follow `worktree_align.go` for output: per-item lines, counted summary, separately
  headed error block.
- Commit as `feat(worktree): ...`.

## Tasks

### [x] 1.0 The `git.RemoveWorktree` Primitive and Its Safety Semantics

The git-level operation, including the two safety behaviors every higher form inherits:
git's refusal on a dirty worktree, and the lock guarantee.

#### 1.0 Proof Artifact(s)

- Test: `go test -run TestRemoveWorktree -v ./internal/git/` passes, asserting the worktree directory is gone and `git worktree list` no longer reports it, demonstrates the core operation (FR U1-2)
- Test: `go test -run TestRemoveWorktree_DirtyRefused -v ./internal/git/` passes, asserting removal of a worktree with uncommitted changes fails **without** `--force` and succeeds **with** it, demonstrates the default safety posture (FR U1-6, U1-7)
- Test: `go test -run TestRemoveWorktree_UntrackedRefused -v ./internal/git/` passes for an untracked file, demonstrating the refusal is git's own and covers more than modified files (FR U1-6)
- Test: `go test -run TestRemoveWorktree_BranchSurvives -v ./internal/git/` passes, asserting `git branch --list <branch>` still reports the branch after its worktree is removed (FR U1-9)
- Test: `go test -run TestRemoveWorktree_ErrorKeepsGitReason -v ./internal/git/` passes, asserting the returned error still contains git's own wording rather than replacing it, demonstrates the wrap-not-replace decision
- CLI: `golangci-lint run ./internal/...` reports `0 issues`, demonstrates the new primitive meets the lint gate

#### 1.0 Tasks

- [x] 1.1 Rebase this branch onto `feat/worktree-add-by-tag`, then confirm `internal/repocontext` and `selectWorktreeAddTargets` are present. Run `make ci` once for a green baseline before any change.
- [x] 1.2 Add `func RemoveWorktree(repoPath, worktreePath string, force bool) error` to `internal/git/worktree.go`, next to `AddWorktree` (line 107), running `gitCommand(repoPath, "worktree", "remove", worktreePath)` and appending `--force` only when `force` is true. Match the existing parameter order convention of `MoveWorktree(repoPath, currentPath, newPath)`.
- [x] 1.3 Wrap a failure as `failed to remove worktree: %w` so git's own text stays visible. Question 5b deferred the dirty check to git, so replacing git's wording would hide the only explanation the user gets for a refusal.
- [x] 1.4 Document on `RemoveWorktree` that it deliberately does **not** check locks: `IsWorktreeLocked` is consulted by callers before the subprocess runs, so the dry-run preview can mark locked entries without a failed git call.
- [x] 1.5 Confirm by inspection that `RemoveWorktree` issues no branch-affecting command — no `branch -d`, no `branch -D`. Removing a checkout must never discard the work on it.
- [x] 1.6 Extend `internal/git/worktree_test.go` with a fixture helper creating a repository plus a real worktree on a named branch, reusing the `initTestRepo` pattern already in that package.
- [x] 1.7 Write `TestRemoveWorktree`: remove a clean worktree and assert the directory is gone and `ListWorktrees` no longer reports it.
- [x] 1.8 Write `TestRemoveWorktree_DirtyRefused`: modify a tracked file in the worktree, assert removal fails with `force=false`, then assert it succeeds with `force=true`.
- [x] 1.9 Write `TestRemoveWorktree_UntrackedRefused`: add an untracked file and assert the same refusal, showing the guard is git's and broader than modified files.
- [x] 1.10 Write `TestRemoveWorktree_BranchSurvives`: after removing the worktree, assert `git branch --list <branch>` still reports the branch.
- [x] 1.11 Write `TestRemoveWorktree_ErrorKeepsGitReason`: assert the error from a refused removal still contains git's own wording rather than a replacement message.
- [x] 1.12 Run `gofmt -l`, `go vet ./internal/...`, and `golangci-lint run ./internal/...`; resolve findings rather than adding exclusions.

### [x] 2.0 Individual Removal: `omgw worktree remove <repo> <branch>`

The command itself in its single-repository forms, including the `rm` alias, the
current-directory form, branch matching, directory cleanup, and persistence.

#### 2.0 Proof Artifact(s)

- CLI: transcript creating a worktree, removing it, then `omgw worktree list` showing it gone and `git branch` showing the branch still present, demonstrates the feature end to end (FR U1-1, U1-9)
- Test: `go test -run TestRunWorktreeRemove_Individual -v ./cmd/omgitworks/` passes, asserting the worktree is gone from disk, from git, and from the saved configuration (FR U1-1, U1-11)
- Test: `go test -run TestRunWorktreeRemove_CurrentRepo -v ./cmd/omgitworks/` passes with one positional from inside the repository, demonstrates the spec 25 resolver form (FR U1-3)
- Test: `go test -run TestRunWorktreeRemove_ExactBranchMatch -v ./cmd/omgitworks/` passes, asserting `feat` does **not** match a worktree on `feat-auth`, demonstrates exact-not-partial matching (FR U1-4)
- Test: `go test -run TestRunWorktreeRemove_NoSuchBranch -v ./cmd/omgitworks/` passes, asserting a non-zero exit and an error naming both repository and branch (FR U1-5)
- Test: `go test -run TestRunWorktreeRemove_LockedSkipped -v ./cmd/omgitworks/` passes **with and without** `--force`, asserting the worktree survives and the lock reason is reported (FR U1-8)
- Test: `go test -run TestRunWorktreeRemove_CleansEmptyDirs -v ./cmd/omgitworks/` passes with two sub-cases: a nested branch (`hotfix/urgent`) leaves no empty `hotfix/` behind, and a repository directory still holding another worktree is left alone (FR U1-10)
- Test: `go test -run TestWorktreeRemoveCmd_Alias -v ./cmd/omgitworks/` passes, asserting `rm` resolves to the same command (FR U1-1)

#### 2.0 Tasks

- [x] 2.1 Create `cmd/omgitworks/worktree_remove.go` with `worktreeRemoveCmd`, `Use: "remove [repo] <branch>"`, `Aliases: []string{"rm"}`, `Args: cobra.RangeArgs(1, 2)`, registered in `init()` via `worktreeCmd.AddCommand`. No command sets `Aliases` today, so verify the alias renders in help — `main.go`'s usage template already has an Aliases block.
- [x] 2.2 Add `func findWorktreeForBranch(repo *config.Repository, branch string) (config.Worktree, bool)` matching `wt.Branch == branch` exactly. Deliberately not `filter.MatchesPattern`: a partial match here would delete the wrong worktree.
- [x] 2.3 Add `func removeWorktreeForRepo(repo *config.Repository, branch string, force bool) (removedPath string, err error)`: locate the worktree, check `git.IsWorktreeLocked` **first**, call `git.RemoveWorktree`, then refresh `repo.Worktrees` via `git.ListWorktrees`. Perform no `config.Save` and no printing, mirroring spec 26's `createWorktreeForRepo`.
- [x] 2.4 Introduce typed sentinels so callers classify outcomes without matching strings: `worktreeNotFoundError{Repo, Branch}` and `worktreeLockedError{Repo, Branch, Reason}`. The locked case must be returned **before** `git.RemoveWorktree` is called, and regardless of `force`.
- [x] 2.5 Add `func cleanupEmptyWorktreeDirs(repoName, removedPath string) error`: walk upward from the removed worktree's parent, removing directories only while they are empty, stopping at the repository's projects directory and never removing the projects root itself. Follow the conservative shape of `removeEmptyLegacyDir` in `worktree_align.go:282`.
- [x] 2.6 Ensure 2.5 handles slashed branch names: removing `hotfix/urgent` must leave no empty `hotfix/` directory behind, while a repository directory still holding another worktree is left untouched.
- [x] 2.7 Add `runWorktreeRemove(cfg, repos, branch string, opts removeOptions, stdout io.Writer, stdin io.Reader) error` as the single removal loop, taking injected I/O from the outset so task 4.0's confirmation path is testable. Follow `runNavigate` in `navigate.go:24`.
- [x] 2.8 Save configuration once after the loop, and only when at least one removal succeeded, matching spec 26's single-save rule.
- [x] 2.9 Register `--force` on the command, plumbed through `removeOptions` to `git.RemoveWorktree`. It must never be implied by `--yes` or by tag selection.
- [x] 2.10 Create `cmd/omgitworks/worktree_remove_test.go` with a fixture creating tracked repositories that each already have a worktree, reusing `setupTaggedFixture` from spec 26 where it fits.
- [x] 2.11 Write `TestRunWorktreeRemove_Individual`: assert the worktree is gone from disk, absent from `git.ListWorktrees`, and absent from the reloaded configuration.
- [x] 2.12 Write `TestRunWorktreeRemove_CurrentRepo`: chdir into the repository, pass one positional, and assert the correct worktree is removed via the spec 25 resolver.
- [x] 2.13 Write `TestRunWorktreeRemove_ExactBranchMatch`: with a worktree on `feat-auth`, assert branch `feat` removes nothing and errors.
- [x] 2.14 Write `TestRunWorktreeRemove_NoSuchBranch`: assert a non-zero result and an error naming both the repository and the branch.
- [x] 2.15 Write `TestRunWorktreeRemove_LockedSkipped` with two sub-cases, `force=false` and `force=true`, asserting in both that the worktree still exists and the lock reason is reported. Create the lock by writing git's `.git/worktrees/<name>/locked` file, which is what `IsWorktreeLocked` reads.
- [x] 2.16 Write `TestRunWorktreeRemove_BranchSurvives` at the command level, asserting `git branch --list` still reports the branch after removal.
- [x] 2.17 Write `TestRunWorktreeRemove_CleansEmptyDirs` with two sub-cases: a nested `hotfix/urgent` branch leaves no empty `hotfix/`, and a repository directory holding a second worktree is left in place.
- [x] 2.18 Write `TestWorktreeRemoveCmd_Alias`, asserting `rm` resolves to `worktreeRemoveCmd` through cobra's command lookup rather than by reading the `Aliases` field directly.

### [x] 3.0 Tag-Scoped Removal on the Shared Targeting Model

Extend removal across a tagged group, reusing spec 26's precedence table rather than
restating it, and adopt its partial-failure contract.

#### 3.0 Proof Artifact(s)

- CLI: transcript of `omgw worktree list`, `omgw worktree remove -t <tag> feat-x`, then `omgw worktree list` again, showing the branch's worktrees gone from tagged repositories only, demonstrates the feature end to end (FR U2-2)
- Test: `go test -run TestWorktreeRemoveBulk_RemovesAcrossTag -v ./cmd/omgitworks/` passes, asserting removal in every tagged repository and none outside the tag (FR U2-1, U2-2)
- Test: `go test -run TestWorktreeRemoveTargets_PrecedenceTable -v ./cmd/omgitworks/` passes with a sub-case per row, demonstrating `remove` and `add` share one table (FR U2-3)
- Test: `go test -run TestWorktreeRemoveBulk_SkipsMissingBranch -v ./cmd/omgitworks/` passes, asserting a tagged repository without that worktree is skipped and reported, not failed (FR U2-4)
- Test: `go test -run TestWorktreeRemoveBulk_SummaryAndExit -v ./cmd/omgitworks/` passes over a mixed run, asserting removed/skipped/failed counts and a non-zero return, with a sub-case asserting zero when only skips occurred (FR U2-5, U2-6)
- Test: `go test -run TestWorktreeRemoveBulk_NoRestore -v ./cmd/omgitworks/` passes, asserting removals before a later failure stay removed (FR U2-7)
- Test: `go test -run TestWorktreeRemoveBulk_SingleSave -v ./cmd/omgitworks/` passes, reloading configuration from disk and asserting every removal is reflected (FR U2-8)
- Test: `go test -run TestWorktreeAddSelection_PrecedenceTable -v ./cmd/omgitworks/` still passes unchanged after the selector moves, demonstrating the extraction was behavior-preserving

#### 3.0 Tasks

- [x] 3.1 Move `selectWorktreeAddTargets` from `worktree_add.go` to `worktree.go`, renaming it `selectWorktreeTargets`, and move `allRepositories`, `selectByName`, and `filterByTag` with it. Update `worktree_add.go`'s three call sites. Pure rename and relocation — no logic change.
- [x] 3.2 Run spec 26's `TestWorktreeAddSelection_*` suite unchanged and confirm it is green. Those tests are the guard that the extraction preserved behavior; if any assertion needs editing, the move was not behavior-preserving.
- [x] 3.3 Register `-t`/`--tag` on `worktreeRemoveCmd` as `StringArrayVarP`, reusing spec 26's `worktreeAddTag`-style single-value validation. Factor that helper into a shared `singleTagValue(flagValues []string) (string, error)` so `add` and `remove` share one rejection message.
- [x] 3.4 Wire `worktreeRemoveCmd`'s `RunE` through `selectWorktreeTargets`, so `remove` inherits all four precedence rows with no second copy of the table.
- [x] 3.5 In the removal loop, classify `worktreeNotFoundError` as a **skip** rather than a failure when the run targets more than one repository: a tagged repository that simply never had this branch is not an error. Keep it a hard error in the single-repository form, mirroring spec 26's `single` flag.
- [x] 3.6 Derive that `single` flag from the **invocation shape** (`tag == "" && len(args) < 2`), not from `len(repos)`. Spec 26's validation found the `len(repos) == 1` derivation turns a legitimate skip into a hard error when a tag matches exactly one repository; do not reintroduce it here.
- [x] 3.7 Collect per-repository failures into `errors []string` as `  %s: %v`, print the counted summary of removed/skipped/failed through `pluralize`, then the separately headed error block, matching `runWorktreeAlign`.
- [x] 3.8 Return spec 26's `errPartialFailure` sentinel when any removal failed, and set `SilenceErrors`/`SilenceUsage` on `worktreeRemoveCmd`, printing non-sentinel errors explicitly as `runWorktreeAddCommand` does. Without the explicit print, every ordinary error on this command becomes silent.
- [x] 3.9 Create `cmd/omgitworks/worktree_remove_bulk_test.go` with a fixture of three tagged repositories holding the same branch's worktree plus one untagged repository that also holds it.
- [x] 3.10 Write `TestWorktreeRemoveBulk_RemovesAcrossTag`, asserting removal in all three tagged repositories and that the untagged repository's worktree survives.
- [x] 3.11 Write `TestWorktreeRemoveTargets_PrecedenceTable` with a sub-case per row, asserting `remove` selects the same repository sets `add` does.
- [x] 3.12 Write `TestWorktreeRemoveBulk_SkipsMissingBranch`: give one tagged repository no worktree for the branch and assert it is announced as a skip, the run returns nil, and the others are removed.
- [x] 3.13 Write `TestWorktreeRemoveBulk_SingleMatchTagStillBulk`, the spec 26 regression: a tag matching exactly one repository that lacks the branch must skip and return nil, not error.
- [x] 3.14 Write `TestWorktreeRemoveBulk_SummaryAndExit` over a mixed run, asserting the three counts appear and the sentinel is returned, with a sub-case asserting nil when only skips occurred.
- [x] 3.15 Write `TestWorktreeRemoveBulk_NoRestore`: break a later repository, and assert earlier removals stay removed on disk and in the saved configuration.
- [x] 3.16 Write `TestWorktreeRemoveBulk_SingleSave`: reload configuration from disk and assert every removal is reflected, guarding the overwriting-save failure mode.

### [x] 4.0 Dry Run and the Confirmation Gate

Ensure nothing is deleted that the user has not seen, including when there is no terminal
to ask on.

#### 4.0 Proof Artifact(s)

- CLI: transcript of `--dry-run` followed by the identical real command, showing the preview list matched the outcome exactly, demonstrates preview fidelity (FR U3-1, U3-2)
- Test: `go test -run TestWorktreeRemove_DryRunChangesNothing -v ./cmd/omgitworks/` passes, asserting every worktree, directory, and configuration entry is byte-for-byte unchanged after a dry run (FR U3-1)
- Test: `go test -run TestWorktreeRemove_DryRunAnnotations -v ./cmd/omgitworks/` passes, asserting the preview marks a locked worktree as skipped and a dirty worktree as one that would fail without `--force`, and carries the `--force` posture annotation (FR U3-3)
- Test: `go test -run TestWorktreeRemove_ConfirmationGate -v ./cmd/omgitworks/` passes with an injected reader: a multi-worktree run prompts, and answering no removes nothing and exits zero (FR U3-4, U3-7)
- Test: `go test -run TestWorktreeRemove_YesSkipsPrompt -v ./cmd/omgitworks/` passes, asserting `--yes` proceeds without reading stdin (FR U3-5)
- Test: `go test -run TestWorktreeRemove_NonInteractiveRefuses -v ./cmd/omgitworks/` passes, asserting a non-TTY stdin without `--yes` exits non-zero, removes nothing, and names `--yes` in the error (FR U3-6)
- Test: `go test -run TestWorktreeRemove_SingleNotPrompted -v ./cmd/omgitworks/` passes, asserting a one-worktree run proceeds without a prompt, per the resolved open question
- Test: `go test -run TestWorktreeRemove_DryRunWithYes -v ./cmd/omgitworks/` passes, asserting the combination behaves as a dry run (FR U3-8)

#### 4.0 Tasks

- [x] 4.1 Define `type removalPlan struct { RepoName, Branch, Path string; Locked bool; LockReason string; Dirty bool }` and `func buildRemovalPlan(repos []*config.Repository, branch string, force bool) []removalPlan`, computing the full plan before anything is removed. One plan structure feeds the dry run, the confirmation prompt, and the real run, so the text a user approves is the text `--dry-run` shows.
- [x] 4.2 Detect the would-fail-dirty condition for the preview **without** removing anything, by checking whether the worktree has uncommitted or untracked changes. Reuse the existing status machinery in `internal/git` rather than adding a second cleanliness check, and mark the plan entry rather than failing.
- [x] 4.3 Add `func renderRemovalPlan(plans []removalPlan, force bool, w io.Writer)`: one block per worktree with repository name, branch, and path; annotations for `(locked: reason — will be skipped)` and `(has uncommitted changes — will fail without --force)`; a header line stating the `--force` posture; and a trailing count via `pluralize`. Follow `runWorktreeAlign`'s dry-run shape.
- [x] 4.4 Register `--dry-run` on the command. When set, render the plan and return **before** any removal, configuration save, or directory cleanup.
- [x] 4.5 Register `--yes`/`-y`. When a plan targets more than one worktree and `--yes` was not given, render the plan and prompt for confirmation before removing anything.
- [x] 4.6 Implement the prompt with `bufio.NewScanner(stdin)` following `navigate.go:210`, accepting `y`/`yes` case-insensitively; anything else declines. Declining returns nil — the user's answer is not an error.
- [x] 4.7 Gate the prompt on the **number of worktrees targeted**, not on `-t`. Per the spec's resolved question 1, a name pattern matching several repositories is gated exactly as a tag-scoped run is, while a single-worktree removal proceeds unprompted because git's own refusal already guards the destructive case.
- [x] 4.8 Add a stdin TTY check for the non-interactive contract: `term.IsTerminal(int(os.Stdin.Fd()))`, as `list.go:112` does for stdout. Expose it as an overridable package variable so tests can force either answer, following the `stdoutIsTerminalFunc` pattern in `cd.go:17`.
- [x] 4.9 When confirmation is required, `--yes` was absent, and stdin is not a terminal, return a non-zero error naming `--yes` and remove nothing. Do not read from the stream, and do not proceed unconfirmed.
- [x] 4.10 Make `--dry-run` take precedence over `--yes`, so the combination previews without removing.
- [x] 4.11 Create `cmd/omgitworks/worktree_remove_confirm_test.go` with fixtures for a multi-worktree plan, a locked worktree, and a dirty worktree.
- [x] 4.12 Write `TestWorktreeRemove_DryRunChangesNothing`: capture `config.json` bytes and the worktree directory listing before and after, asserting both are identical and the plan was printed.
- [x] 4.13 Write `TestWorktreeRemove_DryRunAnnotations`: assert the locked entry is marked as skipped, the dirty entry marked as would-fail-without-force, and the header states the force posture.
- [x] 4.14 Write `TestWorktreeRemove_ConfirmationGate`: feed `"n\n"` through the injected reader and assert nothing was removed and the result is nil; add a sub-case feeding `"y\n"` that proceeds.
- [x] 4.15 Write `TestWorktreeRemove_YesSkipsPrompt`: pass `--yes` with an **empty** reader and assert the removal proceeds, proving stdin was never consulted.
- [x] 4.16 Write `TestWorktreeRemove_NonInteractiveRefuses`: force the TTY check to false, omit `--yes`, and assert a non-zero error naming `--yes` with nothing removed.
- [x] 4.17 Write `TestWorktreeRemove_SingleNotPrompted`: a one-worktree plan with an empty reader must proceed without prompting.
- [x] 4.18 Write `TestWorktreeRemove_DryRunWithYes`, asserting the combination previews and removes nothing.
- [x] 4.19 Write `TestWorktreeRemove_PreviewMatchesRun`: render the plan, perform the real run, and assert every path listed in the preview is exactly the set that changed — the fidelity guarantee behind success metric 4.

### [x] 5.0 Completion, Documentation, and the Full Gate

> **Scope note:** as in spec 26, this spec states no functional requirement for completion
> or documentation. Included on the repository's convention that `docs/site/` moves with
> behavior; flagged in the planning audit. Drop it if you prefer a docs-only follow-up.

#### 5.0 Proof Artifact(s)

- CLI: `omgw __complete worktree remove ""` transcript showing removable branch names for the resolved repository, demonstrates completion is context-aware
- Test: `go test -run TestCompleteWorktreeRemove -v ./cmd/omgitworks/` passes, asserting only branches that actually have worktrees are suggested — completing to a branch with no worktree would only ever produce an error
- Documentation: diff of `docs/site/commands-core.md` adding a `remove` section covering the `rm` alias, `--force`, `--dry-run`, `--yes`, tag-scoped removal, the confirmation rule, the lock and branch guarantees, and the empty-directory cleanup
- CLI: `omgw worktree remove --help` transcript matching the documented flags and showing `rm` in the Aliases block
- CLI: `make ci` exits 0, demonstrates vet, lint, and the race-enabled suite pass on the complete change

#### 5.0 Tasks

- [x] 5.1 Add `func completeRemovableBranches(...)` in `worktree.go`, suggesting only branches that actually have worktrees in the targeted repository. Completing to a branch without a worktree could only ever produce an error.
- [x] 5.2 Register it as `worktreeRemoveCmd.ValidArgsFunction`, resolving the repository the same way `completeWorktreeAddArgs` does, and register `completeAllTags` for `--tag` via `RegisterFlagCompletionFunc`.
- [x] 5.3 Write `TestCompleteWorktreeRemove`, asserting only branches with worktrees are offered, that suggestions are prefix-filtered, and that the directive is `ShellCompDirectiveNoFileComp`.
- [x] 5.4 Add a `Remove Worktrees` section to `docs/site/commands-core.md` covering the signature, the `rm` alias, and tag-scoped removal.
- [x] 5.5 Document the safety model in that section: `--force` is required for uncommitted or untracked changes, locked worktrees are always skipped even with `--force`, branches are never deleted, and empty projects directories are cleaned up.
- [x] 5.6 Document `--dry-run`, `--yes`, the more-than-one-worktree confirmation rule, and the non-interactive requirement, with a worked dry-run example.
- [x] 5.7 Build and capture `omgw worktree remove --help`, checking it against the documentation and confirming the Aliases block shows `rm`.
- [x] 5.8 Run `make ci` and confirm exit 0, then capture the CLI transcripts named in the 1.0-4.0 proof artifacts against a scratch fixture workspace with isolated `XDG_CONFIG_HOME` and `XDG_DATA_HOME`. The spec's Security Considerations require transcripts not to reveal real workspace paths or unrelated repository names.

## Requirement Coverage Map

| Unit | Functional Requirement | Covered By |
| --- | --- | --- |
| 1 | `remove <repo> <branch>` with an `rm` alias | 2.0 |
| 1 | Git-level `git worktree remove <path>` operation | 1.0 |
| 1 | One positional removes from the resolved current repository | 2.0 |
| 1 | Branch matched exactly, not partially | 2.0 |
| 1 | Unmatched branch exits non-zero naming repo and branch | 2.0 |
| 1 | No `--force` by default; git's refusal surfaced clearly | 1.0 |
| 1 | `--force` passes through to git | 1.0 |
| 1 | Locked worktrees skipped and reported, even with `--force` | 2.0 |
| 1 | The checked-out branch is never touched | 1.0 |
| 1 | Empty projects directory removed, non-empty left alone | 2.0 |
| 1 | Worktree data updated and configuration saved | 2.0 |
| 2 | `-t`/`--tag` with `MatchesExact`, single value | 3.0 |
| 2 | `-t <tag> <branch>` removes across every tagged repository | 3.0 |
| 2 | Spec 26's precedence table applies unchanged | 3.0 |
| 2 | Tagged repo without the branch is skipped, not failed | 3.0 |
| 2 | Continue past failures; summary of removed/skipped/failed | 3.0 |
| 2 | Non-zero on failure, zero when only skips occurred | 3.0 |
| 2 | Removals before a later failure are not restored | 3.0 |
| 2 | Configuration saved once at the end | 3.0 |
| 3 | `--dry-run` previews and changes nothing | 4.0 |
| 3 | Preview lists repo, branch, path, and a count | 4.0 |
| 3 | Preview marks locked and would-fail-dirty entries | 4.0 |
| 3 | Multi-worktree runs prompt before removing | 4.0 |
| 3 | `--yes`/`-y` skips the prompt | 4.0 |
| 3 | Non-interactive stdin without `--yes` exits non-zero | 4.0 |
| 3 | Declining exits zero having removed nothing | 4.0 |
| 3 | `--dry-run` with `--yes` behaves as a dry run | 4.0 |
