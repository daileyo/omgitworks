# Task 04 Proofs - Help text and published docs for the current-repository form

## Task Summary

This task makes the new argument shape discoverable. The `assign` usage line becomes `assign [repository] <profile>`, its help documents both forms, the detection rule, the worktree behavior, and the `--use-subdirs` restriction, and every example uses `omgw`. The `user` parent help is rewritten from `gws` to `omgw` and gains the one-argument `assign` example. The published reference `docs/site/commands-user.md` is updated to match.

## What This Task Proves

- Neither the `assign` help nor the `user` help contains a `gws` example any more (success metric 5).
- The `assign` help shows both forms, explains detection from the current directory, and states that `--use-subdirs` needs an explicit repository.
- The shell wrappers and the deprecated root-flag messages are untouched: both existing suites pass with no edits.
- The published docs carry the new usage line, both forms, the worktree note, the error text, the completion rule, and the flag restriction, and the site builds in strict mode with the new cross-page link.

## Evidence Summary

- Two new help-text tests pass alongside the full `TestShell*` and `TestDeprecated*` suites, none of which were modified.
- `omgw user assign --help` and `omgw user --help` from the built binary show the rewritten text.
- `git diff docs/site/commands-user.md` shows the Assign Profile section changes, and `mkdocs build --strict` exits zero.
- `make ci` exits zero.

## Artifact: Help-text tests and unchanged shell and deprecated suites

**What it proves:** The `gws` rewrite is complete for the two help blocks this spec touches, the `Use` string is exact, and the rewrite did not reach the deprecated root-flag messages (spec non-goal 9) or the shell wrappers (spec Unit 2's last requirement).

**Why it matters:** `deprecated.go` has its own `gws user list` string that a careless search-and-replace across `user.go` would not touch, but a careless rewrite of shared text could. These suites passing unmodified is the evidence that nothing outside scope moved.

**Command:**

```bash
go test ./cmd/omgitworks/ -run 'TestUserAssign_Help|TestShell|TestDeprecated' -count=1 -v
```

```text
--- PASS: TestDeprecatedFlagsAreHidden (0.00s)
--- PASS: TestDeprecatedListEmitsWarning (0.00s)
--- PASS: TestDeprecatedMutualExclusivity (0.00s)
--- PASS: TestDeprecatedRecursiveRequiresAdd (0.00s)
--- PASS: TestDeprecatedGoConflictsWithArgs (0.00s)
--- PASS: TestDeprecatedNoFlagsSet (0.00s)
--- PASS: TestDeprecatedAddTagEmitsWarning (0.00s)
--- PASS: TestDeprecatedAddTagDispatch (0.00s)
--- PASS: TestDeprecatedUserEmitsWarning (0.00s)
--- PASS: TestDeprecatedUserUpdateRequiresUser (0.00s)
--- PASS: TestDeprecatedUserDeleteRequiresUser (0.00s)
--- PASS: TestDeprecatedUserUpdateDeleteMutualExclusivity (0.00s)
--- PASS: TestDeprecatedVisibilityFlag (0.00s)
--- PASS: TestDeprecatedListDispatchPopulatesListOptions (0.00s)
--- PASS: TestShellWrapperWorktreePassthrough (1.57s)
--- PASS: TestShellWrapperWorktreeNavigation (0.14s)
--- PASS: TestShellTemplatesRouteSubcommands (0.00s)
--- PASS: TestShellTemplatesContainBinPlaceholder (0.00s)
--- PASS: TestShellTemplatesContainNavigationFallthrough (0.00s)
--- PASS: TestShellTemplatesContainWorktreeNavigation (0.00s)
--- PASS: TestShellTemplatesContainParentNavigation (0.00s)
--- PASS: TestShellTemplatesContainCdNavigation (0.00s)
--- PASS: TestShellTemplatesKeepPrintWorkspacePassthrough (0.00s)
--- PASS: TestUserAssign_HelpTextUsesOmgw (0.00s)
--- PASS: TestUserAssign_HelpDocumentsBothForms (0.00s)
PASS
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	1.721s
```

| Test | Observable assertions |
| --- | --- |
| `_HelpTextUsesOmgw` | `userAssignCmd.Long` and `userCmd.Long` contain no `gws `; `userAssignCmd.Use == "assign [repository] <profile>"` |
| `_HelpDocumentsBothForms` | `assign` help contains `omgw user assign work`, `omgw user assign my-project work`, `current directory`, and `explicit repository argument`; `user` help contains `omgw user assign work` |

`git diff --stat` for the Go side of this task touches only `user.go` (the parent `Long`), `user_assign.go` (`Use` and `Long`), and `user_assign_test.go`. `shellinit.go`, `shellinit_test.go`, `shellinit_exec_test.go`, `deprecated.go`, and `deprecated_test.go` are unchanged on this branch.

## Artifact: `--help` output from the binary

**What it proves:** The rewritten help is what users see, with the `[repository]` usage line rendered by cobra from the `Use` string.

**Artifact path:** `docs/specs/29-spec-user-repo-context/29-proofs/assign-help.md`

**Result summary:** `omgw user assign --help` shows the detection paragraph, the `--use-subdirs` restriction, four `omgw` examples beginning with the one-argument form, and `Usage: omgitworks user assign [repository] <profile> [flags]`. `omgw user --help` lists `omgw user assign work` beside the two-argument example and shows no `gws`.

## Artifact: Published docs diff and strict site build

**What it proves:** The published reference matches the behavior, and the link to the Current-Repository Targeting section in `commands-core.md` resolves.

**Why it matters:** The docs site is built in strict mode by the docs workflow, so a broken anchor would fail the deploy.

**Artifact path:** `docs/specs/29-spec-user-repo-context/29-proofs/docs-diff.md`

**Commands:**

```bash
grep -n 'omgw user assign \[repository\] <profile>' docs/site/commands-user.md
python3 -m venv .venv && .venv/bin/pip install -q -r docs/requirements.txt
.venv/bin/python -m mkdocs build --strict
```

```text
125:omgw user assign [repository] <profile> [flags]

INFO    -  Cleaning site directory
INFO    -  Building documentation to directory: /home/daileyo/gws/omgitworks/site
INFO    -  Documentation built in 0.21 seconds
mkdocs strict exit=0
```

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
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	6.873s
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

The `assign` and `user` help now describe the command users actually have, in the binary name they actually have, and the published docs say the same thing. Nothing outside the two help blocks the spec names was rewritten, which the unmodified deprecated and shell suites confirm. With this task, all four parent tasks of spec 29 are implemented and proven.
