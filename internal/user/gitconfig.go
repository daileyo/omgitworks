package user

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
)

// IncludeIf represents an includeIf directive in gitconfig
type IncludeIf struct {
	Condition string // The condition (e.g., "gitdir:~/work/")
	Path      string // The path to the included config file
}

// ParseIncludedConfig reads a gitconfig file referenced by an includeIf and extracts a Profile
func ParseIncludedConfig(includeIf IncludeIf) (*config.Profile, error) {
	expandedPath := expandPath(includeIf.Path)

	userCfg, err := git.ReadUserConfigFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read included config: %w", err)
	}
	if userCfg.Name == "" && userCfg.Email == "" {
		return nil, fmt.Errorf("no user identity found in %s", expandedPath)
	}

	return &config.Profile{
		Name:          deriveProfileName(includeIf),
		GitName:       userCfg.Name,
		Email:         userCfg.Email,
		SigningKey:    userCfg.SigningKey,
		SignCommits:   userCfg.SignCommits,
		SigningFormat: userCfg.SigningFormat,
	}, nil
}

// DetectProfiles extracts profiles from the includeIf directives in every
// config file git reads (~/.gitconfig, $XDG_CONFIG_HOME/git/config, system
// config and anything they include).
func DetectProfiles() ([]config.Profile, error) {
	directives, err := git.ListIncludeIfDirectives()
	if err != nil {
		return nil, fmt.Errorf("failed to read includeIf directives: %w", err)
	}
	if len(directives) == 0 {
		return nil, nil // No includeIf directives
	}

	// Parse each included config to extract profiles
	var profiles []config.Profile
	seen := make(map[string]bool)

	for _, directive := range directives {
		includeIf := IncludeIf{Condition: directive.Condition, Path: directive.Path}
		profile, err := ParseIncludedConfig(includeIf)
		if err != nil {
			// Skip files that can't be parsed or don't have user info
			continue
		}

		// Avoid duplicate profiles (by email since that's more unique)
		if seen[profile.Email] {
			continue
		}
		seen[profile.Email] = true

		profiles = append(profiles, *profile)
	}

	return profiles, nil
}

// deriveProfileName extracts a meaningful profile name from the includeIf directive
func deriveProfileName(includeIf IncludeIf) string {
	// First, check if the config file path has a clear profile name pattern
	// (e.g., .gitconfig-work, gitconfig-ado)
	configName := extractNameFromConfigPath(includeIf.Path)
	if configName != "" && configName != "default" && !isGenericDirName(configName) {
		return configName
	}

	// Fall back to extracting from the gitdir condition
	condition := includeIf.Condition

	// Handle gitdir: and gitdir/i: conditions
	if strings.HasPrefix(strings.ToLower(condition), "gitdir:") {
		path := strings.TrimPrefix(condition, "gitdir:")
		path = strings.TrimPrefix(path, "gitdir/i:")
		return extractNameFromPath(path)
	}

	if strings.HasPrefix(strings.ToLower(condition), "gitdir/i:") {
		path := strings.TrimPrefix(condition, "gitdir/i:")
		return extractNameFromPath(path)
	}

	// Final fallback
	if configName != "" {
		return configName
	}
	return "default"
}

// isGenericDirName checks if a name is too generic to be a profile name
func isGenericDirName(name string) bool {
	generic := []string{"home", "users", "user", "repos", "repositories", "git", "src", "code"}
	nameLower := strings.ToLower(name)
	for _, g := range generic {
		if nameLower == g {
			return true
		}
	}
	return false
}

// extractNameFromPath extracts a profile name from a directory path
func extractNameFromPath(path string) string {
	// Expand and clean the path
	path = expandPath(path)
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimSuffix(path, "/**")

	// Get the last meaningful directory name
	base := filepath.Base(path)
	if base == "." || base == "/" || base == "" {
		// Try parent directory
		parent := filepath.Dir(path)
		base = filepath.Base(parent)
	}

	// Clean up and return
	base = strings.ToLower(base)
	base = strings.TrimPrefix(base, ".")

	if base == "" || base == "home" || base == "users" {
		return "default"
	}

	return base
}

// extractNameFromConfigPath extracts a profile name from a gitconfig file path
func extractNameFromConfigPath(path string) string {
	base := filepath.Base(path)

	// Handle .gitconfig-<name> pattern
	if strings.HasPrefix(base, ".gitconfig-") {
		return strings.TrimPrefix(base, ".gitconfig-")
	}

	// Handle gitconfig-<name> pattern
	if strings.HasPrefix(base, "gitconfig-") {
		return strings.TrimPrefix(base, "gitconfig-")
	}

	// Handle .gitconfig.<name> pattern
	if strings.HasPrefix(base, ".gitconfig.") {
		return strings.TrimPrefix(base, ".gitconfig.")
	}

	// Handle <name>.gitconfig pattern
	if strings.HasSuffix(base, ".gitconfig") {
		return strings.TrimSuffix(base, ".gitconfig")
	}

	// Fall back to directory name
	return extractNameFromPath(filepath.Dir(path))
}

// expandPath expands ~ to home directory and cleans the path
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	return filepath.Clean(path)
}
