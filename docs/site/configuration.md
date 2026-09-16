# Configuration

omgitworks stores its configuration in `~/.config/gws/config.json`. The configuration includes:

- **version**: Config file format version
- **workspace**: Root directory of the workspace
- **profiles**: Array of git user profiles for managing identity across repositories
- **repositories**: Array of discovered repositories
- **preferences**: Optional preferences for controlling CLI behavior

## Configuration Example

```json
{
  "version": "1.1.0",
  "workspace": "/home/user/projects",
  "profiles": [
    {
      "name": "personal",
      "git_name": "Jane Doe",
      "email": "jane@personal.dev",
      "signing_key": "ABC123DEF456",
      "sign_commits": true
    },
    {
      "name": "work",
      "git_name": "Jane Doe",
      "email": "jane.doe@company.com"
    }
  ],
  "repositories": [
    {
      "name": "omgw",
      "path": "/home/user/projects/omgw",
      "remote_url": "https://github.com/daileyo/omgitworks.git",
      "type": "github",
      "visibility": "unknown",
      "tags": ["personal", "go"],
      "user": "Jane Doe",
      "email": "jane@personal.dev",
      "signing_enabled": true,
      "user_source": "local"
    },
    {
      "name": "my-api",
      "path": "/home/user/projects/my-api",
      "remote_url": "git@gitlab.com:user/my-api.git",
      "type": "gitlab",
      "visibility": "private",
      "tags": ["work", "backend"],
      "user": "Jane Doe",
      "email": "jane.doe@company.com",
      "user_source": "includeif",
      "worktrees": [
        {
          "path": "/home/user/.local/share/gws/projects/my-api/feat-auth",
          "branch": "feat-auth",
          "aligned": true
        }
      ]
    }
  ],
  "preferences": {
    "status_workers": 8
  }
}
```

## Field Reference

### Top-level Fields

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Config file format version (currently `"1.1.0"`) |
| `workspace` | string | Absolute path to the root directory of the workspace |
| `profiles` | array | List of git user profile objects (see below) |
| `repositories` | array | List of all discovered repository objects |
| `preferences` | object | Optional preferences for controlling CLI behavior (see below) |

### Profile Fields

Profiles are managed via `omgw user add`, `omgw user remove`, and related commands. See [User Management](commands-user.md) for details.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Profile identifier (e.g., `"work"`, `"personal"`) |
| `git_name` | string | Git `user.name` value |
| `email` | string | Git `user.email` value |
| `signing_key` | string | GPG signing key ID (optional) |
| `sign_commits` | boolean | Whether to enable `commit.gpgsign` (optional, defaults to `false`) |

### Repository Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Repository name (derived from the directory name) |
| `path` | string | Absolute path to the repository on disk |
| `remote_url` | string | Git remote URL (empty string if no remote is configured) |
| `type` | string | Detected hosting provider: `github`, `gitlab`, `ado`, `bitbucket`, or `unknown` |
| `visibility` | string | Inferred visibility: `private` (SSH URL) or `unknown` (HTTPS URL) |
| `tags` | array | List of custom tag strings managed via `omgw tag add` / `omgw tag remove` |
| `user` | string | Git `user.name` configured for this repository |
| `email` | string | Git `user.email` configured for this repository |
| `signing_enabled` | boolean | Whether commit signing is configured for this repository |
| `user_source` | string | Where the user config comes from: `global`, `local`, `includeif`, or `unknown` |
| `worktrees` | array | List of git worktrees for this repository (omitted when empty). See Worktree Fields below. |

### Worktree Fields

Each entry in the `worktrees` array represents a git worktree associated with a repository. Worktree data is populated during `omgw refresh` and updated by `omgw worktree add`, `omgw worktree align`, and `omgw worktree refresh`, which re-syncs it with git for chosen repositories.

| Field | Type | Description |
|-------|------|-------------|
| `path` | string | Absolute path to the worktree directory on disk |
| `branch` | string | Branch checked out in this worktree |
| `aligned` | boolean | Whether the worktree is inside the projects root |

### Preferences Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `status_workers` | integer | `8` | Number of concurrent workers for fetching git status. Also configurable per-invocation with `omgw list --workers`. |

## What is a "project"?

