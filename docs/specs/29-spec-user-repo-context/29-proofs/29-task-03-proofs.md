# Task 03 Proofs - Context-aware tab completion for `omgw user assign`

## Task Summary

This task attaches a completion function to `assign` so the shell offers the right names for each argument position. Inside a tracked repository the first argument completes to profile names, because the one-argument form is the expected use there. Outside one it completes to repository names. Once a repository has been typed, the second argument completes to profiles, which the two-argument form never had before.

## What This Task Proves

- With no arguments typed and the working directory inside a tracked repository, only profile names are offered.
- With no arguments typed and the working directory outside any tracked repository, only repository names are offered.
- With one argument typed, profile names are offered, and the typed prefix filters them.
- With two arguments typed, nothing is offered and file completion stays off.
- The behavior holds through cobra's real `__complete` entry point, which the installed completion scripts call.

## Evidence Summary

- Four new test functions pass, calling `completeUserAssignArgs` directly from inside and outside a tracked repository.
- The `__complete` transcript shows the same four outcomes through the binary.
- `make ci` exits zero with golangci-lint reporting `0 issues`.

## Artifact: Completion tests

**What it proves:** Each branch of the arity switch returns exactly the expected set, with the `NoFileComp` directive, and nothing from the other set leaks in.

**Why it matters:** The fixture's profile names (`work`, `personal`) and repository names (`alpha`, `beta`) never overlap, so each assertion compares the full sorted result rather than checking for presence.

**Command:**

```bash
go test ./cmd/omgitworks/ -run 'TestCompleteUserAssign' -count=1 -v
```

```text
=== RUN   TestCompleteUserAssign_ProfilesWhenResolved
--- PASS: TestCompleteUserAssign_ProfilesWhenResolved (0.02s)
=== RUN   TestCompleteUserAssign_ReposWhenNotResolved
--- PASS: TestCompleteUserAssign_ReposWhenNotResolved (0.02s)
=== RUN   TestCompleteUserAssign_ProfilesForSecondArg
--- PASS: TestCompleteUserAssign_ProfilesForSecondArg (0.02s)
=== RUN   TestCompleteUserAssign_NothingAfterTwoArgs
--- PASS: TestCompleteUserAssign_NothingAfterTwoArgs (0.01s)
PASS
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.253s
```

| Test | Observable assertions |
| --- | --- |
| `_ProfilesWhenResolved` | from inside `alpha`, no args: result equals `[personal work]`, directive `NoFileComp` |
| `_ReposWhenNotResolved` | from an untracked directory, no args: result equals `[alpha beta]`, directive `NoFileComp` |
| `_ProfilesForSecondArg` | args `[alpha]`: result equals `[personal work]`; with prefix `w`: `[work]` |
| `_NothingAfterTwoArgs` | args `[alpha work]`: empty result, directive `NoFileComp` |

The existing `TestUserProfileCompletion` and all fifteen `TestUserAssign_*` tests still pass alongside them.

## Artifact: `__complete` transcript

**What it proves:** The completion function is wired to the command and behaves the same way through cobra's completion machinery as it does when called directly.

**Why it matters:** A completion function that is written but never attached with `ValidArgsFunction` would pass the unit tests and do nothing in a shell. This transcript rules that out.

**Artifact path:** `docs/specs/29-spec-user-repo-context/29-proofs/completion.md`

**Result summary:** Inside `my-api`, `omgw __complete user assign ""` lists `work` and `personal`. From `$HOME`, the same command lists `my-api` and `web-ui`. With `my-api` typed, the next position lists the profiles, and `w` narrows to `work`. With both typed, nothing is listed. Every call ends with `ShellCompDirectiveNoFileComp`.

## Artifact: Full CI gate

**Command:**

```bash
PATH=$HOME/go/bin:$PATH make ci
```

```text
Running go vet...
Running linter...
0 issues.
Running tests with race detector...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.452s
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

## Reviewer Conclusion

Completion now mirrors the command's own rule: where the one-argument form would resolve, the first argument is a profile; where it would not, it is a repository; and a second argument is always a profile. The unit tests and the `__complete` transcript agree. Unit 2's three completion requirements are implemented and proven.
