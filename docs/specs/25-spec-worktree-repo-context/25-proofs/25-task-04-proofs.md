# Task 04 Proofs - Context-aware tab completion

## Task Summary

`worktree add`, `worktree list`, and `worktree align` had no `ValidArgsFunction` at all
before this task. Each now completes its arguments, and `add`'s first positional
completes to whatever that argument will actually mean: branches when the working
directory resolves to a tracked repository, repository names when it does not.

## What This Task Proves

- `add`'s first argument suggests branch names inside a tracked repository and falls
  back to repository names outside one.
- `add`'s second argument suggests branches of the repository named first, regardless of
  the working directory.
- `list` and `align` offer `.` only when the working directory actually resolves, so the
  shell never suggests an argument that would fail.
- `.` is filtered on the typed prefix like every other suggestion.
- All three commands have a completion function registered.

## Evidence Summary

- 6 `__complete` transcripts, one per behavior, run against the built binary.
- 9 new tests (with 6 sub-cases) pass, plus 2 pre-existing completion tests.

## Artifact: add completion is context-aware

**What it proves:** The same argument position yields branches or repository names
depending on whether the current directory resolves — matching the command's own rule,
so completion never suggests something the command would reject.

**Why it matters:** The optional repository argument makes `add`'s first positional
ambiguous. If completion guessed wrong, the feature would be actively misleading.

**Command:**

~~~bash
omgw __complete worktree add ""                    # inside demo-repo
cd $(mktemp -d) && omgw __complete worktree add "" # outside any tracked repo
omgw __complete worktree add demo-repo ""          # second argument
~~~

**Result summary:** Inside the repository the suggestions are its branches. Outside, the
suggestions are the two tracked repository names. With a repository already named, the
second argument returns that repository's branches even though the working directory
does not resolve.

~~~text
### add: first arg inside a tracked repo -> branch names
$ omgw __complete worktree add ""
feat-arity
feat-arity-ok
feat-arity2
feat-x
feat-y
feature-auth
loose-demo
main
release-1
:4
Completion ended with directive: ShellCompDirectiveNoFileComp

### add: first arg outside any tracked repo -> repo names
$ cd $(mktemp -d) && omgw __complete worktree add ""
demo-repo
other-repo
:4
Completion ended with directive: ShellCompDirectiveNoFileComp

### add: second arg -> branches of the NAMED repo
$ omgw __complete worktree add demo-repo ""
feat-arity
feat-arity-ok
feat-arity2
feat-x
feat-y
feature-auth
loose-demo
main
release-1
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
~~~

## Artifact: "." suggested only where it would work

**What it proves:** `list` and `align` offer `.` alongside repository names inside a
tracked repository, and withhold it outside one.

**Why it matters:** Suggesting `.` where resolution would fail would hand the user a
completion that errors the moment they press enter.

**Command:**

~~~bash
omgw __complete worktree list ""
omgw __complete worktree align ""
cd $(mktemp -d) && omgw __complete worktree list ""
~~~

**Result summary:** Both subcommands list `.` first inside the repository. Outside it,
only the repository names remain.

~~~text
### list: "." offered inside a tracked repo
$ omgw __complete worktree list ""
.
demo-repo
other-repo
:4
Completion ended with directive: ShellCompDirectiveNoFileComp

### align: "." offered inside a tracked repo
$ omgw __complete worktree align ""
.
demo-repo
other-repo
:4
Completion ended with directive: ShellCompDirectiveNoFileComp

### list: "." withheld outside a tracked repo
$ cd $(mktemp -d) && omgw __complete worktree list ""
demo-repo
other-repo
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
~~~

## Artifact: Completion test suite

**What it proves:** Every branch of the completion logic is covered, including prefix
filtering of `.`, the beyond-two-arguments case, a non-repository path, and the fact
that all three commands actually have a function registered.

**Why it matters:** Shell completion is easy to break silently, since nothing fails at
build time if a `ValidArgsFunction` is dropped.
`TestWorktreeCompletionsRegistered` guards exactly that.

**Command:**

~~~bash
go test -v -run 'TestComplete|TestWorktreeCompletions' ./cmd/omgitworks/
~~~

**Result summary:** All cases pass, including the 2 pre-existing completion tests.

~~~text
--- PASS: TestCompleteRepoThenNone (0.00s)
--- PASS: TestCompleteRepoThenTags_SecondArg (0.00s)
--- PASS: TestCompleteWorktreeAdd_BranchesWhenResolved (0.02s)
--- PASS: TestCompleteWorktreeAdd_ReposWhenUnresolved (0.02s)
--- PASS: TestCompleteWorktreeAdd_PrefixFilter (0.02s)
--- PASS: TestCompleteWorktreeAdd_SecondArgument (0.02s)
--- PASS: TestCompleteWorktreeAdd_BeyondTwoArgs (0.01s)
--- PASS: TestCompleteWorktreeRepoOrDot (0.02s)
    --- PASS: TestCompleteWorktreeRepoOrDot/dot_offered_when_resolved (0.00s)
    --- PASS: TestCompleteWorktreeRepoOrDot/dot_withheld_when_unresolved (0.00s)
    --- PASS: TestCompleteWorktreeRepoOrDot/no_completions_after_the_first_argument (0.00s)
--- PASS: TestCompleteWorktreeRepoOrDot_PrefixFiltersDot (0.01s)
--- PASS: TestWorktreeCompletionsRegistered (0.00s)
    --- PASS: TestWorktreeCompletionsRegistered/add (0.00s)
    --- PASS: TestWorktreeCompletionsRegistered/list (0.00s)
    --- PASS: TestWorktreeCompletionsRegistered/align (0.00s)
--- PASS: TestCompleteBranchNames_NotARepo (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.128s
~~~

## Reviewer Conclusion

Completion now exists on all three subcommands and tracks the real meaning of each
argument, including refusing to suggest `.` where it would not resolve.
