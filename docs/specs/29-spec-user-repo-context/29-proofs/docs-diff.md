# `docs/site/commands-user.md` diff

The Assign Profile section after task 4.0, as `git diff docs/site/commands-user.md` on the feature branch before the docs commit. It adds the optional-repository usage line, the omitted-repository and worktree behavior, the not-tracked error, the completion rule, the `--use-subdirs` restriction, and examples for both forms.

```diff
diff --git a/docs/site/commands-user.md b/docs/site/commands-user.md
index c57f739..2b3334e 100644
--- a/docs/site/commands-user.md
+++ b/docs/site/commands-user.md
@@ -122,26 +122,45 @@ Remove a stored user profile. If any repositories are currently using the profil
 ## Assign Profile
 
 ```
-omgw user assign <repository> <profile> [flags]
+omgw user assign [repository] <profile> [flags]
 ```
 
 Assign a user profile to a repository. This sets `user.name` and `user.email` (and optionally signing configuration) in the repository's local `.git/config`.
 
+The repository may be omitted when you are standing inside it, or inside one of its worktrees. The profile is then applied to the repository that owns the current directory. From a worktree that is the main checkout, whose `.git/config` git shares with every worktree of the repository. Naming a repository explicitly always takes precedence over the current directory. See [Current-Repository Targeting](commands-core.md#current-repository-targeting) for how the repository is detected.
+
+If the current directory is not inside a tracked repository, the command fails without changing anything:
+
+```
+Error: current directory is not inside a tracked repository
+  Supply a repository argument, or run 'omgw add' to track this repository
+```
+
+Tab completion follows the same rule: inside a tracked repository the first argument completes to **profile names**, and outside one it completes to **repository names**. The second argument always completes to profile names.
+
 ### Flags
 
 | Flag | Default | Description |
 |------|---------|-------------|
-| `--use-subdirs` | `false` | Move repository to profile subdirectory |
+| `--use-subdirs` | `false` | Move repository to profile subdirectory. Requires an explicit repository argument. |
 | `--dry-run` | `false` | Preview changes without applying them |
 
 **Examples:**
 
 ```bash
-# Assign the "work" profile to a repository
+# Assign the "work" profile to a repository by name
 omgw user assign my-api work
 
+# Omit the repository when you are standing in it
+cd ~/code/my-api
+omgw user assign work
+
+# From inside a worktree, the owning repository receives the profile
+cd ~/.local/share/gws/projects/my-api/feat-x
+omgw user assign personal
+
 # Preview what would change
-omgw user assign my-api work --dry-run
+omgw user assign work --dry-run
 ```
 
 ---
```
