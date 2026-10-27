# 29-tasks-user-repo-context.md

> Source spec: `docs/specs/29-spec-user-repo-context/29-spec-user-repo-context.md`
>
> Depends on: **spec 25** (`worktree-repo-context`) for `repocontext.ResolveCurrent` and `ErrNotTracked`. Base is `main` at `8639b74` (release 3.4.0), which already carries specs 25–28. Implementation should land on a `feat/user-repo-context` branch and open a PR against `main`.

## Planning Notes (Codebase Assessment)

These findings come from reading the code on `main` and refine what the spec assumed. They are recorded here so implementation does not re-derive them.

- **The two-argument form does not use `filter.MatchesPattern`.** `userAssignCmd` calls `findRepositories` (`tag.go:398`), which matches an exact path or a case-insensitive **substring** of the name. The spec calls this "name-pattern lookup" but requires the form to behave "exactly as it does today", so the substring matcher must stay. Do **not** swap in `selectByName` / `selectWorktreeTargets`: `MatchesPattern` is a wildcard matcher and would change which repositories a bare `my` matches. That reconciliation belongs to spec 30 alongside the bulk-selection change.
- **The multi-match error prints to stdout and then returns a generic error.** The current code prints `Multiple repositories match '<id>':` plus one line per match to stdout and returns `please specify a more specific repository identifier`. Success metric 2 requires identical output, so characterization tests must capture both the stdout listing and the error string before anything moves.
- **`assign` has no tests today.** `cmd/omgitworks/user_test.go` covers short-flag dispatch and `completeProfileNames` only. The first parent task therefore writes characterization tests against the existing behavior before touching it, so the "no working invocation changes" guarantee has evidence rather than assertion.
- **The whole command lives in one `RunE` closure.** To test it without cobra, extract a function (`runUserAssign`) that takes the positional arguments and the flag values, following `runWorktreeAddCommand` / `runWorktreeAddResolved` in `worktree_add.go`. Keep `assignUseSubdirs` and `assignDryRun` as the package-level flag variables they are; tests that set them must reset them in `t.Cleanup`.
- **The dry-run header already names the repository.** `Dry run - would assign profile '%s' to repository '%s':` prints `repo.Name`, so the one-argument form needs no new output for the spec's dry-run requirement. The test asserts the resolved name appears in that header.
- **"Resolver not consulted" has no mocking seam.** `repocontext` is called directly with no interface, exactly as in the worktree commands. Prove it the way `TestWorktreeAddSelection_ResolverNotConsulted` does: change directory into a repository that is *not* the one named (or into an untracked directory), run the two-argument form, and assert it acts on the named repository and never returns `ErrNotTracked`.
- **Inside-worktree fixture already exists.** `TestRunWorktreeAdd_ResolvesFromInsideWorktree` builds it: `setupWorktreeTestRepo(t, "my-repo")`, `runWorktreeAdd("my-repo", "feat-x")` to create a worktree and record it in config, `projectsPath(t, "my-repo", "feat-x")` for its path, then `chdirForTest`. Reuse that sequence rather than a new helper. Note a worktree's `.git` is a **file**, so `AssignLocal` run against the worktree path would fail; the resolver returning the owning repository is what makes the write land in `<repo>/.git/config`.
- **Help-text assertions elsewhere do not read `userCmd.Long`.** `deprecated_test.go:237` asserts `gws user list` in the output of the deprecated root flag path, whose strings live in `deprecated.go:113-117`. Rewriting the `user` parent `Long` must not touch those (spec non-goal 9). Verify after the rewrite that `deprecated_test.go` still passes unmodified.
- **Scope of the `gws` → `omgw` rewrite.** The spec limits the rewrite to the `assign` `Long` and the `user` parent `Long`. `user.go` also carries `gws` in the parent `RunE` error strings (`user -a requires a profile name: gws user -a ...`) and in the `list`, `add`, `show`, `remove`, and `sync` `Long` text. Those stay as they are; the success-metric test greps `Long` for the two commands this spec touches, not the whole file.
- **Completion model.** `completeWorktreeAddArgs` (`worktree.go:152`) is the arity switch to copy: with zero arguments, resolve and complete the context-dependent thing, else fall back to `completeRepoNames`; with one argument, complete the second. `completeProfileNames` and `completeRepoNames` both exist and take only `toComplete`.
- **`setupTestConfig` in `user_test.go` is a legacy helper** that writes under `$HOME/.gws` and returns a restore closure. New tests should use `withTempHome` / `setupWorktreeTestRepo`, which set the XDG variables that `config.Load` reads on this branch.
- **No shell wrapper change is needed.** `user` is already in every passthrough list in `shellinit.go` (lines 48, 128, 198), so the `omgw` function hands the whole `user` subtree to the binary. The spec's "shell-init output unchanged" artifact is satisfied by the existing `shellinit_test.go` suite passing with no edits.
- **Go `init()` order across files is by filename.** `user.go` sorts before `user_assign.go`, so `userCmd.AddCommand(userAssignCmd)` runs before the flags and `ValidArgsFunction` are attached in `user_assign.go`'s `init()`. Cobra does not care about that order, so splitting registration (in `user.go`) from flag setup (in `user_assign.go`) is safe, and matches how `worktree.go` and `worktree_add.go` divide the work.

