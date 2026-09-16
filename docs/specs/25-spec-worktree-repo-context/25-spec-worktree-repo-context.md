# 25-spec-worktree-repo-context

> Clarification record: `25-questions-1-worktree-repo-context.md` and `25-questions-2-worktree-repo-context.md` in this directory hold the answered question rounds for this spec **and** for specs 26, 27, and 28. They are the shared source of record for the worktree CRUD series.

## Introduction/Overview

Every `omgw worktree` subcommand that acts on a repository requires the repository to be named explicitly, even when the user is standing inside that repository. This feature adds current-repository detection: when the repository argument is omitted, omgitworks determines which tracked repository the working directory belongs to and uses it. An explicitly supplied repository always takes precedence over the detected one.

This spec is first in a four-spec series because it introduces the shared resolver that specs 26 (`add` by tag), 27 (`remove`), and 28 (`refresh`) all consume.

## Goals

- Introduce a single current-repository resolver that every worktree subcommand can call
- Allow `omgw worktree add <branch>` to be run from inside a repository with no repository argument
- Resolve correctly when the working directory is inside a **worktree**, returning the owning repository
- Preserve the existing meaning of every current command invocation, so no working command changes behavior
- Give a clear, actionable error when the working directory is not inside a tracked repository

## User Stories

- **As a developer working inside a repository**, I want to run `omgw worktree add feat-auth` without naming the repository so that I do not have to retype a name the tool could already determine.
- **As a developer working inside one worktree**, I want to create a sibling worktree for another branch without first navigating back to the main checkout so that I can stay where I am working.
- **As a developer with many tracked repositories**, I want an explicit repository name to always win over detection so that I can act on a different repository without leaving my current directory.
- **As a developer whose current directory is not a tracked repository**, I want an error that tells me how to fix it so that I am not left guessing why the command failed.

## Demoable Units of Work

### Unit 1: Current-Repository Resolution for `worktree add`

**Purpose:** Deliver the resolver and prove it through the command the user asked for, so `omgw worktree add <branch>` works from inside a repository or any of its worktrees.

**Functional Requirements:**

- The system shall provide a reusable resolver that determines the tracked repository owning the current working directory.
- The resolver shall determine the enclosing checkout by running `git rev-parse --show-toplevel` from the current working directory.
- The resolver shall compare the resulting path against each tracked repository's `Path` field, and if no repository matches, against every entry in each repository's `Worktrees[].Path`.
- When the match is found via a worktree path, the resolver shall return the repository that owns that worktree, not the worktree itself.
- The resolver shall resolve symbolic links on both the git-reported path and the stored paths before comparing them, so that a symlinked workspace or data directory still matches.
- The resolver shall compare paths exactly after symlink resolution and path cleaning, rather than by prefix.
- `omgw worktree add <branch>` (one positional argument) shall create the worktree in the resolved current repository.
- `omgw worktree add <repo> <branch>` (two positional arguments) shall continue to behave exactly as it does today, using the named repository and ignoring the working directory.
- When the repository argument is omitted and the resolver finds no match, the system shall exit with a non-zero status and an error that states the working directory is not inside a tracked repository and names both remedies: supply a repository argument, or run `omgw add` to track the current repository.
- When `git rev-parse --show-toplevel` fails, because the directory is not inside any git repository, the system shall produce that same error rather than surfacing git's raw message.
- The resolver shall not modify configuration, create directories, or prompt the user.

**Proof Artifacts:**

- Test: resolver returns the owning repository when the working directory is the repository root, demonstrates the base case
- Test: resolver returns the owning repository when the working directory is a subdirectory several levels deep, demonstrates that git's own upward walk is relied upon
- Test: resolver returns the owning repository when the working directory is inside one of that repository's worktrees, demonstrates the worktree-path fallback required by question 2
- Test: resolver matches correctly when the stored repository path traverses a symlink, demonstrates symlink resolution on both sides
- Test: resolver returns an error naming both remedies when the directory is not inside a tracked repository, demonstrates the error contract
- Test: `worktree add <repo> <branch>` with two arguments ignores the working directory, demonstrates explicit precedence
- CLI: transcript of `cd <repo> && omgw worktree add feat-x` followed by `omgw worktree list <repo>` showing the new worktree, demonstrates the feature end to end
- CLI: transcript of running `omgw worktree add feat-y` from inside an existing worktree of the same repository, showing the worktree created against the owning repository, demonstrates the worktree-relative case

