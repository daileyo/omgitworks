# Task 05 Proofs - Shared cache and workers on worktree list

## Task Summary

`omgw worktree list` now accepts `--workers N`, resolved exactly as `omgw list` resolves it: the flag, then the `status_workers` preference, then 8. Because task 1.0 routed both commands through `fetchStatuses`, worktree status is written to and read from the same cache file as repository status, keyed by worktree path. This task adds the flag and the tests that prove the sharing rather than assuming it.

## What This Task Proves

- After `worktree list -s`, the shared cache file holds one entry per worktree path with its branch and a `LastChecked` time.
- A second run within the TTL reproduces the first output with no `git` binary reachable, so it was served from the cache.
- `--workers 3` beats `status_workers: 5`, which beats the default of 8, and the flag reaches the command's options and its help.
- `worktree_list.go` reads none of `list`'s flag variables, so the two commands cannot clobber each other.

## Evidence Summary

- Three new tests pass: `_CacheWritten`, `_CacheHit`, and `Workers_Resolution`.
- `make ci` exits zero.
- A timed transcript shows a cold run, a warm run, and the cache keys afterwards.
- A grep for `list`'s flag variables in `worktree_list.go` matches only the comment that explains their absence.

## Artifact: Cache and worker tests

**What it proves:** Cache sharing, cache hits without git, and worker resolution.

**Command:**

~~~bash
go test ./cmd/omgitworks -run 'TestWorktreeListStatus_Cache|TestWorktreeListWorkers' -v
~~~

**Result summary:** All pass. The cache-hit test empties `PATH` before the second run, so a miss would have rendered blank icons; the output is byte-identical to the first run.

~~~text
--- PASS: TestWorktreeListStatus_CacheWritten (0.08s)
--- PASS: TestWorktreeListStatus_CacheHit (0.07s)
--- PASS: TestWorktreeListWorkers_Resolution (0.00s)
ok  	github.com/daileyo/omgitworks/cmd/omgitworks	0.152s
~~~

## Artifact: Timed cold and warm runs

**What it proves:** The real binary writes worktree paths into the shared cache file and the second run is served from it.

**Why it matters:** This is the user-visible form of cache sharing. The progress spinner only renders when stderr is a terminal, which it is not under capture, so its absence is not observable here; the TTY-only rule is unchanged and covered by `internal/git/progress_test.go`. Paths are rewritten to `/home/user` after capture, so the PATH column is sized for the original, longer paths.

**Command:** see each `$` line. The cache file was removed before the cold run.

**Result summary:** The cold run fetches four worktrees with two workers; the warm run returns the same table faster; the cache file then lists the four worktree paths (and no repository paths, since `omgw list -s` was not run in this HOME).

~~~text
$ ls ~/.gws/status-cache.json
ls: cannot access '/home/user/.gws/status-cache.json': No such file or directory

$ time omgw worktree list -s --workers 2   # cold cache
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean      ✓  /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty      ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes
other    wip        ✗  /home/user/.local/share/gws/projects/other/wip      yes
elapsed 0.011s

$ time omgw worktree list -s                # warm cache, same TTL
REPO     BRANCH        PATH                                                                                                                                                   ALIGNED
-------  ------------  -----------------------------------------------------------------------------------------------------------------------------------------------------  -------
fixture  ahead   ↑1 ✓  /home/user/.local/share/gws/projects/fixture/ahead  yes
fixture  clean      ✓  /home/user/.local/share/gws/projects/fixture/clean  yes
fixture  dirty      ✗  /home/user/.local/share/gws/projects/fixture/dirty  yes
other    wip        ✗  /home/user/.local/share/gws/projects/other/wip      yes
elapsed 0.003s

$ python3 -c "..."   # keys now in the shared cache file
/home/user/.local/share/gws/projects/fixture/ahead
/home/user/.local/share/gws/projects/fixture/clean
/home/user/.local/share/gws/projects/fixture/dirty
/home/user/.local/share/gws/projects/other/wip
~~~

## Artifact: No shared flag variables with list

**What it proves:** `worktree list` owns its own `--status`, `--show-status`, `--color`, and `--workers` state.

**Command:**

~~~bash
grep -n 'flagWorkers\|flagColor\b\|flagStatus\b\|flagShowStatus\b' cmd/omgitworks/worktree_list.go
~~~

**Result summary:** One match, the comment at the top of the flag block explaining that the variables are deliberately separate.

~~~text
17:// list's flagStatus, flagColor, and flagWorkers so the two commands can never
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

Worktree status and repository status share one cache file, one TTL, and one worker-resolution rule, and a worktree status fetched once is served from the cache on the next run without touching git.
