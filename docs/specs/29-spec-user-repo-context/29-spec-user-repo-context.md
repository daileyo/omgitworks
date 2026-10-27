# 29-spec-user-repo-context

> Clarification record: `29-questions-1-user-repo-context.md` in this directory holds the answered question round for this spec **and** for specs 30 (`assign` by tag), 31 (`unassign`), and 32 (scoped `sync`). It is the shared source of record for the user identity series. Round 1 questions 1, 1a, 4, 5, 6, and 9 govern this spec.
>
> Depends on: **spec 25** (`worktree-repo-context`), which supplies the `internal/repocontext` resolver this spec reuses. No new resolver is built here.

## Introduction/Overview

`omgw user assign` requires the repository to be named explicitly, even when the user is standing inside it. This feature lets the repository argument be omitted: `omgw user assign <profile>` run from inside a tracked repository, or from inside one of its worktrees, applies the profile to that repository. An explicitly named repository always takes precedence over detection, and every invocation that works today keeps working unchanged.

This is the first of a four-spec series that carries the targeting model built for `omgw worktree` in specs 25–28 over to `omgw user`. It is deliberately the smallest step: it changes one command's argument shape and wires in a resolver that already exists.

## Goals

- Allow `omgw user assign <profile>` to be run from inside a repository with no repository argument
- Resolve correctly when the working directory is inside a **worktree**, applying the profile to the owning repository
- Preserve the existing meaning of `omgw user assign <repository> <profile>`, including its multi-match error, so no working invocation changes behavior
- Reuse the spec 25 resolver and its error contract rather than adding a second resolution path
- Replace the stale `gws` binary name in the `user` and `assign` help text, which this spec rewrites anyway

## User Stories

- **As a developer working inside a repository**, I want to run `omgw user assign work` without naming the repository so that I do not have to retype a name the tool could already determine.
- **As a developer working inside one worktree**, I want the profile applied to the repository that owns the worktree so that every checkout of that repository shares one identity, which is how git's local config works anyway.
- **As a developer with many tracked repositories**, I want an explicit repository name to always win over detection so that I can assign a profile to a different repository without leaving my current directory.
- **As a developer whose current directory is not a tracked repository**, I want the same error the worktree commands give so that I learn one remedy and it applies everywhere.
- **As a developer reading `omgw user --help`**, I want the examples to use the binary I actually have so that I can copy them as written.

## Demoable Units of Work

### Unit 1: Current-Repository Resolution for `user assign`

**Purpose:** Let `omgw user assign <profile>` act on the repository the user is standing in, using the shared resolver from spec 25, without changing the two-argument form.

**Functional Requirements:**

- `omgw user assign <profile>` (one positional argument) shall apply the profile to the repository returned by `repocontext.ResolveCurrent`.
- `omgw user assign <repository> <profile>` (two positional arguments) shall continue to behave exactly as it does today: name-pattern lookup, the existing "multiple repositories match" error on ambiguity, and no inspection of the working directory.
- The resolver shall be consulted only when the repository argument is omitted. When a repository is named, the working directory shall not be inspected.
- When the working directory is inside a worktree of a tracked repository, the profile shall be applied to the owning repository's `.git/config`, and that repository's stored `user`, `email`, `signing_enabled`, and `user_source` fields shall be updated, exactly as the two-argument form does.
- When the repository argument is omitted and the resolver returns `repocontext.ErrNotTracked`, the system shall exit non-zero with that error unchanged, so `user assign` and the `worktree` subcommands share one message and one pair of remedies.
- The repository shall be resolved before the profile is looked up, matching the order of the two-argument form, so a missing profile and an unresolvable directory each produce their own error and never both.
- `--dry-run` shall work with the one-argument form and shall print the resolved repository's name in its "would assign" header, so the user can confirm which repository was detected before applying.
- The one-argument form shall print the same success output as the two-argument form, including the resolved repository's name. No additional line announcing that the repository was inferred shall be printed, matching spec 25's silent-resolver decision.
- `--use-subdirs` combined with the one-argument form shall be rejected before any change is made, with a non-zero exit and an error stating that `--use-subdirs` requires an explicit repository argument. This applies to `--dry-run` as well, so the preview never shows a move the real run would refuse.
- The command's argument validation shall change from exactly two positionals to one or two. Zero or three or more positionals shall continue to fail with cobra's usage error.

