# 25-tasks-worktree-repo-context.md

> Source spec: `25-spec-worktree-repo-context.md` in this directory.
> Clarification record: `25-questions-1-worktree-repo-context.md`, `25-questions-2-worktree-repo-context.md`.
> Downstream: specs 26, 27, and 28 consume the resolver built in task 1.0.

## Relevant Files

| File | Why It Is Relevant |
| --- | --- |
| `internal/repocontext/repocontext.go` | **New.** Home of the shared current-repository resolver and its error contract. Kept separate from `internal/git` (no config knowledge) and `internal/config` (no subprocess calls). |
| `internal/repocontext/repocontext_test.go` | **New.** Unit tests for the resolver: root, subdirectory, inside-worktree, symlink, nested-repo, both failure paths, and the read-only guarantee. |
| `internal/git/toplevel.go` | **New.** Exported `Toplevel` and `ListBranches` helpers. Needed because `gitCommand` is unexported, and the spec forbids calling `os/exec` directly from the resolver. |
| `internal/git/toplevel_test.go` | **New.** Unit tests for `Toplevel` and `ListBranches` against real temp-dir git repositories. |
| `internal/git/worktree.go` | Holds `resolvePath` (line 265), the resolve-then-clean helper the spec mandates reusing. Exported here so `repocontext` can call it. |
| `internal/git/worktree_test.go` | Existing worktree tests; must stay green across the `resolvePath` rename. |
| `cmd/omgitworks/worktree.go` | Parent command. Gains the shared `worktreeScope` targeting type and the new completion functions; its `Long` help text lists the subcommand forms that change. |
| `cmd/omgitworks/worktree_add.go` | `Args` widens from `ExactArgs(2)` to `RangeArgs(1, 2)`; `runWorktreeAdd` splits so resolution and explicit naming share one creation path. |
| `cmd/omgitworks/worktree_add_test.go` | Existing add tests (five cases) plus new resolution and explicit-precedence cases. Already provides `setupWorktreeTestRepo`. |
| `cmd/omgitworks/worktree_list.go` | Accepts `.` as the repository argument; `runWorktreeList` takes a `worktreeScope` instead of a raw filter string. |
| `cmd/omgitworks/worktree_list_test.go` | Five existing call sites migrate to the new signature with assertions unchanged; new `.` scoping cases added. |
| `cmd/omgitworks/worktree_align.go` | Same `.` handling as `list`; `runWorktreeAlign` takes a `worktreeScope`. `--dry-run` behavior is untouched. |
| `cmd/omgitworks/worktree_align_test.go` | Eight existing call sites migrate to the new signature; provides the reusable `captureStdoutStr` helper. New `.` scoping cases added. |
| `cmd/omgitworks/worktree_completion_test.go` | **New.** Tests for branch-vs-repo completion on `add` and the conditional `.` suggestion on `list` and `align`. |
| `cmd/omgitworks/shellinit.go` | Must remain byte-identical. Listed as a regression guard, not a change target. |
| `cmd/omgitworks/shellinit_test.go` | Seven existing `TestShellTemplates*` cases serve as the guard that no shell-function behavior shifted. |
| `docs/site/commands-core.md` | Published command reference. The `worktree add`, `list`, and `align` sections document the new invocation forms and the resolution-failure error. |

### Notes

- Tests live alongside the code they cover, per the existing layout.
- Run tests with `go test ./...`, a single package with `go test -v ./internal/repocontext/`, and the full gate with `make ci` (`go vet` + `golangci-lint` + `go test -race`).
- `golangci-lint` runs with `gosec`, `noctx`, `unparam`, and `misspell` enabled; `goimports` uses local prefix `github.com/daileyo/omgitworks`, so the new package's imports group accordingly. Test files are exempt from `gosec`, `noctx`, and `unparam`.
- CI builds against Go 1.24.x and 1.25.x and verifies the binary on Linux, macOS, and Windows, so nothing may depend on POSIX-only path behavior.
- Resolver tests change the working directory. Follow the `os.Chdir` + `t.Cleanup(func() { _ = os.Chdir(orig) })` pattern already used in `cmd/omgitworks/init_test.go:72`, and do not mark those tests `t.Parallel()`.
- Test fixtures that touch config must set `HOME` and clear `xdg.EnvConfigHome` / `xdg.EnvDataHome`, as `setupWorktreeTestRepo` already does.
- Commit as `feat(worktree): ...` (a `feat` triggers a minor bump via Release Please). Note the `commit-msg` hook only pads unscoped `type:` tokens, so a scoped subject is left unpadded — matching `feat(worktree): relocate worktrees to the projects root` in existing history.

