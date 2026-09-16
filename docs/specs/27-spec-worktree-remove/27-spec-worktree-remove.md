# 27-spec-worktree-remove

> Clarification record: the answered question rounds for this spec live at `docs/specs/25-spec-worktree-repo-context/25-questions-1-worktree-repo-context.md` and `25-questions-2-worktree-repo-context.md`. Round 1 questions 5a, 5b, 5c, 5d, 6, and 7 govern this spec.
>
> Depends on: **spec 25** (`worktree-repo-context`) for the current-repository resolver. Shares the targeting precedence model defined in **spec 26**.

## Introduction/Overview

omgitworks can create, list, align, and navigate to worktrees, but it cannot remove them — spec 20 listed removal as an explicit non-goal, leaving users to fall back to `git worktree remove`. This feature adds `omgw worktree remove`, in both an individual form and a tag-scoped form, with the safety rails a destructive bulk command needs: git's own refusal to discard uncommitted work, a dry-run preview, and a confirmation prompt.

## Goals

- Add `omgw worktree remove` with an `rm` alias, covering individual and tag-scoped removal
- Never discard uncommitted work by default, deferring to git's own safety refusal
- Never delete a git branch as a side effect of removing a worktree
- Give the user two independent chances to catch a mistake: a dry-run preview and a confirmation prompt
- Leave the projects root tidy by removing a repository's projects directory once it is empty

## User Stories

- **As a developer who finished a feature**, I want to remove its worktree across every repository tagged `backend` in one command so that I do not clean up one repository at a time.
- **As a developer**, I want removal to refuse when a worktree has uncommitted changes so that a cleanup command cannot silently destroy work in progress.
- **As a cautious developer**, I want to preview exactly what a removal will delete before it happens so that I can confirm the tag matched what I expected.
- **As a developer**, I want my branches left alone when I remove a worktree so that removing a checkout is not the same as discarding the work.
- **As a script author**, I want to skip the confirmation prompt explicitly so that automation is possible without making the interactive path dangerous.

## Demoable Units of Work

### Unit 1: Individual Worktree Removal

**Purpose:** Deliver removal for a single worktree, including the git-level primitive and the safety behavior every other form inherits.

**Functional Requirements:**

- The system shall provide `omgw worktree remove <repo> <branch>`, with `rm` registered as a command alias.
- The system shall provide a git-level removal operation that runs `git worktree remove <path>` against the owning repository.
- `omgw worktree remove <branch>` with one positional argument shall remove that branch's worktree from the repository resolved from the current working directory, per spec 25.
- The system shall match the branch against the repository's stored worktree entries and shall remove only an exactly matching branch, not a partial match.
- When the branch matches no worktree in the targeted repository, the system shall exit non-zero with an error naming the repository and the branch.
- By default the system shall not pass `--force`, so git refuses to remove a worktree containing uncommitted changes, untracked files, or a submodule, and that refusal shall be surfaced as a clear failure.
- The system shall accept a `--force` flag that passes `--force` through to `git worktree remove`.
- The system shall skip any locked worktree, reporting the lock and its reason, using the existing `git.IsWorktreeLocked` helper, and shall not remove a locked worktree even when `--force` is supplied.
- The system shall not delete, modify, or otherwise act on the git branch that the removed worktree had checked out.
- After a successful removal, the system shall remove the repository's directory under the XDG projects root if, and only if, that directory is now empty.
- After removal, the system shall update the affected repository's stored worktree data and save the configuration.

**Proof Artifacts:**

- Test: `remove <repo> <branch>` removes the worktree directory and git's worktree entry, demonstrates the core operation
- Test: removal of a worktree with uncommitted changes fails without `--force` and succeeds with it, demonstrates the default safety posture from question 5b
- Test: a locked worktree is skipped and reported even when `--force` is supplied, demonstrates the lock guarantee
- Test: the branch still exists after its worktree is removed, demonstrates question 5c
- Test: an empty projects directory is cleaned up and a non-empty one is left alone, demonstrates question 5d
- Test: a branch matching no worktree exits non-zero naming repository and branch, demonstrates the error contract
- Test: `rm` alias resolves to the same command, demonstrates question 7
- CLI: transcript of creating a worktree, removing it, and confirming via `omgw worktree list` that it is gone and via `git branch` that the branch remains, demonstrates the feature end to end

### Unit 2: Tag-Scoped Removal

**Purpose:** Remove the same branch's worktree across a group of repositories in one command, reusing the targeting model from spec 26.

**Functional Requirements:**