**Proof Artifacts:**

- Test: `user assign <profile>` from a repository root applies the profile to that repository, demonstrates the base case
- Test: `user assign <profile>` from a nested subdirectory applies the profile to the enclosing repository, demonstrates reliance on git's upward walk through the shared resolver
- Test: `user assign <profile>` from inside a worktree writes to the owning repository's `.git/config` and updates the owning repository's stored fields, demonstrates the worktree case required by question 1 option A
- Test: `user assign <repository> <profile>` with two arguments does not call the resolver and still rejects an ambiguous pattern, demonstrates explicit precedence and preserved behavior
- Test: one-argument form outside any tracked repository exits non-zero with `repocontext.ErrNotTracked`, demonstrates the shared error contract
- Test: one-argument form with an unknown profile from inside a tracked repository reports the profile error, not the resolver error, demonstrates lookup order
- Test: `user assign <profile> --use-subdirs` is rejected before any filesystem or config change, with and without `--dry-run`, demonstrates question 6 option A
- Test: `user assign <profile> --dry-run` names the resolved repository and leaves `.git/config` and `config.json` untouched, demonstrates the preview
- CLI: transcript of `cd <repo> && omgw user assign work` followed by `git config --local user.email`, demonstrates the feature end to end
- CLI: transcript of running `omgw user assign personal` from inside a worktree of the same repository, followed by `git config --local user.email` run in the main checkout, demonstrates the worktree-relative case

### Unit 2: Completion, Help Text, and Documentation

**Purpose:** Make the new argument shape discoverable, and bring the `user` command's help and published docs up to date with the binary name and the new form.

**Functional Requirements:**

- Tab completion for the first positional argument of `omgw user assign` shall suggest profile names (stored and auto-detected) when the working directory resolves to a tracked repository.
- Tab completion for the first positional argument of `omgw user assign` shall suggest repository names when the working directory does not resolve to a tracked repository.
- Tab completion for the second positional argument of `omgw user assign` shall suggest profile names, so the two-argument form gains completion it does not have today.
- The `assign` subcommand's `Use` line shall read `assign [repository] <profile>`, and its `Long` help shall document both forms, state that the repository is detected from the working directory when omitted, and use `omgw` in every example.
- The `user` parent command's `Long` help shall use `omgw` in every example instead of `gws`, and its `assign` example shall show the one-argument form alongside the existing two-argument example.
- `docs/site/commands-user.md` shall document the omitted-repository form of `assign`, the worktree-relative resolution behavior, the not-a-tracked-repository error, and the `--use-subdirs` restriction, and shall update the usage line and examples to match.
- The shell function in `shellinit.go` shall require no changes, because no navigation or `cd` behavior is affected.

**Proof Artifacts:**

- Test: completion returns profile names for the first argument when resolution succeeds, demonstrates context-aware completion
- Test: completion returns repository names for the first argument when resolution fails, demonstrates the fallback
- Test: completion returns profile names for the second argument, demonstrates the new two-argument completion
- Test: the `user` and `assign` help text contains no `gws ` example, demonstrates question 9 option A
- Test: existing shell-init output is unchanged, demonstrates no shell function regression
- CLI: `omgw user assign --help` output showing the `[repository] <profile>` usage and `omgw` examples, demonstrates discoverability
- Documentation: updated `docs/site/commands-user.md` Assign Profile section, demonstrates published documentation matches behavior

## Non-Goals (Out of Scope)

