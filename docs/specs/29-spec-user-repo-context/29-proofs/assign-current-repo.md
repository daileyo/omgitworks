# `omgw user assign <profile>` from the repository root

Recorded against a binary built from this branch (`go build -o omgw ./cmd/omgitworks`), under a temporary `HOME` with `XDG_CONFIG_HOME` and `XDG_DATA_HOME` unset and `GIT_CONFIG_NOSYSTEM=1`, so no real identity or path appears. `$WS` is `$HOME/code`. Temporary paths are shown as `<tmp>`. Profiles use placeholder identities.

## Setup

```text
### setup
$ omgw init <tmp>/code
Initialized workspace at: <tmp>/code
Found 1 repository.
[exit 0]
$ omgw user add work --email work@example.com --name Worker
Added profile 'work'
  Name:  Worker
  Email: work@example.com
[exit 0]
$ omgw user add personal --email personal@example.com --name Person
Added profile 'personal'
  Name:  Person
  Email: personal@example.com
[exit 0]
```

## Transcript

The repository argument is omitted. The profile lands in the repository the shell is standing in, and git reads it back from the local config.

```text
### transcript 1: assign from the repository root
$ cd $WS/my-api
$ omgw user assign work
Assigned profile 'work' to repository 'my-api'
  user.name:  Worker
  user.email: work@example.com
[exit 0]
$ git config --local user.email
work@example.com
[exit 0]
```
