# Task 01 Proofs - `assign` extracted into its own file with today's behavior pinned

## Task Summary

This task moves `omgw user assign` out of `user.go` into `cmd/omgitworks/user_assign.go`, splits its `RunE` closure into a `runUserAssign` function that tests can call without cobra, and extracts the profile lookup into `lookupProfile`. Before any behavior changes in task 2.0, five characterization tests lock the two-argument form's output, errors, ordering, and the prompt-driven `--use-subdirs` path. No user-visible behavior changes in this task.

## What This Task Proves

- The two-argument form still assigns the named repository, writing both git's local config and the stored repository fields.
- The ambiguity error prints the same stdout listing and returns the same error string as before.
- An unknown profile produces the profile-lookup error, never the resolver error, and leaves `.git/config` untouched.
- `--dry-run` prints the same header and change lines and leaves both `.git/config` and `config.json` byte-identical.
- The `--use-subdirs` prompt path (audit flag 1) moves the repository, updates the stored path and source, and writes `includeIf` into the sandboxed global gitconfig; answering `n` moves nothing.
- `assign` is still registered under `user` after the move, and the full CI gate passes.

## Evidence Summary

- Six test functions (five new plus the existing registration test) pass under `-race`, with eight sub-cases in total.
- The extraction removes 155 lines from `user.go` and adds a 191-line `user_assign.go`, with the only logic change being the `io.Reader` seam for the prompt.
- `make ci` exits zero: vet, golangci-lint v2.13.2, and the race-detector suite all pass.

## Artifact: Characterization tests and registration test

**What it proves:** Each pinned behavior has a passing test before task 2.0 changes anything, and `assign` remains a registered `user` subcommand.

**Why it matters:** Success metric 2 requires every current invocation to produce identical output after the feature lands. These tests are the only thing that can detect a regression in the move.

**Command:**

```bash
go test ./cmd/omgitworks/ -run 'TestUserAssign_|TestUserSubcommandsRegistered' -count=1 -v
```

**Result summary:** All six top-level tests pass, including both sub-cases of the `--use-subdirs` test added by the audit remediation.

```text
=== RUN   TestUserAssign_TwoArgs_AssignsNamedRepo
--- PASS: TestUserAssign_TwoArgs_AssignsNamedRepo (0.01s)
=== RUN   TestUserAssign_TwoArgs_MultiMatchError
--- PASS: TestUserAssign_TwoArgs_MultiMatchError (0.01s)
=== RUN   TestUserAssign_TwoArgs_UnknownProfileError
--- PASS: TestUserAssign_TwoArgs_UnknownProfileError (0.01s)
=== RUN   TestUserAssign_TwoArgs_DryRunOutput
--- PASS: TestUserAssign_TwoArgs_DryRunOutput (0.01s)
=== RUN   TestUserAssign_TwoArgs_UseSubdirsStillWorks
=== RUN   TestUserAssign_TwoArgs_UseSubdirsStillWorks/accepted
=== RUN   TestUserAssign_TwoArgs_UseSubdirsStillWorks/canceled
--- PASS: TestUserAssign_TwoArgs_UseSubdirsStillWorks (0.01s)
    --- PASS: TestUserAssign_TwoArgs_UseSubdirsStillWorks/accepted (0.01s)
    --- PASS: TestUserAssign_TwoArgs_UseSubdirsStillWorks/canceled (0.01s)
=== RUN   TestUserSubcommandsRegistered
--- PASS: TestUserSubcommandsRegistered (0.00s)
    --- PASS: TestUserSubcommandsRegistered/list (0.00s)
    --- PASS: TestUserSubcommandsRegistered/add (0.00s)
    --- PASS: TestUserSubcommandsRegistered/show (0.00s)
    --- PASS: TestUserSubcommandsRegistered/remove (0.00s)
    --- PASS: TestUserSubcommandsRegistered/assign (0.00s)
    --- PASS: TestUserSubcommandsRegistered/sync (0.00s)
PASS
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.065s
```

