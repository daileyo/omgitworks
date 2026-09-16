# Task 06 Proofs - `worktree refresh` is documented, discoverable, and usable through `omgw`

## Task Summary

The command is documented in its own help, in the `worktree` subcommand list, and in the
documentation site. The help text says explicitly that it updates worktree data only, so
it isn't mistaken for the workspace-wide `omgw refresh`.

While documenting the shell integration, a bug turned up: the `omgw` shell function would
have made the command unusable. The function routes any `worktree` subcommand it doesn't
recognize to the *navigation* path, which appends `-q`, captures stdout as a directory,
and `cd`s. Through `omgw`, `worktree refresh` lost its entire report. The fix adds
`refresh` to the passthrough list in all three shell wrappers.

## What This Task Proves

- `omgw worktree refresh --help` covers the operation, the worktree-data-only scope, repair
  before prune, `--dry-run`, all targeting forms, and examples.
- `omgw worktree --help` lists `refresh` among the subcommands.
- The documentation site has a `Refresh Worktrees` section and builds under `--strict`, and
  every anchor the new text links to exists in the rendered HTML.
- Through the `omgw` shell function, `worktree refresh` reaches the binary unchanged and
  prints its output, without `-q` and without a `cd`. The tests fail against the original
  wrappers.

## Evidence Summary

- Both help outputs, captured from the built binary.
- `mkdocs build --strict` succeeds. The anchors `#refresh-worktrees` and `#refresh-workspace`
  exist and are linked.
- 3 new wrapper exec-test cases pass in zsh and bash, and the template test now checks all
  three wrappers. Both fail against the original `shellinit.go`.
- `make ci` passes: vet, golangci-lint with 0 issues, race-detector tests.

## Artifact: Command help

**What it proves:** The help covers the spec's naming-overlap concern: "the help text should
still state explicitly that `worktree refresh` updates worktree data only". It also shows
the four targeting forms and `--dry-run`, and that `refresh` is listed under `worktree`.

**Command:** The built binary, run with an isolated `HOME`.

~~~text
$ omgitworks worktree refresh --help
Re-sync stored worktree data with what git reports.

For each repository, runs git worktree repair, then git worktree prune, then
git worktree list, and rebuilds the stored worktree entries from the result.
Worktrees created or deleted outside omgitworks are picked up, and whether
each worktree sits in the projects root is recomputed. Each repository whose
stored data changed is listed with what changed, followed by a summary.

This updates worktree data only. Discovering new repositories, re-detecting
git users, and clearing the status cache remain with 'omgw refresh'.

Repair runs before prune so that a worktree whose link is broken but whose
directory still exists is fixed rather than discarded. Use --dry-run to see
what would change: it runs neither repair nor prune, since both change git
state, and saves nothing.

Targeting:

  refresh                   every tracked repo
  refresh <repo>            every repo matching the name pattern
  refresh .                 the repo of the current directory
  refresh -t <tag>          every repo carrying the tag
  refresh <repo> -t <tag>   repos matching the name AND the tag

Unlike 'worktree add' and 'worktree remove', omitting the repo argument means
every repository, not the current one, and a name pattern matching several
repositories refreshes all of them.

Examples:
  gws worktree refresh                   # Re-sync every repo
  gws worktree refresh my-repo           # Only repos matching my-repo
  gws worktree refresh .                 # Only the current repo
  gws worktree refresh -t backend        # Every repo tagged backend
  gws worktree refresh --dry-run         # Preview without changing anything

Usage:
  omgitworks worktree refresh [repo|.] [flags]

Flags:
      --dry-run           Preview changes without writing them
  -h, --help              help for refresh
  -t, --tag stringArray   Select repositories by tag (single value; not repeatable)

Global Flags:
  -q, --quiet   Suppress verbose output, print only the path
(exit 0)

$ omgitworks worktree --help
Manage git worktrees across your workspace.