## Tasks

### [x] 1.0 Current-Repository Resolver (`internal/repocontext`)

Build the shared resolver that every worktree subcommand calls, including the exported
git helper it needs. This task delivers no user-visible command change; it is proven by
tests and is the dependency for tasks 2.0-4.0 and for specs 26-28.

#### 1.0 Proof Artifact(s)

- Test: `go test -run TestResolve -v ./internal/repocontext/` passes, with a named case showing the resolver returns the owning repository when the working directory is the repository root, demonstrates the base case (FR U1-1, U1-2, U1-3)
- Test: `go test -run TestResolve_Subdirectory -v ./internal/repocontext/` passes from a directory several levels deep, demonstrates reliance on git's own upward walk rather than manual path splitting (FR U1-2)
- Test: `go test -run TestResolve_InsideWorktree -v ./internal/repocontext/` passes and asserts the returned repository is the owner, not the worktree, demonstrates the `Worktrees[].Path` fallback (FR U1-3, U1-4)
- Test: `go test -run TestResolve_SymlinkedPath -v ./internal/repocontext/` passes with a symlinked repository path in config, demonstrates `EvalSymlinks` applied to both sides (FR U1-5)
- Test: `go test -run TestResolve_NestedRepoNotPrefixMatched -v ./internal/repocontext/` passes, demonstrates exact comparison rather than prefix matching (FR U1-6)
- Test: `go test -run TestResolve_NotTracked -v ./internal/repocontext/` passes asserting the error text names both remedies (supply a repository argument, or run `omgw add`), demonstrates the error contract (FR U1-9)
- Test: `go test -run TestResolve_NotAGitRepo -v ./internal/repocontext/` passes asserting the same error is returned rather than git's raw message, demonstrates failure normalization (FR U1-10)
- Test: `go test -run TestResolve_DoesNotMutateConfig -v ./internal/repocontext/` passes comparing `config.json` bytes before and after a failed and a successful resolution, demonstrates the read-only guarantee (FR U1-11)
- Test: `go test -run TestResolve_WindowsDriveLetterCase -v ./internal/repocontext/` passes on Windows and reports `SKIP` elsewhere, demonstrates that comparison tolerates drive-letter case on a case-insensitive filesystem (FR U1-6)
- CLI: `go vet ./internal/repocontext/ && golangci-lint run ./internal/repocontext/...` exits 0, demonstrates the new package meets the repository lint gate

#### 1.0 Tasks

