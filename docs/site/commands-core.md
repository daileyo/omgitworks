# Core Commands

## Repository Classification

### Automatic Classification

When repositories are discovered with `omgw init`, they are automatically classified based on their remote URL:

| Repository Type | Detected From |
|----------------|---------------|
| **GitHub** | `github.com` |
| **GitLab** | `gitlab.com`, `gitlab.*` |
| **Azure DevOps (ADO)** | `dev.azure.com`, `visualstudio.com` |
| **Bitbucket** | `bitbucket.org` |
| **Unknown** | Other or no remote URL |

#### Supported Remote URL Patterns

**GitHub:**

- HTTPS: `https://github.com/user/repo.git`
- SSH: `git@github.com:user/repo.git`

**GitLab:**

- HTTPS: `https://gitlab.com/user/repo.git`
- SSH: `git@gitlab.com:user/repo.git`
- Self-hosted: `https://gitlab.company.com/user/repo.git`

**Azure DevOps:**

- HTTPS: `https://dev.azure.com/org/project/_git/repo`
- HTTPS (legacy): `https://org.visualstudio.com/project/_git/repo`
- SSH: `git@ssh.dev.azure.com:v3/org/project/repo`

**Bitbucket:**

- HTTPS: `https://bitbucket.org/user/repo.git`
- SSH: `git@bitbucket.org:user/repo.git`

### Visibility Detection

Repository visibility is inferred from the remote URL protocol:

- **Private**: SSH URLs (`git@...` or `ssh://...`) typically require authentication
- **Unknown**: HTTPS URLs could be public or private (requires authentication check)

---

## List Repositories

```
omgw list [flags]
```

List all tracked repositories with optional filtering and display options.

By default, only repository names are shown in a compact multi-column layout. Use flags to filter results and control which columns are displayed.

### Flag Convention

`omgw list` uses a **dual-purpose flag convention**:

- **Lowercase flags** (`-t`, `-y`, `-s`, etc.) = **filter only** — narrow results without adding a column
- **Uppercase flags** (`-T`, `-Y`, `-S`, etc.) = **show column** — display the column, optionally filtering when a value is provided

For example:

- `omgw list -t work` — filter by tag "work", no tag column shown
- `omgw list -T work` — filter by tag "work" AND show the tags column
- `omgw list -T` — show the tags column with no filter applied

### Filter Flags (Lowercase)

| Flag | Short | Description |
|------|-------|-------------|
| `--type` | `-y` | Filter by type (exact match: `github`, `gitlab`, `ado`, `bitbucket`, `unknown`) |
| `--visibility` | `-i` | Filter by visibility (exact match: `private`, `unknown`) |
| `--tag` | `-t` | Filter by tag (exact match, single value) |
| `--path` | `-p` | Filter by path pattern (partial match) |
| `--status` | `-s` | Show compact status in name column, or filter by status pattern (partial match) |
| `--show-user` | `-u` | Filter by user name (partial match) |
| `--remote` | `-r` | Filter by remote URL pattern (partial match) |
| `--remote-raw` | `-b` | Filter by raw remote URL pattern (partial match) |
| `--name` | `-n` | Filter by repository name (partial match, supports wildcards) |

### Show Column Flags (Uppercase)

These flags show a column in the output. When provided with a value, they simultaneously show the column and filter by that value.

| Flag | Short | Description |
|------|-------|-------------|
| `--show-type` | `-Y` | Show type column, or show and filter by type |
| `--show-visibility` | `-I` | Show visibility column, or show and filter by visibility |
| `--show-tag` | `-T` | Show tags column, or show and filter by tag |
| `--show-path` | `-P` | Show path column, or show and filter by path |
| `--show-status` | `-S` | Show status column, or show and filter by status |
| `--show-user-col` | `-U` | Show user column, or show and filter by user |
| `--show-remote` | `-R` | Show remote column, or show and filter by remote |
| `--show-remote-raw` | `-B` | Show raw remote column, or show and filter by raw remote |

### Display Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | `table` | Output format: `table` or `json` |
| `--verbose` | `-v` | | Verbose output: `-v` shows type, visibility, tags, path; `-vv` shows all columns |
| `--workers` | | `8` | Number of concurrent workers for status fetching |
| `--color` | | `auto` | Color output: `auto`, `always`, `never` |

!!! note "Filter behavior"
    - `--type`, `--tag`, and `--visibility` use **exact matching**.
    - `--tag`/`-t` accepts a **single value** (not repeatable).
    - All other filter flags use partial/pattern matching.