What each test asserts:

| Test | Observable assertions |
| --- | --- |
| `_AssignsNamedRepo` | `git config --local user.email` and `user.name` equal the profile; stored `User`, `Email`, `SigningEnabled`, `UserSource == local`; stdout has the `Assigned profile` block |
| `_MultiMatchError` | error string is `please specify a more specific repository identifier`; stdout lists `Multiple repositories match 'api':` with one `  - name (path)` line each; neither `.git/config` changed |
| `_UnknownProfileError` | error equals `user.GetProfile`'s error for the same name; not `repocontext.ErrNotTracked`; `.git/config` unchanged |
| `_DryRunOutput` | header `Dry run - would assign profile 'work' to repository 'alpha':`, `user.name` and `user.email` change lines; `.git/config` and `config.json` byte-identical |
| `_UseSubdirsStillWorks/accepted` | repository exists at `<workspace>/work/alpha` and not at the old path; stored `Path` updated; `UserSource == includeif`; `$HOME/.gitconfig` contains `[includeIf`; stdout has `Moved repository to:` |
| `_UseSubdirsStillWorks/canceled` | stdout has `Canceled.`; old path still exists; `.git/config` unchanged |

## Artifact: Extraction diffstat

**What it proves:** The command moved out of `user.go` into its own file, and the only additions are the test file and the small amount of glue (`userAssignOptions`, `lookupProfile`, the `init()` registering flags).

**Why it matters:** A reviewer can see at a glance that this is a move, not a rewrite, and that `user.go` lost exactly the block that `user_assign.go` gained.

**Command:**

```bash
git diff --stat
```

```text
 cmd/omgitworks/user.go             | 155 --------------------
 cmd/omgitworks/user_assign.go      | 191 ++++++++++++++++++++++++
 cmd/omgitworks/user_assign_test.go | 293 +++++++++++++++++++++++++++++++++++++
 3 files changed, 484 insertions(+), 155 deletions(-)
```

The 36-line difference between the removed and added source is the `userAssignOptions` type, the `runUserAssign` and `lookupProfile` declarations with doc comments, the new `init()`, and the `io` import. `user.go` keeps `userCmd.AddCommand(userAssignCmd)` in its own `init()`, and its `path/filepath` import was dropped because the only use moved.

## Artifact: Full CI gate

**What it proves:** The refactor passes vet, lint, and the complete test suite under the race detector, on every package.

**Why it matters:** `make ci` is the repository's merge gate; the pre-push hook runs the same three steps.

**Command:**

```bash
PATH=$HOME/go/bin:$PATH make ci
```

**Result summary:** Exit zero. golangci-lint v2.13.2 (the version CI pins) reports no issues after two US-spelling fixes in test names. All nine packages pass with `-race`.

```text
Running go vet...
go vet ./...
Running linter...
golangci-lint run --timeout=5m
Running tests with race detector...
go test -v -race ./...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.348s
ok  	github.com/daileyo/omgitworks/internal/classifier	1.014s
ok  	github.com/daileyo/omgitworks/internal/config	1.046s
ok  	github.com/daileyo/omgitworks/internal/discovery	1.093s
ok  	github.com/daileyo/omgitworks/internal/filter	1.033s
ok  	github.com/daileyo/omgitworks/internal/git	1.570s
ok  	github.com/daileyo/omgitworks/internal/repocontext	1.212s
ok  	github.com/daileyo/omgitworks/internal/user	1.113s
ok  	github.com/daileyo/omgitworks/internal/xdg	1.032s
Tests complete
All CI checks passed!
```

## Reviewer Conclusion

`assign` now lives in `user_assign.go` behind a cobra-free entry point, and the behavior task 2.0 must preserve is pinned by tests that pass on this commit. The prompt-driven `--use-subdirs` path, which had no test before, is covered in both the accept and cancel cases. Task 2.0 can change the argument shape with a safety net in place.
