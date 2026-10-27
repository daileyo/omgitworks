# 30-spec-worktree-status

> Clarification record: the answered question round for this spec lives at `docs/specs/30-spec-worktree-status/30-questions-1-worktree-status.md`. All four questions were answered with the recommended options.
>
> Builds on: **spec 19** (`status-compact`) for the `-s`/`-S` convention and icon formatting, **spec 20** (`worktree-management`) for `omgw worktree list`, and **spec 25** (`worktree-repo-context`) for `.` targeting.

## Introduction/Overview

`omgw list -s` and `omgw list -S` show git status for every tracked repository: compact icons for behind, ahead, clean, and dirty, with optional filtering on those terms. Worktrees have no equivalent. `omgw worktree list` shows where each worktree is and whether it is aligned, but says nothing about whether it has uncommitted changes or unpushed commits. A developer with several worktrees per repository has to visit each one to find out which need attention. This feature brings the same status flags, icons, filter vocabulary, colors, parallel fetching, and cache to `omgw worktree list`, so worktree status reads the same way repository status already does.

## Goals

- Add `-s`/`--status` and `-S`/`--show-status` to `omgw worktree list` with the same meaning they have on `omgw list`
- Show the same icons, in the same colors, using the same filter terms, so nothing has to be relearned
- Fetch status for worktrees through the same cache and worker pool as `omgw list`, so the two commands are equally fast and never disagree about a path
- Rename the existing alignment column so that "status" means git status in both commands
- Keep status opt-in, so `omgw worktree list` without flags is exactly as fast and quiet as it is today

## User Stories

- **As a developer with several worktrees in one repository**, I want to see at a glance which ones have uncommitted changes so that I know where I left work unfinished.
- **As a developer about to clean up worktrees**, I want to see which ones are ahead of their remote so that I do not remove a worktree with unpushed commits.
- **As a developer who already uses `omgw list -s`**, I want `omgw worktree list -s` to look and behave the same so that I do not have to learn a second status convention.
- **As a developer with many worktrees**, I want status fetched in parallel and cached so that the command stays fast on a large workspace.
- **As a developer scripting around the output**, I want the alignment column to have an unambiguous name so that "status" is not two different things one command apart.

## Demoable Units of Work

### Unit 1: Worktree Status Display

**Purpose:** Show git status for each worktree in `omgw worktree list`, in the same shape `omgw list` uses.

**Functional Requirements:**

- The system shall rename the existing aligned/unaligned column of `omgw worktree list` from `STATUS` to `ALIGNED`, with the values `yes` and `no`.
- The system shall accept `-s`/`--status` on `omgw worktree list`. Given bare, it shall append the compact status icons (`↓n`, `↑n`, `✓` or `✗`) after the branch name inside the `BRANCH` column, with the column widened to fit and the icons aligned in a sub-column so they line up across rows, as the compact mode of `omgw list` does.
- The system shall accept `-S`/`--show-status` on `omgw worktree list`. Given bare, it shall add a `STATUS` column containing the status icons only, since the branch already has its own column.
- When both `-s` and `-S` are given, `-S` shall win and the `STATUS` column shall be shown without compact icons in `BRANCH`, matching `omgw list`.
- The system shall produce the icons with the same formatter `omgw list` uses, so icon glyphs, ordering, and colors cannot differ between the two commands.
- The system shall accept `--color auto|always|never` with the same resolution rule as `omgw list`: color on a TTY by default, forced on or off by the flag.
- A worktree whose status could not be read (for example, its directory no longer exists) shall show no icons and no `STATUS` value, and the command shall still succeed.
- A detached worktree shall continue to show `(detached)` in `BRANCH`, with icons appended after it when status is requested.
- Without a status flag, the output shall be unchanged apart from the `ALIGNED` column rename, and no git command shall run.
- The `omgw worktree list` help text and the documentation page for worktree management shall describe the status flags, the filter terms, and the renamed column, and the example output shall be updated.

**Proof Artifacts:**

