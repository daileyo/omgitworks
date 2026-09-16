# 28-spec-worktree-refresh

> Clarification record: the answered question rounds for this spec live at `docs/specs/25-spec-worktree-repo-context/25-questions-1-worktree-repo-context.md` and `25-questions-2-worktree-repo-context.md`. Round 1 questions 4, 6, and 7 govern this spec.
>
> Depends on: **spec 25** (`worktree-repo-context`) for the current-repository resolver. Shares the targeting precedence model defined in **spec 26**.

## Introduction/Overview

Worktree metadata in `config.json` is only updated by a full `omgw refresh`, which re-scans the entire workspace, re-detects git users, and clears the status cache. A user who creates or deletes a worktree outside omgitworks has no cheap way to bring the stored data back in line for one repository or one group of repositories. This feature adds `omgw worktree refresh`, a scoped metadata re-sync that repairs, prunes, and re-discovers worktrees for the targeted repositories and nothing else.

## Goals

- Add `omgw worktree refresh` covering individual, current-directory, tag-scoped, and whole-workspace targeting
- Re-sync worktree metadata without the cost or side effects of a full `omgw refresh`
- Repair recoverable worktree links before pruning dead ones, so nothing salvageable is discarded
- Report what actually changed, rather than reporting only that a refresh ran
- Leave everything a full refresh touches but worktree data — repository discovery, user detection, status cache — untouched

## User Stories

- **As a developer who created a worktree with plain git**, I want to re-sync one repository's worktree data so that omgitworks sees it without re-scanning my whole workspace.
- **As a developer who deleted a worktree directory by hand**, I want the stale entry cleared so that `omgw worktree list` stops showing something that no longer exists.
- **As a developer whose worktrees were broken by moving a repository**, I want recoverable links repaired rather than pruned so that I do not lose worktrees that are merely mislinked.
- **As a developer maintaining a group of repositories**, I want to re-sync every repository carrying a tag in one command so that a group stays consistent.
- **As a developer**, I want to see what a refresh would change before it changes anything so that I can understand an unexpected result.

## Demoable Units of Work

### Unit 1: Scoped Worktree Metadata Re-sync

**Purpose:** Deliver the core operation — bring stored worktree data back in line with git reality for a chosen set of repositories.

**Functional Requirements:**

- The system shall provide `omgw worktree refresh [repo]`.
- For each targeted repository, the system shall run `git worktree repair`, then `git worktree prune`, then `git worktree list --porcelain`, in that order.
- The system shall rebuild each targeted repository's stored `Worktrees` entries from the re-discovered list, recomputing the `Aligned` value for each entry against the XDG projects root.
- The system shall exclude any discovered worktree whose path no longer exists on disk, matching the behavior of the existing workspace-wide discovery.
- When a targeted repository ends with no worktrees, the system shall clear its stored worktree data rather than leaving stale entries.
- The system shall save the configuration once at the end of the run, not once per repository.
- The system shall not re-scan the workspace for new repositories, shall not run git user detection, and shall not clear the git status cache, since those belong to the full `omgw refresh`.
- A repository whose repair, prune, or list step fails shall be reported and skipped, with the run continuing to the remaining repositories.
- The system shall exit non-zero when one or more repositories failed, and zero otherwise.

**Proof Artifacts:**

- Test: a worktree created outside omgitworks appears in stored data after refresh, demonstrates discovery of external additions
- Test: a worktree whose directory was deleted by hand is removed from stored data after refresh, demonstrates stale-entry clearing
- Test: repair runs before prune, demonstrating that a recoverable worktree survives rather than being pruned, demonstrates the ordering requirement
- Test: `Aligned` is recomputed, so a worktree moved into the projects root is reported aligned after refresh, demonstrates correct recomputation
- Test: a repository ending with no worktrees has its stored data cleared, demonstrates the empty case
- Test: the status cache and repository list are unchanged by a worktree refresh, demonstrates scope containment
- CLI: transcript showing `omgw worktree list` before a hand-made change, the change, `omgw worktree refresh`, and `omgw worktree list` after, demonstrates the feature end to end