### Unit 2: Explicit Current-Repository Targeting for `list` and `align`

**Purpose:** Extend current-repository targeting to the subcommands whose repository argument is already optional, without changing what their existing invocations mean.

**Functional Requirements:**

- The system shall accept `.` as the repository argument on `omgw worktree list` and `omgw worktree align`, meaning the repository resolved from the current working directory.
- `omgw worktree list` with no argument shall continue to list worktrees across all tracked repositories, exactly as it does today.
- `omgw worktree align` with no argument shall continue to align worktrees across all tracked repositories, exactly as it does today.
- `omgw worktree list .` shall list only the resolved current repository's worktrees.
- `omgw worktree align .` shall align only the resolved current repository's worktrees, and shall honor the existing `--dry-run` flag.
- When `.` is supplied and the resolver finds no match, the system shall produce the same error contract defined in Unit 1.
- A literal `.` shall not be interpreted as a repository name pattern, even if a tracked repository's name happens to contain a period.

**Proof Artifacts:**

- Test: `worktree list` with no argument returns worktrees for all repositories, demonstrates preserved existing behavior
- Test: `worktree list .` returns only the current repository's worktrees, demonstrates scoped targeting
- Test: `worktree align .` plans moves only for the current repository, demonstrates scoped targeting on the mutating command
- Test: `.` resolution failure produces the Unit 1 error, demonstrates a consistent error contract across subcommands
- CLI: transcript comparing `omgw worktree list` and `omgw worktree list .` from inside a repository, demonstrates the difference in scope

### Unit 3: Completion and Documentation

**Purpose:** Make the new argument shapes discoverable, and keep the published documentation accurate.

**Functional Requirements:**

- Tab completion for the first positional argument of `omgw worktree add` shall suggest branch names from the resolved current repository when the working directory resolves to a tracked repository.
- Tab completion for the first positional argument of `omgw worktree add` shall continue to suggest repository names when the working directory does not resolve to a tracked repository.
- Tab completion for `omgw worktree list` and `omgw worktree align` shall include `.` among the suggested repository values when the working directory resolves to a tracked repository.
- The shell function in `shellinit.go` shall require no changes, because no navigation or `cd` behavior is affected.
- `docs/site/commands-core.md` shall document the omitted-repository form of `worktree add`, the `.` argument for `list` and `align`, the worktree-relative resolution behavior, and the not-a-tracked-repository error.

**Proof Artifacts:**

- Test: completion returns branch names for the first argument when resolution succeeds, demonstrates context-aware completion
- Test: completion falls back to repository names when resolution fails, demonstrates the fallback
- Test: existing shell-init output is unchanged, demonstrates that no shell function regression was introduced
- Documentation: updated `docs/site/commands-core.md` section showing the new invocation forms, demonstrates published documentation matches behavior

## Non-Goals (Out of Scope)

1. **Changing `worktree navigate`**: The bare `omgw worktree <branch>` navigate form stays global across the workspace. Preferring the current repository's worktrees was explicitly declined (question 1, option C).
2. **Tag-based targeting**: The `-t` / `--tag` flag is specified in spec 26 and is not part of this spec.
3. **`remove` and `refresh` subcommands**: These do not exist yet. They are specified in specs 27 and 28 and will consume this resolver when they are built.
4. **Auto-tracking untracked repositories**: Running from inside an untracked git repository produces an error, never a silent `omgw add` (question 3, option D declined).
5. **Interactive repository selection**: Resolution failure never falls back to a picker (question 3, option C declined).
6. **Changing the meaning of a bare `list` or `align`**: Omitting the argument continues to mean "all repositories" for these two commands.

## Design Considerations

No new visual design is introduced. Two interface decisions carry through:

- The error for a failed resolution follows the repository's existing style of stating the problem and then the remedy, as `git.MoveWorktree` does for cross-device moves.
- The `.` argument was chosen over a `--here` flag so that current-repository targeting occupies the same argument position as an explicit repository name, which keeps `list`, `align`, and the future `refresh` uniform.

## Repository Standards

- **Go conventions**: Command files live in `cmd/omgitworks/`, shared logic in `internal/`.
- **Cobra structure**: Subcommands register through `init()` with `worktreeCmd.AddCommand`, matching `worktree_add.go` and `worktree_list.go`.
- **Testing**: Go standard `testing`, `t.TempDir()` for filesystem fixtures, `bytes.Buffer` for output capture, `t.Cleanup()` for teardown, following `worktree_add_test.go`.
- **Git invocation**: Route through the existing `gitCommand` helper in `internal/git` rather than calling `os/exec` directly.
- **Path comparison**: Reuse the established resolve-then-clean approach from `resolvePath` in `internal/git/worktree.go`.
- **Commit messages**: Conventional commits; `feat:` for the new capability, which triggers a minor version bump via Release Please.