1. **Tag-based targeting**: The `-t` / `--tag` flag on `assign` is spec 30. This spec adds no flags.
2. **Bulk selection by name pattern**: Question 4a chose to treat a multi-match pattern as a bulk selection. That change rides with the bulk machinery (confirmation prompt, `--yes`, summary counts) in spec 30. Until then, the two-argument form keeps today's "multiple repositories match" error.
3. **Confirmation prompt and `--yes`**: Question 5 option A applies a prompt when more than one repository would change. This spec can only ever change one repository, so no prompt or `--yes` flag is added here. Spec 30 introduces them.
4. **`unassign`**: Clearing a repository's identity is spec 31. The deprecation table's `--delete` row stays as it is until then.
5. **Scoped `sync`**: Targeting and per-repository reporting for `user sync` is spec 32.
6. **A `.` argument on `assign`**: `.` is not accepted as the repository argument. Spec 25 introduced `.` only where omitting the argument already meant "all repositories". On `assign`, omission is unambiguous, so `.` has nothing to disambiguate.
7. **`--use-subdirs` with detection**: Declined by question 6 option A. Moving the repository the shell is standing in is a separate feature with its own failure modes.
8. **Auto-tracking untracked repositories and interactive selection**: Resolution failure produces the spec 25 error, never an `omgw add` or a picker.
9. **The short-flag form `omgw user -u`**: The deprecated root flags are untouched. The deprecation table's `--update` row already points at the two-argument subcommand, which still exists.

## Design Considerations

No new visual design is introduced. Three interface decisions carry through:

- Resolution is silent. The success and dry-run output already name the repository, which is enough for the user to confirm detection without a separate "detected repository X" line. This matches spec 25's resolution of its open question 2.
- The `--use-subdirs` refusal follows the repository's problem-then-remedy error style: it says the flag needs an explicit repository and shows the two-argument form.
- The `Use` line uses `[repository] <profile>` with square brackets on the optional argument, consistent with how `worktree add [repo] <branch>` is documented in `commands-core.md`.

## Repository Standards

- **Go conventions**: Command files live in `cmd/omgitworks/`, shared logic in `internal/`. The `user` subcommands are all declared in `cmd/omgitworks/user.go` today; implementers may split `assign` into its own file as `worktree_add.go` is, but must keep registration in the `user.go` `init()`.
- **Cobra structure**: `cobra.RangeArgs(1, 2)` replaces `cobra.ExactArgs(2)`, exactly as spec 25 did for `worktree add`. Completion is attached through `ValidArgsFunction`.
- **Resolver reuse**: Call `repocontext.ResolveCurrent(cfg)` directly. Do not add a second `git rev-parse` path or duplicate path comparison.
- **Completion helpers**: Reuse `completeProfileNames` from `user.go` and `completeRepoNames` from `tag.go`. Model the arity switch on `completeWorktreeAddArgs` in `worktree.go`.
- **Testing**: Go standard `testing`, `t.TempDir()` for filesystem fixtures, `t.Cleanup()` for teardown. `cmd/omgitworks/user_test.go` has no `assign` tests today, so fixture helpers for a tracked repository with a worktree should be modelled on `worktree_add_test.go` and the resolver tests in `internal/repocontext/repocontext_test.go`. Tests that change the working directory must restore it in `t.Cleanup`.
- **Documentation**: `docs/site/commands-user.md` is the published reference and follows the usage-block, flags-table, examples pattern used by every section in that file.
- **Commit messages**: Conventional commits with the `user` scope, `feat(user):` for the new capability, `docs(user):` for the site docs. A `feat:` commit triggers a minor version bump via Release Please.

## Technical Considerations