When called with a branch pattern, navigates to the matching worktree:
  gws worktree feat-auth                # Same as: gws worktree navigate feat-auth
  gws worktree "feat-*"                 # Wildcard match with interactive selection

Subcommands:
  gws worktree navigate <branch>        # Navigate to a worktree by branch name
  gws worktree list [repo|.]            # List all worktrees (. means the current repo)
  gws worktree add [repo] <branch>      # Create a new worktree in the projects root
  gws worktree align [repo|.]           # Move unaligned worktrees into the projects root
  gws worktree remove [repo] <branch>   # Remove a worktree (alias: rm)
  gws worktree refresh [repo|.]         # Re-sync stored worktree data with git

Usage:
  omgitworks worktree [branch-pattern] [flags]
  omgitworks worktree [command]

Available Commands:
  add         Create a new worktree in the projects root
  align       Move unaligned worktrees into the projects root
  list        List worktrees across tracked repositories
  navigate    Navigate to a worktree by branch name
  refresh     Re-sync stored worktree data with git
  remove      Remove a worktree

Flags:
  -h, --help    help for worktree
  -q, --quiet   Suppress verbose output, print only the path

Use "omgitworks worktree [command] --help" for more information about a command.
(exit 0)
~~~

**Result summary:** The refresh help states "This updates worktree data only", names what
remains with `omgw refresh`, and lists every targeting form, `--dry-run`, and examples.
The `worktree` help lists `refresh` both in its hand-written subcommand list and in cobra's
generated command list.

The inherited `-q, --quiet` global flag ("print only the path") also appears, as it does for
`list` and `align`. It comes from `worktreeCmd`'s persistent flags and is unchanged by this
task.

## Artifact: Documentation site

**What it proves:** The documentation builds cleanly and its internal links resolve.

**Why it matters:** CI builds the site with `mkdocs build`. `--strict` turns warnings into
failures, which is a stronger check.

**Command:**

~~~bash
python3 -m venv <scratch>/docsvenv
<scratch>/docsvenv/bin/pip install -r docs/requirements.txt   # mkdocs 1.5.3, mkdocs-material 9.5.18
<scratch>/docsvenv/bin/python -m mkdocs build --strict -d <scratch>/site-build
~~~

**Result summary:** The build succeeds with no warnings.

~~~text
INFO    -  Cleaning site directory
INFO    -  Building documentation to directory: <scratch>/site-build
INFO    -  Documentation built in 0.22 seconds
~~~

**Anchor check:** MkDocs 1.5.3 doesn't validate in-page anchors, so the rendered
`commands-core/index.html` was checked directly. Both linked anchors exist, and the new table
and dry-run example rendered.

~~~text
refresh-worktrees              id present: 1
refresh-workspace              id present: 1
current-repository-targeting   id present: 1
--- links to those anchors across the whole site
      3 href="#refresh-workspace"
      3 href="#refresh-worktrees"
~~~

**What changed in the docs:**

| File | Change |
| --- | --- |
| `docs/site/commands-core.md` | New `Refresh Worktrees` section after `Remove Worktrees`: usage, examples, what it runs, the worktree-data-only scope, a targeting table, example output with each change kind, repair-before-prune, and dry run. `Refresh Workspace` now points to it. `Current-Repository Targeting` now includes `refresh` and recommends it over a full `omgw refresh` for an unrecorded worktree. |
| `docs/site/configuration.md` | Names `worktree refresh` among the commands that update worktree data. |
| `docs/site/shell-integration.md` | Adds `refresh` to the list of `worktree` subcommands passed through without `cd`. |

## Artifact: The shell wrapper routes refresh correctly

**What it proves:** `omgw worktree refresh`, as users actually type it through the shell
function, runs the binary with the user's arguments and shows its output.

**Why it matters:** Without this fix the feature was effectively unusable through `omgw`.
The unit tests call Go functions directly and the CLI transcripts call the binary, so
neither could catch it.

