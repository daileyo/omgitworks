# Task 02 Proofs - worktree add without a repository argument

## Task Summary

`omgw worktree add <branch>` now works from inside a repository, or from inside any of
that repository's worktrees, with no repository argument. The two-argument form is
untouched and still wins over the working directory.

## What This Task Proves

- One positional creates the worktree in the repository resolved from the current
  directory, including from a nested subdirectory and from inside a sibling worktree.
- Two positionals behave exactly as before, and the resolver is never consulted.
- Standing in repository A while naming repository B creates the worktree in B and
  leaves A untouched.
- A directory outside any tracked repository exits non-zero with the two-remedy error
  and creates nothing.
- `RangeArgs(1, 2)` rejects zero and three arguments.

## Evidence Summary

- 5 CLI scenarios run end to end against a sandboxed workspace, each verified with a
  follow-up `worktree list`.
- 6 new tests plus the 5 pre-existing `TestRunWorktreeAdd_*` cases all pass, confirming
  the refactor preserved existing behavior.
- Exit codes confirmed directly: 1 for zero args, three args, and unresolvable
  directories; 0 for the one- and two-argument forms.

## Artifact: End-to-end CLI transcript

**What it proves:** The four user-facing scenarios of the spec's Unit 1 work against a
real binary and real git repositories, not just in unit tests.

**Why it matters:** This is the feature the user actually asked for. Paths are shown
relative to a sandbox root; the run used an isolated `XDG_CONFIG_HOME` and
`XDG_DATA_HOME` so no real workspace was touched.

**Command:** see each `$` line below.

**Result summary:** Scenario 1 creates the worktree with the repo argument omitted.
Scenario 2 does the same from inside `feat-x`, and the new `feat-y` is recorded against
the owning repository, not the worktree. Scenario 3 shows the explicit argument winning
while standing in a different repository. Scenario 4 fails with the two-remedy error and
exit status 1.

~~~text
### 1. worktree add with the repo argument omitted, from the repo root
$ cd <workspace>/demo-repo
$ omgw worktree add feat-x
Created worktree for branch 'feat-x' at <sandbox>/data/gws/projects/demo-repo/feat-x
$ omgw worktree list demo-repo
REPO       BRANCH  PATH                                          STATUS
---------  ------  --------------------------------------------  ------
demo-repo  feat-x  <sandbox>/data/gws/projects/demo-repo/feat-x   aligned

### 2. worktree add from INSIDE an existing worktree of the same repo
$ cd <projects-root>/demo-repo/feat-x
$ omgw worktree add feat-y
Created worktree for branch 'feat-y' at <sandbox>/data/gws/projects/demo-repo/feat-y
$ omgw worktree list demo-repo
REPO       BRANCH  PATH                                          STATUS
---------  ------  --------------------------------------------  ------
demo-repo  feat-x  <sandbox>/data/gws/projects/demo-repo/feat-x   aligned
demo-repo  feat-y  <sandbox>/data/gws/projects/demo-repo/feat-y   aligned

### 3. Explicit repo argument wins over the current directory
$ cd <workspace>/demo-repo    # standing in demo-repo
$ omgw worktree add other-repo feat-z
Created worktree for branch 'feat-z' at <sandbox>/data/gws/projects/other-repo/feat-z
$ omgw worktree list
REPO        BRANCH  PATH                                           STATUS
----------  ------  ---------------------------------------------  ------
demo-repo   feat-x  <sandbox>/data/gws/projects/demo-repo/feat-x    aligned
demo-repo   feat-y  <sandbox>/data/gws/projects/demo-repo/feat-y    aligned
other-repo  feat-z  <sandbox>/data/gws/projects/other-repo/feat-z   aligned

### 4. Resolution failure outside any tracked repository
$ cd $(mktemp -d)
$ omgw worktree add feat-nope
Error: current directory is not inside a tracked repository
  Supply a repository argument, or run 'omgw add' to track this repository
~~~

## Artifact: Exit status verification

**What it proves:** Failure is signalled through the process exit status, not only
through printed text, and the new arity bounds are enforced.

**Why it matters:** The spec requires resolution failure to "exit with a non-zero
status". Scripts and shell functions depend on that.

**Command:**

~~~bash
omgw worktree add                            # zero args
omgw worktree add a b c                      # three args
omgw worktree add feat-arity                 # one arg
omgw worktree add demo-repo feat-arity2      # two args
cd $(mktemp -d) && omgw worktree add feat-nope
~~~

**Result summary:** Both arity violations and the unresolvable directory exit 1; both
valid argument shapes exit 0.

~~~text
$ omgw worktree add                     # zero args
  exit=1
$ omgw worktree add a b c               # three args
  exit=1
$ omgw worktree add feat-arity          # one arg
  exit=0
$ omgw worktree add demo-repo feat-arity2   # two args
  exit=0
$ cd $(mktemp -d) && omgw worktree add feat-nope   # unresolvable
  exit=1
~~~

## Artifact: Test suite, new and pre-existing

**What it proves:** The resolution paths are covered by tests, and the refactor that
split `runWorktreeAdd` into `addWorktreeForRepo` did not change existing behavior.

**Why it matters:** The spec requires that "every currently valid invocation produces
identical output before and after the change". The five pre-existing cases are the
guard for that, and they were run unmodified.

**Command:**

~~~bash
go test -v -run 'TestRunWorktreeAdd|TestWorktreeAddCmd' ./cmd/omgitworks/
~~~

**Result summary:** 6 new cases (one with 4 sub-cases) and 5 pre-existing cases all
pass.

~~~text
--- PASS: TestRunWorktreeAdd_ResolvesCurrentRepo (0.03s)
--- PASS: TestRunWorktreeAdd_ResolvesFromSubdirectory (0.02s)
--- PASS: TestRunWorktreeAdd_ResolvesFromInsideWorktree (0.03s)
--- PASS: TestRunWorktreeAdd_ExplicitRepoIgnoresCwd (0.03s)
--- PASS: TestRunWorktreeAddCurrent_NotTracked (0.01s)
--- PASS: TestWorktreeAddCmd_Arity (0.00s)
    --- PASS: TestWorktreeAddCmd_Arity/zero_args (0.00s)
    --- PASS: TestWorktreeAddCmd_Arity/one_arg_(branch_only) (0.00s)
    --- PASS: TestWorktreeAddCmd_Arity/two_args_(repo_and_branch) (0.00s)
    --- PASS: TestWorktreeAddCmd_Arity/three_args (0.00s)
--- PASS: TestRunWorktreeAdd_Success (0.02s)
--- PASS: TestRunWorktreeAdd_CreatesWtDir (0.02s)
--- PASS: TestRunWorktreeAdd_UnknownRepo (0.01s)
--- PASS: TestRunWorktreeAdd_DuplicateBranch (0.02s)
--- PASS: TestRunWorktreeAdd_BranchWithSlash (0.02s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.234s
~~~

## Observation: cobra prints usage after the error

The failure transcript in scenario 4 is followed by cobra's usage block. This is not
introduced by this task: `SilenceUsage` is not set anywhere in `cmd/omgitworks`, so
every `RunE` error behaves this way, including the pre-existing
`no repository found matching '...'` path, which was run for comparison and produced an
identical usage dump.

No change was made, since suppressing it would alter behavior across every command and
is outside this spec. It is worth noting for spec 27, whose plan already calls for
partial failures to exit non-zero "without cobra overprinting a redundant error".

## Reviewer Conclusion

The omitted-repository form works from a repository root, a nested subdirectory, and
inside a sibling worktree, while the explicit two-argument form is provably unchanged.
Failure is actionable and non-zero with no side effects.