- Test: the header reads `ALIGNED` and rows show `yes` or `no`, demonstrates the rename
- Test: `-s` appends icons after the branch inside the `BRANCH` column and adds no `STATUS` column, demonstrates compact mode
- Test: `-S` adds a `STATUS` column holding icons only, demonstrates full mode
- Test: `-s` and `-S` together produce the `STATUS` column and leave `BRANCH` plain, demonstrates precedence
- Test: `--color always` wraps icons in the same ANSI codes `omgw list` uses, and `--color never` leaves them plain, demonstrates color parity
- Test: a worktree with no status entry renders a blank icon area and the run exits zero, demonstrates graceful handling
- Test: with no status flag, no status is fetched, demonstrates that status remains opt-in
- CLI: transcript of `omgw worktree list .`, `omgw worktree list . -s`, and `omgw worktree list . -S` against a fixture repository with one clean, one dirty, and one ahead worktree, demonstrates the three display modes

### Unit 2: Status Filtering and Argument Handling

**Purpose:** Let the user narrow worktrees by status using the same vocabulary as `omgw list`, without breaking the existing repository argument.

**Functional Requirements:**

- `-s=<pattern>` and `--status=<pattern>` shall show compact icons and keep only worktrees whose status text matches the pattern; `-S=<pattern>` and `--show-status=<pattern>` shall do the same with the `STATUS` column.
- The status text matched against shall be built by the same helper `omgw list` uses, producing `clean` or `dirty` plus `ahead` and `behind` when applicable, and matching with the same partial, case-insensitive pattern rule.
- The positional argument of `omgw worktree list` shall always be the repository argument (a name pattern or `.`). It shall not be reassigned to a status flag, so `omgw worktree list -s my-repo` means compact status for `my-repo`. Filter values must be attached to the flag with `=` (`-s=dirty`, `--status=dirty`). *Amended during implementation (task 4.0):* the joined short form `-sdirty` was planned as a third spelling, but pflag parses the letters after a sentinel-valued short flag as further short flags and rejects it, so it is not supported.
- The help text shall show the attached form in its examples, so the one difference from `omgw list` is visible where the user looks for it.
- A worktree with no readable status shall not match any status filter and shall be omitted when a filter is active.
- Status filtering shall apply after repository scoping, so `omgw worktree list my-repo -s=dirty` lists only dirty worktrees of `my-repo`.
- When scoping and filtering together leave no worktrees, the system shall print the existing "No worktrees found" message, naming the scope label when there is one, and exit zero.

**Proof Artifacts:**

- Test: `-s=dirty` keeps only worktrees with uncommitted changes, demonstrates filtering
- Test: `-S=ahead` keeps only worktrees ahead of their remote and shows the `STATUS` column, demonstrates show-and-filter
- Test: `omgw worktree list -s my-repo` scopes to `my-repo` with compact status rather than treating `my-repo` as a filter, demonstrates the positional contract
- Test: a worktree with no status is excluded when a filter is active, demonstrates the no-status rule
- Test: a filter that matches nothing prints the no-worktrees message and exits zero, demonstrates the empty case
- CLI: transcript of `omgw worktree list -s=dirty` across the fixture workspace, demonstrates cross-repository filtering

### Unit 3: Shared Status Pipeline, Cache, and Workers

**Purpose:** Make worktree status as fast and as consistent as repository status by running both through one pipeline.

**Functional Requirements:**

- The system shall extract the status fetching pipeline currently inline in `omgw list` (load cache, serve fresh entries, fall back to stale entries, fetch uncached paths in parallel with a progress spinner, save the cache, and refresh stale entries in the background) into one shared helper that takes a list of paths, and both `omgw list` and `omgw worktree list` shall call it.
- `omgw worktree list` shall read and write the same status cache file as `omgw list`, with the same time-to-live, keyed by worktree path.
- The system shall accept `--workers N` on `omgw worktree list`, resolved with the same rule as `omgw list`: the flag, then the `status_workers` preference, then the default of 8.
- The progress spinner shall render on stderr only when stderr is a terminal, as it does for `omgw list`.
- The behavior of `omgw list -s` and `omgw list -S` shall be unchanged by the extraction, as shown by the existing `list` tests continuing to pass.
- The configuration documentation for `status_workers` shall mention that `omgw worktree list --workers` also overrides it.

