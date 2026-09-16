# Task 03 Proofs - Refresh targets all, a pattern, the current repo, or a tag

## Task Summary

`omgw worktree refresh` accepts the targeting forms from spec Unit 2:

| Form | Selects |
| --- | --- |
| `refresh` | every tracked repository |
| `refresh <repo>` | every repository matching the name pattern |
| `refresh .` | the repository of the current directory |
| `refresh -t <tag>` | every repository carrying the tag |
| `refresh <repo> -t <tag>` / `refresh . -t <tag>` | repositories satisfying both |

Unmatched filters exit non-zero with an error naming them, and a repeated `--tag` is rejected.

The main risk in this task was reusing `selectWorktreeTargets`, the selection function
that `worktree add` and `worktree remove` share. It gets two of refresh's cases wrong:
with no arguments it picks the current repository, where refresh needs every repository,
and it rejects a name pattern matching several repositories, where refresh needs all of
them. Refresh builds its base set from `worktreeScopeFor`, the model `list` and `align`
use, and narrows it by tag.

## What This Task Proves

- Each form selects exactly the expected set. The no-argument case is run from *inside*
  a tracked repository and still selects all four.
- A name pattern matching several repositories selects all of them, with no ambiguity error.
- A name or `.` combined with a tag applies AND, not OR.
- A tag matching one repository takes the same path, and gives the same output, as a tag
  matching several. No single-repository mode is derived from the match count.
- `-t` given twice is rejected. Unmatched filters are errors naming the filter values,
  not partial failures.
- `.` outside a tracked repository returns the resolver's own error and refreshes nothing.
- Completion offers `.` only where it resolves, and `--tag` completes the tags in use.

## Evidence Summary

- 11 targeting and completion tests pass, 24 counting subtests.
- 6 of 6 mutants are caught, including the two reuse bugs that specs 26 and 27 found only
  at validation.
- CLI: `-t backend` refreshes the two backend repositories and leaves `web-ui` stale;
  error cases exit 1 with clear messages.
- `make ci` passes: vet, golangci-lint with 0 issues, race-detector tests.

## Artifact: Targeting tests

**What it proves:** The targeting contract, form by form, against a fixture of four
repositories. `api-core` and `api-edge` are tagged `backend`, `web-ui` is tagged
`frontend`, and `api-legacy` is untagged. The name pattern `api` deliberately spans
tagged and untagged repositories.

**Command:**

~~~bash
go test ./cmd/omgitworks -count=1 -v -run 'TestWorktreeRefreshTargets|TestWorktreeRefreshCompletion|TestWorktreeCompletionsRegistered'
~~~

**Result summary:** All pass.