### Examples

**Default listing (compact multi-column names):**

```bash
omgw list
```

```
Found 15 repositories:

my-project       work-api         client-site      my-api
frontend-app     docs-site        infra-tools      backend-svc (wt)
mobile-app       shared-libs      auth-service     data-pipeline
ml-models        cli-tools        test-harness
```

Repositories with git worktrees show an orange `(wt)` indicator next to their name. This indicator appears in all display modes (compact, table, and JSON).

**Verbose listing (stored data columns):**

```bash
omgw list -v
```

```
Found 15 repositories:

NAME              TYPE       VISIBILITY  TAGS              PATH
----------------  ---------  ----------  ----------------  ------------------------------------
my-project        github     private     personal, web     /home/user/projects/my-project
work-api          gitlab     private     work              /home/user/projects/work-api
client-site       bitbucket  unknown     client, archived  /home/user/projects/client-site
```

**Show specific columns:**

```bash
omgw list -YTSP
```

**Compact status (icons in name column):**

```bash
omgw list -s
```

```
Found 15 repositories:

NAME
------------------------------
my-project                   ✓
work-api                ↑2   ✗
client-site          ↓1      ✓
```

Use `-s` with a value to filter by status:

```bash
omgw list -s dirty         # Show only dirty repos
omgw list -s clean         # Show only clean repos
omgw list -s ahead         # Show only repos ahead of remote
```

When both `-s` and `-S` are specified, `-S` wins and the full STATUS column is shown.

**Full git status column:**

```bash
omgw list -S
```

```
Found 15 repositories:

NAME              STATUS
----------------  -----------------------------------------
my-project        main                                    ✓
work-api          develop                            ↑2   ✗
client-site       main                          ↓1        ✓
```

**Status Indicators:**

- `✓` = Clean working tree (no uncommitted changes)
- `✗` = Dirty working tree (uncommitted changes)
- `↑N` = N commits ahead of remote
- `↓N` = N commits behind remote

Status icons are displayed in order: behind → ahead → clean/dirty. Status icons are colorized when the terminal supports it (controlled by `--color`).

Branch names longer than 30 characters are truncated with `...`.

A `⚠` indicator in the NAME column means the repository's git user configuration has drifted from the assigned profile.

**Filtering:**

```bash
# Filter by repository type (exact match)
omgw list -y github

# Filter by visibility
omgw list -i private

# Filter by single tag (exact match)
omgw list -t personal

# Filter by repository name (partial match, case-insensitive)
omgw list -n project

# Filter by remote URL pattern
omgw list -r github.com

# Filter by status pattern
omgw list -s dirty

# Combine multiple filters
omgw list -y gitlab -t work -n api

# Filter and show column simultaneously
omgw list -T work -S
```

**JSON output:**

By default, JSON output only includes the `name` field:

```bash
omgw list -o json
```

```json
[
  {
    "name": "my-project"
  }
]
```

Add show-column flags to include additional fields:

```bash
omgw list -o json -YTP
```

```json
[
  {
    "name": "my-project",
    "type": "github",
    "tags": ["personal", "web"],
    "path": "/home/user/projects/my-project"
  }
]
```

---

## Initialize Workspace

```
omgw init [directory]
```

Initialize a workspace by scanning a directory for git repositories. Defaults to the current directory if no path is given.

**What it does:**