**How it was found:** `shell-integration.md` lists which `worktree` subcommands run "(no
cd)". The generated wrappers hard-code that list: `list|align|add` in the zsh and bash
templates, and `'list', 'align', 'add'` in PowerShell. A temporary probe using the
repository's own exec-test harness then confirmed the routing in both available shells:

~~~text
zsh:  "worktree refresh"            -> invoked "worktree refresh -q", stdout "", cwd changed
zsh:  "worktree refresh --dry-run"  -> invoked "worktree refresh --dry-run -q", stdout "", cwd changed
bash: identical
~~~

**The new tests fail against the original wrappers.** The failing cases were added before
the fix. Without a terminal, the navigation path even fails on `/dev/tty` before running
the binary:

~~~text
shellinit_exec_test.go:233: stdout = "", want it to contain "stub refresh"
shellinit_exec_test.go:236: unexpected stderr: "omgw:18: no such device or address: /dev/tty\n"
shellinit_exec_test.go:242: binary invoked with "", want "worktree refresh" (no -q)
~~~

The template string test, extended to require `refresh`, fails for all three shells when
`shellinit.go` is restored from `HEAD`:

~~~text
shellinit_test.go:90: zsh template missing worktree subcommand passthrough
shellinit_test.go:90: bash template missing worktree subcommand passthrough
shellinit_test.go:117: powershell template missing worktree subcommand passthrough
~~~

**With the fix:**

~~~bash
go test ./cmd/omgitworks -count=1 -v -run 'TestShellWrapperWorktreePassthrough|TestShellTemplatesContainWorktreeNavigation'
~~~

