# 26-spec-worktree-add-by-tag

> Clarification record: the answered question rounds for this spec live at `docs/specs/25-spec-worktree-repo-context/25-questions-1-worktree-repo-context.md` and `25-questions-2-worktree-repo-context.md`. Round 1 questions 2, 2a, 3, and 3a and round 2 question 4 are the ones that govern this spec.
>
> Depends on: **spec 25** (`worktree-repo-context`), which supplies the current-repository resolver referenced in the precedence rules below.

## Introduction/Overview

Creating the same worktree across a group of related repositories currently means running `omgw worktree add` once per repository. This feature adds tag-scoped creation: `omgw worktree add -t <tag> <branch>` creates a worktree for that branch in every tracked repository carrying the tag, reporting per-repository results and continuing past individual failures.

## Goals

- Add a `-t` / `--tag` flag to `omgw worktree add` that selects repositories by tag
- Create one worktree per matched repository using the existing single-repository creation behavior
- Continue past per-repository failures and report a complete summary rather than stopping at the first error
- Define one unambiguous precedence order across the three targeting modes: explicit repository, tag, and current-directory detection
- Exit non-zero when any repository failed, so the command composes correctly in scripts

## User Stories

- **As a developer maintaining a group of related services**, I want to create a `feat-auth` worktree across every repository tagged `backend` in one command so that I can start coordinated work without repeating myself.
- **As a developer running a bulk creation**, I want repositories that fail to be reported without aborting the rest so that one bad repository does not block the other nine.
- **As a developer**, I want to narrow a tag to a subset by also giving a name pattern so that I can act on part of a tagged group.
- **As a script author**, I want a non-zero exit status when any repository failed so that my automation notices partial success.

## Demoable Units of Work

### Unit 1: Tag-Scoped Worktree Creation

**Purpose:** Deliver the core capability — one branch, many repositories, selected by tag.

**Functional Requirements:**

- The system shall accept a `-t` / `--tag` flag on `omgw worktree add` that selects tracked repositories by tag.
- The tag shall be matched using the existing exact, case-insensitive matching with wildcard support provided by `filter.MatchesExact`.
- The `--tag` flag shall accept a single tag value; supplying it more than once shall be rejected with an error rather than silently using one value.
- When `-t <tag>` is supplied with one positional argument, that argument shall be the branch name.
- For each matched repository, the system shall create a worktree at the repository's XDG projects path for the given branch, using the same behavior as single-repository creation: if the branch exists locally it is checked out, otherwise a new branch is created from the repository's current HEAD.
- The system shall create any missing parent directories for branch names containing slashes, matching current single-repository behavior.
- The system shall skip and report, without treating it as a hard failure, any repository that already has a worktree for the given branch.
- When the tag matches no repositories, the system shall exit non-zero with an error naming the tag.
- The system shall update each affected repository's stored worktree data and save the configuration once at the end of the run, not once per repository.

**Proof Artifacts:**

- Test: `-t <tag> <branch>` creates a worktree in every tagged repository and in no untagged repository, demonstrates correct selection
- Test: tag matching is case-insensitive and honors wildcards, demonstrates reuse of the established matching rule
- Test: a repository that already has a worktree for the branch is skipped and reported, demonstrates idempotent-ish handling
- Test: an unmatched tag exits non-zero with an error naming the tag, demonstrates the empty-selection contract
- Test: configuration is saved once and reflects every created worktree, demonstrates correct persistence
- CLI: transcript of `omgw worktree add -t <tag> feat-x` across three tagged repositories followed by `omgw worktree list`, demonstrates the feature end to end

### Unit 2: Targeting Precedence and Combined Filters

**Purpose:** Make it unambiguous which repositories a given invocation will act on, across all three targeting modes.

**Functional Requirements:**

- The system shall resolve targeting according to this table, where "positionals" counts positional arguments:

  | Invocation | Repositories targeted | Branch |
  |---|---|---|
  | `add <branch>` (1 positional, no `-t`) | the repository resolved from the current directory | positional 1 |
  | `add -t <tag> <branch>` (1 positional, `-t`) | all repositories carrying the tag | positional 1 |
  | `add <repo> <branch>` (2 positionals, no `-t`) | repositories matching the name pattern | positional 2 |
  | `add <repo> <branch> -t <tag>` (2 positionals, `-t`) | repositories matching the name pattern **and** carrying the tag | positional 2 |

- When both a repository name pattern and a tag are supplied, the system shall require a repository to satisfy both conditions, matching the AND semantics already used by `omgw tag add --repo X --path Y`.
- The repository name pattern shall continue to be matched with `filter.MatchesPattern`, the partial case-insensitive matching used today.
- The system shall invoke the current-directory resolver only when no repository name pattern and no tag are supplied.
- When a combined name-and-tag filter matches no repositories, the system shall exit non-zero with an error naming both the pattern and the tag.
- The existing two-positional form without `-t` shall continue to reject an ambiguous name pattern that matches multiple repositories, preserving today's `multiple repositories match '%s', narrow your query` behavior.

**Proof Artifacts:**

- Test: each of the four rows in the precedence table selects the expected repositories, demonstrates the full precedence contract
- Test: name pattern combined with a tag applies AND logic, demonstrates question 4's chosen semantics
- Test: the current-directory resolver is not consulted when a tag or name pattern is present, demonstrates precedence ordering
- Test: a combined filter matching nothing exits non-zero naming both filters, demonstrates the error contract
- CLI: transcript showing `omgw worktree add api feat-x -t backend` acting only on repositories satisfying both conditions, demonstrates combined filtering