**Proof Artifacts:**

- Test: status fetched by `omgw worktree list -s` is written to the cache under the worktree path, and a second run reads it without calling git, demonstrates cache sharing
- Test: `--workers` on the flag wins over the preference, and the preference wins over the default, demonstrates worker resolution
- Test: the shared helper returns fresh entries for cached paths and fetched entries for uncached paths, demonstrates the extracted pipeline
- Test: the full existing `list` test suite passes after the extraction, demonstrates that `omgw list` is unchanged
- CLI: transcript of `time omgw worktree list -s` run twice in a row, showing the second run completes without a spinner, demonstrates the cache hit

## Non-Goals (Out of Scope)

1. **Worktree status inside `omgw list`**: Expanding repositories into per-worktree sub-rows was declined (round 1, question 1, option B). `omgw list` continues to show only the `(wt)` presence marker.
2. **JSON output for `omgw worktree list`**: Declined (round 1, question 4, option D). `worktree list` keeps its table-only output; `-o json` would be a separate spec.
3. **Main checkout as a row**: Declined (round 1, question 4, option E). `omgw worktree list` continues to list linked worktrees only; the main checkout's status is available through `omgw list -s`.
4. **Fetching from remotes**: Ahead and behind counts come from the local tracking ref, as in `omgw list`. No network access is added.
5. **New status indicators**: No stash, untracked-only, or merge-state indicators. The icon set stays identical to `omgw list`.
6. **Changes to `omgw worktree list` scoping**: Repository name patterns and `.` continue to behave exactly as spec 25 defined them.

## Design Considerations

The output must read as the same system as `omgw list -s`. Icons, their order (`↓n ↑n` then `✓`/`✗`), and their colors (magenta behind, cyan ahead, red dirty, green clean) are reused rather than redrawn. Compact icons sit in a sub-column after the branch so a column of worktrees scans the way a column of repositories does.

Illustrative output for `omgw worktree list my-repo -s`:

```
REPO      BRANCH                PATH                                                        ALIGNED
--------  --------------------  ----------------------------------------------------------  -------
my-repo   feat-auth       ↑2 ✓  /home/user/.local/share/gws/projects/my-repo/feat-auth      yes
my-repo   fix-login          ✗  /home/user/.local/share/gws/projects/my-repo/fix-login      yes
my-repo   experiment      ↓1 ✓  /tmp/experiment                                             no
```

Illustrative output for `omgw worktree list my-repo -S`:

```
REPO      BRANCH      PATH                                                        ALIGNED  STATUS
--------  ----------  ----------------------------------------------------------  -------  ------
my-repo   feat-auth   /home/user/.local/share/gws/projects/my-repo/feat-auth      yes      ↑2 ✓
my-repo   fix-login   /home/user/.local/share/gws/projects/my-repo/fix-login      yes      ✗
my-repo   experiment  /tmp/experiment                                             no       ↓1 ✓
```

## Repository Standards

- **Go conventions**: Changes live in `cmd/omgitworks/worktree_list.go` and `cmd/omgitworks/list.go`, with the shared pipeline helper in a new file alongside them or in `internal/git`, whichever keeps the import graph clean.
- **Cobra structure**: Flags registered in `init()`; the sentinel constant for "flag present without a value" is reused rather than redefined.
- **Flag variables**: `list.go` already owns package-level `flagStatus`, `flagShowStatus`, `flagWorkers`, and `flagColor`. The worktree list flags need their own variables so the two commands cannot clobber each other.
- **Testing**: Go standard `testing`, `t.TempDir()` with real git repositories for status, `bytes.Buffer` output capture, and `t.Setenv` for the config home, following `worktree_list_test.go`.
- **Commit messages**: Conventional commits with the aligned-scope style used in this repository: `refactor(list):` for the pipeline extraction, `feat(worktree):` for the flags, `docs(worktree):` for the documentation.
- **CI**: `go vet`, `go test -race`, and golangci-lint must stay green.