## Technical Considerations

- **Resolver placement**: The resolver needs both a git invocation and access to the loaded configuration. Placing it in a small `internal/repocontext` package keeps `internal/git` free of configuration knowledge and `internal/config` free of subprocess calls. Implementers may choose otherwise, but the resolver must be callable from all worktree subcommands without duplication, since specs 26–28 depend on it.
- **Resolution method**: Question 2a selected `git rev-parse --show-toplevel` over pure path-prefix matching. This delegates the upward directory walk and git-boundary detection to git itself, so subdirectories, nested repositories, and submodule boundaries are handled by git's own rules. The cost is one subprocess per invocation on commands where the repository argument is omitted; the resolver should not be invoked when an explicit repository argument is present.
- **Exact matching is required, not prefix matching**: Because `--show-toplevel` returns a checkout root rather than an arbitrary directory, comparison must be exact after symlink resolution. Prefix matching here would incorrectly resolve a repository nested inside another repository's directory tree.
- **Worktree path lookup order**: Repository paths are checked before worktree paths. A worktree can never share a path with a tracked repository root, so the order is a matter of cost rather than correctness.
- **Symlinks**: Tracked repositories may be symlinked into the workspace, and the XDG data directory is a symlink on some systems. `filepath.EvalSymlinks` must be applied to both sides, falling back to `filepath.Clean` when a path does not exist, exactly as `resolvePath` already does.
- **Stale worktree data**: The resolver reads `Worktrees[].Path` from `config.json`, which is populated at `omgw refresh` time. A worktree created outside omgitworks and not yet refreshed will not resolve. This is acceptable: spec 28 adds targeted refresh, and the error message already points at a remedy.
- **Forward constraint for spec 26**: Question 4 selected AND logic, so an explicit repository argument and `-t` may be combined. This means the resolver must be the lowest-precedence targeting source: explicit argument and tag filter are both honored when present, and detection applies only when neither narrows the selection. Spec 26 defines the full precedence table.
- **Argument arity as the disambiguator**: With the repository argument optional, `worktree add` accepts one or two positionals. One means branch, two means repository then branch. `cobra.RangeArgs(1, 2)` replaces the current `cobra.ExactArgs(2)`.
- **Git version**: `git rev-parse --show-toplevel` has been available far longer than the Git 2.17 floor the worktree feature already requires, so it introduces no new version constraint.

## Security Considerations

No specific security considerations identified. Resolution is a read-only local filesystem and git metadata operation. No credentials, tokens, or network access are involved. The resolver must not write to configuration, so a failed resolution cannot leave persistent state behind.

## Success Metrics

1. **`omgw worktree add <branch>` succeeds from a repository root, from a nested subdirectory, and from inside one of that repository's worktrees**, creating the worktree against the owning repository in all three cases.
2. **Every currently valid invocation produces identical output before and after the change**, verified for `worktree add <repo> <branch>`, bare `worktree list`, and bare `worktree align`.
3. **Resolution failure exits non-zero with an error naming both remedies**, with no partial side effects.
4. **The resolver is called from a single shared implementation**, with no duplicated resolution logic across command files.
5. **Test coverage** for resolver success paths (root, subdirectory, worktree, symlink), the failure path, and each command wiring.

## Open Questions

All questions below are **resolved**. No blocking questions remain.

1. **Resolved — keep `.` for `list` and `align`.** The `.` convention was a spec-authoring decision rather than an answered question: round 2 question 1 selected option (B), extending detection to every subcommand with a repository argument, but `list` and `align` already use the omitted argument to mean "all repositories", so omission could not be reused without breaking existing behavior.

   *User decision:* "this is fine for now. we can see how it works and then decide if we want to make the breaking chagne later."

   Implementation proceeds with `.` as specified in Unit 2. Changing omission to mean "current repository" on these two commands remains available as a future breaking change, and would be its own spec.

2. **Resolved — the resolver stays silent.** No informational line is printed when the repository is inferred, matching how the rest of the CLI treats inferred values.

   *User decision:* "assuming silent is fine"
