# Task 03 Proofs - The STATUS column, precedence, and help text

## Task Summary

`omgw worktree list` now has `-S/--show-status`, which adds a `STATUS` column after `ALIGNED` holding the icons only, since the branch already has its own column. When `-s` and `-S` are both given, `-S` wins and `BRANCH` stays plain, matching `omgw list`. Detached worktrees keep `(detached)` in `BRANCH` and get their icons wherever the active mode puts them. The subcommand help now carries the flag convention, the filter terms, the attached-value rule, and examples.

## What This Task Proves

- `-S` renders a `STATUS` header and per-row icons with `BRANCH` unchanged.
- `-s -S` together produce the `STATUS` column and a plain `BRANCH`.
- A detached worktree shows `(detached)` with its icon in `STATUS` under `-S` and after the label under `-s`.
- Help lists `--status`, `--show-status`, and `--color`, names the four filter terms, and shows the `-s=dirty` form.

## Evidence Summary

- Four new tests (`_Full`, `_ShowStatusWins`, `_Detached`, `Help_StatusFlags`) pass alongside the task 2.0 tests.
- `make ci` exits zero.
- A transcript from the real binary shows `-S`, `-s -S`, and `--help`.

## Artifact: Fixture-backed tests

**What it proves:** Full mode, precedence, detached handling, and the help contract.

**Why it matters:** These complete the Unit 1 display requirements and the help portion of Unit 2.

**Command:**

~~~bash
go test ./cmd/omgitworks -run 'TestWorktreeListStatus|TestWorktreeListHelp' -v
~~~

**Result summary:** All status and help tests pass.

~~~text
--- PASS: TestWorktreeListStatus_Compact (0.07s)
--- PASS: TestWorktreeListStatus_CompactAligned (0.07s)
--- PASS: TestWorktreeListStatus_Color (0.08s)
--- PASS: TestWorktreeListStatus_MissingWorktree (0.07s)
--- PASS: TestWorktreeListStatus_OptIn (0.06s)
--- PASS: TestWorktreeListStatus_Full (0.07s)
--- PASS: TestWorktreeListStatus_ShowStatusWins (0.07s)
--- PASS: TestWorktreeListStatus_Detached (0.07s)
--- PASS: TestWorktreeListHelp_StatusFlags (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.586s
~~~

## Artifact: Real binary against the scratch fixture

**What it proves:** The `-S` column and `-s -S` precedence from the command line, plus the rendered help.

**Why it matters:** Shows the flag wiring, including the sentinel-valued `-S`, through cobra rather than through `runWorktreeList` directly.

**Command:** see each `$` line. Paths are rewritten to `/home/user` after capture, so the `PATH` column is sized for the original, longer paths.

**Result summary:** `-S` places `↑1 ✓`, `✓`, and `✗` in a `STATUS` column after `ALIGNED`; adding `-s` changes nothing, proving `-S` wins; the help shows both flags without leaking the sentinel default.

~~~text
$ omgw worktree list . -S
REPO     BRANCH  PATH                                                                                                                                                   ALIGNED  STATUS
-------  ------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------  ------
fixture  ahead   /home/user/.local/share/gws/projects/fixture/ahead  yes      ↑1 ✓
fixture  clean   /home/user/.local/share/gws/projects/fixture/clean  yes      ✓
fixture  dirty   /home/user/.local/share/gws/projects/fixture/dirty  yes      ✗

$ omgw worktree list . -s -S
REPO     BRANCH  PATH                                                                                                                                                   ALIGNED  STATUS
-------  ------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------  ------
fixture  ahead   /home/user/.local/share/gws/projects/fixture/ahead  yes      ↑1 ✓
fixture  clean   /home/user/.local/share/gws/projects/fixture/clean  yes      ✓
fixture  dirty   /home/user/.local/share/gws/projects/fixture/dirty  yes      ✗

$ omgw worktree list --help
List all git worktrees across tracked repositories, optionally filtered to a
single repository.

Each entry shows the repository name, branch, path, and whether the worktree
is aligned (inside the projects root).

Passing "." lists only the repository the current directory belongs to. Omitting
the argument continues to list every tracked repository. The positional
argument is always the repository, so a status filter value must be attached
to its flag: -s=dirty, -sdirty, or --status=dirty.

Git status is opt-in and follows the same flag convention as "gws list":
  -s  (lowercase) = compact status icons after the branch name
  -S  (uppercase) = a separate STATUS column holding the icons
  Either flag with a value also filters: clean, dirty, ahead, behind.
  When both are given, -S wins.

Examples:
  gws worktree list                 # List all worktrees
  gws worktree list my-repo         # List worktrees for my-repo only
  gws worktree list .               # List worktrees for the current repo
  gws worktree list . -s            # Compact status after each branch
  gws worktree list my-repo -S      # Full STATUS column for my-repo
  gws worktree list -s=dirty        # Only worktrees with uncommitted changes
  gws worktree list my-repo -S=ahead  # Only my-repo worktrees ahead of origin
  gws worktree list -s my-repo      # my-repo is the repository, not a filter

Usage:
  omgitworks worktree list [repo|.] [flags]

Flags:
      --color string         Color output: auto, always, never (default "auto")
  -h, --help                 help for list
  -S, --show-status string   Show a STATUS column, or show and filter with -S=<pattern>
  -s, --status string        Show compact status after the branch, or show and filter with -s=<pattern>

Global Flags:
  -q, --quiet   Suppress verbose output, print only the path
~~~

## Artifact: Full quality gate

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit zero.

~~~text
Tests complete
All CI checks passed!
~~~

## Reviewer Conclusion

Both display modes now exist with the same precedence rule as `omgw list`, detached worktrees are handled in each, and the help text documents the one deliberate difference from `list`: filter values attach to the flag because the positional is the repository.
