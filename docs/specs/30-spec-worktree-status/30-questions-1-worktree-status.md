# 30 Questions Round 1 - Worktree Status

Please answer each question below (select one or more options, or add your own notes). Feel free to add additional context under any question.

Context for all questions: today `omgw list -s` shows compact git status icons (`↓n ↑n ✓/✗`) beside each repository name, `omgw list -S` adds a full STATUS column (branch + icons), and a value on either flag filters on `clean`, `dirty`, `ahead`, `behind`. Status is fetched per path through `internal/git.GetStatus`, cached in the shared status cache keyed by path, and fetched in parallel with `--workers`. Worktrees are separate paths, so the same machinery can produce a status per worktree. `omgw worktree list` currently shows REPO, BRANCH, PATH, and a column named STATUS that means aligned/unaligned, with no git status at all.

## 1. Where should worktree status appear?

Which command surface should gain per-worktree git status?

- [X] (A) `omgw worktree list` only: add status flags to the worktree list subcommand, so `omgw worktree list my-repo -s` shows status for that repo's worktrees
- [ ] (B) `omgw list` only: when `-s`/`-S` is used, expand each repository that has worktrees into indented sub-rows, one per worktree, each with its own status
- [ ] (C) Both (A) and (B) in this spec
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` matches the request wording ("stat info on the worktrees in a repo") and lands where worktrees are already listed, with the repo argument and `.` scoping already in place.
- `(A)` leaves `omgw list` untouched. Spec 20 deliberately excluded worktree status from `omgw list` because its layout is per-repository, and sub-rows would break the compact multi-column mode and JSON shape that other users and scripts depend on.
- `(B)` is a much larger change with layout decisions of its own. If you want it, `(C)` is better split into a follow-up spec after `(A)` lands, so each has a clean demo and validation.

## 2. The existing STATUS column means aligned/unaligned

`omgw worktree list` already prints a column headed STATUS whose values are `aligned` or `(unaligned)`. Git status needs a home that does not collide with it.

- [X] (A) Rename the existing column to ALIGNED (values `yes` / `no`, or keep `aligned` / `(unaligned)`), and let STATUS mean git status, matching `omgw list`
- [ ] (B) Keep the existing STATUS column as is, and head the new git status column something else, such as GIT
- [ ] (C) Fold alignment into the PATH column (for example a `(unaligned)` suffix on the path) and free up STATUS for git status
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` makes "status" mean the same thing in `omgw list` and `omgw worktree list`, which is the whole point of extending the feature rather than inventing a new one.
- `(A)` is a visible output change: the header and docs example change, and existing tests that assert on the STATUS header need updating. This is a one-time cost and the column is not consumed by scripts today (there is no JSON output for worktree list).
- `(B)` avoids the rename but leaves two different meanings of "status" one command apart, which is confusing in docs and help text.
- `(C)` hides alignment in a column that is already the widest, and makes the path harder to copy.

## 3. Flag shape and display convention

How closely should the worktree list flags follow the `omgw list` dual-purpose convention?

- [X] (A) Mirror `omgw list` exactly: `-s/--status` bare appends compact icons to the BRANCH column, `-s <pattern>` also filters; `-S/--show-status` bare adds a full STATUS column (icons, since branch is already its own column), `-S <pattern>` also filters; filter terms are `clean`, `dirty`, `ahead`, `behind`
- [ ] (B) A single `-s/--status` flag: bare shows a STATUS column with icons, a value shows and filters; no separate compact mode or `-S`
- [ ] (C) Always show git status when the command runs, no flag needed; add `-s <pattern>` only as a filter
- [ ] (D) Other (describe)

**Recommended answer(s):** [(A)]

**Why these are recommended:**

- `(A)` means anyone who knows `omgw list -s` already knows `omgw worktree list -s`, including the filter terms, and the help text can reuse the same convention paragraph.
- `(A)` reuses the existing icon formatter and filter text helper in `list.go`, so the icons, colors, and filter vocabulary cannot drift between the two commands.
- `(B)` is simpler but introduces a third status flag shape in the same binary.
- `(C)` makes every `worktree list` run touch git for every worktree, which is slow on large workspaces and changes the default output everyone already has. Status should stay opt-in, as it is in `omgw list`.

## 4. How much `omgw list` parity should come along?

Select everything that should be in scope for this spec. Anything unselected is recorded as a non-goal.

- [X] (A) `--color auto|always|never`, so icons are colored on a TTY exactly as in `omgw list`
- [X] (B) `--workers N`, honoring the same `status_workers` preference, so large worktree sets fetch in parallel
- [X] (C) Shared status cache: read and write the same cache file `omgw list` uses, with the same TTL and stale-then-refresh behavior, so a `worktree list -s` right after `list -s` is instant
- [ ] (D) `-o json` output for `worktree list`, with status fields included only when a status flag is set (same rule as `omgw list -o json`)
- [ ] (E) Include the main repository checkout as a row alongside its worktrees, so status for the whole project is visible in one table
- [ ] (F) Other (describe)

**Recommended answer(s):** [(A), (B), (C)]

**Why these are recommended:**

- `(A)`, `(B)`, and `(C)` are the pieces that make the feature behave like `omgw list -s` rather than a slower, plainer imitation. All three reuse existing code, so they add little implementation cost and keep one status pipeline in the codebase.
- `(C)` in particular matters for correctness as well as speed: a shared cache means the two commands never disagree about the same path's status.
- `(D)` is a reasonable addition but it is new surface for `worktree list`, which has no JSON today. It can be its own small spec unless you want it now.
- `(E)` changes what `worktree list` means (today it lists linked worktrees only) and affects every row count assertion in the tests. It is a judgment call; leaving it out keeps the spec focused on the worktrees themselves, and `omgw list -s` already covers the main checkout.
