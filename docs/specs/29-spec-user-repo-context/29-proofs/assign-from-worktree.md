# `omgw user assign <profile>` from inside a worktree

Recorded against a binary built from this branch (`go build -o omgw ./cmd/omgitworks`), under a temporary `HOME` with `XDG_CONFIG_HOME` and `XDG_DATA_HOME` unset and `GIT_CONFIG_NOSYSTEM=1`, so no real identity or path appears. `$WS` is `$HOME/code`. Temporary paths are shown as `<tmp>`. Profiles use placeholder identities.

Continues from the setup in `assign-current-repo.md`, where `my-api` already carries the `work` profile.

## Transcript

The shell is inside the `feat-x` worktree. `git rev-parse --show-toplevel` confirms the working directory is the worktree, not the main checkout. The profile is applied to the owning repository `my-api`: the main checkout's local config reads the new email, and so does the worktree, because git shares one `.git/config` across all worktrees of a repository.

```text
### transcript 2: assign from inside a worktree
$ omgw worktree add feat-x
Created worktree for branch 'feat-x' at <tmp>/.local/share/gws/projects/my-api/feat-x
[exit 0]
$ cd $HOME/.local/share/gws/projects/my-api/feat-x
$ git rev-parse --show-toplevel
<tmp>/.local/share/gws/projects/my-api/feat-x
[exit 0]
$ omgw user assign personal
Assigned profile 'personal' to repository 'my-api'
  user.name:  Person
  user.email: personal@example.com
[exit 0]
$ git -C $WS/my-api config --local user.email
personal@example.com
[exit 0]
$ git config --local user.email
personal@example.com
[exit 0]
```

## Failure and preview cases

From an untracked directory the one-argument form fails with the spec 25 resolver error. `--use-subdirs` without a repository is refused before anything is loaded, with and without `--dry-run`. A one-argument `--dry-run` names the resolved repository.

```text
### failure cases
$ cd $HOME
$ omgw user assign work
Error: current directory is not inside a tracked repository
  Supply a repository argument, or run 'omgw add' to track this repository
Usage:
  omgitworks user assign <repository> <profile> [flags]

Flags:
      --dry-run       Preview changes without applying
  -h, --help          help for assign
      --use-subdirs   Move repository to profile subdirectory

[exit 1]
$ cd $WS/my-api
$ omgw user assign work --use-subdirs
Error: --use-subdirs requires an explicit repository argument: omgw user assign <repository> <profile> --use-subdirs
Usage:
  omgitworks user assign <repository> <profile> [flags]

Flags:
      --dry-run       Preview changes without applying
  -h, --help          help for assign
      --use-subdirs   Move repository to profile subdirectory

[exit 1]
$ omgw user assign work --use-subdirs --dry-run
Error: --use-subdirs requires an explicit repository argument: omgw user assign <repository> <profile> --use-subdirs
Usage:
  omgitworks user assign <repository> <profile> [flags]

Flags:
      --dry-run       Preview changes without applying
  -h, --help          help for assign
      --use-subdirs   Move repository to profile subdirectory

[exit 1]
$ omgw user assign personal --dry-run
Dry run - would assign profile 'personal' to repository 'my-api':

  No changes needed (already configured)
[exit 0]
```

Observation: cobra prints the usage block after each runtime error because `userAssignCmd` does not set `SilenceUsage`, unlike the worktree commands. That is pre-existing `assign` behavior, unchanged by this spec, and the `Usage:` line shows the `Use` string that task 4.0 updates.