- Recursively scans the directory for git repositories
- Extracts repository metadata (name, path, remote URL)
- Detects repository type and git user configuration
- Saves the configuration to `~/.config/gws/config.json` (see [Configuration](configuration.md#file-locations))

**Examples:**

```bash
# Initialize in current directory
omgw init

# Initialize in a specific directory
omgw init ~/projects

# Initialize with absolute path
omgw init /path/to/your/workspace
```

---

## Add Repository

```
omgw add [path] [flags]
```

Add a single git repository to the workspace. Defaults to the current directory if no path is given.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--recursive` | `-r` | `false` | Recursively add all git repositories found under the path |

!!! note
    When `--recursive` is used, the scan always operates from the **current working directory**, regardless of the path argument.

When adding a repository that lives **outside the workspace root**, a symlink is automatically created inside the workspace directory pointing to the external repository.

**Examples:**

```bash
# Add current directory as a repository
omgw add

# Add a specific repository
omgw add ~/projects/my-repo

# Recursively add all repos in current directory
omgw add -r

# Recursively add all repos under a path
omgw add ~/projects -r
```

---

## Refresh Workspace

```
omgw refresh
```

Re-scan the workspace and update repository metadata.

**What it does:**

- Re-scans workspace for new repositories
- Removes repositories whose paths are no longer valid
- Updates remote URLs and classification
- Re-detects git user configuration
- Discovers git worktrees for all tracked repositories
- Clears and rebuilds git status cache
- Preserves all custom tags

**When to use:**

- After adding new repositories to your workspace
- When remote URLs have changed
- To force update of cached git status
- After bulk repository operations

**Example output:**

```
Refreshing workspace at: /home/user/projects
Detecting git user configuration...
Cleared git status cache

Refresh complete!
Total repositories: 15
Removed 1 repository (path no longer valid)
Found 2 new repositories
Updated 3 repositories
Repositories with user configuration: 12
Repositories with worktrees: 3
```

The conditional lines (Removed, Found, Updated, Repositories with user configuration, Repositories with worktrees) only appear when their counts are greater than zero.

---

## Navigate to Workspace Root

```
omgw cd [flags]
```

Navigate to the workspace root directory.

This requires [shell integration](shell-integration.md): the binary prints the path and the
`omgw` shell function performs the directory change. Without it, `omgw cd` only prints the path
and says so.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--quiet` | `-q` | `false` | Suppress verbose output, print only the path |

**Examples:**

```bash
# Navigate to the workspace root
omgw cd

# Print only the path
omgw cd -q
```

`omgw cd` takes no arguments — use `omgw <repo>` to navigate to a repository.

---

## Print Workspace

```
omgw print-workspace
```

Print the workspace root path to stdout.

`print-workspace` is the scripting primitive; `omgw cd` is the interactive command. Because
`print-workspace` only ever writes the path to stdout, it stays the right choice inside
scripts and command substitution:

```bash
cd "$(omgw print-workspace)"
```

---

## Worktree Management

omgitworks provides first-class support for [git worktrees](https://git-scm.com/docs/git-worktree). A **project** is a repository plus its worktrees: the repository stays wherever you keep it, and its worktrees are collected under the XDG projects root.

```
~/projects/
  my-repo/                              # Main repository — stays put

~/.local/share/gws/projects/
  my-repo/                              # This repo's worktrees
    feat-auth/                          # Worktree for feat-auth branch
    fix-login/                          # Worktree for fix-login branch
```

The root is `$XDG_DATA_HOME/gws/projects`, defaulting to `~/.local/share/gws/projects`. Keeping
worktrees out of your project directory is the point: no `.wt` sibling appears next to every
repo you branch. See [Configuration](configuration.md#file-locations) for the full layout and
[Upgrading](configuration.md#upgrading-from-earlier-versions) if you have worktrees in the old
`<repo>.wt/` location.

### List Worktrees

```
omgw worktree list [repo|.]
```

List all git worktrees across tracked repositories. Optionally filter to a single repo, or to the current one with `.`.

```bash
# List all worktrees across all repos
omgw worktree list

# List worktrees for a specific repo
omgw worktree list my-repo

# List worktrees for the repo you are standing in
omgw worktree list .
```

Omitting the argument still means **all** repositories. Use `.` to scope to the current one — see [Current-Repository Targeting](#current-repository-targeting).

**Example output:**

```
REPO          BRANCH        PATH                                        STATUS
------------  ------------  ------------------------------------------  ----------
my-repo       feat-auth     /home/user/.local/share/gws/projects/my-repo/feat-auth    aligned
my-repo       fix-login     /home/user/.local/share/gws/projects/my-repo/fix-login    aligned
other-repo    experiment    /tmp/other-experiment                        (unaligned)
```

Worktrees inside the projects root are marked `aligned`. Worktrees elsewhere — including any still in a legacy `<repo>.wt/` directory — are marked `(unaligned)`.

### Add Worktree

```
omgw worktree add [repo] <branch>
```

Create a new worktree. It is created at `<projects-root>/<repo>/<branch>`, defaulting to `~/.local/share/gws/projects/<repo>/<branch>`.

```bash
# Create a worktree for a new branch
omgw worktree add my-repo feat-new-feature

# Branch names with slashes are preserved
omgw worktree add my-repo hotfix/urgent-fix

# Omit the repo when you are standing in it
cd ~/code/my-repo
omgw worktree add feat-new-feature
```

The directory is created automatically if it doesn't exist. If the branch already exists in the repo, it is checked out into the worktree. If the branch doesn't exist, a new branch is created.

When the repo argument is omitted, it is determined from your current directory — see [Current-Repository Targeting](#current-repository-targeting). Naming a repo explicitly always takes precedence, so `omgw worktree add other-repo feat-x` acts on `other-repo` no matter where you are standing.

Tab completion follows the same rule: inside a tracked repository the first argument completes to that repository's **branch names**, and outside one it completes to **repository names**.

### Align Worktrees

```
omgw worktree align [repo|.] [--dry-run]
```

Move all unaligned worktrees into the projects root using `git worktree move` (requires Git 2.17+). This is also how you migrate worktrees from the legacy `<repo>.wt/` layout.

```bash
# Preview what would be moved
omgw worktree align --dry-run

# Align all repos
omgw worktree align

# Align only a specific repo
omgw worktree align my-repo

# Align only the repo you are standing in
omgw worktree align .

# Preview just the current repo
omgw worktree align . --dry-run
```

As with `list`, omitting the argument still means **all** repositories, and `.` scopes to the current one.

**Example dry-run output:**

```
Dry run — no changes will be made:

Would move [my-repo] experiment
  from: /tmp/my-experiment
  to:   /home/user/.local/share/gws/projects/my-repo/experiment

Total: 1 worktree to align
```

**Behavior details:**

- Locked worktrees are skipped (with a message explaining why)
- If two worktrees would produce the same directory name, a `-dup-NN` suffix is appended
- If a move fails partway, the worktree is rolled back to its original location
- After alignment, worktree data is re-discovered and saved to config

### Current-Repository Targeting

`omgw worktree add` lets you omit the repo argument, and `omgw worktree list` and `omgw worktree align` accept `.`, when you want to act on the repository you are already standing in.

In every case the repository is determined the same way:

- The enclosing checkout is found with `git rev-parse --show-toplevel`, so any subdirectory works, at any depth.
- That path is matched against your tracked repositories, then against their recorded worktrees.
- **Standing inside a worktree resolves to the repository that owns it**, so you can create a sibling worktree without returning to the main checkout.

Symlinked paths are resolved on both sides, so a symlinked workspace or data directory still matches.

If the current directory is not inside a tracked repository — including when it is not a git repository at all — the command exits non-zero with:

```
Error: current directory is not inside a tracked repository
  Supply a repository argument, or run 'omgw add' to track this repository
```

Worktree paths come from your config, which is populated by `omgw refresh`. A worktree created outside omgitworks and not yet refreshed will not resolve; run `omgw refresh` first.

Two things deliberately do **not** change:

- Bare `omgw worktree list` and bare `omgw worktree align` still mean *all* repositories. Only `.` narrows them.
- `omgw worktree <branch>` navigation stays global across the workspace; it does not prefer the current repository.

### Navigate to Worktrees

Navigate directly to a worktree by branch name across all repos:

```bash
# Navigate to a worktree by branch name (searches all repos)
omgw worktree feat-auth

# Canonical form (same behavior)
omgw worktree navigate feat-auth

# Wildcard matching
omgw worktree "feat-*"
```

When multiple worktrees match, an interactive selection list is displayed:

```
Multiple worktrees match 'feat-*':

  1) my-repo / feat-auth  /home/user/.local/share/gws/projects/my-repo/feat-auth
  2) my-repo / feat-new   /home/user/.local/share/gws/projects/my-repo/feat-new
  3) other-repo / feat-x  /home/user/.local/share/gws/projects/other-repo/feat-x

Select worktree [1-3]:
```

Tab completion is available for worktree branch names.

You can also navigate to a specific repo's worktree using the `-wt` shorthand:

```bash
# Navigate to a specific worktree within a repo
omgw my-repo -wt feat-auth

# List all worktrees for a repo with interactive selection
omgw my-repo -wt
```

See [Shell Integration](shell-integration.md) for details on how `-wt` navigation works.

---

## Parent Navigation

```
omgw parent <repo> [flags]
```

Print the parent directory path of a repository. Useful for navigating to the directory that contains a repository.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--quiet` | `-q` | `false` | Suppress verbose output, print only the path |

**Examples:**

```bash
# Print parent directory path
omgw parent my-repo

# Navigate to parent directory
cd "$(omgw parent my-repo)"
```

See [Shell Integration](shell-integration.md) for shorthand navigation forms (`omgw -p my-repo`, `omgw my-repo -p`).