- [x] 1.1 Create `internal/git/toplevel.go` with `func Toplevel(dir string) (string, error)`, implemented as `gitCommand(dir, "rev-parse", "--show-toplevel")`. Document that it returns the enclosing checkout root and that the error is git's, to be normalized by callers.
- [x] 1.2 Rename `resolvePath` to exported `ResolvePath` in `internal/git/worktree.go:265`, update both call sites in `IsAligned` (lines 256-257), and keep the existing doc comment explaining the symlink-then-clean fallback.
- [x] 1.3 Create `internal/git/toplevel_test.go` covering `Toplevel` from a repository root, from a nested subdirectory (asserting the same root is returned), and from a non-git temp directory (asserting an error).
- [x] 1.4 Create `internal/repocontext/repocontext.go` with package doc explaining that it resolves the tracked repository owning a directory, and define the exported error value `ErrNotTracked` whose message states the problem and then both remedies, following the two-part style of the cross-device error in `internal/git/worktree.go:166`: the current directory is not inside a tracked repository; supply a repository argument, or run `omgw add` to track it.
- [x] 1.5 Implement `func Resolve(cfg *config.Config, dir string) (*config.Repository, error)`: call `git.Toplevel(dir)`, and on any error return `ErrNotTracked` without surfacing git's message; otherwise `git.ResolvePath` the returned root.
- [x] 1.6 In `Resolve`, compare the resolved root against `git.ResolvePath(repo.Path)` for each `cfg.Repositories` entry using exact equality, never `strings.HasPrefix`. Add a small `samePath(a, b string) bool` helper that uses `strings.EqualFold` when `runtime.GOOS == "windows"` and `==` everywhere else, so a drive-letter case difference between git's output and the stored path does not defeat the match on a case-insensitive filesystem. Return `&cfg.Repositories[i]` so callers can mutate and save the repository, as `runWorktreeAdd` does.
- [x] 1.7 In `Resolve`, when no repository path matches, make a second pass over every `repo.Worktrees[].Path`, comparing through the same `samePath` helper, and return the **owning repository** for a hit. Return `ErrNotTracked` when neither pass matches.
- [x] 1.8 Add `func ResolveCurrent(cfg *config.Config) (*config.Repository, error)` that calls `os.Getwd` and delegates to `Resolve`, returning `ErrNotTracked` if `os.Getwd` fails. Confirm by inspection that neither function calls `config.Save`, `os.MkdirAll`, or reads stdin.
- [x] 1.9 Create `internal/repocontext/repocontext_test.go` with a helper that builds a temp workspace, runs `git init` plus an empty initial commit (mirroring `setupWorktreeTestRepo` in `cmd/omgitworks/worktree_add_test.go:13`), and returns an in-memory `*config.Config`. Resolve symlinks on the temp dir so macOS `/var` → `/private/var` does not defeat comparisons.
- [x] 1.10 Write `TestResolve` and `TestResolve_Subdirectory`: resolve from the repository root and from a directory nested at least three levels deep, asserting the same repository is returned in both.
- [x] 1.11 Write `TestResolve_InsideWorktree`: create a real worktree via `git worktree add`, record it in `Worktrees[].Path`, chdir into it, and assert the returned repository is the owner, with `Path` equal to the main checkout and not the worktree path.
- [x] 1.12 Write `TestResolve_SymlinkedPath`: store a path that traverses a symlink in `cfg.Repositories[0].Path` while the working directory is the real path, and assert resolution still matches.
- [x] 1.13 Write `TestResolve_NestedRepoNotPrefixMatched`: track an outer repository and `git init` an inner repository inside its tree that is **not** tracked, chdir into the inner one, and assert `ErrNotTracked` rather than a match on the outer repository.
- [x] 1.14 Write `TestResolve_NotTracked` and `TestResolve_NotAGitRepo`: assert `errors.Is(err, ErrNotTracked)` in both, and assert the message contains both remedy phrases and does not contain `rev-parse`.
- [x] 1.15 Write `TestResolve_DoesNotMutateConfig`: `config.Save` a fixture, read `config.json` bytes, run one successful and one failing resolution, re-read, and assert the bytes are identical.
- [x] 1.16 Write `TestResolve_WindowsDriveLetterCase`, guarded by `if runtime.GOOS != "windows" { t.Skip("windows-only path casing") }`: store the repository path with the drive letter lower-cased while git reports it upper-cased (or the reverse), and assert resolution still returns the repository. This pins the `samePath` rule from sub-task 1.6; CI's test job runs on ubuntu only, so the case is otherwise unexercised.
- [x] 1.17 Run `gofmt -l`, `go vet ./internal/...`, and `golangci-lint run ./internal/...`; fix any finding before moving on.

### [ ] 2.0 `omgw worktree add <branch>` Without a Repository Argument

Wire the resolver into `worktree add`, widening its arity so one positional means branch
and two continue to mean repository then branch.

#### 2.0 Proof Artifact(s)

- CLI: transcript of `cd <repo> && omgw worktree add feat-x` followed by `omgw worktree list <repo>` showing the new worktree row, demonstrates the feature end to end (FR U1-7)
- CLI: transcript of `cd <repo-worktree-dir> && omgw worktree add feat-y` followed by `omgw worktree list <repo>`, showing `feat-y` created under the owning repository's projects root, demonstrates the worktree-relative case (FR U1-7, U1-4)
- CLI: transcript of `omgw worktree add feat-z` run from `$(mktemp -d)`, showing non-zero exit (`echo $?` prints non-zero) and the two-remedy error, demonstrates the failure contract at the command layer (FR U1-9, U1-10)
- Test: `go test -run TestRunWorktreeAdd_ResolvesCurrentRepo -v ./cmd/omgitworks/` passes, demonstrates one-positional resolution (FR U1-7)
- Test: `go test -run TestRunWorktreeAdd_ExplicitRepoIgnoresCwd -v ./cmd/omgitworks/` passes, with the working directory set inside repository A while the command names repository B and the worktree lands in B, demonstrates explicit precedence (FR U1-8)
- Test: `go test -run TestRunWorktreeAdd -v ./cmd/omgitworks/` shows all five pre-existing `TestRunWorktreeAdd_*` cases still passing unchanged, demonstrates no regression in the two-argument form (FR U1-8)
- CLI: `omgw worktree add` with zero arguments and with three arguments each exit non-zero with cobra's arity error, demonstrates `RangeArgs(1, 2)` bounds

