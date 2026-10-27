# 29 Questions Round 1 - User CRUD

Please answer each question below (select one or more options, or add your own notes). Feel free to add additional context under any question.

Context: `omgw user` currently supports `list`, `add`, `show`, `remove` (deletes a **profile**), `assign <repo> <profile> [--use-subdirs] [--dry-run]`, and `sync` (all repositories, no arguments). There is no non-deprecated way to clear a repository's local identity: the only implementation is the deprecated root flag `omgw --user -D [--tag ...] [--all]` (`runUserDelete` / `user.DeleteLocal`). Specs 25–28 gave `omgw worktree` current-repository detection, `-t` tag targeting, removal with dry-run and confirmation, and a scoped refresh. This round decides how much of that model carries over to `user`.

This file is intended to be the shared clarification record for the whole user series, the same way `25-questions-*` serves specs 25–28. Question 1a selected one spec per capability, so this directory is `29-spec-user-repo-context` and specs 30–32 point back to this file, the same way 26–28 point at `25-questions-1-worktree-repo-context.md`.

## 1. Capabilities in Scope

Which capabilities should the user series deliver?

- [X] (A) **Current-repository context**: `omgw user assign <profile>` run from inside a repository (or one of its worktrees) targets that repository, reusing the `internal/repocontext` resolver from spec 25
- [X] (B) **Assign by tag**: `omgw user assign -t <tag> <profile>` applies a profile to every repository carrying the tag
- [X] (C) **Unassign**: clear a repository's local git identity, individually, for the current repository, and by tag (replaces the deprecated `--user -D`)
- [X] (D) **Scoped sync**: `omgw user sync [repo|.] [-t <tag>] [--dry-run]` with per-repository change reporting, mirroring `worktree refresh`
- [ ] (E) **Profile update**: an `omgw user update <name> [--email ...] [--name ...]` subcommand (`user.UpdateProfile` already exists internally but has no command), optionally re-applying the change to repositories using that profile
- [ ] (F) **Profile removal cascade**: `omgw user remove <name>` offers to unassign the profile from the repositories that use it
- [ ] (G) Other (describe)

**Recommended answer(s):** [(A), (B), (C), (D)]

**Why these are recommended:**

- `(A)`–`(D)` line up one-to-one with specs 25–28 (context, add-by-tag, remove, refresh), which is what "very similar feature additions" most directly means, and they include your tag assign/remove example.
- `(C)` closes a real gap: the deprecation table points `--delete` users at "`user assign` (remove local config)", which does not exist.
- `(E)` and `(F)` are reasonable but are profile management, not repository targeting. They are better as a follow-up spec than as a fifth item that widens this series.

### 1a. Spec Split

How should the selected capabilities be packaged?

- [X] (A) One spec per capability (e.g. 29 context, 30 assign-by-tag, 31 unassign, 32 sync-scope), with this questions file shared across them, exactly like 25–28
- [ ] (B) Two specs: 29 = context + assign-by-tag + unassign; 30 = scoped sync
- [ ] (C) A single spec covering everything selected
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` keeps each spec to 2–3 demoable units, which is the size the 25–28 series proved workable, and lets each land as its own `feat:` release.
- `(C)` would produce a spec with 10+ demoable units, too large for one reviewable task list.
- `(B)` is workable, but the first spec would carry three separate safety concerns (overwrite, delete, current-directory inference) at once.

## 2. Unassign Command Name

`omgw user remove` already deletes a profile, so the `worktree remove` / `rm` naming cannot be reused. What should clearing a repository's identity be called?

- [X] (A) `omgw user unassign [repo]` (no alias)
- [ ] (B) `omgw user unassign [repo]` with alias `ua`
- [ ] (C) A flag on assign: `omgw user assign --remove [repo]`
- [ ] (D) `omgw user clear [repo]`
- [ ] (E) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` is the direct inverse of `assign`, so the pair reads naturally in help output and docs, and it cannot be confused with profile `remove`.
- `(C)` overloads `assign` with an operation that takes no profile, which makes its argument count depend on a flag, and that fights the arity-based targeting rules from spec 26.
- `(B)`'s alias saves little typing and adds another name to document. Add it later if you find you want it.

## 3. What Unassign Removes

`user assign` writes `user.name`, `user.email`, and, when the profile has them, `user.signingkey` and `commit.gpgsign` into the repository's local `.git/config`. What should unassign remove?

- [ ] (A) `user.name` and `user.email` only by default; `--all` also removes `user.signingkey` and `commit.gpgsign` (matches the deprecated `--user -D [--all]`)
- [X] (B) Always every key `assign` can write (name, email, signing key, gpgsign), making unassign the exact inverse of assign
- [ ] (C) Only name and email, with no option to remove signing configuration
- [ ] (D) Other (describe)

**Recommended answer(s):** [(B)]

**Why these are recommended:**

- `(B)` makes `assign` then `unassign` a true round trip. With `(A)`'s default, a repository can end up signing commits with a leftover key that belongs to the old identity while the name and email come from global config, which is a confusing half-state.
- `(A)` keeps backward-compatible behavior with the deprecated flag. The tradeoff is that it removes a `commit.gpgsign` you may have set by hand, independently of any profile. If you do that often, choose `(A)`.

## 4. Targeting Model

How should repositories be selected for `assign` and `unassign`?

- [X] (A) Mirror the spec 26 precedence table exactly: `-t`/`--tag` takes a single value matched with `filter.MatchesExact`, a repository name pattern and `-t` combine with AND, and the current-directory resolver applies only when neither is given
- [ ] (B) Same as (A), but `--tag` is repeatable with AND logic, matching `omgw list -t` and the deprecated `--user -u --tag`
- [ ] (C) Other (describe)