## Relevant Files

| File | Why It Is Relevant |
| --- | --- |
| `cmd/omgitworks/user_assign.go` | New. Home of `userAssignCmd`, its flag variables, `runUserAssign`, `selectAssignTarget`, `lookupProfile`, and `completeUserAssignArgs`, extracted from `user.go`. |
| `cmd/omgitworks/user_assign_test.go` | New. Fixture helpers plus the characterization tests (task 1.0), the one-argument-form tests (task 2.0), and the help-text tests (task 4.0). |
| `cmd/omgitworks/user_assign_completion_test.go` | New. The four completion tests (task 3.0). |
| `cmd/omgitworks/user.go` | Loses the `assign` block; keeps `userCmd.AddCommand(userAssignCmd)` in its `init()`. The parent `Long` is rewritten to `omgw` and gains the one-argument `assign` example. |
| `cmd/omgitworks/user_test.go` | Existing `TestUserSubcommandsRegistered` must keep passing after the move. No edits planned. |
| `cmd/omgitworks/tag.go` | Provides `findRepositories` (substring matcher the two-argument form must keep) and `completeRepoNames`. Consumed unchanged. |
| `cmd/omgitworks/worktree.go` | Provides `completeWorktreeAddArgs`, the completion arity switch to model on. Read only. |
| `cmd/omgitworks/worktree_add_current_test.go` | Provides `chdirForTest` and the inside-worktree fixture sequence to reuse. Read only. |
| `cmd/omgitworks/worktree_add_test.go` | Provides `setupWorktreeTestRepo` and `projectsPath`. Read only. |
| `cmd/omgitworks/worktree_align_test.go` | Provides `captureStdoutStr`. Read only. |
| `internal/repocontext/repocontext.go` | Provides `ResolveCurrent` and `ErrNotTracked`. Consumed unchanged. |
| `internal/user/assign.go` | Provides `AssignLocal`, `PreviewAssignLocal`, `AssignWithSubdirs`. Consumed unchanged. |
| `internal/user/profile.go` | Provides `GetProfile`; `DetectProfiles` lives in `gitconfig.go`. Consumed unchanged. |
| `internal/config/config.go` | Defines `Repository.User/Email/SigningEnabled/UserSource` and `Profile`. Consumed unchanged. |
| `docs/site/commands-user.md` | Published reference; the Assign Profile section gains the one-argument form, worktree note, error, and `--use-subdirs` restriction. |
| `docs/site/commands-core.md` | Holds the `[repo]` usage convention and the `#current-repository-targeting` anchor the user docs link to. Read only. |
| `docs/specs/29-spec-user-repo-context/29-proofs/` | New directory for CLI transcripts and the docs diff named in the proof artifacts. |

### Notes

- Tests live beside the code in `cmd/omgitworks/`, named `*_test.go`, using `t.TempDir()` with real git repositories. Tests that change directory use `chdirForTest` and must not call `t.Parallel`.
- Reuse existing fixture helpers rather than writing new ones where possible: `setupWorktreeTestRepo`, `setupTwoWorktreeTestRepos`, `runWorktreeAdd`, `projectsPath`, `chdirForTest`, `captureStdoutStr`, `configBytes`, `withTempHome`.
- Profile fixtures use placeholder identities only (`Worker <work@example.com>`, `Person <personal@example.com>`). Transcripts in `29-proofs/` are produced under a temporary `HOME` so no real identity or path appears.
- Run `make ci` (vet + golangci-lint + `go test -race`) before considering any parent task complete.
- Commit messages follow Conventional Commits; the `.githooks/commit-msg` hook pads the type. Use `refactor(user):` for the extraction, `feat(user):` for the new capability and completion, and `docs(user):` for the site docs. A `feat:` commit triggers a minor version bump via Release Please.
- `goimports` local prefix is `github.com/daileyo/omgitworks`; group imports accordingly.

