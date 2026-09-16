# 25 Questions Round 1 - Worktree CRUD

Please answer each question below (select one or more options, or add your own notes). Feel free to add additional context under any question.

Context: `omgw worktree` currently supports `navigate`, `list`, `add <repo> <branch>`, and `align [repo]`. This spec adds tag-scoped creation, removal (individual + tag), and refresh (individual + tag). Removal and refresh are new verbs with no existing implementation to pattern-match, so several behaviors need to be pinned down before the spec can be written.

## 1. Scope Confirmation — "both worktree"

Your request said "add more CRUD capability to both worktree". The word "both" is ambiguous. Which reading is correct?

- [X] (A) Typo/shorthand — the scope is the `omgw worktree` command family only (add-by-tag, remove, refresh)
- [ ] (B) Both the `worktree` subcommand family **and** the top-level repo commands (e.g. tag-scoped bulk operations on repos themselves)
- [X] (C) Both the CLI **and** the shell function/tab-completion integration must be updated in this spec
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A), (C)]

**Why these are recommended:**

- `(A)` keeps the spec to one coherent command family and matches all three bullets you listed, which are worktree-scoped.
- `(C)` is a small necessary add-on: `worktree` is already in the shell passthrough list, but new subcommands need completion entries, so folding that in avoids a half-finished feature.
- `(B)` would roughly double the scope and pull in repo-level lifecycle operations that none of your three bullets mention; if you do want that, it is better as a separate spec.

## 2. Tag Targeting Syntax

How should a tag be specified for the bulk (tag-scoped) forms of add, remove, and refresh?

- [X] (A) A `--tag` / `-t` flag, matching the `omgw list -t <tag>` filter flag already in the CLI (e.g. `omgw worktree add -t work feat-auth`)
- [ ] (B) A dedicated subcommand namespace (e.g. `omgw worktree add-tag <tag> <branch>`)
- [ ] (C) Positional, disambiguated from a repo name by a prefix (e.g. `omgw worktree add @work feat-auth`)
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` reuses the exact `-t/--tag` convention already established by `omgw list` and the `internal/filter` package (`Criteria.Tags`, AND logic, exact match with wildcard support), so users get one mental model.
- `(A)` also lets `--tag` compose with the existing `--repo`/`--path` targeting style used by `omgw tag add`, if you later want combined filters.
- `(B)` doubles the number of subcommands and duplicates help text; `(C)` introduces a sigil that does not appear anywhere else in the CLI.

### 2a. Repeatable tags

Should `--tag` be repeatable with AND logic (repo must have all listed tags), matching `omgw list`?

- [ ] (A) Yes — repeatable, AND logic (consistent with `omgw list -t`)
- [X] (B) No — single tag only, keep it simple
- [ ] (C) Repeatable with OR logic
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` matches `filter.Criteria.Tags` behavior that already exists, so the implementation is a direct reuse rather than a new matching rule.
- `(C)` would make worktree commands behave differently from `omgw list` for the same flag, which is a surprising inconsistency.

## 3. Bulk Create Semantics — `worktree add` by tag

"Create worktrees for all repos with a specified tag" implies one branch name applied across many repos. What should happen per repo?

- [X] (A) Same branch name for every matched repo; if the branch exists locally, check it out; if not, create it from the repo's current HEAD (this is what `git.AddWorktree` already does for the single-repo case)
- [ ] (B) Same as (A), but new branches are created from a configurable base ref (e.g. `--from main`), defaulting to the repo's default branch
- [ ] (C) Only create worktrees for repos where the branch **already exists**; skip repos where it does not
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` is exactly the existing single-repo behavior applied N times, so bulk create is a loop over proven code rather than new git semantics — the smallest correct slice.
- `(B)` is a reasonable later enhancement, but "default branch" detection is not currently implemented anywhere in the codebase and would add real work plus new failure modes.
- `(C)` makes the command unpredictable (you would not know how many worktrees you got without checking first), which is poor for a bulk operation.

### 3a. Partial failure behavior for bulk create

If some repos succeed and others fail (branch conflict, worktree already exists, git error), what should happen?

- [X] (A) Continue through all repos, collect errors, print a per-repo summary, exit non-zero if any failed — matches the existing `worktree align` error-collection pattern
- [ ] (B) Stop at the first failure and roll back nothing
- [ ] (C) Stop at the first failure and roll back worktrees already created in this run
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` matches how `runWorktreeAlign` already handles multi-repo work (collect `errors`, report count and detail at the end), so the output shape is familiar.
- Bulk operations across independent repos are naturally partial-success; `(B)` leaves the user in an undocumented half-state, and `(C)` adds rollback complexity for operations that are individually harmless and easy to undo.