#### 2.0 Tasks

- [ ] 2.1 In `cmd/omgitworks/worktree_add.go`, split `runWorktreeAdd` by extracting everything from the duplicate-branch check onward into `func addWorktreeForRepo(cfg *config.Config, repo *config.Repository, branch string) error`, leaving the existing behavior byte-for-byte identical. Verify the five existing `TestRunWorktreeAdd_*` cases still pass before continuing.
- [ ] 2.2 Reduce `runWorktreeAdd(repoName, branch string)` to config load, the existing name-pattern lookup with its "multiple repositories match" and "no repository found" errors, then a call to `addWorktreeForRepo`.
- [ ] 2.3 Add `func runWorktreeAddCurrent(branch string) error`: load config, call `repocontext.ResolveCurrent`, return the resolver's error unwrapped on failure, and otherwise call `addWorktreeForRepo`.
- [ ] 2.4 Change `Args: cobra.ExactArgs(2)` to `cobra.RangeArgs(1, 2)` and dispatch in `RunE`: one argument calls `runWorktreeAddCurrent(args[0])`, two call `runWorktreeAdd(args[0], args[1])`. The resolver must not be invoked in the two-argument path.
- [ ] 2.5 Update the command's `Use` to `add [repo] <branch>` and extend `Long` with the omitted-repository form, a note that it works from inside a worktree of the repository, and a matching example. Keep the existing `gws` prefix used by the surrounding examples: rebranding help text is a separate change across all 18 command files, not part of this spec.
- [ ] 2.6 Update the `Subcommands:` block in `worktreeCmd.Long` (`cmd/omgitworks/worktree.go:28-31`) so the `add` line reads `add [repo] <branch>`.
- [ ] 2.7 Add `TestRunWorktreeAdd_ResolvesCurrentRepo` to `worktree_add_test.go`: chdir into the repository created by `setupWorktreeTestRepo`, call `runWorktreeAddCurrent("feat-x")`, and assert the worktree exists at the projects path and is recorded in the reloaded config.
- [ ] 2.8 Add `TestRunWorktreeAdd_ResolvesFromInsideWorktree`: create a first worktree, chdir into it, add a second branch via `runWorktreeAddCurrent`, and assert the second worktree is created under the owning repository's projects directory.
- [ ] 2.9 Add `TestRunWorktreeAdd_ExplicitRepoIgnoresCwd`: build a fixture with two tracked repositories, chdir into repository A, call `runWorktreeAdd("repo-b", "feat-x")`, and assert the worktree lands in repository B's projects directory and repository A's `Worktrees` is untouched.
- [ ] 2.10 Add `TestRunWorktreeAddCurrent_NotTracked`: chdir to a bare `t.TempDir()`, call `runWorktreeAddCurrent`, and assert the error matches `repocontext.ErrNotTracked` and that no directory was created under the projects root.
- [ ] 2.11 Add an arity test asserting `worktreeAddCmd.Args` rejects zero and three arguments and accepts one and two, calling the `Args` function directly rather than executing the command.

### [ ] 3.0 `.` Targeting for `omgw worktree list` and `omgw worktree align`

Accept a literal `.` as the repository argument on the two subcommands whose argument is
already optional, without changing what omission means on either.

#### 3.0 Proof Artifact(s)

