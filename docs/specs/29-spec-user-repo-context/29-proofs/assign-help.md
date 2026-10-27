# `omgw user assign --help` and `omgw user --help`

Recorded against a binary built from this branch. The `Usage:` line comes from the `Use` string and shows the optional `[repository]`; every example uses `omgw`. The second block is the `user` parent help, which now lists the one-argument `assign` form.

```text
$ omgw user assign --help
Assign a user profile to a repository, setting user.name and user.email
in the repository's local .git/config file.

The repository can be named explicitly (partial match on the name, or a
path) or omitted. When omitted, the repository is determined from the
current directory. This also works from inside one of that repository's
worktrees, where the owning repository receives the identity, which git
then shares with every worktree. Naming a repository explicitly always
takes precedence over the current directory.

The profile must exist (either stored or auto-detected).

Options:
  --use-subdirs  Move the repository to a profile subdirectory and set up
                 includeIf in ~/.gitconfig (advanced workflow). Requires an
                 explicit repository argument.
  --dry-run      Preview changes without applying them

Examples:
  omgw user assign work                       # Repo from the current directory
  omgw user assign my-project work            # Set work profile on my-project
  omgw user assign my-project personal        # Set personal profile
  omgw user assign my-project work --dry-run  # Preview changes

Usage:
  omgitworks user assign [repository] <profile> [flags]

Flags:
      --dry-run       Preview changes without applying
  -h, --help          help for assign
      --use-subdirs   Move repository to profile subdirectory

$ omgw user --help
Manage git user profiles for different contexts (work, personal, etc).

Profiles define git user.name, user.email, and optional signing configuration.
They can be manually created or auto-detected from the includeIf directives in your git config.

Commands:
  omgw user list                                   # List all profiles
  omgw user add work --email work@company.com      # Add a new profile
  omgw user show work                              # Show profile details
  omgw user remove work                            # Remove a profile
  omgw user assign my-repo work                    # Assign profile to repo
  omgw user assign work                            # Assign profile to the current repo
  omgw user sync                                   # Sync user info

Short flags:
  omgw user -l                                     # List all profiles
  omgw user -a work --email work@company.com       # Add a new profile
  omgw user -s work                                # Show profile details
  omgw user -d work                                # Remove a profile

Usage:
  omgitworks user [flags]
  omgitworks user [command]

Available Commands:
  add         Add a new user profile
  assign      Assign a user profile to a repository
  list        List all user profiles
  remove      Remove a user profile
  show        Show profile details
  sync        Sync stored user info with effective git config

Flags:
  -a, --add                  Add a profile (equivalent to 'user add')
  -d, --delete               Remove a profile (equivalent to 'user remove')
      --email string         Email address for git commits (required with -a)
  -h, --help                 help for user
  -l, --list                 List all profiles (equivalent to 'user list')
      --name string          Git user name (defaults to profile name)
  -s, --show                 Show profile details (equivalent to 'user show')
      --sign-commits         Enable commit signing
      --signing-key string   GPG signing key ID

Use "omgitworks user [command] --help" for more information about a command.
```
