# Shell Integration

## Setup

### Linux / macOS (bash / zsh)

Add two lines to your `~/.zshrc` (or `~/.bashrc`):

```bash
export PATH="$HOME/.local/bin:$PATH"
eval "$(omgitworks shell-init zsh)"   # or: shell-init bash
```

### Windows (PowerShell)

Add to your PowerShell `$PROFILE` (run `echo $PROFILE` to find its path):

```powershell
$env:Path = "$HOME\.local\bin;$env:Path"
Invoke-Expression (& omgitworks shell-init powershell | Out-String)
```

This works with both PowerShell 5.1 (Windows PowerShell) and PowerShell 7+ (pwsh).

The `shell-init` command outputs the `omgw` function and tab completion setup directly from the
binary, so you never hand-maintain the function — each new shell picks up whatever the
installed binary provides.

### Upgrading

**The `omgw` function is a snapshot taken when your shell starts.** Your rc file evaluates
`shell-init` once, at startup. Upgrading the binary afterwards does not update the function
that is already loaded in your running shells, so commands added by the new version are not
routed until you re-evaluate it:

```bash
eval "$(omgitworks shell-init zsh)"   # same shell, no restart needed
```

Or simply open a new shell.

Until you do, a newly added command falls through to repository navigation and reports
something like `No repositories found matching 'cd'` — confusing, because the real problem is
a stale function rather than a missing repository.

Two related traps when testing a local build:

- `make build` writes only to `./build/`. It does **not** install. Use `make install` (or
  `make use-dev`) to update the binary that `omgw` actually calls.
- Every local build reports `version dev`, so `--version` alone cannot tell an old build from
  a new one. Compare the `commit:` line instead:

  ```
  $ omgitworks --version
  omgitworks version dev
    commit: 907d3ae        <- this is what identifies the build
    built:  2026-08-11T05:20:05Z
  ```

To try a build without touching your installed setup, point `PATH` at it in a throwaway shell:

```bash
zsh -c 'export PATH="$PWD/build:$PATH"; eval "$(omgitworks shell-init zsh)"; omgw cd; pwd'
```

---

## Repository Navigation

Once set up, use `omgw` to jump to any tracked repository by name:

```bash
# Navigate to a repository by name — changes your directory
omgw my-repo

# Navigate to the workspace root
omgw cd

# Use subcommands directly through omgw
omgw list
omgw list --tag personal -S
omgw refresh
```

**Wildcard matching:**

```bash
# Wildcards work too (* = zero or more, ? = single character)
omgw "api-*"
omgw "?rontend"
```

**Multiple matches:**

When multiple repositories match, omgitworks displays a numbered list for selection:

```
Multiple repositories match 'api':

  1) my-api (github) /home/user/projects/my-api
  2) my-api-v2 (github) /home/user/projects/my-api-v2
  3) work-api (gitlab) /home/user/projects/work-api

Select repository [1-3]:
```

When piped (non-TTY), all matching paths are printed without prompting.

**No match suggestions:**

When no repositories match, omgitworks suggests similar names:

```
No repositories found matching 'aip'

Did you mean:
  my-api
  work-api
```

**Using the binary directly:**

```bash
# Print path without changing directory (useful for scripting)
omgitworks -g my-repo -q
# Output: /home/user/projects/my-repo
```

---

## Parent Navigation

Navigate to the **parent directory** of a repository (the directory that contains it):

```bash
# These all navigate to the parent directory of "my-repo"
omgw parent my-repo
omgw -p my-repo
omgw my-repo -p
```

This is useful when you want to work in the directory that contains a repository, rather than inside the repository itself.

**Using the binary directly:**

```bash
# Print parent path without changing directory
omgitworks parent my-repo -q
# Output: /home/user/projects
```

---

## Worktree Navigation

Navigate to git worktrees directly from the shell:

```bash
# Navigate to a worktree by branch name (searches all repos)
omgw worktree feat-auth

# Wildcard matching with interactive selection
omgw worktree "feat-*"

# Navigate to a worktree within a specific repo
omgw my-repo -wt feat-auth

# List all worktrees for a repo and choose interactively
omgw my-repo -wt
```

The shell function handles the `cd` automatically — `omgw worktree <branch>` and `omgw <repo> -wt <branch>` both change your working directory to the matched worktree path.

The `worktree` subcommands that don't navigate (`list`, `align`, `add`, `remove`/`rm`) are passed through to the binary without `cd`, so their output and any confirmation prompt reach your terminal:

```bash
omgw worktree list              # Lists worktrees (no cd)
omgw worktree align --dry-run   # Previews alignment (no cd)
omgw worktree add my-repo feat  # Creates worktree (no cd)
omgw worktree remove my-repo feat --dry-run  # Previews removal (no cd)
```

**Using the binary directly:**

```bash
# Print worktree path without changing directory
omgitworks worktree feat-auth -q
# Output: /home/user/.local/share/gws/projects/my-repo/feat-auth
```