- The system shall accept a `-t` / `--tag` flag on `omgw worktree remove`, matching tags with `filter.MatchesExact` and accepting a single value.
- `omgw worktree remove -t <tag> <branch>` shall remove the named branch's worktree from every tracked repository carrying the tag.
- The system shall apply the same targeting precedence table defined in spec 26: two positionals name a repository pattern, one positional with `-t` scopes by tag, one positional alone uses the current repository, and a repository pattern combined with `-t` requires both conditions.
- Repositories carrying the tag that have no worktree for the branch shall be skipped and reported, not treated as failures.
- The system shall continue past per-repository failures, collect errors, and report a summary of removed, skipped, and failed counts, following the reporting shape used by `worktree align`.
- The system shall exit non-zero when one or more removals failed, and zero when all targeted repositories either succeeded or were skipped.
- Worktrees successfully removed before a later failure shall not be restored.
- The system shall save the configuration once at the end of the run rather than once per repository.

**Proof Artifacts:**

- Test: `-t <tag> <branch>` removes the worktree in every tagged repository and none outside the tag, demonstrates correct selection
- Test: a tagged repository without that branch's worktree is skipped rather than failing, demonstrates skip-versus-fail handling
- Test: a mixed-outcome run reports removed, skipped, and failed counts and exits non-zero, demonstrates partial-failure reporting
- Test: configuration after a bulk run reflects every removal, demonstrates correct single-save persistence
- CLI: transcript of `omgw worktree remove -t <tag> feat-x` across tagged repositories with `omgw worktree list` before and after, demonstrates the feature end to end

### Unit 3: Dry Run and Confirmation

**Purpose:** Ensure a destructive command cannot delete anything the user has not seen first.

**Functional Requirements:**

- The system shall accept a `--dry-run` flag that prints exactly what would be removed and exits without removing anything, modifying configuration, or deleting directories.
- The dry-run output shall list, per worktree, the repository name, branch, and filesystem path, and shall end with a count, following the format established by `worktree align --dry-run`.
- The dry-run output shall mark worktrees that would be skipped because they are locked, and worktrees that would fail because they contain uncommitted changes and `--force` was not supplied.
- When a run would remove more than one worktree, the system shall print the full list and prompt for confirmation before removing anything.
- The system shall accept a `--yes` / `-y` flag that skips the confirmation prompt.
- When confirmation is required, `--yes` was not supplied, and standard input is not an interactive terminal, the system shall exit non-zero with an error instructing the user to pass `--yes`, rather than prompting into a non-interactive stream or proceeding unconfirmed.
- Declining the confirmation prompt shall exit zero without removing anything.
- `--dry-run` and `--yes` used together shall behave as a dry run, since no removal occurs.

**Proof Artifacts:**

- Test: `--dry-run` produces the planned removal list and leaves every worktree, directory, and configuration entry untouched, demonstrates preview safety
- Test: dry-run output flags locked worktrees and dirty worktrees that would fail, demonstrates preview accuracy
- Test: a multi-worktree run prompts before removing and removes nothing when declined, demonstrates the confirmation gate
- Test: `--yes` skips the prompt and proceeds, demonstrates the scripted path
- Test: confirmation required with a non-interactive stdin and no `--yes` exits non-zero without removing anything, demonstrates the non-interactive contract
- CLI: transcript of a `--dry-run` followed by the real run of the same command, demonstrates that the preview matched the outcome

## Non-Goals (Out of Scope)

1. **Removing all worktrees of a tagged group**: `remove -t <tag>` without a branch argument is not supported (round 1, question 5a, options B and C declined). A branch argument is always required.
2. **Branch deletion**: No `--delete-branch` flag and no implicit branch removal (round 1, question 5c).
3. **Forcing past a lock**: Locked worktrees are always skipped. `--force` covers dirty worktrees only, matching git's own division.
4. **Interactive per-worktree prompting**: Confirmation is a single up-front gate over the whole plan, not one prompt per worktree (round 1, question 5b, option C declined).
5. **Rollback**: Successful removals are not restored when a later removal fails.
6. **Pruning unrelated stale worktrees**: Metadata repair and pruning belong to spec 28.
7. **Repeatable tags**: `--tag` takes a single value (round 1, question 2a).

## Design Considerations

No new visual design is introduced. Two interface points matter:

- The dry-run and confirmation listings share one plan-rendering format, so the preview a user approves is textually the same as the preview `--dry-run` shows.
- The confirmation prompt follows the numbered-list-and-prompt conventions already used for interactive selection in `navigate.go`, adapted to a yes/no question.