~~~text
--- PASS: TestWorktreeCompletionsRegistered (0.00s)
--- PASS: TestWorktreeRefreshTargets_EachForm (0.03s)
--- PASS: TestWorktreeRefreshTargets_NameAndTagAreAnded (0.02s)
--- PASS: TestWorktreeRefreshTargets_DotAndTagAreAnded (0.02s)
--- PASS: TestWorktreeRefreshTargets_SingleMatchTagIsBulk (0.03s)
--- PASS: TestWorktreeRefreshTargets_RepeatedTagRejected (0.02s)
--- PASS: TestWorktreeRefreshTargets_UnmatchedFilters (0.03s)
--- PASS: TestWorktreeRefreshTargets_DotOutsideRepo (0.02s)
--- PASS: TestWorktreeRefreshCompletion_OffersDot (0.03s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.218s
~~~

| Spec Unit 2 targeting requirement | Test |
| --- | --- |
| No arguments and no tag refreshes every repository | `EachForm/no_argument_selects_every_repo,_even_inside_one` |
| `<repo>` matches by name pattern | `EachForm/name_pattern_selects_every_match_without_an_ambiguity_error` |
| `.` refreshes the current repository | `EachForm/dot_selects_the_current_repo`, `DotOutsideRepo` |
| `-t <tag>` matches exactly, single value | `EachForm/tag_selects_every_tagged_repo`, `RepeatedTagRejected` |
| `<repo> -t <tag>` applies AND | `NameAndTagAreAnded`, `DotAndTagAreAnded` |
| Unmatched filters exit non-zero, naming the filters | `UnmatchedFilters` (4 subtests) |

## Artifact: The tests fail against wrong implementations

**What it proves:** The tests catch the specific mistakes this task was most exposed to.

**Why it matters:** Every test passed on its first run. Two of these mutants reproduce bugs
that passed implementation tests in specs 26 and 27 and were caught only at validation.

**Method:** Each mutant was applied to `worktree_refresh.go` in turn, the refresh tests
run, and the file restored and verified byte-identical with `cmp`.

| Mutant | Caught by |
| --- | --- |
| Selection function delegates to `selectWorktreeTargets` | `EachForm` (no-argument and name-pattern subtests) |
| Command calls `selectWorktreeTargets` instead of the refresh selection | `UnmatchedFilters`, `DotOutsideRepo`, and every `TestWorktreeRefresh_*` core test |
| Name and tag combined with OR | `NameAndTagAreAnded`, `DotAndTagAreAnded`, `UnmatchedFilters` |
| Single-repository mode when exactly one repository matched | `SingleMatchTagIsBulk` |
| `.` that fails to resolve falls back to every repository | `DotOutsideRepo` |
| A repeated `--tag` silently keeps the last value | `RepeatedTagRejected` |

The delegation mutant failed in exactly the two predicted ways: no arguments collapsed
to the current repository, and a multi-match pattern was rejected.

~~~text
## MUTANT: selection function delegates to add/remove's table
    worktree_refresh_target_test.go:76: selected [web-ui], want [api-core api-edge api-legacy web-ui]
    worktree_refresh_target_test.go:73: unexpected error: multiple repositories match 'api', narrow your query
--- FAIL: TestWorktreeRefreshTargets_EachForm (0.03s)
RESTORED
~~~

**Note on the command-level mutant:** `EachForm` calls the selection function directly,
so a mutant that bypasses that function at the command level can't reach it. It was
caught by the tests that go through the command instead. Both entry points are therefore
covered, by different tests.

## Artifact: Scoped refresh from the real binary

**What it proves:** Tag, pattern, and `.` targeting, the error contract, and flag
parsing all work through cobra in the built binary.

**Why it matters:** The unit tests call the entry point with flag state set directly.
This transcript also exercises real `-t` parsing, including the repeated flag.

**Method:** Built with `go build`, run in a scratch workspace with an isolated `HOME`.
Three repositories were tagged with `omgitworks tag add`, and a worktree was created in
each with plain git. The scratch path is replaced by `$DEMO`, and table padding is
compacted after that substitution. Per-repository change output comes in task 4.0, so
successful runs are silent here.

**Result summary:** `-t backend` records worktrees for `api-core` and `api-edge` only,
and `web-ui` stays unrecorded until `.` is run inside it. Each rejected invocation exits 1
with an error naming its filters.

~~~text
# Setup (output hidden): api-core and api-edge tagged backend, web-ui tagged frontend.
# A worktree is then created with plain git in every repository, so all three are stale.
$ omgitworks worktree list
No worktrees found
(exit 0)

# Refresh only the backend repositories
$ omgitworks worktree refresh -t backend
(exit 0)

$ omgitworks worktree list
REPO  BRANCH  PATH  STATUS
--------  ------  --------  ------
api-core  feat-x  $DEMO/elsewhere/api-core  (unaligned)
api-edge  feat-x  $DEMO/elsewhere/api-edge  (unaligned)
(exit 0)

# web-ui was not refreshed: its external worktree is still unrecorded
# Unmatched filters and a repeated tag are rejected
$ omgitworks worktree refresh -t nope
Error: no repository found tagged 'nope'
(exit 1)

$ omgitworks worktree refresh web -t backend
Error: no repository found matching 'web' and tagged 'backend'
(exit 1)

$ omgitworks worktree refresh -t backend -t frontend
Error: --tag accepts a single value, but was given 2 times
(exit 1)

# A name pattern refreshes every match; '.' refreshes the current repository
$ omgitworks worktree refresh api
(exit 0)

$ cd $DEMO/ws/web-ui
$ omgitworks worktree refresh .
(exit 0)

$ omgitworks worktree list
REPO  BRANCH  PATH  STATUS
--------  ------  --------  ------
api-core  feat-x  $DEMO/elsewhere/api-core  (unaligned)
api-edge  feat-x  $DEMO/elsewhere/api-edge  (unaligned)
web-ui  feat-x  $DEMO/elsewhere/web-ui  (unaligned)
(exit 0)
~~~

## Artifact: Repository quality gate

**Command:**

~~~bash
PATH=$HOME/go/bin:$PATH make ci
~~~

**Result summary:** Vet is clean, lint reports 0 issues, and every package passes under
`-race`.

~~~text
Running linter...
0 issues.
...
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	7.773s
All CI checks passed!
~~~

## Deviations From the Task Plan

- **3.4:** `.` combined with a tag the current repository lacks reports
  `current repository '<name>' is not tagged '<tag>'`, since "matching '.'" would be
  meaningless. With no filters and nothing tracked, the command selects nothing and exits 0.
- **3.8:** Added `TestWorktreeRefreshTargets_DotAndTagAreAnded`, a `refresh` row in
  `TestWorktreeCompletionsRegistered`, and a `--tag` completion check.
- **Help text:** `worktree refresh --help` states that an omitted repository means every
  repository, unlike `add` and `remove`, so users of those commands aren't surprised.
  Full documentation is task 6.0.

## Reviewer Conclusion

Refresh's targeting matches spec Unit 2 and deliberately departs from add and remove
where the spec requires it. The two ways reuse could have broken it are each pinned by a
test that fails when the reuse is introduced.
