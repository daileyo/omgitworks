# Task 01 Proofs - Tag flag and targeting precedence

## Task Summary

`omgw worktree add` gains `-t` / `--tag`, and all four rows of the spec's precedence
table are implemented in one selection function. This task decides *which* repositories
an invocation targets; creation is task 2.0.

## What This Task Proves

- Each of the four precedence rows selects exactly the expected repository set.
- Tag matching reuses `filter.MatchesExact`: exact, case-insensitive, wildcard-aware.
- A name pattern combined with a tag requires **both** (AND), matching `omgw tag add`.
- The current-directory resolver is consulted **only** when neither a name nor a tag is
  given, making detection the lowest-precedence source as spec 25 requires.
- An ambiguous name pattern is still rejected without `-t`; with `-t` the tag narrows the
  group and the run proceeds over the intersection.
- `--tag` given twice is rejected rather than silently using the last value.

## Evidence Summary

- 20 selection cases pass, including a sub-case per precedence row.
- CLI: `--tag` twice exits 1 with a clear message; an unmatched tag exits 1 naming the tag.

## Artifact: Precedence table test

**What it proves:** Every row of the spec's table resolves to the documented repository
set, checked against a four-repository fixture where the pattern `api` deliberately spans
a tagged and an untagged repository.

**Why it matters:** The precedence table is the whole of Unit 2 and the part of this spec
most likely to be got subtly wrong. Each row is asserted independently rather than
inferred.

**Command:**

~~~bash
go test -v -run TestWorktreeAddSelection_PrecedenceTable ./cmd/omgitworks/
~~~

**Result summary:** All four rows pass. Note the second row runs from inside `web-ui`
while selecting `backend`, proving the working directory does not leak into a tagged run.

~~~text
--- PASS: TestWorktreeAddSelection_PrecedenceTable
    --- PASS: .../1_positional,_no_tag_->_current_repo
    --- PASS: .../1_positional_with_tag_->_all_tagged
    --- PASS: .../2_positionals,_no_tag_->_name_pattern
    --- PASS: .../2_positionals_with_tag_->_name_AND_tag
~~~

## Artifact: Tag matching, AND semantics, and precedence ordering

**What it proves:** Tag matching is the established `MatchesExact` rule; the combined
filter excludes repositories failing either condition; and detection never overrides an
explicit filter.

**Command:**

~~~bash
go test -v -run 'TestWorktreeAddSelection_(TagMatching|CombinedAnd|ResolverNotConsulted)' ./cmd/omgitworks/
~~~

**Result summary:** All pass, including wildcard forms `back*` and `backen?`.

~~~text
--- PASS: TestWorktreeAddSelection_TagMatching
    --- PASS: .../exact
    --- PASS: .../case-insensitive
    --- PASS: .../wildcard
    --- PASS: .../wildcard_single_char
--- PASS: TestWorktreeAddSelection_CombinedAnd
--- PASS: TestWorktreeAddSelection_ResolverNotConsulted
    --- PASS: .../tag_ignores_the_working_directory
    --- PASS: .../pattern_ignores_the_working_directory
    --- PASS: .../detection_applies_only_when_neither_is_given
~~~

## Artifact: Ambiguity handling (pins the planning assumption)

**What it proves:** `api` alone matches three repositories and is rejected with today's
exact wording. The same pattern with `-t backend` returns both tagged repositories
without error.

**Why it matters:** The spec is ambiguous here — FR U2-6 scopes the ambiguity error to the
no-tag form, while the Technical Considerations imply the check still applies with a tag.
The plan resolved this in favour of the user story ("act on part of a tagged group"), and
this test is what pins that decision so a future reader sees it was deliberate.

**Command:**

~~~bash
go test -v -run TestWorktreeAddSelection_AmbiguityAfterTagFilter ./cmd/omgitworks/
~~~

**Result summary:** Passes. Reverting the decision changes this test and two sub-tasks,
nothing else.

~~~text
--- PASS: TestWorktreeAddSelection_AmbiguityAfterTagFilter (0.04s)
~~~

## Artifact: Flag and empty-selection contracts (CLI)

**What it proves:** `--tag` rejects repetition, and an unmatched selection exits non-zero
naming what was searched for.

**Result summary:** Both exit 1 with a single clear line and no usage block.

~~~text
$ omgw worktree add -t backend -t frontend feat-x
Error: --tag accepts a single value, but was given 2 times
   exit=1

$ omgw worktree add -t nosuchtag feat-x
Error: no repository found tagged 'nosuchtag'
   exit=1
~~~

Note `--tag` is declared as `StringArrayVarP` rather than `StringVarP` precisely so a
repeat is detectable; a plain string would silently keep the last value.

## Reviewer Conclusion

Selection is correct for every row of the precedence table, reuses the repository's own
matching rules, and keeps detection subordinate to explicit filters.