## Technical Considerations

- **The positional argument is the one place the mirror is not exact.** `omgw list -s dirty` works because `list` has no other use for a positional and reassigns a trailing argument to the last bare status flag. `worktree list` already uses its positional for the repository, so the same trick would make `omgw worktree list -s my-repo` ambiguous. The spec resolves this by keeping the positional as the repository and requiring attached filter values (`-s=dirty`, `--status=dirty`). These attached forms also work on `omgw list`, so a user who learns them has one habit that works everywhere.
- **Extracting the pipeline is the main design work.** The two-phase cache logic in `runList` (fresh, then stale fallback, then foreground fetch of the rest, then background refresh) is about sixty lines and has subtle ordering: the cache is saved after the foreground fetch and the background prefetch is awaited on exit. The extraction must preserve both, and the existing `list` tests are the regression net. Spec 28 took the same approach for worktree discovery.
- **Cache keys are paths, so sharing is free.** Worktree paths are distinct from repository paths, so entries never collide, and a `omgw list -s` followed by `omgw worktree list -s` shares nothing but the file. The cache file location is unchanged by this spec.
- **Ahead/behind uses the worktree's own branch.** `git.GetStatus` runs `rev-parse --abbrev-ref HEAD` in the given directory, so a worktree reports its checked-out branch, not the main repository's. No change to `internal/git/status.go` is expected.
- **Column width with ANSI codes.** When color is on, widths must be computed on the visible text, using the existing `displayWidth` helper, or columns drift. `omgw list` already handles this; the worktree table must use the same helper.
- **The `ALIGNED` rename is a visible output change.** Two existing tests assert on `aligned` and `(unaligned)` text and must be updated. There is no JSON for `worktree list`, so no script contract breaks.
- **No external standards research was needed.** The work is local git status through existing helpers and cobra patterns already established in the repository.

## Security Considerations

No specific security considerations identified. Status is read with local git commands in directories already recorded in `config.json`; no network access, credentials, or tokens are involved. The shared cache file already exists and gains worktree paths, which are no more sensitive than the repository paths it already holds. Proof artifact transcripts should be produced against a scratch fixture workspace so they do not disclose private repository or branch names.

## Success Metrics

1. **`omgw worktree list -s` and `-S` produce the same icons and colors as `omgw list -s` and `-S`** for a worktree in the same state, confirmed by shared formatter tests.
2. **A second `omgw worktree list -s` within the cache time-to-live runs no git status commands**, confirming cache sharing.
3. **`omgw worktree list -s my-repo` scopes to `my-repo`**, confirming the positional contract is intact.
4. **The status pipeline exists in exactly one place** after implementation, and the full existing `list` test suite passes unchanged.
5. **`omgw worktree list` with no flags runs no git commands**, confirming status stays opt-in.
6. **Test coverage** for the rename, both display modes, precedence, color, filtering, the positional contract, no-status rows, cache sharing, worker resolution, and the extracted pipeline.

## Open Questions

Both items below are **resolved** by accepting the stated assumptions. Neither affects scope, Demoable Units, or Proof Artifacts, and either can be flipped during task planning without touching the rest of the spec.

1. **Resolved — `ALIGNED` values are `yes` and `no`.** Round 1 question 2 left the values open. With the header now naming the property, short boolean values read better than repeating `aligned` and `(unaligned)`.

2. **Resolved — filter values attach to the flag.** Because the positional stays the repository argument, `-s dirty` with a space is not supported on `worktree list`; `-s=dirty` and `--status=dirty` are. (The joined `-sdirty` form turned out not to parse; see the amendment in Unit 2.) If space-separated filter values matter more than an unambiguous repository argument, this can be revisited, but it would require a rule for deciding whether a bare word is a repository or a status pattern.
