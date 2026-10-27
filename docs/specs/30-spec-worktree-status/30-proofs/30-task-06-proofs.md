# Task 06 Proofs - Worktree status is documented

## Task Summary

The site documentation now describes worktree status. The `List Worktrees` subsection of the core commands page gained a flags table, the filter terms, the attached-value rule, and example output for plain, `-s`, and `-S` listings with the renamed `ALIGNED` column. The `omgw list` `--status` row cross-references the new section. The configuration page's `status_workers` row names both commands, and its upgrade note shows the `ALIGNED` column's `no` value instead of the old `(unaligned)` marker. The parent `omgw worktree` help names the status flags on its `list` line.

## What This Task Proves

- The documentation matches the implementation: same flags, same terms, same `=` rule, same column name.
- No stale `(unaligned)` output remains anywhere in the site docs.
- The site builds under `mkdocs --strict`, which fails on broken links such as the new in-page cross-reference.
- The Go help and the documentation agree.

## Evidence Summary

- `mkdocs build --strict` exits zero.
- A grep for `(unaligned)` across the site docs returns nothing.
- `make ci` exits zero after the one-line help change in `worktree.go`.
- The rendered help is reproduced beside the documented flags below.

## Artifact: Strict site build

**What it proves:** The documentation builds, and the `#worktree-status` anchor and `#flag-convention` link resolve.

**Command:**

~~~bash
.venv/bin/mkdocs build --strict
~~~

**Result summary:** Exit zero. Tail of the output:

~~~text
INFO    -  Cleaning site directory
INFO    -  Building documentation to directory: site
INFO    -  Documentation built in 0.35 seconds
~~~

## Artifact: No stale alignment markers

**What it proves:** Every example that showed `(unaligned)` now shows the `ALIGNED` column.

**Command:**

~~~bash
grep -n '(unaligned)' docs/site/*.md
~~~

**Result summary:** No matches.

## Artifact: Help and documentation agree

**What it proves:** The flags, terms, precedence rule, and attached-value rule in `omgw worktree list --help` are the ones the docs describe.

**Command:**

~~~bash
omgw worktree list --help | sed -n '/^Git status/,/^Usage/p'
omgw worktree --help | grep 'worktree list'
~~~

**Result summary:** The help names `-s`, `-S`, the four terms, `-S wins`, and the `-s=dirty` form; the documentation's flags table and note say the same.

~~~text
Git status is opt-in and follows the same flag convention as "gws list":
  -s  (lowercase) = compact status icons after the branch name
  -S  (uppercase) = a separate STATUS column holding the icons
  Either flag with a value also filters: clean, dirty, ahead, behind.
  When both are given, -S wins.

Examples:
  gws worktree list                 # List all worktrees
  gws worktree list my-repo         # List worktrees for my-repo only
  gws worktree list .               # List worktrees for the current repo
  gws worktree list . -s            # Compact status after each branch
  gws worktree list my-repo -S      # Full STATUS column for my-repo
  gws worktree list -s=dirty        # Only worktrees with uncommitted changes
  gws worktree list my-repo -S=ahead  # Only my-repo worktrees ahead of origin
  gws worktree list -s my-repo      # my-repo is the repository, not a filter

Usage:
  gws worktree list [repo|.]            # List all worktrees, with git status via -s/-S
~~~

Documented flags, from `docs/site/commands-core.md`:

~~~text
#### Worktree status

Git status is opt-in and uses the same [flag convention](#flag-convention) and icons as `omgw list`: lowercase shows compact icons after the branch, uppercase adds a `STATUS` column, and a value on either flag also filters.

| Flag | Short | Description |
|------|-------|-------------|
| `--status` | `-s` | Show compact status icons after the branch, or show and filter with `-s=<pattern>` |
| `--show-status` | `-S` | Show a `STATUS` column, or show and filter with `-S=<pattern>` |
| `--color` | | Color output: `auto`, `always`, `never` |
| `--workers` | | Number of concurrent workers for status fetching (default: `8`, or the `status_workers` preference) |

Filter terms are `clean`, `dirty`, `ahead`, and `behind`, matched as partial patterns. When both `-s` and `-S` are given, `-S` wins.

~~~

## Artifact: Full quality gate

**Command:**

~~~bash
make ci
~~~

**Result summary:** Exit zero.

~~~text
Tests complete
All CI checks passed!
~~~

## Reviewer Conclusion

A reader of either the `omgw list` page or the worktree section can find the other, the examples show the column and icons as the binary renders them, and the one deliberate difference from `omgw list`, attaching filter values with `=`, is called out in an admonition.