## Repository Standards

- **Go conventions**: New command file `cmd/omgitworks/worktree_remove.go`, with the git primitive added to `internal/git/worktree.go` alongside `AddWorktree` and `MoveWorktree`.
- **Cobra structure**: Registered through `init()` with `worktreeCmd.AddCommand`, using cobra's native `Aliases` field for `rm`.
- **Testing**: Go standard `testing`, `t.TempDir()` for real git fixtures, `bytes.Buffer` for output capture, and an injected reader for prompt input, following the I/O-injection pattern in `worktree_list.go` and `navigate.go`.
- **Error summary format**: Follow the `errors []string` collection and `pluralize` reporting in `worktree_align.go`.
- **Directory cleanup**: Follow the conservative pattern of `removeEmptyLegacyDir`, which removes only genuinely empty directories.
- **Commit messages**: Conventional commits; `feat:` for the new command.

## Technical Considerations

- **New git primitive required**: `internal/git/worktree.go` has `AddWorktree`, `MoveWorktree`, `RepairWorktrees`, `PruneWorktrees`, and `IsWorktreeLocked`, but no removal function. A `RemoveWorktree(repoPath, worktreePath string, force bool) error` is new work, not a wiring exercise.
- **Deferring to git's dirty check**: Question 5b chose to let `git worktree remove` refuse rather than reimplementing a cleanliness check. This means the failure text comes from git; it should be wrapped with the repository and branch for context but not replaced, so the underlying reason stays visible.
- **Lock checking happens before the git call**: `IsWorktreeLocked` reads `.git/worktrees/<name>/locked` directly. Checking it up front lets the dry-run preview mark locked entries accurately, and lets the real run skip them without a failed subprocess.
- **Empty-directory cleanup is two levels**: A branch name containing slashes nests below the repository's projects directory, so cleanup must walk upward from the removed worktree's parent, removing empty directories until it reaches a non-empty one or the repository's projects directory itself. It must never remove the projects root.
- **Testing needs real git repositories**: Removal behavior around dirty worktrees is git's, not omgitworks', so tests must build actual repositories with `git init` and real worktrees in `t.TempDir()` rather than stubbing the git layer. The repository's existing worktree tests establish this pattern.
- **Prompt input must be injectable**: Confirmation reads from standard input, so the run function must take an `io.Reader` for input and `io.Writer` for output, as `runNavigate` already does, or the confirmation path cannot be tested.
- **TTY detection**: The non-interactive requirement needs a terminal check on standard input. The repository already performs TTY detection for `--color` handling in `list.go`, so the approach exists to follow.
- **Exit status**: As in spec 26, partial failure must exit non-zero after the summary is printed, without cobra overprinting a redundant error.
- **Dependency on spec 25**: The one-positional form requires the current-repository resolver. If implemented before spec 25, that form must be deferred rather than reimplemented.

## Security Considerations

This is the most destructive command in the worktree family, so the relevant concerns are data-loss rather than credential ones:

- No credentials, tokens, or network access are involved; all operations are local.
- The default path must never discard uncommitted or untracked work. `--force` is the only route to that, and it is never implied by `--yes` or by tag-scoped selection.
- Committed work is never at risk, because branches are never deleted.
- Proof artifacts must not capture real workspace paths that reveal unrelated private repository names; transcripts should be produced against a scratch fixture workspace.

## Success Metrics

1. **A worktree with uncommitted changes is never removed without `--force`**, verified against a real git fixture.
2. **A locked worktree is never removed**, with or without `--force`.
3. **Branches survive removal of their worktrees** in every tested path.
4. **`--dry-run` output matches the subsequent real run exactly** for the same command and workspace state.
5. **A multi-worktree run cannot proceed unconfirmed**, either interactively or with a non-interactive stdin and no `--yes`.
6. **Test coverage** for individual removal, tag-scoped removal, force, locks, branch preservation, directory cleanup, dry run, confirmation, non-interactive refusal, partial failure, and exit status.

## Open Questions

All questions below are **resolved**. No blocking questions remain.

1. **Resolved — the confirmation gate is keyed to the number of worktrees targeted, not to `-t`.** Any run targeting more than one worktree prompts, so a name pattern matching several repositories is gated just as a tag-scoped run is. A single-worktree removal is not prompted, since git's own refusal already guards the only destructive case.

   *User decision:* "the assumption is fine."

2. **Resolved — `--force` is annotated in the dry-run header.** The preview states which safety posture the real run will use, as a one-line annotation.

   *User decision:* "assumption is fine"