### Unit 2: Targeting and Change Reporting

**Purpose:** Let the user choose what to re-sync and see what it actually did.

**Functional Requirements:**

- `omgw worktree refresh` with no arguments and no tag shall refresh every tracked repository.
- `omgw worktree refresh <repo>` shall refresh repositories matching the name pattern, using `filter.MatchesPattern`.
- `omgw worktree refresh .` shall refresh the repository resolved from the current working directory, per spec 25.
- `omgw worktree refresh -t <tag>` shall refresh every repository carrying the tag, matched with `filter.MatchesExact` and accepting a single tag value.
- `omgw worktree refresh <repo> -t <tag>` shall refresh repositories satisfying both the name pattern and the tag, matching the AND semantics defined in spec 26.
- When a name pattern or tag matches no repositories, the system shall exit non-zero with an error naming the filters used.
- The system shall report, per repository that changed, the worktrees added to and removed from stored data, and any entry whose `Aligned` value changed.
- The system shall print a final summary giving the number of repositories refreshed, the number changed, and the number that failed.
- A repository whose stored data did not change shall not be listed individually in the output.

**Proof Artifacts:**

- Test: each targeting form selects the expected repositories, demonstrates the targeting contract
- Test: name pattern combined with a tag applies AND logic, demonstrates consistency with spec 26
- Test: an unmatched filter exits non-zero naming the filters, demonstrates the error contract
- Test: output lists added, removed, and realigned worktrees per changed repository, demonstrates change reporting
- Test: an unchanged repository produces no per-repository output, demonstrates quiet-when-nothing-happened behavior
- CLI: transcript of `omgw worktree refresh -t <tag>` across tagged repositories showing the per-repository change lines and the summary, demonstrates scoped bulk refresh

### Unit 3: Dry Run

**Purpose:** Let the user see what a refresh would change before it writes anything.

**Functional Requirements:**

- The system shall accept a `--dry-run` flag that reports the changes a refresh would make and exits without saving configuration.
- In dry-run mode the system shall not run `git worktree repair` or `git worktree prune`, since both mutate git state, and shall base its report on `git worktree list --porcelain` plus a check of each path's existence on disk.
- The dry-run output shall state that repair and prune were not run, so the user understands the preview may differ where a repair would have salvaged an entry.
- The dry-run output shall use the same per-repository change format and final summary as a real run, with the counts framed as what would change.
- `--dry-run` shall leave configuration, git state, and the filesystem untouched.

**Proof Artifacts:**

- Test: `--dry-run` reports pending changes and leaves stored configuration byte-identical, demonstrates preview safety
- Test: `--dry-run` does not invoke repair or prune, demonstrates that the preview is non-mutating
- Test: dry-run output includes the repair-and-prune caveat, demonstrates the honesty requirement
- CLI: transcript of `omgw worktree refresh --dry-run` followed by the real run, demonstrates that the preview matched the outcome

## Non-Goals (Out of Scope)

1. **Fetching or pulling**: Refresh never touches network state or updates branch contents (round 1, question 4, options B and D declined). A `worktree sync` or `--fetch` capability would be a separate feature.
2. **Recreating worktrees**: Refresh never removes and re-adds a worktree (round 1, question 4, option C declined).
3. **Moving worktrees**: Alignment is recomputed and reported, never acted on. Moving remains `omgw worktree align`.
4. **Confirmation prompts**: Round 1 question 6's confirmation gate covers the destructive removal flow. Refresh only rewrites stored metadata to match git reality, so it gets `--dry-run` but no prompt.
5. **Full workspace refresh behavior**: Repository discovery, user detection, and status cache clearing stay with `omgw refresh`.
6. **Repeatable tags**: `--tag` takes a single value (round 1, question 2a).

## Design Considerations

No new visual design is introduced. Output follows the repository's established convention of reporting only what changed and then a counted summary, as the existing `omgw refresh` does with its conditional summary lines.

## Repository Standards