- CLI: side-by-side transcript of `omgw worktree list` and `omgw worktree list .` run from inside a repository in a workspace with at least two repositories that both have worktrees, showing the first lists both repositories and the second lists only one, demonstrates the difference in scope (FR U2-1, U2-2, U2-4)
- CLI: transcript of `omgw worktree align . --dry-run` from inside a repository, showing planned moves only for that repository, demonstrates scoped targeting on the mutating command with the existing flag honored (FR U2-5)
- Test: `go test -run TestRunWorktreeList_NoArgListsAllRepos -v ./cmd/omgitworks/` passes, demonstrates preserved bare-argument behavior (FR U2-2)
- Test: `go test -run TestRunWorktreeList_DotScopesToCurrentRepo -v ./cmd/omgitworks/` passes, demonstrates scoped targeting (FR U2-4)
- Test: `go test -run TestRunWorktreeAlign_NoArgProcessesAllRepos -v ./cmd/omgitworks/` passes, demonstrates preserved bare-argument behavior (FR U2-3)
- Test: `go test -run TestRunWorktreeAlign_DotScopesToCurrentRepo -v ./cmd/omgitworks/` passes and asserts an out-of-scope repository's unaligned worktree is not planned, demonstrates scoped targeting (FR U2-5)
- Test: `go test -run TestRunWorktreeList_DotNotTreatedAsNamePattern -v ./cmd/omgitworks/` passes with a tracked repository named `my.repo` present, asserting `.` resolves by working directory and never reaches `filter.MatchesPattern`, demonstrates that `.` is intercepted before pattern matching (FR U2-7)
- Test: `go test -run TestWorktreeDotResolutionFailure -v ./cmd/omgitworks/` passes asserting both `list .` and `align .` return the task 1.0 error verbatim, demonstrates a consistent error contract across subcommands (FR U2-6)

#### 3.0 Tasks

- [ ] 3.1 In `cmd/omgitworks/worktree.go`, add `type worktreeScope struct { NamePattern string; RepoPath string; Label string }`, where an empty struct means all repositories, `NamePattern` is the legacy pattern path, and a non-empty `RepoPath` is an already-resolved exact repository path. Document that `RepoPath` and `NamePattern` are never both set.
- [ ] 3.2 Add `func (s worktreeScope) matches(repo *config.Repository) bool`: return `true` when the scope is empty; when `RepoPath` is set compare `git.ResolvePath(repo.Path) == s.RepoPath` exactly; otherwise fall through to `filter.MatchesPattern(repo.Name, s.NamePattern)`.
- [ ] 3.3 Add `func worktreeScopeFor(cfg *config.Config, arg string) (worktreeScope, error)`: return the empty scope for `""`; for the literal `"."` call `repocontext.ResolveCurrent` and build a `RepoPath` scope with `Label` set to the resolved repository name, returning the resolver error unchanged on failure; for anything else return a `NamePattern` scope with `Label` set to the argument. The `.` branch must return before any call to `filter.MatchesPattern`.
- [ ] 3.4 Change `runWorktreeList(repoFilter string, stdout io.Writer)` to `runWorktreeList(scope worktreeScope, stdout io.Writer)`, replacing the `repoFilter != "" && !filter.MatchesPattern(...)` guard with `!scope.matches(&repo)` and the `No worktrees found for '%s'` message with `scope.Label`. The all-repositories message stays unchanged.
- [ ] 3.5 Update `worktreeListCmd.RunE` to build the scope via `worktreeScopeFor` and return its error before doing any work. `Args` stays `cobra.MaximumNArgs(1)`.
- [ ] 3.6 Change `runWorktreeAlign(repoFilter string, dryRun bool)` to `runWorktreeAlign(scope worktreeScope, dryRun bool)`, replacing the filter guard in the planning loop with `!scope.matches(repo)`. Leave the plan construction, dry-run output, move execution, and config re-save untouched.
- [ ] 3.7 Update `worktreeAlignCmd.RunE` to build the scope via `worktreeScopeFor` and return its error before any repair or prune call runs.
- [ ] 3.8 Migrate the five existing `runWorktreeList` call sites in `worktree_list_test.go` and the eight in `worktree_align_test.go` to the new signature (`worktreeScope{}` for `""`, `worktreeScope{NamePattern: "repo-a"}` for a name). Do not change any assertion; confirm the suite is green before adding new cases.
- [ ] 3.9 Add `TestRunWorktreeList_NoArgListsAllRepos` and `TestRunWorktreeAlign_NoArgProcessesAllRepos` as explicit guards that omission still means all repositories, using a two-repository fixture where both have worktrees.
- [ ] 3.10 Add `TestRunWorktreeList_DotScopesToCurrentRepo`: two-repository fixture, chdir into repository A, build the scope through `worktreeScopeFor(cfg, ".")`, and assert the output contains A's worktree branch and not B's.
- [ ] 3.11 Add `TestRunWorktreeAlign_DotScopesToCurrentRepo`: give both repositories an unaligned worktree, chdir into A, run with `dryRun` true, and assert via `captureStdoutStr` that only A appears in the planned moves and B's worktree is still at its original path.
- [ ] 3.12 Add `TestRunWorktreeAlign_DotHonorsDryRun`: assert the `.`-scoped dry run prints `Would move` and leaves the worktree in place, matching the existing `TestRunWorktreeAlign_DryRun` assertions.
- [ ] 3.13 Add `TestRunWorktreeList_DotNotTreatedAsNamePattern`: track a repository literally named `my.repo` alongside the current one, chdir into the current one, and assert `worktreeScopeFor(cfg, ".")` returns a `RepoPath` scope with an empty `NamePattern` and that `my.repo` is absent from the output.
- [ ] 3.14 Add `TestWorktreeDotResolutionFailure`: chdir to a bare `t.TempDir()` and assert `worktreeScopeFor(cfg, ".")` returns an error satisfying `errors.Is(err, repocontext.ErrNotTracked)`, then assert both `worktreeListCmd.RunE` and `worktreeAlignCmd.RunE` surface that same error for the `.` argument.
- [ ] 3.15 Update the `list` and `align` `Long` help text and the `Subcommands:` block in `worktreeCmd.Long` to document the `.` form alongside the existing `[repo]` form.