~~~text
--- PASS: TestShellWrapperWorktreePassthrough (3.71s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_-h (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_--help (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_--version (0.24s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_navigate_--help (0.25s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_feat-auth_-h (0.25s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_list (0.25s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_align (0.25s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_add_my-repo_feat-auth (0.27s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_refresh (0.24s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_refresh_--dry-run (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_refresh_my-repo_-t_backend (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_-h (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_--help (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_--version (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_navigate_--help (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_feat-auth_-h (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_list (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_align (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_add_my-repo_feat-auth (0.06s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_refresh (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_refresh_--dry-run (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_refresh_my-repo_-t_backend (0.04s)
    --- SKIP: TestShellWrapperWorktreePassthrough/powershell/worktree_refresh (0.03s)
    --- SKIP: TestShellWrapperWorktreePassthrough/powershell/worktree_refresh_--dry-run (0.04s)
    --- SKIP: TestShellWrapperWorktreePassthrough/powershell/worktree_refresh_my-repo_-t_backend (0.04s)
--- PASS: TestShellTemplatesContainWorktreeNavigation (0.00s)
    --- PASS: TestShellTemplatesContainWorktreeNavigation/zsh (0.00s)
    --- PASS: TestShellTemplatesContainWorktreeNavigation/bash (0.00s)
    --- PASS: TestShellTemplatesContainWorktreeNavigation/powershell (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	3.717s
~~~

**PowerShell coverage limit:** `pwsh` isn't installed here, so the PowerShell exec cases
skip, as they already did for the existing `list`, `align`, and `add` cases. The PowerShell
list is guarded only by the template string test, which verifies the change is present
but not how it behaves. The edit mirrors the existing entries exactly.

**`remove` / `rm`:** the same routing bug affected spec 27's `worktree remove`: its removal
plan and confirmation preview were captured instead of shown. It was reported to the session
that owns spec 27, with the probe evidence and exact line references, and not fixed here.
That session fixed it on spec 27 in `13db8d0` (PR #102).

## Artifact: Rebase onto spec 27's routing fix

**What happened:** `feat/worktree-refresh` was rebased from `9bae1a6` onto `13db8d0` with
`git rebase --onto 13db8d0 9bae1a6 feat/worktree-refresh`, using the SHA pair sent by the
spec 27 session, with the user's approval. Eight commits replayed cleanly. This task's
commit conflicted in exactly the three expected files, because both fixes extended the same
lists. Every conflict was the list-union shape the user approved, resolved by keeping both
sides:

| Location | Resolved to |
| --- | --- |
| `shellinit.go` zsh and bash lists | `list\|align\|add\|remove\|rm\|refresh\|""\|-*` |
| `shellinit.go` PowerShell list | `'list', 'align', 'add', 'remove', 'rm', 'refresh'` |
| Exec-test stub case line | `…"worktree remove"\|"worktree rm"\|"worktree refresh"` |
| `TestShellWrapperWorktreePassthrough` cases | all three `remove`/`rm` cases and all three `refresh` cases |
| `shell-integration.md` passthrough sentence and examples | both sets |

Two further edits the conflict markers didn't show. The `shellinit_test.go` string checks
had merged automatically but required the contiguous substring `list|align|add|refresh`, so
they were updated to the union. And spec 27 had added `remove|rm` to the two example function
listings in `shell-integration.md` without conflict, so `refresh` was added there too.

**No loss check:** against `13db8d0`, every changed line in `shellinit.go` and
`shell-integration.md` replaces spec 27's line with a superset that adds `refresh`. No
removal of spec 27's content remains.

**Result on the rebased tree:** all six `remove`/`rm`/`refresh` passthrough cases pass in
zsh and bash, navigation still captures the destination and `cd`s, `make ci` passes, and
`mkdocs build --strict` succeeds.

~~~text
--- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_remove_my-repo_feat-auth (0.25s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_rm_feat-auth (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_remove_-t_backend_feat-auth_--dry-run (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_refresh (0.24s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_refresh_--dry-run (0.26s)
    --- PASS: TestShellWrapperWorktreePassthrough/zsh/worktree_refresh_my-repo_-t_backend (0.25s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_remove_my-repo_feat-auth (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_rm_feat-auth (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_remove_-t_backend_feat-auth_--dry-run (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_refresh (0.04s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_refresh_--dry-run (0.05s)
    --- PASS: TestShellWrapperWorktreePassthrough/bash/worktree_refresh_my-repo_-t_backend (0.04s)
--- PASS: TestShellWrapperWorktreeNavigation (0.35s)
    --- PASS: TestShellWrapperWorktreeNavigation/zsh (0.25s)
    --- PASS: TestShellWrapperWorktreeNavigation/bash (0.06s)
--- PASS: TestShellTemplatesContainWorktreeNavigation (0.00s)
    --- PASS: TestShellTemplatesContainWorktreeNavigation/zsh (0.00s)
    --- PASS: TestShellTemplatesContainWorktreeNavigation/bash (0.00s)
    --- PASS: TestShellTemplatesContainWorktreeNavigation/powershell (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	5.044s
~~~

## Artifact: Repository quality gate

**Command:**

~~~bash
PATH=$HOME/go/bin:$PATH make ci
~~~

**Result summary:** Vet is clean, lint reports 0 issues, and every package passes under
`-race`, including the shell exec tests. This run is on the rebased tree.

~~~text
Running linter...
0 issues.
...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	10.131s
All CI checks passed!
~~~

## Deviations From the Task Plan

- **6.5 / proof artifact:** `make docs` runs `mkdocs serve`, which never exits and would
  create `.venv` inside the repository. The evidence is `mkdocs build --strict` from a
  scratch virtualenv with the pinned requirements, plus an anchor check of the rendered
  HTML, which MkDocs 1.5.3 doesn't do itself.
- **6.4:** Beyond the new section, three existing passages that would otherwise have become
  inaccurate were updated. They are listed in the table above.
- **6.6 (added):** The shell wrapper fix described above.

## Reviewer Conclusion

`worktree refresh` is documented where users look: its own help, the `worktree` help, and
the site. The help makes the worktree-only scope explicit. A routing bug that would have
hidden all of the command's output when run through `omgw` was found while documenting
it, fixed in all three shell wrappers, and pinned by tests that fail against the original
templates.
