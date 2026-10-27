package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/user"
)

// User subcommand flags for short-flag invocation (user -a / -d / -l / -s).
var (
	userFlagAdd    bool
	userFlagDelete bool
	userFlagList   bool
	userFlagShow   bool
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage git user profiles",
	Long: `Manage git user profiles for different contexts (work, personal, etc).

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
  omgw user -d work                                # Remove a profile`,
	Args:              cobra.ArbitraryArgs,
	ValidArgsFunction: completeProfileThenNone,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Count active short flags for mutual exclusivity
		activeCount := 0
		if userFlagAdd {
			activeCount++
		}
		if userFlagDelete {
			activeCount++
		}
		if userFlagList {
			activeCount++
		}
		if userFlagShow {
			activeCount++
		}

		if activeCount > 1 {
			return fmt.Errorf("only one of -a, -d, -l, -s can be used at a time")
		}

		// Dispatch based on short flag
		if userFlagList {
			return userListCmd.RunE(cmd, args)
		}

		if userFlagAdd {
			if len(args) < 1 {
				return fmt.Errorf("user -a requires a profile name: gws user -a <name> --email <email>")
			}
			return userAddCmd.RunE(cmd, args)
		}

		if userFlagShow {
			if len(args) != 1 {
				return fmt.Errorf("user -s requires exactly 1 argument: gws user -s <name>")
			}
			return userShowCmd.RunE(cmd, args)
		}

		if userFlagDelete {
			if len(args) != 1 {
				return fmt.Errorf("user -d requires exactly 1 argument: gws user -d <name>")
			}
			return userRemoveCmd.RunE(cmd, args)
		}

		// No sub-operation specified — show help
		return cmd.Help()
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all user profiles",
	Long: `List all stored and auto-detected user profiles.

Stored profiles are ones you've added via 'gws user add'.
Auto-detected profiles are discovered from the includeIf directives in your git config
(~/.gitconfig, ~/.config/git/config and anything they include).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		stored, detected := user.ListProfiles(cfg)

		if len(stored) == 0 && len(detected) == 0 {
			fmt.Println("No profiles found.")
			fmt.Println("\nAdd a profile with: gws user add <name> --email <email>")
			return nil
		}

		// Display stored profiles
		if len(stored) > 0 {
			fmt.Printf("Stored Profiles (%d):\n\n", len(stored))
			displayProfileTable(stored)
		}

		// Display detected profiles
		if len(detected) > 0 {
			if len(stored) > 0 {
				fmt.Println()
			}
			fmt.Printf("Auto-Detected Profiles (%d):\n\n", len(detected))
			displayProfileTable(detected)
			fmt.Println("\n(Auto-detected from git config includeIf directives)")
		}

		return nil
	},
}

// Flags for user add command
var (
	addEmail      string
	addGitName    string
	addSigningKey string
	addSignCommit bool
)

var userAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new user profile",
	Long: `Add a new user profile with the specified name.

The profile name is used to identify this profile when assigning it to repositories.
Common names: work, personal, opensource, client-name, etc.

Examples:
  gws user add work --email work@company.com --name "John Doe"
  gws user add personal --email john@personal.com --name "John"
  gws user add secure --email john@company.com --signing-key ABCD1234 --sign-commits`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		// Validate required email flag
		if addEmail == "" {
			return fmt.Errorf("--email is required")
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		profile := config.Profile{
			Name:        profileName,
			GitName:     addGitName,
			Email:       addEmail,
			SigningKey:  addSigningKey,
			SignCommits: addSignCommit,
		}

		// If git name not provided, use profile name as default
		if profile.GitName == "" {
			profile.GitName = profileName
		}

		if err := user.AddProfile(cfg, profile); err != nil {
			return err
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Added profile '%s'\n", profileName)
		fmt.Printf("  Name:  %s\n", profile.GitName)
		fmt.Printf("  Email: %s\n", profile.Email)
		if profile.SigningKey != "" {
			fmt.Printf("  Signing Key: %s\n", profile.SigningKey)
		}
		if profile.SignCommits {
			fmt.Printf("  Sign Commits: yes (%s)\n", signingFormatLabel(profile.SigningFormat))
		}

		return nil
	},
}

var userShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show profile details",
	Long: `Show detailed information about a user profile.

Examples:
  gws user show work
  gws user show personal`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// First check stored profiles
		profile, err := user.GetProfile(cfg, profileName)
		if err != nil {
			// Try to find in auto-detected profiles
			detected, detectErr := user.DetectProfiles()
			if detectErr != nil {
				return err // Return original error
			}

			for i := range detected {
				if strings.EqualFold(detected[i].Name, profileName) {
					profile = &detected[i]
					break
				}
			}

			if profile == nil {
				return err // Return original error
			}
		}

		fmt.Printf("Profile: %s\n\n", profile.Name)
		fmt.Printf("  Git Name:     %s\n", profile.GitName)
		fmt.Printf("  Email:        %s\n", profile.Email)

		if profile.SigningKey != "" {
			fmt.Printf("  Signing Key:  %s\n", profile.SigningKey)
		} else {
			fmt.Printf("  Signing Key:  (not configured)\n")
		}

		if profile.SignCommits {
			fmt.Printf("  Sign Commits: yes (%s)\n", signingFormatLabel(profile.SigningFormat))
		} else {
			fmt.Printf("  Sign Commits: no\n")
		}

		// Show repositories using this profile
		repoCount := 0
		for _, repo := range cfg.Repositories {
			if repo.User == profile.GitName && repo.Email == profile.Email {
				repoCount++
			}
		}

		fmt.Printf("\n  Repositories using this profile: %d\n", repoCount)

		return nil
	},
}

var userRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a user profile",
	Long: `Remove a user profile from the configuration.

If repositories are using this profile, you will be prompted for confirmation.
Note: This does not change the git configuration in repositories.

Examples:
  gws user remove work
  gws user remove old-project`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Check if profile exists first
		_, err = user.GetProfile(cfg, profileName)
		if err != nil {
			return err
		}

		// Get repos using this profile (dry run)
		reposUsing, err := user.RemoveProfile(cfg, profileName)
		if err != nil {
			return err
		}

		// If repos are using this profile, ask for confirmation
		if len(reposUsing) > 0 {
			fmt.Printf("Warning: %d %s using this profile:\n",
				len(reposUsing), pluralize(len(reposUsing), "repository is", "repositories are"))
			for _, name := range reposUsing {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Print("\nProceed with removal? [y/N]: ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" && response != "yes" {
				fmt.Println("Canceled.")
				return nil
			}
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Removed profile '%s'\n", profileName)
		return nil
	},
}

var userSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync stored user info with effective git config",
	Long: `Update stored user information for all repositories to match their
current effective git configuration.

This is useful after manually changing git config files or when repositories
may have been modified outside of gws.

Examples:
  gws user sync                    # Sync all repositories`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println("Syncing user information for all repositories...")

		updated, err := user.SyncUserInfo(cfg)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		fmt.Printf("Sync complete. Updated %d %s.\n",
			updated, pluralize(updated, "repository", "repositories"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(userCmd)

	// Add subcommands
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userShowCmd)
	userCmd.AddCommand(userRemoveCmd)
	userCmd.AddCommand(userAssignCmd)
	userCmd.AddCommand(userSyncCmd)

	// Short flags on the user command for quick invocation
	userCmd.Flags().BoolVarP(&userFlagAdd, "add", "a", false, "Add a profile (equivalent to 'user add')")
	userCmd.Flags().BoolVarP(&userFlagDelete, "delete", "d", false, "Remove a profile (equivalent to 'user remove')")
	userCmd.Flags().BoolVarP(&userFlagList, "list", "l", false, "List all profiles (equivalent to 'user list')")
	userCmd.Flags().BoolVarP(&userFlagShow, "show", "s", false, "Show profile details (equivalent to 'user show')")

	// Add-related flags on userCmd so they work with -a (same backing vars as userAddCmd)
	userCmd.Flags().StringVar(&addEmail, "email", "", "Email address for git commits (required with -a)")
	userCmd.Flags().StringVar(&addGitName, "name", "", "Git user name (defaults to profile name)")
	userCmd.Flags().StringVar(&addSigningKey, "signing-key", "", "GPG signing key ID")
	userCmd.Flags().BoolVar(&addSignCommit, "sign-commits", false, "Enable commit signing")

	// Add flags for user add subcommand
	userAddCmd.Flags().StringVar(&addEmail, "email", "", "Email address for git commits (required)")
	userAddCmd.Flags().StringVar(&addGitName, "name", "", "Git user name (defaults to profile name)")
	userAddCmd.Flags().StringVar(&addSigningKey, "signing-key", "", "GPG signing key ID")
	userAddCmd.Flags().BoolVar(&addSignCommit, "sign-commits", false, "Enable commit signing")

	_ = userAddCmd.MarkFlagRequired("email")
}

// completeProfileThenNone completes profile names for the first arg, nothing after.
func completeProfileThenNone(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return completeProfileNames(toComplete)
}

// completeProfileNames returns profile names (stored + auto-detected) matching the prefix.
func completeProfileNames(toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	seen := make(map[string]bool)
	var names []string

	// Stored profiles
	for _, p := range cfg.Profiles {
		if strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(toComplete)) && !seen[p.Name] {
			seen[p.Name] = true
			names = append(names, p.Name)
		}
	}

	// Auto-detected profiles
	detected, err := user.DetectProfiles()
	if err == nil {
		for _, p := range detected {
			if strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(toComplete)) && !seen[p.Name] {
				seen[p.Name] = true
				names = append(names, p.Name)
			}
		}
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// signingFormatLabel names the signing mechanism for a gpg.format value.
// Git signs with OpenPGP when gpg.format is unset, which is shown as "gpg".
func signingFormatLabel(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "openpgp":
		return "gpg"
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

// displayProfileTable shows profiles in a formatted table
func displayProfileTable(profiles []config.Profile) {
	// Calculate column widths
	maxNameLen := 4  // "NAME"
	maxGitLen := 8   // "GIT NAME"
	maxEmailLen := 5 // "EMAIL"

	for _, p := range profiles {
		if len(p.Name) > maxNameLen {
			maxNameLen = len(p.Name)
		}
		if len(p.GitName) > maxGitLen {
			maxGitLen = len(p.GitName)
		}
		if len(p.Email) > maxEmailLen {
			maxEmailLen = len(p.Email)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s  %s\n",
		maxNameLen, "NAME",
		maxGitLen, "GIT NAME",
		maxEmailLen, "EMAIL",
		"SIGN")
	fmt.Printf("%s  %s  %s  %s\n",
		strings.Repeat("-", maxNameLen),
		strings.Repeat("-", maxGitLen),
		strings.Repeat("-", maxEmailLen),
		strings.Repeat("-", 4))

	// Print profiles
	for _, p := range profiles {
		signStr := ""
		if p.SignCommits {
			signStr = "✓ (" + signingFormatLabel(p.SigningFormat) + ")"
		}

		fmt.Printf("%-*s  %-*s  %-*s  %s\n",
			maxNameLen, p.Name,
			maxGitLen, p.GitName,
			maxEmailLen, p.Email,
			signStr)
	}
}