## Tasks

### [x] 1.0 Extract `assign` into its own file with a testable entry point and pin today's two-argument behavior

Move `userAssignCmd` and its flags from `user.go` into `cmd/omgitworks/user_assign.go`, keeping registration in `user.go`'s `init()`. Split the `RunE` closure into a function tests can call without cobra. Before any behavior changes, write characterization tests that lock the two-argument form's output, errors, and ordering, so success metric 2 ("every currently valid invocation produces identical output") is checked by tests rather than by inspection. This task changes no user-visible behavior.

#### 1.0 Proof Artifact(s)

- Test: `TestUserAssign_TwoArgs_AssignsNamedRepo` in `user_assign_test.go` passes, asserting `git config --local user.email` in the repository equals the profile's email and the stored `User`, `Email`, `SigningEnabled`, and `UserSource` fields are updated, demonstrating the existing happy path is covered before refactoring
- Test: `TestUserAssign_TwoArgs_MultiMatchError` passes, asserting the stdout listing `Multiple repositories match '<id>':` with one line per match and the error `please specify a more specific repository identifier`, demonstrating the ambiguity contract spec non-goal 2 preserves
- Test: `TestUserAssign_TwoArgs_UnknownProfileError` passes, asserting the error is the profile-lookup error and that `.git/config` is unchanged, demonstrating today's lookup order (repository first, then profile) is pinned
- Test: `TestUserAssign_TwoArgs_DryRunOutput` passes, asserting the `Dry run - would assign profile ... to repository '<name>':` header and the per-key change lines, and that `.git/config` and `config.json` are byte-identical before and after, demonstrating the two-argument dry-run output is pinned
- Test: `TestUserAssign_TwoArgs_UseSubdirsStillWorks` passes, piping `y` to the prompt and asserting the repository moved to `<workspace>/work/alpha`, the stored path and `UserSource` updated, and the sandboxed `$HOME/.gitconfig` gained an `includeIf` line, with a second sub-case piping `n` asserting nothing moved, demonstrating the prompt-driven `--use-subdirs` path survives the extraction (audit flag 1)
- CLI: `git diff --stat` of the move shows `user.go` shrinking and `user_assign.go` appearing, with `go build ./...` and `make ci` exiting zero, demonstrating the extraction compiles and passes vet, lint, and the race detector
- Test: the existing `TestUserSubcommandsRegistered` passes without modification, demonstrating `assign` is still registered under `user` after the move

#### 1.0 Tasks