### [ ] 4.0 Context-Aware Tab Completion for `add`, `list`, and `align`

None of these three subcommands currently registers a `ValidArgsFunction`; this task adds
completion so the new argument shapes are discoverable.

#### 4.0 Proof Artifact(s)

- CLI: transcript of `omgw __complete worktree add ""` run from inside a tracked repository, showing that repository's branch names in the output, demonstrates context-aware completion (FR U3-1)
- CLI: transcript of `omgw __complete worktree add ""` run from `$(mktemp -d)`, showing tracked repository names instead, demonstrates the fallback (FR U3-2)
- CLI: transcript of `omgw __complete worktree list ""` and `omgw __complete worktree align ""` run from inside a tracked repository, each showing `.` among the suggestions, demonstrates the `.` suggestion (FR U3-3)
- Test: `go test -run TestCompleteWorktreeAdd_BranchesWhenResolved -v ./cmd/omgitworks/` passes, demonstrates branch completion on successful resolution (FR U3-1)
- Test: `go test -run TestCompleteWorktreeAdd_ReposWhenUnresolved -v ./cmd/omgitworks/` passes, demonstrates the repository-name fallback (FR U3-2)
- Test: `go test -run TestCompleteWorktreeRepoOrDot -v ./cmd/omgitworks/` passes asserting `.` is present when resolution succeeds and absent when it fails, demonstrates conditional `.` suggestion (FR U3-3)
- Test: `go test -run TestCompleteWorktreeAdd_SecondArgument -v ./cmd/omgitworks/` passes asserting the second positional completes branches for the named repository, demonstrates completion tracks the two-argument form

#### 4.0 Tasks