### Unit 3: Partial-Failure Reporting

**Purpose:** Give the user a complete, trustworthy account of a bulk run in which some repositories succeeded and others did not.

**Functional Requirements:**

- The system shall attempt creation for every matched repository, continuing past failures rather than aborting.
- The system shall collect per-repository errors and print them after the run, following the error-collection and summary format already used by `runWorktreeAlign`.
- The summary shall report the number of worktrees created, the number of repositories skipped because the branch already had a worktree, and the number that failed.
- Each reported failure shall name the repository and the underlying reason.
- The system shall exit non-zero when one or more repositories failed, and zero when all matched repositories either succeeded or were skipped as already present.
- Worktrees successfully created before a later repository failed shall be retained, not rolled back.

**Proof Artifacts:**

- Test: a run where some repositories fail creates worktrees for the rest and reports both counts, demonstrates continue-past-failure behavior
- Test: exit status is non-zero when any repository failed and zero when only skips occurred, demonstrates the exit-status contract
- Test: successful creations are retained after a later failure, demonstrates the no-rollback decision
- Test: summary output names each failed repository with its reason, demonstrates diagnosability
- CLI: transcript of a mixed-outcome run showing the created, skipped, and failed summary, demonstrates the reporting format

## Non-Goals (Out of Scope)

1. **Configurable base ref**: New branches are created from the repository's current HEAD. A `--from <ref>` flag and default-branch detection were declined (round 1, question 3, option B).
2. **Existing-branches-only mode**: Repositories lacking the branch still get one created. Skipping them was declined (round 1, question 3, option C).
3. **Repeatable tags with AND logic**: `--tag` takes a single value (round 1, question 2a, option B). This differs from `omgw list -t`, which is repeatable.
4. **Rollback on failure**: Partial success is the defined outcome (round 1, question 3a).
5. **Removal and refresh**: Specified in specs 27 and 28.
6. **Dry-run and confirmation prompts**: The safety rails from round 1 question 6 apply to destructive operations. Creation is non-destructive and additive, so it gets neither.

## Design Considerations

No new visual design is introduced. Output follows the established bulk-operation shape from `worktree align`: per-item lines during the run, then a counted summary, then a separately headed error block when failures occurred.

## Repository Standards

- **Go conventions**: Command file in `cmd/omgitworks/`, extending `worktree_add.go` rather than adding a parallel command.
- **Cobra structure**: The `-t` flag registers on the existing `worktreeAddCmd` in its `init()`.
- **Filtering**: Reuse `internal/filter` — `MatchesExact` for tags, `MatchesPattern` for name patterns — rather than writing new matching logic.
- **Testing**: Go standard `testing`, `t.TempDir()`, `bytes.Buffer` output capture, following `worktree_add_test.go`.
- **Error summary format**: Follow the `errors []string` collection and `pluralize` reporting already in `worktree_align.go`.
- **Commit messages**: Conventional commits; `feat:` for the new flag.

## Technical Considerations

- **Reuse of `git.AddWorktree`**: The per-repository operation is unchanged from today. Bulk creation is a loop over existing, proven behavior, which is why round 1 question 3 selected option (A).
- **Single configuration save**: Today `runWorktreeAdd` loads, mutates, and saves configuration for one repository. The bulk path must hold one loaded configuration across the whole run and save once, or repositories processed early will be overwritten by a later save of stale data.
- **Worktree re-discovery cost**: Current single-repository creation re-runs `git worktree list` for the affected repository afterwards. Doing this per repository in a bulk run is acceptable, since it is one subprocess per repository that actually changed.
- **`-t` shorthand consistency**: `-t` matches `omgw list -t`. Round 1 question 2a made it single-valued here while `list` accepts it repeatably, so the flag description must state the single-value constraint explicitly to avoid a silent surprise.
- **Exit status**: The existing worktree commands return an error from `RunE` or nothing at all. Partial failure needs a non-zero exit without cobra printing a confusing top-level error over an already-printed summary. Returning a sentinel error after printing the summary, with cobra's `SilenceErrors`/`SilenceUsage` configured on the command, is the approach that fits the existing structure.
- **Dependency on spec 25**: The one-positional-no-tag row of the precedence table requires the resolver from spec 25. If specs are implemented out of order, that row must be deferred rather than reimplemented here.
- **Ambiguous name patterns**: The two-positional form currently errors when a pattern matches multiple repositories. Combining it with a tag narrows the set, so the ambiguity check must run after the tag filter is applied, not before.

## Security Considerations

No specific security considerations identified. All operations are local git and filesystem work. No credentials, tokens, or network access are involved. Bulk creation increases the number of repositories touched per invocation, which is a correctness and blast-radius concern rather than a security one, and is addressed by the reporting requirements in Unit 3.

## Success Metrics

1. **`omgw worktree add -t <tag> <branch>` creates exactly one worktree per tagged repository**, verified against a fixture workspace with tagged and untagged repositories.
2. **All four rows of the precedence table behave as specified**, with no invocation selecting an unexpected repository set.
3. **A mixed-outcome run reports created, skipped, and failed counts accurately** and exits non-zero.
4. **Configuration after a bulk run reflects every created worktree**, with no repository's data lost to an overwriting save.
5. **Test coverage** for tag selection, combined filtering, precedence, partial failure, exit status, and persistence.

## Open Questions

All questions below are **resolved**. No blocking questions remain.

1. **Resolved — report skips per repository.** A repository skipped because it already has a worktree for the branch is announced during the run as well as counted in the final summary, matching how `worktree align` announces skipped locked worktrees.

   *User decision:* "this assumption is fine for now"
