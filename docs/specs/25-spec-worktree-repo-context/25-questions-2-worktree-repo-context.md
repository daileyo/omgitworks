# 25 Questions Round 2 - Worktree CRUD

Round 1 answers are recorded in `25-questions-1-worktree-crud.md` and are treated as settled. This round covers only what your new request changed.

Context: you chose **9(C) — split into three specs, one per verb**, and then added a fourth capability: implicit repository detection, so `omgw worktree add <branch>` infers the repo from the current working directory, with an explicit `<repo>` argument taking precedence.

Relevant codebase finding: **there is no current-repo resolver in the codebase today.** `os.Getwd()` is used in `add.go` and `init.go`, but only to pick a scan root when registering new repositories — nothing maps a working directory back to a tracked repo. This spec would introduce that helper, which is why the questions below matter: every later worktree command will inherit its behavior.

## 1. Scope of Implicit Repo Detection

Which commands should infer the repo from the current directory?

- [ ] (A) `omgw worktree add` only — exactly what you asked for, nothing more
- [X] (B) Every `worktree` subcommand that takes an optional/required repo argument: `add`, `list`, `align`, plus the new `remove` and `refresh`
- [ ] (C) (B) plus the bare navigate form, so `omgw worktree <branch>` prefers worktrees of the current repo before searching the whole workspace
- [ ] (D) Other (describe)

**Recommended answer(s):** [(B)]

**Why these are recommended:**

- `(B)` makes one rule true everywhere — "omit the repo and I'll use the one you're standing in" — which is much easier to learn than a rule that holds for `add` but not `remove`.
- Building the resolver once and wiring it into all five call sites costs barely more than wiring it into one, whereas retrofitting `remove` and `refresh` later means reopening specs you have already validated.
- `(A)` is the literal ask and is perfectly defensible if you want the smallest change, but it will very likely leave you typing the repo name for `remove` right after not having to type it for `add`.
- `(C)` is the riskiest: `worktree navigate` is deliberately global today, and silently re-ranking results by current directory changes the behavior of a command that already works. Better as a separate follow-up if you want it.

## 2. Resolving "the repo I am in" — Worktrees

This is the important one. If you are standing inside a **worktree** — say `~/.local/share/gws/projects/my-repo/feat-auth` — `git rev-parse --show-toplevel` returns that worktree's path, which matches no tracked `repo.Path`. It does match an entry in `repo.Worktrees[].Path`.

Given omgitworks collects worktrees under the projects root, being inside a worktree is a *likely* place to run `omgw worktree add`. What should happen?

- [X] (A) Resolve to the owning repository — match the directory against `repo.Path` first, then against every `repo.Worktrees[].Path`, and return the owning repo either way. From inside `my-repo`'s `feat-auth` worktree, `omgw worktree add feat-b` creates `feat-b` for `my-repo`
- [ ] (B) Only `repo.Path` counts — running from inside a worktree is an error telling you to cd to the main repo
- [ ] (C) Resolve to the owning repo, but require a confirmation prompt when resolution came via a worktree rather than the main checkout
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` is almost certainly what you would want in the moment: you are working in one worktree and want a sibling worktree for another branch. There is exactly one correct answer for which repo that is, so there is nothing to disambiguate.
- Since every worktree path is already stored in `config.json`, this lookup is a map build over data omgitworks already has — no extra git calls.
- `(B)` makes the feature fail in the exact directory where worktree commands are most likely to be typed.
- `(C)` adds a prompt for a resolution that is never actually ambiguous.

### 2a. Resolution method

How should the current directory be matched to a tracked repo?

- [ ] (A) Match `os.Getwd()` (symlinks resolved) against tracked repo paths and worktree paths by prefix, walking up parent directories until something matches — no git invocation at all
- [X] (B) Run `git rev-parse --show-toplevel` to find the enclosing checkout, then match that exact path against tracked repos and worktrees
- [ ] (C) (B) first, falling back to (A) if the git call fails
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` needs no subprocess, so the common case stays fast, and it works from a subdirectory deep inside a repo (`src/internal/foo`) exactly as well as from the root.
- `(A)` also naturally handles the case where you are inside a git repo that omgitworks does not track: nothing matches, and you get the clean error from question 3 rather than a confusing partial resolution.
- `(B)` is more "correct" about git boundaries but adds a subprocess to every invocation and still requires the same matching step afterwards.
- The codebase already resolves symlinks for exactly this kind of comparison (`git.IsAligned`, `resolvePath`), so the pattern exists to reuse.