See [Core Commands](commands-core.md#worktree-management) for full worktree command reference.

---

## Tab Completion

Tab completion is set up automatically by `shell-init` — no separate step needed if you followed the setup above.

For other shells:

**fish:**

```bash
omgitworks completion fish > ~/.config/fish/completions/omgitworks.fish
```

**Manual zsh setup (without shell-init):**

```bash
# omgitworks shell integration — do not edit, managed via shell-init
if ! type compdef &>/dev/null; then
  autoload -U compinit && compinit
fi
function omgw() {
  local _dest
  if [[ $# -eq 0 ]]; then
    omgitworks
    return
  fi
  case "$1" in
    list|init|add|refresh|print-workspace|tag|user|completion|shell-init|help|__*) omgitworks "$@" ;;
    worktree)
      case "$2" in
        list|align|add|remove|rm|"") omgitworks "$@" ;;
        *)
          _dest="$(omgitworks "$@" -q 2>/dev/tty </dev/tty)"
          [[ -n "$_dest" ]] && cd "$_dest"
          ;;
      esac
      ;;
    -p|--parent|parent)
      _dest="$(omgitworks parent "$2" -q 2>/dev/tty </dev/tty)"
      [[ -n "$_dest" ]] && cd "$_dest"
      ;;
    -*)
      omgitworks "$@"
      ;;
    *)
      if [[ "$2" == "-p" || "$2" == "--parent" ]]; then
        _dest="$(omgitworks parent "$1" -q 2>/dev/tty </dev/tty)"
      elif [[ "$2" == "-wt" ]]; then
        if [[ -n "$3" ]]; then
          _dest="$(omgitworks "$1" --worktree "$3" -q 2>/dev/tty </dev/tty)"
        else
          _dest="$(omgitworks "$1" --worktree -q 2>/dev/tty </dev/tty)"
        fi
        [[ -n "$_dest" ]] && cd "$_dest"
        return
      else
        _dest="$(omgitworks "$1" -q 2>/dev/tty </dev/tty)"
      fi
      [[ -n "$_dest" ]] && cd "$_dest"
      ;;
  esac
}
source <(omgitworks completion zsh)
compdef _omgitworks omgw
```

**Manual PowerShell setup (without shell-init):**

```powershell
# omgitworks shell integration — do not edit, managed via shell-init
function omgw {
    if ($args.Count -eq 0) {
        & omgitworks
        return
    }

    $first = $args[0]
    $rest = @()
    if ($args.Count -gt 1) {
        $rest = $args[1..($args.Count - 1)]
    }

    switch -Regex ($first) {
        '^(list|init|add|refresh|print-workspace|tag|user|completion|shell-init|help|__.*)$' {
            & omgitworks @args
            return
        }
        '^worktree$' {
            if ($rest.Count -eq 0) {
                & omgitworks @args
                return
            }
            $second = $rest[0]
            switch ($second) {
                { $_ -in 'list', 'align', 'add', 'remove', 'rm' } {
                    & omgitworks @args
                    return
                }
                default {
                    $dest = & omgitworks @args -q 2>&1 | Where-Object { $_ -is [string] }
                    if ($dest) { Set-Location $dest }
                    return
                }
            }
        }
        '^(-p|--parent|parent)$' {
            $second = if ($rest.Count -gt 0) { $rest[0] } else { $null }
            $dest = & omgitworks parent $second -q 2>&1 | Where-Object { $_ -is [string] }
            if ($dest) { Set-Location $dest }
            return
        }
        '^-' {
            & omgitworks @args
            return
        }
        default {
            $second = if ($rest.Count -gt 0) { $rest[0] } else { $null }

            if ($second -eq '-p' -or $second -eq '--parent') {
                $dest = & omgitworks parent $first -q 2>&1 | Where-Object { $_ -is [string] }
                if ($dest) { Set-Location $dest }
                return
            }
            elseif ($second -eq '-wt') {
                if ($rest.Count -gt 1) {
                    $branch = $rest[1]
                    $dest = & omgitworks $first --worktree $branch -q 2>&1 | Where-Object { $_ -is [string] }
                } else {
                    $dest = & omgitworks $first --worktree -q 2>&1 | Where-Object { $_ -is [string] }
                }
                if ($dest) { Set-Location $dest }
                return
            }
            else {
                $dest = & omgitworks $first -q 2>&1 | Where-Object { $_ -is [string] }
                if ($dest) { Set-Location $dest }
                return
            }
        }
    }
}
& omgitworks completion powershell | Invoke-Expression
(& omgitworks completion powershell) -replace 'omgitworks', 'omgw' | Invoke-Expression
```

---

## Workspace Navigation

Jump to the workspace root with `omgw cd`:

```bash
# Changes your directory to the workspace root
omgw cd

# Print only the path
omgw cd -q
```

`omgw cd` works in bash, zsh, and PowerShell. It takes no arguments — use `omgw <repo>` to
navigate to a repository.

### How it works

A process cannot change its parent shell's working directory, so `omgw cd` is a two-part
mechanism, the same one repository navigation uses:

1. `omgitworks cd -q` prints the workspace root to stdout
2. The `omgw` shell function captures that output and runs `cd` (or `Set-Location`) on it

Running the binary directly — `omgitworks cd` rather than `omgw cd` — therefore only prints
the path. The command detects this and tells you shell integration is missing rather than
appearing to do nothing.

### Scripting

`print-workspace` remains the primitive for scripts, since it only ever writes the path to
stdout:

```bash
cd "$(omgw print-workspace)"
```

Use `omgw cd` interactively; use `omgw print-workspace` in scripts.