## 4. What "refresh" Means for a Worktree

This is the most ambiguous of the three bullets — "refresh" could mean several materially different things. Which do you want?

- [X] (A) **Metadata re-sync only** — run `git worktree repair` + `prune` + `list --porcelain` and rewrite the `Worktrees` entries in `config.json` for the targeted repos (this is what `omgw refresh` already does workspace-wide, scoped down to specific repos/tags)
- [ ] (B) **Git content update** — run `git fetch` (and/or `pull --ff-only`) inside each targeted worktree so the checked-out branches are up to date
- [ ] (C) **Recreate** — remove and re-add the worktree at the same path from the same branch, discarding local state
- [ ] (D) Both (A) and (B): metadata re-sync plus a fetch, possibly gated by a flag
- [ ] (E) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` is the meaning consistent with the existing `omgw refresh` command, where "refresh" already means "re-scan and update stored metadata, clear caches" — reusing the word for a different meaning inside `worktree` would be confusing.
- `(A)` is also non-destructive and needs no network, which keeps it safe to run by tag across many repos.
- `(B)` introduces network operations, auth failures, and merge/divergence handling into a bulk command — worth having, but it is really a different feature (`omgw worktree sync` or a `--fetch` flag) and deserves its own scope.
- `(C)` is destructive and overlaps heavily with remove-then-add; it would need all the safety rails from question 6.

## 5. Remove Semantics

### 5a. What gets targeted

For `omgw worktree remove`, what identifies the worktrees to remove in each form?

- [X] (A) Individual: `omgw worktree remove <repo> <branch>` removes one worktree. By tag: `omgw worktree remove -t <tag> <branch>` removes the worktree for that branch in every repo carrying the tag
- [ ] (B) Individual: as in (A). By tag: `omgw worktree remove -t <tag>` removes **all** worktrees of every repo carrying the tag (no branch argument)
- [ ] (C) Support both forms in (A) and (B): branch argument optional, omitting it means "all worktrees for the matched repos"
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` mirrors the symmetry of bulk create ("this branch, across these repos"), which is the most likely real workflow: you finished a feature and want its worktrees gone everywhere.
- `(B)` is a much blunter and more dangerous operation — a single typo in the tag wipes every worktree for a group of repos.
- `(C)` is defensible if you want both, but it makes the destructive form the one with *fewer* arguments, which is the wrong ergonomic default. If you pick `(C)`, the no-branch form should require an explicit confirmation or `--all` flag.

### 5b. Dirty and locked worktrees

`git worktree remove` refuses to remove a worktree with uncommitted changes, untracked files, or a submodule, unless `--force` is passed. It also refuses locked worktrees. What should omgitworks do?

- [X] (A) Respect git's refusal by default; provide `--force` that passes through to `git worktree remove --force`; always skip locked worktrees with a clear message (matching how `worktree align` already skips locked worktrees via `git.IsWorktreeLocked`)
- [ ] (B) Always pass `--force`
- [ ] (C) Prompt interactively per dirty worktree
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` inherits git's own safety guarantee rather than reimplementing a dirty-check, and the locked-skip behavior already has precedent in `runWorktreeAlign`.
- `(B)` makes data loss the default, which is unacceptable for a command that can run across many repos at once.
- `(C)` breaks non-interactive/scripted use and does not compose with tag-scoped bulk operation.

### 5c. Branch deletion

Should removing a worktree also delete the branch it had checked out?

- [X] (A) No — never delete branches; removal only removes the worktree directory and git's worktree entry
- [ ] (B) Yes, always delete the branch too
- [ ] (C) No by default, with an opt-in flag (e.g. `--delete-branch`) that runs `git branch -d` (safe delete, fails if unmerged)
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` keeps the command's blast radius equal to its name: "remove worktree" removes a worktree. Branch deletion is separately recoverable-but-alarming, and `git worktree remove` itself never touches branches.
- `(C)` is a fine follow-up, but adding it now means specifying merged-vs-unmerged handling across N repos in the bulk form, which materially grows the spec.
- `(B)` risks destroying unmerged work with no undo.

