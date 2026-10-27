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
	"github.com/daileyo/omgitworks/internal/user"
)

// Flags for user assign command
var (
	assignUseSubdirs bool
	assignDryRun     bool
)

var userAssignCmd = &cobra.Command{
	Use:   "assign <repository> <profile>",
	Short: "Assign a user profile to a repository",
	Long: `Assign a user profile to a repository, setting user.name and user.email
in the repository's local .git/config file.

The repository can be specified by name (partial match) or path.
The profile must exist (either stored or auto-detected).

Options:
  --use-subdirs  Move the repository to a profile subdirectory and set up
                 includeIf in ~/.gitconfig (advanced workflow)
  --dry-run      Preview changes without applying them

Examples:
  gws user assign my-project work         # Set work profile on my-project
  gws user assign my-project personal     # Set personal profile
  gws user assign my-project work --dry-run  # Preview changes`,
	Args: cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		opts := userAssignOptions{useSubdirs: assignUseSubdirs, dryRun: assignDryRun}
		return runUserAssign(args, opts, os.Stdin)
	},
}

func init() {
	userAssignCmd.Flags().BoolVar(&assignUseSubdirs, "use-subdirs", false, "Move repository to profile subdirectory")
	userAssignCmd.Flags().BoolVar(&assignDryRun, "dry-run", false, "Preview changes without applying")
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
	repoIdentifier := args[0]
	profileName := args[1]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Find the repository
	repos := findRepositories(cfg, repoIdentifier)
	if len(repos) == 0 {
		return fmt.Errorf("no repository found matching: %s", repoIdentifier)
	}
	if len(repos) > 1 {
		fmt.Printf("Multiple repositories match '%s':\n", repoIdentifier)
		for _, r := range repos {
			fmt.Printf("  - %s (%s)\n", r.Name, r.Path)
		}
		return fmt.Errorf("please specify a more specific repository identifier")
	}

	repo := repos[0]

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