- [ ] 4.1 Add `func ListBranches(repoPath string) ([]string, error)` to `internal/git/toplevel.go`, implemented as `gitCommand(repoPath, "for-each-ref", "--format=%(refname:short)", "refs/heads")` split on newlines with empty lines dropped. Cover it in `toplevel_test.go` with a repository holding two branches.
- [ ] 4.2 In `cmd/omgitworks/worktree.go`, add `func completeBranchNames(repoPath, toComplete string) ([]string, cobra.ShellCompDirective)` that calls `git.ListBranches` and applies the same case-insensitive prefix filter used by `completeRepoNames` in `cmd/omgitworks/tag.go:197`, returning `cobra.ShellCompDirectiveNoFileComp`.
- [ ] 4.3 Add `func completeWorktreeAddArgs(_ *cobra.Command, args []string, toComplete string)`: for the first positional, load config and try `repocontext.ResolveCurrent`; on success return that repository's branches, on `ErrNotTracked` fall back to `completeRepoNames`. For the second positional, return branches for the repository named by `args[0]` via `findRepositories`. Return no completions beyond two arguments.
- [ ] 4.4 Add `func completeWorktreeRepoOrDot(_ *cobra.Command, args []string, toComplete string)`: return nothing once one argument is present; otherwise start from `completeRepoNames` and prepend `.` when `repocontext.ResolveCurrent` succeeds and `.` matches `toComplete`.
- [ ] 4.5 Register `worktreeAddCmd.ValidArgsFunction = completeWorktreeAddArgs` in `worktree_add.go`'s `init`, and `worktreeListCmd.ValidArgsFunction` / `worktreeAlignCmd.ValidArgsFunction = completeWorktreeRepoOrDot` in their respective `init` functions. Leave `completeWorktreeBranches` on `worktreeCmd` and `worktreeNavigateCmd` unchanged.
- [ ] 4.6 Create `cmd/omgitworks/worktree_completion_test.go` with `TestCompleteWorktreeAdd_BranchesWhenResolved`: fixture repository with an extra branch, chdir into it, and assert the returned slice contains the branch names and the directive is `ShellCompDirectiveNoFileComp`.
- [ ] 4.7 Add `TestCompleteWorktreeAdd_ReposWhenUnresolved`: chdir to a bare `t.TempDir()` and assert the completions are the tracked repository names, not branches.
- [ ] 4.8 Add `TestCompleteWorktreeAdd_SecondArgument`: assert `completeWorktreeAddArgs` with `args = []string{"my-repo"}` returns that repository's branches regardless of the working directory.
- [ ] 4.9 Add `TestCompleteWorktreeRepoOrDot`: assert `.` is included when the working directory resolves, absent when it does not, and that supplying a first argument yields no completions.
- [ ] 4.10 Add a prefix-filter case asserting `completeWorktreeRepoOrDot` with `toComplete = "my"` omits `.`, confirming `.` is filtered on the typed prefix like every other suggestion.

### [ ] 5.0 Documentation and Shell-Integration Regression Guard

Publish the new invocation forms and prove the shell function was untouched.

#### 5.0 Proof Artifact(s)

- Documentation: diff of `docs/site/commands-core.md` showing the `worktree add` section documenting the omitted-repository form, the `worktree list` and `worktree align` sections documenting `.`, a statement of the worktree-relative resolution behavior, and the not-a-tracked-repository error text, demonstrates published documentation matches behavior (FR U3-5)
- Test: `go test -run TestShellTemplates -v ./cmd/omgitworks/` passes with all seven pre-existing `TestShellTemplates*` cases green and `git diff --stat cmd/omgitworks/shellinit.go` reporting no change, demonstrates no shell-function regression (FR U3-4)
- CLI: `omgw worktree add --help`, `omgw worktree list --help`, and `omgw worktree align --help` transcripts, showing usage strings and examples that match the documented forms, demonstrates in-tool help and published docs agree
- CLI: `make ci` exits 0, demonstrates vet, lint, and the race-enabled test suite all pass on the complete change

#### 5.0 Tasks

- [ ] 5.1 Update the `omgw worktree add` section of `docs/site/commands-core.md` (around line 488): change the signature line to `omgw worktree add [repo] <branch>`, add an example run from inside a repository with the repository argument omitted, and state that the two-argument form is unchanged and always wins over detection.
- [ ] 5.2 Add a short subsection under `worktree add` explaining that the repository is resolved from the current directory, that running from inside one of the repository's worktrees resolves to the owning repository, and that resolution reads worktree paths recorded by `omgw refresh`.
- [ ] 5.3 Update the `omgw worktree list` section (around line 460) and the `omgw worktree align` section (around line 506): document `.` as the current-repository argument, show an example of each, and state explicitly that omitting the argument still means all repositories.
- [ ] 5.4 Document the resolution-failure error once, quoting the exact message from `repocontext.ErrNotTracked` and both remedies, and cross-reference it from the `add`, `list`, and `align` sections rather than repeating it three times.
- [ ] 5.5 Confirm `cmd/omgitworks/shellinit.go` is unmodified with `git diff --stat cmd/omgitworks/shellinit.go`, and run `go test -run TestShellTemplates -v ./cmd/omgitworks/` to capture the seven passing cases as the regression proof.
- [ ] 5.6 Build with `make build` and capture `--help` output for `worktree add`, `worktree list`, and `worktree align`, checking each against the documentation edited above and correcting whichever is wrong.
- [ ] 5.7 Run `make ci` and capture the passing output. Resolve any `golangci-lint` finding in the new package rather than adding an exclusion.
- [ ] 5.8 Record the CLI transcripts named in the 1.0-4.0 proof artifacts, using a scratch workspace and placeholder repository names, and confirm no absolute home paths or identifying values remain in the captured output before it is committed.