- **Go conventions**: New command file `cmd/omgitworks/worktree_refresh.go`, reusing the git helpers already in `internal/git/worktree.go`.
- **Cobra structure**: Registered through `init()` with `worktreeCmd.AddCommand`.
- **Shared discovery logic**: The repair-prune-list-rebuild sequence already exists three times, in `refresh.go`'s `discoverWorktrees`, in `worktree_add.go`, and in `worktree_align.go`. This spec should extract one shared helper and have all call sites use it, rather than adding a fourth copy.
- **Testing**: Go standard `testing`, `t.TempDir()` with real git repositories, `bytes.Buffer` output capture.
- **Commit messages**: Conventional commits; `feat:` for the new command.

## Technical Considerations

- **Naming overlap with `omgw refresh`**: Round 1 question 4 chose metadata re-sync precisely so that `refresh` means the same thing at both levels. The help text should still state explicitly that `worktree refresh` updates worktree data only, so users do not expect repository discovery from it.
- **Extracting the shared discovery helper is the main design work**: The three existing copies of the repair-prune-list-rebuild sequence differ subtly. `refresh.go` skips worktrees whose paths no longer exist and clears the field when none remain; `worktree_add.go` does neither and silently ignores list errors. Consolidating them is a behavior change for the `add` path, which must be covered by tests rather than assumed harmless.
- **Change detection requires a before-and-after comparison**: Reporting added, removed, and realigned entries means capturing each repository's stored worktrees before the rebuild and diffing against the result. Comparison should be keyed on path, since a branch can be absent for a detached HEAD.
- **Dry run cannot be a faithful preview**: `repair` and `prune` mutate git state, so a non-mutating preview must skip them. The consequence is that a broken-but-repairable worktree will appear in the dry run as it currently stands rather than as repair would leave it. The spec requires stating this in the output rather than papering over it.
- **Repair before prune is load-bearing**: Prune removes entries whose directories are gone, and repair fixes entries whose git pointers are broken but whose directories exist. Reversing the order discards worktrees that were recoverable. The existing code already orders them this way; the requirement records why, so it is not "simplified" later.
- **Single configuration save**: As in specs 26 and 27, one load and one save across the run, or repositories processed early are lost to a later save of stale data.
- **Dependency on spec 25**: The `.` targeting form requires the current-repository resolver. If implemented before spec 25, that form must be deferred rather than reimplemented.

## Security Considerations

No specific security considerations identified. All operations are local git metadata and filesystem reads, with writes confined to `config.json`. No credentials, tokens, or network access are involved. Refresh does not delete worktree directories or branches, so it carries no data-loss risk beyond rewriting stored metadata to match what git already reports. Proof artifact transcripts should be produced against a scratch fixture workspace so they do not disclose unrelated private repository names.

## Success Metrics

1. **A worktree created or deleted outside omgitworks is reflected in stored data after a scoped refresh**, without a full workspace scan.
2. **A repairable worktree survives a refresh**, confirming repair runs before prune.
3. **A worktree refresh leaves the repository list, user data, and status cache unchanged**, confirming scope containment.
4. **`--dry-run` leaves stored configuration byte-identical** while reporting the same changes the real run then makes.
5. **The repair-prune-list-rebuild sequence exists in exactly one place in the codebase** after this spec is implemented, down from three.
6. **Test coverage** for each targeting form, external additions and deletions, repair ordering, alignment recomputation, change reporting, dry run, partial failure, and exit status.

## Open Questions

Both questions below are **resolved** by accepting the stated assumptions, consistent with the resolutions recorded in specs 25, 26, and 27. Neither affects implementation planning. No blocking questions remain.

1. **Resolved — consolidate the shared discovery helper within this spec.** This spec would otherwise add a fourth copy of the repair-prune-list-rebuild sequence. Consolidation does change behavior in the `add` path, which is covered by spec 26, so that change must be covered by tests rather than assumed harmless. If the refactor is later preferred as a standalone change, it can be split out during task planning without altering any requirement in this spec.

2. **Resolved — no verbose flag.** Repositories whose stored data did not change produce no output. Silence is the default and no "no changes" line is specified.