## 3. When the Current Directory Is Not a Tracked Repo

If no repo argument is given and the working directory does not resolve to a tracked repo, what should happen?

- [X] (A) Error with a clear message and a hint — e.g. "not inside a tracked repository; specify a repo name or run `omgw add` to track this one"
- [ ] (B) Error with the generic existing "no repository found matching ''" message
- [ ] (C) Fall back to prompting the user to pick a repo from the tracked list
- [ ] (D) Auto-track the current directory if it is a git repo, then proceed
- [ ] (E) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` tells the user both what went wrong and the two ways out, which matters because "I'm in a git repo, why doesn't this work?" is the predictable confusion when a repo is not yet tracked.
- `(D)` makes a config-mutating side effect happen as a consequence of a worktree command, which is a surprising amount of action for an omitted argument.
- `(C)` breaks scripted and non-interactive use for what is really a user error.

## 4. Flag and Argument Precedence

With `-t <tag>` (round 1) and implicit detection both in play, `worktree add` has three targeting modes. Confirm the precedence rules:

- [ ] (A) Explicit `<repo>` argument beats everything; `-t <tag>` beats implicit detection; implicit detection is the fallback. Passing both an explicit `<repo>` and `-t` is an error (mutually exclusive)
- [X] (B) Same, but `<repo>` plus `-t` is allowed and means "repos matching the tag AND the name pattern" (AND logic, like `omgw tag add --repo X --path Y`)
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` keeps each invocation to exactly one targeting mode, so the argument count unambiguously tells you what the command will do: two positionals = explicit repo, one positional with `-t` = tag scope, one positional bare = current repo.
- `(B)`'s AND semantics do exist elsewhere in the CLI (`tag add`), but there it applies to `--repo`/`--path` *flags*, not to a positional; mixing a positional repo with a tag filter reads as a contradiction rather than a refinement.
- Erroring on the combination is cheap to implement and cheap to relax later if you find you want `(B)`.

## 5. Spec Split and Ordering

With the fourth capability added, here is the proposed layout. The implicit-detection spec lands first because the other three consume its resolver.

| # | Spec | Delivers |
|---|------|----------|
| 25 | `worktree-repo-context` | Current-repo resolver; `add`, `list`, `align` accept an omitted repo |
| 26 | `worktree-add-by-tag` | `-t <tag>` bulk creation with partial-failure summary |
| 27 | `worktree-remove` | `remove`/`rm`, individual + tag, `--force`, `--dry-run`, `--yes` |
| 28 | `worktree-refresh` | `refresh`, individual + tag, metadata re-sync |

- [X] (A) Accept this layout and ordering
- [ ] (B) Accept the four specs but implement in a different order (say which)
- [ ] (C) Fold implicit detection into the `add` spec instead of giving it its own, leaving three specs
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- Putting the resolver first means `remove` and `refresh` are specified against a helper that already exists, rather than each re-specifying "and also it should detect the current repo."
- Spec 25 is still independently demoable — `omgw worktree add feat-x` from inside a repo is a visible, testable behavior change, not just internal plumbing.
- `(C)` would work, but it couples a cross-cutting helper to one verb's spec, and `remove`/`refresh` would then depend on a spec whose title suggests it is only about adding.

### 5a. Where the round 1 answers live

The answered `25-questions-1-worktree-crud.md` covers decisions spanning all four specs. Proposed: keep it in the `25-` directory as the shared clarification record and cross-reference it from specs 26–28. Any objection, or a preferred arrangement?

- [X] (A) Keep in `25-`, cross-reference from the others
- [ ] (B) Copy the relevant excerpts into each spec directory
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- A single source of record avoids the four copies drifting apart as specs are refined.
- Each spec will restate its own decisions in its Technical Considerations section anyway, so the cross-reference is for provenance rather than for reading.

## 6. Shell Integration Impact

Round 1 question 1 confirmed `(C)`: shell function and tab completion are in scope. Does implicit repo detection change anything there?

- [X] (A) Tab completion only — when completing `omgw worktree add <TAB>` from inside a tracked repo, suggest that repo's branch names first; no shell function changes needed
- [ ] (B) No changes needed at all; completion stays as it is
- [ ] (C) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- Completion is the one place the ambiguity is visible to the user: with one positional now meaning "branch," the existing repo-name completion on the first argument becomes misleading.
- No `cd` behavior changes, so the shell function itself is untouched — this stays a completion-only concern.