### 5d. Empty directory cleanup

After removal, should an empty `<projects-root>/<repo>/` directory be cleaned up?

- [X] (A) Yes — remove the repo's projects directory if it is empty after removal, mirroring `removeEmptyLegacyDir` in `worktree_align.go`
- [ ] (B) No — leave directories in place
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` matches precedent already in the codebase and keeps the projects root tidy, which is the stated purpose of the XDG projects layout.
- The existing helper only removes directories that are genuinely empty, so nothing the user still has there is at risk.

## 6. Safety Rails for Destructive Bulk Operations

Which safety mechanisms should the tag-scoped forms have?

- [ ] (A) `--dry-run` on remove and refresh, matching the existing `worktree align --dry-run` pattern
- [ ] (B) An interactive confirmation prompt listing what will be removed, skippable with `--yes` / `-y`
- [X] (C) Both (A) and (B)
- [ ] (D) Neither — rely on `--force` semantics from question 5b alone
- [ ] (E) Other (describe)

**Recommended answer(s):** [(C)]

**Why these are recommended:**

- `(A)` already exists in this command family, so users expect it and the output format is established.
- `(B)` catches the specific failure mode that `--dry-run` does not: the user who never thinks to run the dry run. A one-time list-and-confirm is cheap, and `--yes` keeps scripted use working.
- `(D)` is too thin for an operation that can delete work across many repositories in one command.

## 7. Command Naming

What should the new subcommands be called?

- [X] (A) `omgw worktree remove` and `omgw worktree refresh`, with `rm` as an alias for remove
- [ ] (B) `omgw worktree remove` and `omgw worktree refresh`, no aliases
- [ ] (C) `omgw worktree delete` and `omgw worktree sync`
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `remove` matches `git worktree remove` exactly, so muscle memory carries over; `refresh` matches the existing top-level `omgw refresh`.
- The `rm` alias is a cheap ergonomic win and cobra supports aliases natively.
- `(C)` uses `sync`, which would be a better name if you chose option (B) or (D) in question 4 — flag that here if so.

## 8. Documentation and Proof Artifacts

What should count as proof that this feature works?

- [X] (A) Go unit tests per subcommand (following the existing `*_test.go` pattern with `t.TempDir()` and `bytes.Buffer` capture), plus updated `docs/site/commands-core.md`
- [X] (B) (A) plus captured real CLI transcripts against a scratch workspace, committed as proof artifacts
- [ ] (C) (A) plus a new dedicated `docs/site/commands-worktree.md` page added to the mkdocs nav
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A), (B)]

**Why these are recommended:**

- `(A)` matches the repository's established testing and docs conventions — worktree documentation currently lives in `commands-core.md`.
- `(B)` gives validation something observable to check beyond "tests pass", which matters most for destructive commands where the interesting behavior is what *did not* get deleted.
- `(C)` is a reasonable idea since the worktree section of `commands-core.md` is getting long, but it is a docs restructure that affects the nav and is easy to do separately.

## 9. Scope Sizing

This spec would add three capabilities (tag-scoped create, remove in two forms, refresh in two forms) to one command family. That sits at the upper end of a single SDD spec but stays coherent because all three share the same targeting logic.

- [ ] (A) Keep as one spec with three demoable units (tag-scoped add / remove / refresh)
- [ ] (B) Split into two specs: one for shared tag-targeting infrastructure + remove, a second for refresh
- [X] (C) Split into three specs, one per verb
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` works because the three verbs share one targeting resolver and one output/error-summary pattern; splitting would mean either duplicating that or landing a spec whose only deliverable is internal plumbing.
- Each verb is still an independently demoable vertical slice, so incremental progress is preserved within the single spec.
- `(C)` would triple the planning-audit and validation overhead for three closely related commands.