## Requirement Coverage Map

| Spec Unit | Functional Requirement | Covered By | Planned Test Artifact |
| --- | --- | --- | --- |
| 1 | Reusable resolver callable from all worktree subcommands | 1.4-1.8 | `TestResolve` |
| 1 | Determine enclosing checkout via `git rev-parse --show-toplevel` | 1.1, 1.5 | `TestToplevel`, `TestResolve_Subdirectory` |
| 1 | Compare against repository `Path`, then `Worktrees[].Path` | 1.6, 1.7 | `TestResolve`, `TestResolve_InsideWorktree` |
| 1 | Worktree match returns the owning repository | 1.7 | `TestResolve_InsideWorktree` |
| 1 | Resolve symlinks on both git-reported and stored paths | 1.2, 1.5, 1.6 | `TestResolve_SymlinkedPath` |
| 1 | Exact comparison after resolution, not prefix | 1.6, 1.7 | `TestResolve_NestedRepoNotPrefixMatched`, `TestResolve_WindowsDriveLetterCase` |
| 1 | `worktree add <branch>` uses the resolved repository | 2.3, 2.4 | `TestRunWorktreeAdd_ResolvesCurrentRepo`, `TestRunWorktreeAdd_ResolvesFromInsideWorktree` |
| 1 | `worktree add <repo> <branch>` behavior unchanged | 2.1, 2.2, 2.4 | `TestRunWorktreeAdd_ExplicitRepoIgnoresCwd`, five existing `TestRunWorktreeAdd_*` |
| 1 | Resolution failure exits non-zero naming both remedies | 1.4, 2.3 | `TestResolve_NotTracked`, `TestRunWorktreeAddCurrent_NotTracked` |
| 1 | `rev-parse` failure produces the same error, not git's message | 1.5 | `TestResolve_NotAGitRepo` |
| 1 | Resolver does not write config, create directories, or prompt | 1.8 | `TestResolve_DoesNotMutateConfig` |
| 2 | `.` accepted as repository argument on `list` and `align` | 3.3, 3.5, 3.7 | `TestRunWorktreeList_DotScopesToCurrentRepo`, `TestRunWorktreeAlign_DotScopesToCurrentRepo` |
| 2 | Bare `worktree list` still lists all repositories | 3.4, 3.8 | `TestRunWorktreeList_NoArgListsAllRepos` |
| 2 | Bare `worktree align` still aligns all repositories | 3.6, 3.8 | `TestRunWorktreeAlign_NoArgProcessesAllRepos` |
| 2 | `worktree list .` scopes to the resolved repository | 3.2, 3.4 | `TestRunWorktreeList_DotScopesToCurrentRepo` |
| 2 | `worktree align .` scopes and honors `--dry-run` | 3.6, 3.11, 3.12 | `TestRunWorktreeAlign_DotScopesToCurrentRepo`, `TestRunWorktreeAlign_DotHonorsDryRun` |
| 2 | `.` resolution failure yields the Unit 1 error | 3.3 | `TestWorktreeDotResolutionFailure` |
| 2 | Literal `.` is never a repository name pattern | 3.2, 3.3 | `TestRunWorktreeList_DotNotTreatedAsNamePattern` |
| 3 | `add` first-argument completion suggests branches when resolved | 4.1-4.3, 4.5 | `TestCompleteWorktreeAdd_BranchesWhenResolved` |
| 3 | `add` first-argument completion falls back to repository names | 4.3 | `TestCompleteWorktreeAdd_ReposWhenUnresolved` |
| 3 | `list` and `align` completion includes `.` | 4.4, 4.5 | `TestCompleteWorktreeRepoOrDot` |
| 3 | `shellinit.go` requires no changes | 5.5 | seven existing `TestShellTemplates*` cases |
| 3 | `docs/site/commands-core.md` documents the new forms | 5.1-5.4 | Documentation diff plus `--help` cross-check (5.6) |