A **project** is a git repository plus any worktrees associated with it. The repository is the
main checkout, wherever you keep it; its worktrees live together under the projects root
described below.

## File Locations

omgitworks follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/).

| What | Location | Default |
|------|----------|---------|
| Config | `$XDG_CONFIG_HOME/gws/config.json` | `~/.config/gws/config.json` |
| Worktrees | `$XDG_DATA_HOME/gws/projects/<repo>/<branch>` | `~/.local/share/gws/projects/<repo>/<branch>` |

Worktrees are stored under the **data** directory rather than cache or state because they hold
real work — losing one loses uncommitted changes.

### Resolution order

| Step | Config | Worktrees |
|------|--------|-----------|
| 1 | `$XDG_CONFIG_HOME/gws` | `$XDG_DATA_HOME/gws/projects` |
| 2 | `<home>/.config/gws` | `<home>/.local/share/gws/projects` |

A value in `XDG_CONFIG_HOME` or `XDG_DATA_HOME` is only honored when it is an **absolute**
path. Per the specification, a relative value is treated as unset — otherwise the location
would depend on your current working directory.

### The same layout on every platform

The layout is identical on Linux, macOS, and Windows. On Windows, `<home>` is `%USERPROFILE%`:

```
C:\Users\<user>\.config\omgw\config.json
C:\Users\<user>\.local\share\omgw\projects\<repo>\<branch>
```

Windows has no XDG specification, but this is not an invention — **git does the same thing**.
Per `git-config(1)`, when `XDG_CONFIG_HOME` is unset git uses `$HOME/.config`, and Git for
Windows sets `$HOME` to `%USERPROFILE%`. So `C:\Users\<user>\.config\git\config` is
already a real, supported path there. omgw keeps its config beside git's own.

The benefit is that one set of instructions works everywhere, and a dotfile manager or backup
rule that knows `~/.config` knows where omgw lives too.

`XDG_CONFIG_HOME` and `XDG_DATA_HOME` are honored on Windows as well, so if you prefer the
native `%AppData%` location you can point them there explicitly.

!!! note "Windows path length"
    `C:\Users\<user>\.local\share\omgw\projects\<repo>\<branch>` plus a deep branch name
    can approach the 260-character `MAX_PATH` limit. If you hit it, enable long-path support:
    `Set-ItemProperty -Path "HKLM:\SYSTEM\CurrentControlSet\Control\FileSystem" -Name LongPathsEnabled -Value 1`
    (run as Administrator, then reboot), or set `XDG_DATA_HOME` to a shorter path such as `C:\gws`.

To view the raw config at any time:

```bash
cat "${XDG_CONFIG_HOME:-$HOME/.config}/gws/config.json"
```

## Upgrading from earlier versions

Earlier versions kept the config at `~/.gws/config.json` and worktrees in a `<repo>.wt/`
directory beside each repository. Both have moved.

**Your config migrates automatically.** The first time you run any command, omgw moves
`~/.gws/config.json` to its new location and says so:

```
note: moved config /home/user/.gws/config.json -> /home/user/.config/gws/config.json
note: removed empty /home/user/.gws
```

The old directory is removed only if the config was all it contained. If you kept anything
else there, it is left alone and named in the output.

**Your worktrees do not move on their own.** A worktree can hold uncommitted work, so omgw will
never relocate one without being asked. Until you ask, existing worktrees report as
`(unaligned)`:

```bash
omgw worktree list
# my-repo   feat-auth   /home/user/projects/my-repo.wt/feat-auth   (unaligned)
```

**This is expected, not an error.** "Aligned" now means "inside the projects root", so
worktrees in the old location no longer qualify. To move them:

```bash
omgw worktree align --dry-run   # preview
omgw worktree align             # move them
```

`align` relocates each worktree with `git worktree move`, updates the config, and removes the
emptied `<repo>.wt/` directory — which is the point: your project directory ends up holding
projects, not a `.wt` sibling for every repo you have ever branched.

!!! warning "Worktrees on a different filesystem"
    `git worktree move` is ultimately a rename and cannot cross filesystems. If your home
    directory and your repositories are on different mounts, `align` will say so and name both
    paths. Set `XDG_DATA_HOME` to a location on the same filesystem as your repositories.
