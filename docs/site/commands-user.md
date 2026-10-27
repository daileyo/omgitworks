# User Management

Manage git user profiles across your workspace. Profiles store identity information (name, email, signing key) that can be assigned to repositories to configure their local git `user.*` settings.

Profiles can be created manually or auto-detected from `includeIf` directives in your git config. Identity is read through `git config` itself, so `~/.gitconfig`, `~/.config/git/config`, `$GIT_CONFIG_GLOBAL`, the system config and relative `include` paths are all honored.

---

## Short-Flag Aliases

The `user` parent command supports short-flag aliases for common operations:

| Short Flag | Equivalent |
|------------|------------|
| `-l` | `omgw user list` |
| `-a` | `omgw user add` |
| `-s` | `omgw user show` |
| `-d` | `omgw user remove` |

When using `-a`, you can also pass `--email`, `--name`, `--signing-key`, and `--sign-commits` inline:

```bash
omgw user -a work --email work@company.com --name "Jane Doe"
```

---

## List Profiles

```
omgw user list
```

List all stored and auto-detected user profiles.

Profiles are created and kept up to date by `omgw init`, `omgw add`, and `omgw refresh`
from the identities git reports: `includeIf` config files, the global default identity,
and the identity in effect for each tracked repository. Each profile carries the signing
key and `commit.gpgsign` setting git uses with that identity. A stored profile that is
missing signing settings is filled in on the next sync; settings already on a profile
are never replaced.

**Example output:**

```
Stored Profiles:

  NAME        GIT NAME     EMAIL                    SIGN
  ----        --------     -----                    ----
  personal    Jane Doe     jane@personal.dev        ABC123 (signing on)
  work        Jane Doe     jane.doe@company.com     —

Auto-Detected Profiles:

  NAME        GIT NAME     EMAIL                    SIGN
  ----        --------     -----                    ----
  oss         Jane Doe     jane@opensource.org       —

(Auto-detected from git config includeIf directives)
```

---

## Add Profile

```
omgw user add <name> [flags]
```

Create a new user profile.

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--email` | Yes | | Email address for git commits |
| `--name` | No | Profile name | Git `user.name` value |
| `--signing-key` | No | | GPG signing key ID |
| `--sign-commits` | No | `false` | Enable commit signing (`commit.gpgsign`) |

**Examples:**

```bash
# Add a basic profile
omgw user add personal --email jane@personal.dev

# Add with a different git name
omgw user add work --email jane.doe@company.com --name "Jane Doe"

# Add with signing enabled
omgw user add secure --email jane@secure.dev --signing-key ABC123DEF456 --sign-commits
```

---

## Show Profile

```
omgw user show <name>
```

Display detailed information about a profile, including how many repositories use it.

**Example output:**

```
Profile: personal

  Git Name:       Jane Doe
  Email:          jane@personal.dev
  Signing Key:    ABC123DEF456
  Sign Commits:   true

  Used by 3 repositories
```

---

## Remove Profile

```
omgw user remove <name>
```

Remove a stored user profile. If any repositories are currently using the profile, you will be prompted before removal.

---

## Assign Profile

```
omgw user assign [repository] <profile> [flags]
```

Assign a user profile to a repository. This sets `user.name` and `user.email` (and optionally signing configuration) in the repository's local `.git/config`.

The repository may be omitted when you are standing inside it, or inside one of its worktrees. The profile is then applied to the repository that owns the current directory. From a worktree that is the main checkout, whose `.git/config` git shares with every worktree of the repository. Naming a repository explicitly always takes precedence over the current directory. See [Current-Repository Targeting](commands-core.md#current-repository-targeting) for how the repository is detected.

If the current directory is not inside a tracked repository, the command fails without changing anything:

```
Error: current directory is not inside a tracked repository
  Supply a repository argument, or run 'omgw add' to track this repository
```

Tab completion follows the same rule: inside a tracked repository the first argument completes to **profile names**, and outside one it completes to **repository names**. The second argument always completes to profile names.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--use-subdirs` | `false` | Move repository to profile subdirectory. Requires an explicit repository argument. |
| `--dry-run` | `false` | Preview changes without applying them |

**Examples:**

```bash
# Assign the "work" profile to a repository by name
omgw user assign my-api work

# Omit the repository when you are standing in it
cd ~/code/my-api
omgw user assign work

# From inside a worktree, the owning repository receives the profile
cd ~/.local/share/gws/projects/my-api/feat-x
omgw user assign personal

# Preview what would change
omgw user assign work --dry-run
```

---

## Sync Profiles

```
omgw user sync
```

Synchronize stored user information with the effective git configuration for all tracked repositories. This re-reads each repository's effective git configuration and updates the cached `user`, `email`, `signing_enabled`, and `user_source` fields in the workspace configuration.

**When to use:**

- After manually editing a repository's `.git/config`
- After changing your global git identity or `includeIf` rules
- To ensure the workspace config reflects the current state
