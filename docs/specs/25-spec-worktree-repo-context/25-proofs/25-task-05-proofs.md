# Task 05 Proofs - Documentation and shell-integration guard

## Task Summary

The published command reference now documents every new invocation form, and the shell
integration is proven untouched. This task also runs the full repository gate over the
complete change.

## What This Task Proves

- `docs/site/commands-core.md` documents the omitted-repository form of `worktree add`,
  `.` on `list` and `align`, worktree-relative resolution, and the failure error.
- The resolution rules are documented once in a shared section and cross-referenced,
  rather than repeated three times.
- `cmd/omgitworks/shellinit.go` is byte-identical to `main`, and all seven shell
  template tests pass.
- In-tool `--help` output matches the documentation.
- `make ci` — vet, lint, and the race-enabled suite — passes on the complete change.

## Evidence Summary

- `git diff --stat main...HEAD -- cmd/omgitworks/shellinit.go` is empty.
- 7 shell template tests pass.
- `make ci` exits 0: 397 top-level tests pass (1034 including sub-tests), 0 fail, and
  10 skip — 1 the Windows-only drive-letter case, 9 pre-existing `pwsh not installed`
  sub-cases in a file this branch never touched.

## Artifact: Documentation diff

**What it proves:** The new forms are published, not just implemented.

**Why it matters:** The spec's Unit 3 requires the site to match behavior. Documentation
drift is what spec 18 was created to fix, so this repository treats it as part of the
change rather than a follow-up.

**Artifact path:** `docs/site/commands-core.md`

**Result summary:** Three command sections gained the new argument forms and examples,
and a new `Current-Repository Targeting` section explains the resolution rules once.
Both `list` and `align` state explicitly that omitting the argument still means all
repositories, and the section closes by naming the two behaviors that deliberately did
not change.

Sections changed:

- **List Worktrees** — signature is now `omgw worktree list [repo|.]`, with a `.`
  example and a note that the bare form still means all repositories.
- **Add Worktree** — signature is now `omgw worktree add [repo] <branch>`, with a
  `cd` + omitted-argument example, a statement that an explicit repo always wins, and a
  note on how tab completion follows the same rule.
- **Align Worktrees** — signature is now `omgw worktree align [repo|.] [--dry-run]`,
  with `.` and `. --dry-run` examples.
- **Current-Repository Targeting** (new) — how the repository is determined, the
  worktree-relative rule, symlink handling, the exact error text, the `omgw refresh`
  caveat for worktrees created outside omgitworks, and the two non-goals.

## Artifact: shellinit.go regression guard

**What it proves:** No shell-function behavior changed, as the spec requires.

**Why it matters:** The shell function is what makes navigation work at all. A silent
change there would break `omgw` for every user, and nothing else in the suite would
catch it.

**Command:**

~~~bash
git diff --stat main...HEAD -- cmd/omgitworks/shellinit.go
go test -v -run TestShellTemplates ./cmd/omgitworks/
~~~

**Result summary:** The diff is empty — the file is byte-identical to `main`. All seven
template tests pass.

~~~text
=== shellinit.go changed on this branch? ===
(empty above = unchanged since main)
=== shell template tests ===
--- PASS: TestShellTemplatesRouteSubcommands (0.00s)
--- PASS: TestShellTemplatesContainBinPlaceholder (0.00s)
--- PASS: TestShellTemplatesContainNavigationFallthrough (0.00s)
--- PASS: TestShellTemplatesContainWorktreeNavigation (0.00s)
--- PASS: TestShellTemplatesContainParentNavigation (0.00s)
--- PASS: TestShellTemplatesContainCdNavigation (0.00s)
--- PASS: TestShellTemplatesKeepPrintWorkspacePassthrough (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.007s
~~~

## Artifact: --help cross-check

**What it proves:** The binary's own help agrees with the published documentation, so a
user gets the same answer either way.

**Command:**

~~~bash
make build
./build/omgitworks worktree add --help
./build/omgitworks worktree list --help
./build/omgitworks worktree align --help
~~~

**Result summary:** All three usage lines match the documented signatures:
`worktree add [repo] <branch>`, `worktree list [repo|.]`, and
`worktree align [repo|.]`. Each `Long` text carries the same rules as the docs.

~~~text
Usage:
  omgitworks worktree add [repo] <branch> [flags]

Usage:
  omgitworks worktree list [repo|.] [flags]

Usage:
  omgitworks worktree align [repo|.] [flags]
~~~

## Artifact: Full repository gate

**What it proves:** The complete change passes every gate CI enforces.

**Why it matters:** `make ci` is `go vet` plus `golangci-lint` plus `go test -race` over
the whole module. It is the same set CI runs, and the linter was installed at v2.13.2 to
match the version pinned in `.github/workflows/ci.yml`.

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit 0 across all 9 packages. 397 top-level tests pass, 1034
including sub-tests, and none fail. Of the 10 skips, 1 is this spec's Windows-only
drive-letter case; the other 9 are pre-existing `pwsh not installed` sub-cases in
`TestShellWrapperWorktreePassthrough`, whose file is unchanged on this branch.

~~~text
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	5.998s
ok  	github.com/daileyo/omgitworks/internal/classifier	1.028s
ok  	github.com/daileyo/omgitworks/internal/config	1.047s
ok  	github.com/daileyo/omgitworks/internal/discovery	1.108s
ok  	github.com/daileyo/omgitworks/internal/filter	1.041s
ok  	github.com/daileyo/omgitworks/internal/git	1.536s
ok  	github.com/daileyo/omgitworks/internal/repocontext	1.250s
ok  	github.com/daileyo/omgitworks/internal/user	1.134s
ok  	github.com/daileyo/omgitworks/internal/xdg	1.026s
All CI checks passed!
~~~

## Security check

All five proof files were scanned for credentials and for absolute home paths:

~~~bash
grep -rniE "(api[_-]?key|token|password|secret|bearer)" docs/specs/25-spec-worktree-repo-context/25-proofs/
grep -rn "/home/<user>" docs/specs/25-spec-worktree-repo-context/25-proofs/
~~~

Both return no matches. CLI transcripts were captured in a sandbox with isolated
`XDG_CONFIG_HOME` and `XDG_DATA_HOME`, and paths are shown relative to `<sandbox>`.

## Reviewer Conclusion

Published documentation matches the shipped behavior and the binary's own help; the
shell integration is provably untouched; and the whole change clears the repository's
full CI gate.
