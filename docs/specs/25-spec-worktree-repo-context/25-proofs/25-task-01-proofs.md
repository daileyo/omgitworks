# Task 01 Proofs - Current-repository resolver

## Task Summary

This task delivers `internal/repocontext`, the shared resolver that answers "which
tracked repository does this directory belong to?". It is the dependency for every
other task in spec 25, and for specs 26, 27, and 28. It changes no user-visible
command on its own, so the evidence here is the test suite that pins its contract.

## What This Task Proves

- Resolution works from a repository root, from a deeply nested subdirectory, and from
  inside one of that repository's worktrees, where it returns the **owning repository**.
- Comparison is exact rather than prefix-based, so a repository nested inside another
  repository's tree does not wrongly resolve to the outer one.
- Symlinks are resolved on both the git-reported path and the stored path, so a
  symlinked workspace still matches.
- Failure is a single, actionable error naming both remedies, with git's own wording
  never leaking through.
- Resolution is read-only: `config.json` is byte-identical after both a successful and
  a failed resolve.

## Evidence Summary

- 11 resolver tests pass and 1 skips (Windows-only), covering every functional
  requirement in Unit 1 of the spec.
- 5 tests for the new `internal/git` helpers pass.
- `golangci-lint run ./internal/...` reports `0 issues` and `go vet ./...` is silent.

## Artifact: Resolver test suite

**What it proves:** Each named case maps to a Unit 1 functional requirement: the base
case, git's upward walk, the worktree-path fallback, symlink resolution, exact matching,
the error contract, git-message normalization, and the read-only guarantee.

**Why it matters:** The resolver has no CLI surface of its own, so these tests are the
only place its contract is enforced. Specs 26-28 will build directly on it.

**Command:**

~~~bash
go test -v ./internal/repocontext/
~~~

**Result summary:** 11 passed, 1 skipped. `TestResolve_WindowsDriveLetterCase` skips on
Linux by design; it guards drive-letter case folding on Windows, where the filesystem is
case-insensitive and CI's test job does not run.

~~~text
=== RUN   TestResolve
--- PASS: TestResolve (0.01s)
=== RUN   TestResolve_Subdirectory
--- PASS: TestResolve_Subdirectory (0.01s)
=== RUN   TestResolve_InsideWorktree
--- PASS: TestResolve_InsideWorktree (0.02s)
=== RUN   TestResolve_InsideWorktreeSubdirectory
--- PASS: TestResolve_InsideWorktreeSubdirectory (0.02s)
=== RUN   TestResolve_SymlinkedPath
--- PASS: TestResolve_SymlinkedPath (0.01s)
=== RUN   TestResolve_NestedRepoNotPrefixMatched
--- PASS: TestResolve_NestedRepoNotPrefixMatched (0.02s)
=== RUN   TestResolve_NotTracked
--- PASS: TestResolve_NotTracked (0.03s)
=== RUN   TestResolve_NotAGitRepo
--- PASS: TestResolve_NotAGitRepo (0.01s)
=== RUN   TestResolveCurrent
--- PASS: TestResolveCurrent (0.01s)
=== RUN   TestResolve_DoesNotMutateConfig
--- PASS: TestResolve_DoesNotMutateConfig (0.01s)
=== RUN   TestResolve_WindowsDriveLetterCase
--- SKIP: TestResolve_WindowsDriveLetterCase (0.00s)
=== RUN   TestResolve_NilConfig
--- PASS: TestResolve_NilConfig (0.00s)
ok  	github.com/daileyo/omgitworks/internal/repocontext	0.181s
~~~

## Artifact: New internal/git helpers

**What it proves:** `Toplevel` delegates the upward directory walk to git itself, so a
directory three levels deep reports the checkout root, and a non-git directory errors.
`ListBranches` returns local branch names for the completion work in task 4.0.

**Why it matters:** `gitCommand` is unexported, and the spec forbids the resolver from
calling `os/exec` directly. These exported wrappers are how that constraint is met.

**Command:**

~~~bash
go test -v -run 'TestToplevel|TestListBranches' ./internal/git/
~~~

**Result summary:** All 5 cases pass.

~~~text
--- PASS: TestToplevel_RepoRoot (0.01s)
--- PASS: TestToplevel_Subdirectory (0.01s)
--- PASS: TestToplevel_NotAGitRepo (0.00s)
--- PASS: TestListBranches (0.01s)
--- PASS: TestListBranches_NotAGitRepo (0.00s)
ok  	github.com/daileyo/omgitworks/internal/git	0.046s
~~~

## Artifact: Quality gates

**What it proves:** The new package satisfies the repository's lint and vet
configuration, including `gosec`, `noctx`, `unparam`, and `misspell`, with no exclusion
added to accommodate it.

**Why it matters:** CI runs `golangci-lint` v2.13.2 as a blocking job. Clearing it
locally at the same pinned version means this task will not fail the pipeline.

**Command:**

~~~bash
golangci-lint run ./internal/...
go vet ./...
~~~

**Result summary:** `0 issues.` from the linter; `go vet` produced no output, which is
its pass condition.

~~~text
=== golangci-lint run ./internal/... ===
0 issues.

=== go vet ./... ===
(no output = pass)
~~~

## Note on test isolation

The first run of the resolver suite failed every fixture with
`commit-msg: ERROR: malformed header`. A global `core.hooksPath` applies to every
repository on the machine, including throwaway fixtures in a temp directory, and
rejected the scaffolding commit message `init`.

This is a solved problem in this repository: `internal/git/hooks_isolation_test.go`
neutralizes it through `GIT_CONFIG_*`. The same helper was mirrored into
`internal/repocontext/hooks_isolation_test.go` rather than inventing a new approach.
No production code was changed to accommodate it.

## Reviewer Conclusion

The resolver meets every Unit 1 functional requirement, proven case by case rather
than in aggregate, and passes the repository's own quality gates at the version CI
pins. It is ready to be consumed by `worktree add` in task 2.0.
