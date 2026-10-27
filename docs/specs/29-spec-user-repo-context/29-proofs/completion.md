# Tab completion for `omgw user assign`

Recorded against a binary built from this branch, under a temporary `HOME` with `XDG_CONFIG_HOME` and `XDG_DATA_HOME` unset, through cobra's hidden `__complete` entry point, which is what the installed shell completion scripts call. The workspace tracks `my-api` and `web-ui`; the stored profiles are `work` and `personal`, with placeholder identities. `:4` is cobra's encoding of `ShellCompDirectiveNoFileComp`.

```text
### inside a tracked repository: first argument completes to profiles
$ cd $WS/my-api
$ omgw __complete user assign 
work
personal
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
[exit 0]

### outside any tracked repository: first argument completes to repositories
$ cd $HOME
$ omgw __complete user assign 
my-api
web-ui
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
[exit 0]

### a repository already typed: second argument completes to profiles
$ omgw __complete user assign my-api 
work
personal
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
[exit 0]
$ omgw __complete user assign my-api w
work
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
[exit 0]

### both arguments typed: nothing more
$ omgw __complete user assign my-api work 
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
[exit 0]
```