- **Argument arity as the disambiguator**: With the repository optional, `assign` accepts one or two positionals. The profile is always the last argument, and a repository is present only when there are two. This is the same rule `worktree add` uses, so there is no ambiguity in parsing.
- **Resolver is lowest precedence**: Spec 26's precedence table, which question 4 option A adopts for `user`, makes detection apply only when neither a name pattern nor a tag is present. This spec implements the no-pattern row. Spec 30 must slot the tag rows in without re-ordering this one, so the one-argument path should branch on "repository argument absent" rather than on `len(args) == 1`, leaving room for `-t` to be present with one positional.
- **Shared target selection**: `selectWorktreeTargets` in `worktree.go` already encodes the full precedence table for worktrees, including the resolver fallback. Spec 30 will want the equivalent for `user`. Implementers may either call the resolver inline in this spec, or extract the no-tag rows of a `selectUserTargets` helper now. Either is acceptable, but the multi-match behavior of the two-argument form must stay as it is today, because question 4a's bulk-selection change belongs to spec 30.
- **Worktree writes land on the owning repository**: `AssignLocal` writes to `<repo.Path>/.git/config`. The resolver returns the owning repository when the working directory is inside a worktree, so no special handling is needed: the write goes to the main checkout's config, which git shares across all worktrees of that repository. The proof artifact that reads `user.email` from the main checkout after assigning from a worktree is what demonstrates this.
- **Pointer aliasing**: `repocontext.Resolve` returns a pointer into `cfg.Repositories`, so the existing code that mutates `repo.User`, `repo.Email`, and so on before `config.Save(cfg)` works unchanged for the resolved repository.
- **`--use-subdirs` check ordering**: The refusal must run after argument parsing and before `config.Load()` or any resolver call, so it costs nothing and cannot leave partial state. It must also run before the `--dry-run` branch, otherwise `--dry-run --use-subdirs` with one argument would print a move preview the real run refuses.
- **Help text rewrite scope**: Question 9 scopes the `gws` to `omgw` rewrite to the help text a spec touches. This spec touches the `assign` `Long` and the `user` parent `Long`, so both are rewritten in full rather than leaving a half-converted block. Other `user` subcommands' help (`list`, `add`, `show`, `remove`, `sync`) is not rewritten here. The `sync` help is rewritten by spec 32.
- **Stale worktree data**: Detection from inside a worktree depends on `Worktrees[].Path` in `config.json`, populated by `omgw worktree refresh`. A worktree created outside omgitworks and not yet refreshed will not resolve. This is the same limitation spec 25 accepted, and the error already points at the remedy.
- **Git version**: No new git invocations are introduced beyond the resolver's `git rev-parse --show-toplevel`, which spec 25 already requires.

## Security Considerations

No specific security considerations identified. The feature changes which repository receives a write that the command already performs; it does not change what is written. The resolver is read-only, and the `--use-subdirs` refusal happens before any filesystem or `~/.gitconfig` change. Proof artifacts should use fixture profiles with placeholder names and emails, not the user's real identity.

## Success Metrics

1. **`omgw user assign <profile>` succeeds from a repository root, from a nested subdirectory, and from inside one of that repository's worktrees**, writing the identity to the owning repository in all three cases.
2. **Every currently valid `assign` invocation produces identical output before and after the change**, including the multi-match error and `--dry-run` output for the two-argument form.
3. **Resolution failure exits non-zero with the spec 25 error**, with no change to `.git/config` or `config.json`.
4. **`--use-subdirs` with one argument exits non-zero before any change**, with and without `--dry-run`.
5. **No `gws` remains in the `user` or `assign` help text.**
6. **Test coverage** for the three resolution success paths, the failure path, the lookup-order case, the `--use-subdirs` refusal, both completion branches, and the preserved two-argument behavior.

## Open Questions

Both questions below are **resolved** by accepting the stated assumptions, consistent with how specs 25–28 handled residual authoring decisions. Neither affects implementation planning, and either can be revisited at spec review.

1. **Resolved — first-argument completion follows the spec 25 precedent.** When the working directory resolves to a tracked repository, the first argument completes to profile names, not repository names, because the one-argument form is the expected use from inside a repository. A user who wants the two-argument form from inside a repository types the repository name without completion help, exactly as `worktree add` behaves today. Offering both lists merged was considered and declined: profile and repository names could collide, and a merged list hides which is which.

2. **Resolved — `.` is not accepted on `assign`.** Omitting the repository is unambiguous on `assign`, so `.` is not needed and is listed as a non-goal. If spec 32 ships `sync .` and the inconsistency proves irritating in practice, accepting `.` on `assign` and `unassign` as an alias for omission would be a small follow-up.