For reference, (A) yields:

| Invocation | Repositories targeted |
|---|---|
| `assign <profile>` | current repository |
| `assign -t <tag> <profile>` | all repositories with the tag |
| `assign <repo> <profile>` | repositories matching the name pattern |
| `assign <repo> <profile> -t <tag>` | name pattern **and** tag |
| `unassign` | current repository |
| `unassign -t <tag>` | all repositories with the tag |
| `unassign <repo>` | repositories matching the name pattern |
| `unassign <repo> -t <tag>` | name pattern **and** tag |

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` gives `worktree` and `user` one targeting model, so the flag means the same thing in both families, and it reuses the argument-handling code specs 26–28 already built.
- `(B)` matches the deprecated flags, but it would make `-t` single-valued under `worktree` and repeatable under `user`. You already chose single-value for worktrees in round 1 question 2a.
- **Note on unassign with no arguments:** under `(A)`, a bare `omgw user unassign` acts on the current repository. That is consistent with `worktree remove <branch>`, but it is a destructive command with no required argument. Question 5 covers the safety rail.

### 4a. Ambiguous Name Pattern

Today `user assign my <profile>` errors if `my` matches several repositories. What should happen without `-t`?

- [ ] (A) Keep erroring on multiple matches, as `worktree add` does; bulk selection requires `-t`
- [X] (B) Treat a multi-match pattern as a bulk selection (with the confirmation rules from question 5)
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` preserves today's behavior and makes a tag the only way to act on many repositories, so a short typo cannot rewrite several identities.
- `(B)` is more flexible, but it changes what an existing invocation does.

## 5. Safety Rails

`worktree add` got no prompt because it only adds. `assign` overwrites an existing local identity, and `unassign` deletes one. What rails should they get?

- [X] (A) Both `assign` and `unassign`: `--dry-run` preview, a confirmation prompt when more than one repository would change, `--yes`/`-y` to skip it, and a refusal (non-zero exit) when confirmation is needed but stdin is not a terminal. This is the spec 27 model.
- [ ] (B) Same as (A), but `assign` prompts only when a targeted repository already has a **different** local identity that would be overwritten
- [ ] (C) `--dry-run` on both, confirmation on `unassign` only
- [ ] (D) `--dry-run` only, no prompts
- [ ] (E) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- Committing under the wrong identity to many repositories is easy to do with one mistyped tag and tedious to undo, so a bulk overwrite deserves the same gate as a bulk delete.
- `(A)` keys the prompt on a simple rule users can predict: more than one repository means a prompt, the same rule spec 27 uses. `(B)` is smarter, but you can't tell ahead of time whether a run will prompt.
- Repositories that are already in the requested state (assign: identity already matches; unassign: no local identity) are skipped and reported. They don't count toward the "more than one" threshold.

## 6. `--use-subdirs` With Bulk or Current-Directory Targeting

`assign --use-subdirs` physically moves the repository and edits `~/.gitconfig`. Should it work with the new targeting forms?

- [X] (A) Allow it only with an explicit single repository argument; reject it with `-t` and with current-directory resolution
- [ ] (B) Allow it everywhere, including tag-scoped bulk moves
- [ ] (C) Allow it with `-t` but not with current-directory resolution
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- Moving the repository you are standing in leaves your shell in a deleted directory, and bulk moves combine path rewrites, worktree breakage, and `includeIf` edits across many repositories. That is a separate feature with its own failure modes.
- `(A)` leaves today's `--use-subdirs` behavior unchanged and keeps the series focused on local identity.

## 7. Partial Failure

How should bulk `assign` / `unassign` handle mixed results?

- [X] (A) Continue through every repository, report changed / skipped / failed counts, exit non-zero if any failed, no rollback, and save `config.json` once at the end (the spec 26/27 model)
- [ ] (B) Stop at the first failure
- [ ] (C)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` is the behavior you already chose for every bulk worktree operation, and its summary format and `errPartialFailure` exit handling exist and can be reused.
- Each repository's `.git/config` edit is independent, so there is nothing a stop-at-first-failure would protect.

## 8. Scoped Sync

If scoped sync (question 1, option D) is in scope, what should it include?

- [X] (A) Full `worktree refresh` parity: `sync` with no argument = all repositories (unchanged), `sync <repo>`, `sync .`, `sync -t <tag>`, AND-combined filters, per-repository change lines (old → new identity/source) for changed repositories only, a counted summary, and `--dry-run`
- [ ] (B) Targeting only (`<repo>`, `.`, `-t`), keeping today's single "Updated N repositories" line
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- Today `user sync` reports only a count, so you can't tell which repositories drifted. Change lines are the part of `worktree refresh` that made it useful.
- Sync never touches git, only `config.json`, so a dry run here is an exact preview. `worktree refresh` had to skip repair and prune and warn that its preview could differ; sync has no such caveat.
- The `.` convention (instead of omitting the argument) is required for the same reason as in spec 25: a bare `sync` already means "all repositories".

## 9. Completion, Docs, and Stale `gws` Text

The `user` help text still shows `gws` in every example (for instance `gws user assign my-repo work`). What should the series update?

- [X] (A) Each spec updates tab completion for the commands it touches, `docs/site/commands-user.md`, and the deprecation table's `--delete` pointer (to `user unassign`), and rewrites the `user` help examples it touches to `omgw`
- [ ] (B) Completion and docs only; leave the `gws` help text for a separate cleanup
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- These specs rewrite the `assign` and `user` help text anyway, so fixing the binary name at the same time costs almost nothing, and it avoids shipping new examples next to old `gws` ones.
- Pointing the deprecated `--delete` flag at a command that actually exists is part of finishing `(C)` in question 1.