- [x] 1.1 Create `cmd/omgitworks/user_assign.go` and move the `assignUseSubdirs` / `assignDryRun` variables and the whole `userAssignCmd` declaration into it, verbatim. Add an `init()` in the new file that registers the two flags (`--use-subdirs`, `--dry-run`); delete those two `Flags()` lines from `user.go`'s `init()` but leave `userCmd.AddCommand(userAssignCmd)` where it is.
- [x] 1.2 Define `type userAssignOptions struct { useSubdirs, dryRun bool }` and `func runUserAssign(args []string, opts userAssignOptions, in io.Reader) error` in `user_assign.go`. Move the `RunE` body into it unchanged, replacing reads of `assignUseSubdirs` / `assignDryRun` with `opts.useSubdirs` / `opts.dryRun`, and replacing `bufio.NewReader(os.Stdin)` in the `--use-subdirs` prompt with `bufio.NewReader(in)`, the same seam `runWorktreeRemove` uses. The `RunE` becomes a one-line call passing the package-level flag values and `os.Stdin`. Tests that never reach the prompt pass `strings.NewReader("")`.
- [x] 1.3 Extract the stored-then-auto-detected profile lookup block into `func lookupProfile(cfg *config.Config, name string) (*config.Profile, error)` in `user_assign.go` and call it from `runUserAssign`. Leave the duplicate in `userShowCmd` untouched; this spec does not revise `show`.
- [x] 1.4 Create `cmd/omgitworks/user_assign_test.go` with three helpers: `setupAssignFixture(t, repoNames ...string) map[string]string` (temp `HOME` and XDG via the same calls as `setupWorktreeTestRepo`, one initialized git repository per name with a `user.email` of `seed@example.com`, stored profiles `work` = `Worker <work@example.com>` and `personal` = `Person <personal@example.com>`, saved once); `localGitValue(t, repoDir, key string) string` running `git config --local <key>` in `repoDir`; and `gitConfigBytes(t, repoDir) []byte` reading `<repoDir>/.git/config`. Use `configBytes(t)` for `config.json` snapshots. *As built:* the fixture returns only the path map (the workspace is `filepath.Dir` of any path), and four further helpers were added: `assignOut` / `assignWithInput` wrap `runUserAssign` under `captureStdoutStr`, `loadRepo` reloads one repository by name, and `assertAssignedWork` holds the shared happy-path assertions used by tasks 1.5 and 2.6–2.8.
- [x] 1.5 Write `TestUserAssign_TwoArgs_AssignsNamedRepo`: fixture with `alpha`; run `runUserAssign([]string{"alpha", "work"}, userAssignOptions{}, strings.NewReader(""))` under `captureStdoutStr`; assert `localGitValue(..., "user.email") == "work@example.com"`, the reloaded repository's `User`, `Email`, `SigningEnabled`, `UserSource == config.UserSourceLocal`, and that stdout contains `Assigned profile 'work' to repository 'alpha'` followed by the `user.name:` and `user.email:` lines.
- [x] 1.6 Write `TestUserAssign_TwoArgs_MultiMatchError`: fixture with `api-core` and `api-web`; run with `[]string{"api", "work"}`; assert the error string equals `please specify a more specific repository identifier`, stdout contains `Multiple repositories match 'api':` and one `  - <name> (<path>)` line per repository, and neither `.git/config` changed.
- [x] 1.7 Write `TestUserAssign_TwoArgs_UnknownProfileError`: fixture with `alpha`; run with `[]string{"alpha", "nope"}`; assert the error is non-nil, does not satisfy `errors.Is(err, repocontext.ErrNotTracked)`, and `gitConfigBytes` is unchanged.
- [x] 1.8 Write `TestUserAssign_TwoArgs_DryRunOutput`: fixture with `alpha`; snapshot `gitConfigBytes` and `configBytes`; run with `dryRun: true`; assert stdout contains `Dry run - would assign profile 'work' to repository 'alpha':` and a `user.email: "seed@example.com" -> "work@example.com"` line, and that both snapshots are byte-identical afterwards.
- [x] 1.8a *Added by audit remediation (flag 1).* Write `TestUserAssign_TwoArgs_UseSubdirsStillWorks`: fixture with `alpha`; run `[]string{"alpha", "work"}` with `useSubdirs: true` and `strings.NewReader("y\n")` as `in`, under `captureStdoutStr`; assert the repository directory now exists at `filepath.Join(workspace, "work", "alpha")` and the old path does not, the reloaded repository's `Path` equals the new location and its `UserSource` is `config.UserSourceIncludeIf`, `$HOME/.gitconfig` contains an `[includeIf` line naming the `work` subdirectory, and stdout contains `Moved repository to:`. Also add a sub-case piping `n\n` that asserts `Canceled.` is printed and nothing moved. This pins the prompt-driven path before task 2.3 inserts the refusal ahead of it. *As built:* the sub-case is named `canceled`; golangci-lint's `misspell` linter enforces US spelling.
- [x] 1.9 Run `make ci`. Confirm `TestUserSubcommandsRegistered` and the rest of `user_test.go` pass unmodified. Capture `git diff --stat` for the proof artifact. Commit as `refactor(user): extract assign into its own file`. *As built:* golangci-lint was not on `PATH`; v2.13.2 (CI's pin) was installed to `~/go/bin` and `make ci` run with that on `PATH`.

### [x] 2.0 Add the one-argument form resolved from the working directory

Implement spec Unit 1. When the repository argument is absent, resolve it with `repocontext.ResolveCurrent(cfg)`; when present, keep the exact `findRepositories` path from task 1.0. Enforce the ordering the spec requires: the `--use-subdirs` refusal runs before `config.Load()`, the repository is resolved before the profile is looked up, and `--dry-run` names the resolved repository. Branch on "repository argument absent", not on `len(args) == 1`, so spec 30 can add `-t` without re-ordering this path.

#### 2.0 Proof Artifact(s)

- Test: `TestUserAssign_OneArg_FromRepoRoot` passes, asserting the profile lands in the repository's `.git/config` and stored fields, demonstrating the base case
- Test: `TestUserAssign_OneArg_FromSubdirectory` passes, running from a nested directory inside the repository, demonstrating reliance on git's upward walk through the shared resolver
- Test: `TestUserAssign_OneArg_FromInsideWorktree` passes, running from a worktree created by `runWorktreeAdd` and asserting `git config --local user.email` read in the **main checkout** equals the profile's email and the owning repository's stored fields are updated, demonstrating the worktree case from question 1 option A
- Test: `TestUserAssign_OneArg_OutputMatchesTwoArgForm` passes, asserting the captured stdout of `assign work` run inside `alpha` is byte-for-byte equal to the stdout of `assign alpha work`, demonstrating the silent-resolver requirement (no "inferred" line)
- Test: `TestUserAssign_TwoArgs_ResolverNotConsulted` passes, running the two-argument form from inside a *different* tracked repository and from an untracked directory, asserting the named repository is the one changed and `ErrNotTracked` is never returned, demonstrating explicit precedence
- Test: `TestUserAssign_OneArg_NotTracked` passes, asserting `errors.Is(err, repocontext.ErrNotTracked)` and that no `.git/config` or `config.json` changed, demonstrating the shared error contract and success metric 3
- Test: `TestUserAssign_OneArg_UnknownProfile_ReportsProfileError` passes, asserting the error is the profile-lookup error, not `ErrNotTracked`, demonstrating lookup order
- Test: `TestUserAssign_OneArg_UseSubdirsRefused` passes with two sub-cases (with and without `--dry-run`), asserting a non-nil error whose message says `--use-subdirs` requires an explicit repository argument, that no output mentioning "Would move" is printed, and that the repository directory, `.git/config`, and `config.json` are all unchanged, demonstrating question 6 option A and success metric 4
- Test: `TestUserAssign_OneArg_DryRunNamesResolvedRepo` passes, asserting the dry-run header contains the resolved repository's name and that `.git/config` and `config.json` are byte-identical before and after, demonstrating the preview
- Test: `TestUserAssign_ArgCount` passes, asserting zero and three positionals are rejected by `cobra.RangeArgs(1, 2)`, demonstrating the arity change
- CLI: transcript `29-proofs/assign-current-repo.md` of `cd <repo> && omgw user assign work` followed by `git config --local user.email`, demonstrating the feature end to end with placeholder profile values
- CLI: transcript `29-proofs/assign-from-worktree.md` of `omgw user assign personal` run inside a worktree, followed by `git config --local user.email` run in the main checkout, demonstrating the worktree-relative case
- CLI: `make ci` exits zero, demonstrating vet, lint, and race-detector tests pass

#### 2.0 Tasks

- [x] 2.1 Change `userAssignCmd.Args` from `cobra.ExactArgs(2)` to `cobra.RangeArgs(1, 2)`. (The `Use` line changes in task 4.0.)
- [x] 2.2 At the top of `runUserAssign`, derive the two inputs: `profileName := args[len(args)-1]`; `repoArg := ""`, set to `args[0]` only when `len(args) == 2`. Every later branch tests `repoArg == ""`, never `len(args)`.
- [x] 2.3 Immediately after 2.2 and **before** `config.Load()`, add the refusal: `if opts.useSubdirs && repoArg == "" { return fmt.Errorf("--use-subdirs requires an explicit repository argument: omgw user assign <repository> <profile> --use-subdirs") }`. Because it precedes the dry-run branch, `--dry-run --use-subdirs` with one argument is refused too.
- [x] 2.4 Add `func selectAssignTarget(cfg *config.Config, repoArg string) (*config.Repository, error)` in `user_assign.go`. When `repoArg == ""` return `repocontext.ResolveCurrent(cfg)` unchanged (no wrapping, so `errors.Is(err, repocontext.ErrNotTracked)` holds). Otherwise move the existing `findRepositories` block here verbatim, including the stdout listing and the `please specify a more specific repository identifier` error. Comment that spec 30 extends this function with the tag rows.
- [x] 2.5 Call `selectAssignTarget` before `lookupProfile` in `runUserAssign`, preserving today's order. Confirm no other code between them touches the filesystem.
- [x] 2.6 Write `TestUserAssign_OneArg_FromRepoRoot`: fixture `alpha`; `chdirForTest` into it; run `[]string{"work"}`; assert as in 1.5.
- [x] 2.7 Write `TestUserAssign_OneArg_FromSubdirectory`: same fixture; `os.MkdirAll(filepath.Join(repo, "pkg", "deep"))`; `chdirForTest` there; assert as in 1.5.
- [x] 2.8 Write `TestUserAssign_OneArg_FromInsideWorktree`: `setupWorktreeTestRepo(t, "my-repo")`, add the `work` and `personal` profiles to the saved config, `runWorktreeAdd("my-repo", "feat-x")`, `chdirForTest(t, projectsPath(t, "my-repo", "feat-x"))`; run `[]string{"personal"}`; assert `localGitValue(mainCheckout, "user.email") == "personal@example.com"`, the same value read from the worktree path (git shares local config), and the reloaded `Repositories[0]` fields.
- [x] 2.9 Write `TestUserAssign_OneArg_OutputMatchesTwoArgForm`: fixture `alpha`; capture stdout of `[]string{"alpha", "work"}`, reset the repository's `.git/config` and `config.json` to the snapshot, then `chdirForTest` into `alpha` and capture stdout of `[]string{"work"}`; assert the two strings are equal. *As built:* a `restoreFile` helper writes the two snapshots back between runs.
- [x] 2.10 Write `TestUserAssign_TwoArgs_ResolverNotConsulted` with two sub-tests: (a) fixture `alpha`, `beta`; `chdirForTest` into `beta`; run `[]string{"alpha", "work"}`; assert `alpha`'s email changed and `beta`'s did not; (b) `chdirForTest(t, t.TempDir())`; run the same; assert success and `alpha` changed.
- [x] 2.11 Write `TestUserAssign_OneArg_NotTracked`: fixture `alpha`; `chdirForTest(t, t.TempDir())`; snapshot; run `[]string{"work"}`; assert `errors.Is(err, repocontext.ErrNotTracked)` and both snapshots unchanged.
- [x] 2.12 Write `TestUserAssign_OneArg_UnknownProfile_ReportsProfileError`: fixture `alpha`; `chdirForTest` into it; run `[]string{"nope"}`; assert err non-nil and `!errors.Is(err, repocontext.ErrNotTracked)`.
- [x] 2.13 Write `TestUserAssign_OneArg_UseSubdirsRefused` with sub-tests `{dryRun: false}` and `{dryRun: true}`: fixture `alpha`; `chdirForTest` into it; snapshot `.git/config`, `config.json`, and `os.Stat(repoDir)`; run `[]string{"work"}` with `useSubdirs: true`; assert the error message contains `--use-subdirs requires an explicit repository argument`, captured stdout does not contain `Would move`, and all three snapshots are unchanged. *As built:* stronger than planned; stdout is asserted empty and `$HOME/.gitconfig` is asserted absent.
- [x] 2.14 Write `TestUserAssign_OneArg_DryRunNamesResolvedRepo`: fixture `alpha`; `chdirForTest` into it; snapshot; run `[]string{"work"}` with `dryRun: true`; assert stdout contains `to repository 'alpha':` and snapshots unchanged.
- [x] 2.15 Write `TestUserAssign_ArgCount`: call `userAssignCmd.Args(userAssignCmd, args)` for 0, 1, 2, and 3 arguments and assert error for 0 and 3, nil for 1 and 2.
- [x] 2.16 Produce the two transcripts. Under a temporary `HOME` (and `XDG_CONFIG_HOME` / `XDG_DATA_HOME` unset), `make build`, `omgw init` a scratch workspace, `omgw user add work --email work@example.com --name Worker` and the same for `personal`, then record the commands and output for `29-proofs/assign-current-repo.md` (`cd <repo> && omgw user assign work && git config --local user.email`) and `29-proofs/assign-from-worktree.md` (`omgw worktree add feat-x`, `cd` into it, `omgw user assign personal`, then `git -C <main checkout> config --local user.email`). Scrub the temporary paths to `<tmp>`. *As built:* the worktree transcript also records the failure and preview cases (resolver error, both `--use-subdirs` refusals, one-argument dry run), and notes that cobra prints usage after runtime errors because `assign` never set `SilenceUsage`; left unchanged as pre-existing behavior.
- [x] 2.17 Run `make ci`. Commit as `feat(user): resolve the repository for assign from the current directory`.

### [ ] 3.0 Add context-aware tab completion for both argument positions

Implement the completion requirements of spec Unit 2. Attach a `completeUserAssignArgs` function modelled on `completeWorktreeAddArgs`: with no arguments typed, complete profile names when the working directory resolves and repository names otherwise; with one argument typed, complete profile names; otherwise nothing.

#### 3.0 Proof Artifact(s)

- Test: `TestCompleteUserAssign_ProfilesWhenResolved` passes, calling the completion function with `args` empty from inside a tracked repository and asserting only profile names are returned, demonstrating context-aware completion
- Test: `TestCompleteUserAssign_ReposWhenNotResolved` passes, calling it from an untracked temporary directory and asserting only repository names are returned, demonstrating the fallback
- Test: `TestCompleteUserAssign_ProfilesForSecondArg` passes, calling it with one argument already present and asserting profile names are returned, demonstrating the new two-argument completion
- Test: `TestCompleteUserAssign_NothingAfterTwoArgs` passes, asserting `ShellCompDirectiveNoFileComp` with no candidates for a third position, demonstrating completion stops at the arity limit
- CLI: `omgw __complete user assign ""` run inside a tracked repository lists profile names, and the same command run from an untracked directory lists repository names, captured in `29-proofs/completion.md`, demonstrating the behavior through cobra's completion entry point

#### 3.0 Tasks

- [ ] 3.1 Add `func completeUserAssignArgs(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective)` to `user_assign.go`. Load config; on error return `nil, cobra.ShellCompDirectiveNoFileComp`. `switch len(args)`: `case 0`: if `repocontext.ResolveCurrent(cfg)` succeeds return `completeProfileNames(toComplete)`, else `completeRepoNames(toComplete)`; `case 1`: return `completeProfileNames(toComplete)`; `default`: `nil, cobra.ShellCompDirectiveNoFileComp`.
- [ ] 3.2 In `user_assign.go`'s `init()`, set `userAssignCmd.ValidArgsFunction = completeUserAssignArgs`.
- [ ] 3.3 Create `cmd/omgitworks/user_assign_completion_test.go`. Reuse `setupAssignFixture(t, "alpha", "beta")`, whose profile names (`work`, `personal`) and repository names do not overlap, so each assertion can check the returned set exactly.
- [ ] 3.4 Write `TestCompleteUserAssign_ProfilesWhenResolved`: `chdirForTest` into `alpha`; call with `nil` args and `""`; assert the result equals `{work, personal}` in any order and contains neither repository name.
- [ ] 3.5 Write `TestCompleteUserAssign_ReposWhenNotResolved`: `chdirForTest(t, t.TempDir())`; same call; assert the result equals `{alpha, beta}` and contains neither profile name.
- [ ] 3.6 Write `TestCompleteUserAssign_ProfilesForSecondArg`: from the untracked directory, call with `[]string{"alpha"}`; assert `{work, personal}`.
- [ ] 3.7 Write `TestCompleteUserAssign_NothingAfterTwoArgs`: call with `[]string{"alpha", "work"}`; assert empty result and `cobra.ShellCompDirectiveNoFileComp`.
- [ ] 3.8 Capture `29-proofs/completion.md` using the scratch workspace from 2.16: `omgw __complete user assign ""` from inside a repository and from an untracked directory, plus `omgw __complete user assign alpha ""`.
- [ ] 3.9 Run `make ci`. Commit as `feat(user): complete assign arguments by context`.

### [ ] 4.0 Rewrite the `assign` and `user` help text and update the published docs

Implement the help and documentation requirements of spec Unit 2. Change `Use` to `assign [repository] <profile>`; rewrite the `assign` `Long` to document both forms, the detection rule, the `--use-subdirs` restriction, and `omgw` examples; rewrite the `user` parent `Long` to `omgw` and add the one-argument `assign` example. Update `docs/site/commands-user.md`'s Assign Profile section with the new usage line, both forms, worktree-relative resolution, the not-tracked error, and the `--use-subdirs` restriction, following the `[repo]` convention `commands-core.md` uses for `worktree add`.

#### 4.0 Proof Artifact(s)

- Test: `TestUserAssign_HelpTextUsesOmgw` passes, asserting `userAssignCmd.Long` and `userCmd.Long` contain no `gws ` and that `userAssignCmd.Use` equals `assign [repository] <profile>`, demonstrating question 9 option A and success metric 5
- Test: `TestUserAssign_HelpDocumentsBothForms` passes, asserting the `assign` `Long` contains a one-argument example and the phrase describing detection from the current directory, and that `userCmd.Long` contains the one-argument `assign` example, demonstrating discoverability
- Test: the existing `shellinit_test.go` and `shellinit_exec_test.go` suites pass with no edits, demonstrating no shell function regression
- Test: the existing `deprecated_test.go` passes with no edits, demonstrating the deprecated root flag messages (non-goal 9) were not touched by the `Long` rewrite
- CLI: `omgw user assign --help` output captured in `29-proofs/assign-help.md`, showing the `[repository] <profile>` usage and `omgw` examples, demonstrating discoverability
- Documentation: `git diff docs/site/commands-user.md` captured in `29-proofs/docs-diff.md`, showing the updated usage block, both forms, the worktree note, the error, and the `--use-subdirs` restriction, demonstrating published documentation matches behavior
- CLI: `grep -n 'omgw user assign \[repository\] <profile>' docs/site/commands-user.md` returns the usage line, demonstrating the published usage matches the `Use` string
- CLI: `make ci` exits zero, demonstrating the full gate passes on the finished feature

#### 4.0 Tasks

- [ ] 4.1 Set `userAssignCmd.Use = "assign [repository] <profile>"`.
- [ ] 4.2 Rewrite `userAssignCmd.Long`, modelled on `worktreeAddCmd.Long`: one paragraph on what is written; one on omitting the repository ("determined from the current directory, including from inside one of its worktrees, where the owning repository is used; naming a repository explicitly always takes precedence"); the options block with `--use-subdirs` annotated "requires an explicit repository argument"; and an `Examples:` block whose every line starts with `omgw`, including `omgw user assign work` commented `# Repo from the current directory` and the three existing two-argument examples.
- [ ] 4.3 Rewrite `userCmd.Long`: replace every `gws` with `omgw` in the `Commands:` and `Short flags:` blocks, and add `omgw user assign work` with the comment `# Assign profile to the current repo` directly after the existing `assign my-repo work` line. Do not touch the `RunE` error strings or any other subcommand's `Long`.
- [ ] 4.4 Write `TestUserAssign_HelpTextUsesOmgw` in `user_assign_test.go`: assert `!strings.Contains(userAssignCmd.Long, "gws ")`, `!strings.Contains(userCmd.Long, "gws ")`, and `userAssignCmd.Use == "assign [repository] <profile>"`.
- [ ] 4.5 Write `TestUserAssign_HelpDocumentsBothForms`: assert `userAssignCmd.Long` contains `omgw user assign work` and `current directory`, and `userCmd.Long` contains `omgw user assign work`.
- [ ] 4.6 Update `docs/site/commands-user.md` Assign Profile section: usage block to `omgw user assign [repository] <profile> [flags]`; a paragraph after the description stating the repository may be omitted when standing inside it or one of its worktrees, that the owning repository receives the identity, that an explicit name always wins, and linking to `commands-core.md#current-repository-targeting`; a sentence giving the not-tracked error and its two remedies; the `--use-subdirs` flag row amended with "requires an explicit repository argument"; and an example block adding `cd ~/code/my-api` + `omgw user assign work` and a worktree example.
- [ ] 4.7 Run `go test ./cmd/omgitworks -run 'TestShell|TestDeprecated' -count=1` and confirm both suites pass with no edits, then `make ci`.
- [ ] 4.8 Capture `29-proofs/assign-help.md` (`omgw user assign --help`) and `29-proofs/docs-diff.md` (`git diff docs/site/commands-user.md`), and record the `grep` from the proof artifacts.
- [ ] 4.9 Commit the Go changes as `feat(user): document assign's current-repository form in help` and the site docs as `docs(user): document assign from the current repository`.
