# Task 02 Proofs - `omgw user assign <profile>` resolved from the working directory

## Task Summary

This task makes the repository argument optional. `omgw user assign <profile>` now applies the profile to the repository the shell is standing in, resolved through the spec 25 resolver, including from a nested subdirectory and from inside one of the repository's worktrees. The two-argument form is untouched, `--use-subdirs` is refused before anything is loaded when no repository is named, and `--dry-run` names the resolved repository.

## What This Task Proves

- One positional argument resolves the repository from the working directory at the root, in a subdirectory, and inside a worktree, where the write lands on the owning repository's `.git/config`.
- Naming a repository never consults the working directory, whether the shell is in another tracked repository or in an untracked directory.
- Outside any tracked repository the command fails with `repocontext.ErrNotTracked` unchanged and writes nothing.
- The repository is resolved before the profile is looked up, so an unknown profile inside a tracked repository reports the profile error.
- `--use-subdirs` with one argument is refused before `config.Load()`, with and without `--dry-run`, and nothing is printed, moved, or written.
- The one-argument success output is byte-for-byte the two-argument output, so no "inferred" line was added.
- Zero and three positionals are rejected by `cobra.RangeArgs(1, 2)`.

## Evidence Summary

- Ten new test functions pass, thirteen cases counting sub-tests, alongside the five characterization tests from task 1.0.
- Two transcripts against a real binary show the feature end to end, with `git config --local` reading back the assigned email from the main checkout after assigning from a worktree.
- `make ci` exits zero.

## Artifact: One-argument form, precedence, and refusal tests

**What it proves:** Every functional requirement of spec Unit 1 has a passing test, as mapped in the audit's traceability table.

**Why it matters:** These tests are the evidence the validation phase will re-run. The worktree test in particular proves the resolver returns the owning repository, without which `AssignLocal` would fail on a worktree's `.git` file.

**Command:**

```bash
go test ./cmd/omgitworks/ -run 'TestUserAssign_OneArg|TestUserAssign_TwoArgs_ResolverNotConsulted|TestUserAssign_ArgCount' -count=1 -v
```

**Result summary:** All pass. The `_FromInsideWorktree` test reads `user.email` from both the main checkout and the worktree and asserts the stored repository path is the owning repository's.

```text
=== RUN   TestUserAssign_OneArg_FromRepoRoot
--- PASS: TestUserAssign_OneArg_FromRepoRoot (0.01s)
=== RUN   TestUserAssign_OneArg_FromSubdirectory
--- PASS: TestUserAssign_OneArg_FromSubdirectory (0.01s)
=== RUN   TestUserAssign_OneArg_FromInsideWorktree
--- PASS: TestUserAssign_OneArg_FromInsideWorktree (0.02s)
=== RUN   TestUserAssign_OneArg_OutputMatchesTwoArgForm
--- PASS: TestUserAssign_OneArg_OutputMatchesTwoArgForm (0.01s)
=== RUN   TestUserAssign_TwoArgs_ResolverNotConsulted
--- PASS: TestUserAssign_TwoArgs_ResolverNotConsulted (0.03s)
    --- PASS: TestUserAssign_TwoArgs_ResolverNotConsulted/from_inside_another_tracked_repository (0.02s)
    --- PASS: TestUserAssign_TwoArgs_ResolverNotConsulted/from_an_untracked_directory (0.01s)
=== RUN   TestUserAssign_OneArg_NotTracked
--- PASS: TestUserAssign_OneArg_NotTracked (0.01s)
=== RUN   TestUserAssign_OneArg_UnknownProfile_ReportsProfileError
--- PASS: TestUserAssign_OneArg_UnknownProfile_ReportsProfileError (0.01s)
=== RUN   TestUserAssign_OneArg_UseSubdirsRefused
--- PASS: TestUserAssign_OneArg_UseSubdirsRefused (0.02s)
    --- PASS: TestUserAssign_OneArg_UseSubdirsRefused/real_run (0.01s)
    --- PASS: TestUserAssign_OneArg_UseSubdirsRefused/dry_run (0.01s)
=== RUN   TestUserAssign_OneArg_DryRunNamesResolvedRepo
--- PASS: TestUserAssign_OneArg_DryRunNamesResolvedRepo (0.01s)
=== RUN   TestUserAssign_ArgCount
--- PASS: TestUserAssign_ArgCount (0.00s)
PASS
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.149s
```

What each test asserts:

| Test | Observable assertions |
| --- | --- |
| `_FromRepoRoot` | from the repository root, `[]string{"work"}` sets local `user.email` / `user.name`, stored fields, `UserSource == local`, and the standard success block |
| `_FromSubdirectory` | the same from `<repo>/pkg/deep` |
| `_FromInsideWorktree` | from a worktree created by `runWorktreeAdd`, `user.email` read in the **main checkout** and in the worktree equals the profile; stored `Path` is the owning repository; output names `my-repo` |
| `_OutputMatchesTwoArgForm` | stdout of `assign work` inside `alpha` equals stdout of `assign alpha work` byte for byte, after restoring both files between runs |
| `_ResolverNotConsulted` | two-argument form from inside `beta` changes `alpha` only; from an untracked directory it succeeds and never returns `ErrNotTracked` |
| `_NotTracked` | from an untracked directory, `errors.Is(err, repocontext.ErrNotTracked)`; no stdout; `.git/config` and `config.json` unchanged |
| `_UnknownProfile_ReportsProfileError` | inside a tracked repository, `[]string{"nope"}` returns `user.GetProfile`'s error, not the resolver error |
| `_UseSubdirsRefused` (real run, dry run) | error contains `--use-subdirs requires an explicit repository argument`; stdout empty; repository not moved; `.git/config` and `config.json` unchanged; no global gitconfig created |
| `_DryRunNamesResolvedRepo` | dry-run header reads `to repository 'alpha':`; both files byte-identical |
| `_ArgCount` | `userAssignCmd.Args` rejects 0 and 3 positionals, accepts 1 and 2 |

## Artifact: Transcript, assign from the repository root

**What it proves:** The feature works end to end through the real binary, not only through the Go entry point.

**Why it matters:** It exercises cobra's argument validation, config loading from the XDG path, and the resolver's `git rev-parse` subprocess together.

**Artifact path:** `docs/specs/29-spec-user-repo-context/29-proofs/assign-current-repo.md`

**Result summary:** `cd <repo> && omgw user assign work` prints `Assigned profile 'work' to repository 'my-api'` and `git config --local user.email` returns `work@example.com`.

## Artifact: Transcript, assign from inside a worktree

**What it proves:** From inside a worktree the profile is applied to the owning repository, and the worktree sees it too.

**Why it matters:** This is the worktree-relative case from question 1 option A, and the behavior that distinguishes the shared resolver from a naive "current directory is the repository" lookup.

**Artifact path:** `docs/specs/29-spec-user-repo-context/29-proofs/assign-from-worktree.md`

**Result summary:** With the shell inside `projects/my-api/feat-x`, `omgw user assign personal` names `my-api`, and `git -C <main checkout> config --local user.email` returns `personal@example.com`. The same file also records the failure and preview cases: the resolver error from an untracked directory, the `--use-subdirs` refusal with and without `--dry-run`, and a one-argument dry run naming the resolved repository.

## Artifact: Full CI gate

**What it proves:** Vet, lint, and the race-detector suite pass on every package.

**Command:**

```bash
PATH=$HOME/go/bin:$PATH make ci
```

```text
Running go vet...
Running linter...
Running tests with race detector...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.270s
ok  	github.com/daileyo/omgitworks/internal/classifier	(cached)
ok  	github.com/daileyo/omgitworks/internal/config	(cached)
ok  	github.com/daileyo/omgitworks/internal/discovery	(cached)
ok  	github.com/daileyo/omgitworks/internal/filter	(cached)
ok  	github.com/daileyo/omgitworks/internal/git	(cached)
ok  	github.com/daileyo/omgitworks/internal/repocontext	(cached)
ok  	github.com/daileyo/omgitworks/internal/user	(cached)
ok  	github.com/daileyo/omgitworks/internal/xdg	(cached)
All CI checks passed!
```

## Observation for validation

`userAssignCmd` does not set `SilenceUsage`, so cobra prints the usage block after every runtime error, including the resolver error and the `--use-subdirs` refusal. The worktree commands silence it. This is pre-existing `assign` behavior that the spec does not mention, so it was left as is to keep every existing error path's output unchanged. It is recorded here so validation can decide whether it matters.

## Reviewer Conclusion

The one-argument form works from the root, a subdirectory, and a worktree, always writing to the owning repository. Explicit names win, resolution failure is the shared spec 25 error with nothing written, lookup order and the `--use-subdirs` refusal are enforced in the required order, and the success output is identical to the two-argument form. Unit 1 of the spec is implemented and proven.
