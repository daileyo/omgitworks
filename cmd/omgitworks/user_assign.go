package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/repocontext"
	"github.com/daileyo/omgitworks/internal/user"
)

// Flags for user assign command
var (
	assignUseSubdirs bool
	assignDryRun     bool
)

var userAssignCmd = &cobra.Command{
	Use:   "assign [repository] <profile>",
	Short: "Assign a user profile to a repository",
	Long: `Assign a user profile to a repository, setting user.name and user.email
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
  omgw user assign my-project work --dry-run  # Preview changes`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(_ *cobra.Command, args []string) error {
		opts := userAssignOptions{useSubdirs: assignUseSubdirs, dryRun: assignDryRun}
		return runUserAssign(args, opts, os.Stdin)
	},
}

func init() {
	userAssignCmd.Flags().BoolVar(&assignUseSubdirs, "use-subdirs", false, "Move repository to profile subdirectory")
	userAssignCmd.Flags().BoolVar(&assignDryRun, "dry-run", false, "Preview changes without applying")
	userAssignCmd.ValidArgsFunction = completeUserAssignArgs
}

// completeUserAssignArgs follows the same rule as the one-argument form:
// inside a tracked repository the first argument is expected to be a profile,
// outside one it is expected to be a repository. Once a repository has been
// typed, the second argument is always a profile.
func completeUserAssignArgs(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	switch len(args) {
	case 0:
		if _, err := repocontext.ResolveCurrent(cfg); err == nil {
			return completeProfileNames(toComplete)
		}
		return completeRepoNames(toComplete)
	case 1:
		return completeProfileNames(toComplete)
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

// userAssignOptions carries the assign flags so runUserAssign can be called
// without cobra.
type userAssignOptions struct {
	useSubdirs bool
	dryRun     bool
}

// runUserAssign implements 'user assign'. in answers the --use-subdirs
// confirmation prompt, so tests can drive it without a terminal.
func runUserAssign(args []string, opts userAssignOptions, in io.Reader) error {
	// The profile is always the last positional; a repository is present only
	// when there are two. Later branches test repoArg, never len(args), so
	// spec 30 can add -t without reordering them.
	profileName := args[len(args)-1]
	repoArg := ""
	if len(args) == 2 {
		repoArg = args[0]
	}

	// --use-subdirs moves the repository the shell may be standing in, so it
	// needs an explicit repository. This runs before configuration is loaded
	// and before the dry-run branch, so a preview never shows a move the real
	// run would refuse.
	if opts.useSubdirs && repoArg == "" {
		return fmt.Errorf("--use-subdirs requires an explicit repository argument: omgw user assign <repository> <profile> --use-subdirs")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// The repository is resolved before the profile is looked up, matching
	// the historical order, so each failure produces its own error.
	repo, err := selectAssignTarget(cfg, repoArg)
	if err != nil {
		return err
	}

	profile, err := lookupProfile(cfg, profileName)
	if err != nil {
		return err
	}

	// Dry run mode
	if opts.dryRun {
		fmt.Printf("Dry run - would assign profile '%s' to repository '%s':\n\n", profileName, repo.Name)

		changes, err := user.PreviewAssignLocal(repo.Path, *profile)
		if err != nil {
			return err
		}

		if len(changes) == 0 {
			fmt.Println("  No changes needed (already configured)")
		} else {
			for _, change := range changes {
				fmt.Printf("  %s\n", change)
			}
		}

		if opts.useSubdirs {
			newPath := filepath.Join(cfg.Workspace, profile.Name, repo.Name)
			fmt.Printf("\n  Would move repository to: %s\n", newPath)
			fmt.Printf("  Would add includeIf directive to ~/.gitconfig\n")
		}

		return nil
	}

	// Subdirectory mode
	if opts.useSubdirs {
		fmt.Printf("Warning: This will move the repository to %s/%s/%s\n",
			cfg.Workspace, profile.Name, repo.Name)
		fmt.Print("Continue? [y/N]: ")

		reader := bufio.NewReader(in)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Canceled.")
			return nil
		}

		newPath, err := user.AssignWithSubdirs(cfg, repo.Path, *profile, cfg.Workspace)
		if err != nil {
			return fmt.Errorf("failed to assign with subdirs: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Moved repository to: %s\n", newPath)
		fmt.Printf("Added includeIf directive to ~/.gitconfig\n")
		fmt.Printf("Profile '%s' is now active for this repository\n", profileName)
		return nil
	}

	// Standard local assignment
	if err := user.AssignLocal(repo.Path, *profile); err != nil {
		return fmt.Errorf("failed to assign profile: %w", err)
	}

	// Update stored repo info
	repo.User = profile.GitName
	repo.Email = profile.Email
	repo.SigningEnabled = profile.SignCommits
	repo.UserSource = config.UserSourceLocal

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Assigned profile '%s' to repository '%s'\n", profileName, repo.Name)
	fmt.Printf("  user.name:  %s\n", profile.GitName)
	fmt.Printf("  user.email: %s\n", profile.Email)
	if profile.SignCommits {
		fmt.Printf("  signing:    enabled\n")
	}

	return nil
}

// selectAssignTarget picks the repository assign acts on. With no argument the
// repository is resolved from the working directory, including from inside
// one of its worktrees, and the resolver's error is returned unwrapped so it
// reads the same as the worktree commands'. With an argument the working
// directory is never inspected: the historical substring matcher applies, and
// an ambiguous match is still an error. Bulk selection arrives with the tag
// rows in spec 30, which extends this function.
func selectAssignTarget(cfg *config.Config, repoArg string) (*config.Repository, error) {
	if repoArg == "" {
		return repocontext.ResolveCurrent(cfg)
	}

	repos := findRepositories(cfg, repoArg)
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repository found matching: %s", repoArg)
	}
	if len(repos) > 1 {
		fmt.Printf("Multiple repositories match '%s':\n", repoArg)
		for _, r := range repos {
			fmt.Printf("  - %s (%s)\n", r.Name, r.Path)
		}
		return nil, fmt.Errorf("please specify a more specific repository identifier")
	}
	return repos[0], nil
}

// lookupProfile finds a profile by name, checking stored profiles first and
// profiles auto-detected from includeIf directives second. When neither has
// it, the stored-profile error is returned so the message names the profile.
func lookupProfile(cfg *config.Config, name string) (*config.Profile, error) {
	profile, err := user.GetProfile(cfg, name)
	if err == nil {
		return profile, nil
	}

	detected, detectErr := user.DetectProfiles()
	if detectErr != nil {
		return nil, err
	}
	for i := range detected {
		if strings.EqualFold(detected[i].Name, name) {
			return &detected[i], nil
		}
	}
	return nil, err
}
