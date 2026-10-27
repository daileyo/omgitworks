package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/daileyo/omgitworks/internal/xdg"
)

// Version of the config format for future migrations
const ConfigVersion = "1.2.0"

// Config represents the gws workspace configuration
type Config struct {
	Version      string       `json:"version"`
	Workspace    string       `json:"workspace"`
	Profiles     []Profile    `json:"profiles,omitempty"`
	Repositories []Repository `json:"repositories"`
	Preferences  *Preferences `json:"preferences,omitempty"`
}

// Preferences holds user-configurable settings
type Preferences struct {
	StatusWorkers int `json:"status_workers,omitempty"`
}

// RepositoryType represents the type of git hosting provider
type RepositoryType string

const (
	TypeGitHub    RepositoryType = "github"
	TypeGitLab    RepositoryType = "gitlab"
	TypeADO       RepositoryType = "ado"
	TypeBitbucket RepositoryType = "bitbucket"
	TypeUnknown   RepositoryType = "unknown"
)

// RepositoryVisibility represents whether a repository is public or private
type RepositoryVisibility string

const (
	VisibilityPublic  RepositoryVisibility = "public"
	VisibilityPrivate RepositoryVisibility = "private"
	VisibilityUnknown RepositoryVisibility = "unknown"
)

// UserSource represents where the git user configuration comes from
type UserSource string

const (
	UserSourceGlobal    UserSource = "global"
	UserSourceLocal     UserSource = "local"
	UserSourceIncludeIf UserSource = "includeif"
	UserSourceUnknown   UserSource = "unknown"
)

// Profile represents a git user profile with identity and signing configuration
type Profile struct {
	Name          string `json:"name"`                     // Profile identifier (e.g., "work", "personal")
	GitName       string `json:"git_name"`                 // user.name value
	Email         string `json:"email"`                    // user.email value
	SigningKey    string `json:"signing_key,omitempty"`    // user.signingkey value (optional)
	SignCommits   bool   `json:"sign_commits,omitempty"`   // commit.gpgsign setting
	SigningFormat string `json:"signing_format,omitempty"` // gpg.format value (ssh, openpgp, x509); empty means git's default
}

// Worktree represents a git worktree associated with a repository
type Worktree struct {
	Path    string `json:"path"`              // Absolute filesystem path to the worktree
	Branch  string `json:"branch"`            // Branch checked out in the worktree
	Aligned bool   `json:"aligned,omitempty"` // Whether the worktree is inside <repo>.wt/
}

// Repository represents a discovered git repository
type Repository struct {
	Name           string               `json:"name"`
	Path           string               `json:"path"`
	RemoteURL      string               `json:"remote_url,omitempty"`
	Type           RepositoryType       `json:"type,omitempty"`
	Visibility     RepositoryVisibility `json:"visibility,omitempty"`
	Tags           []string             `json:"tags,omitempty"`
	User           string               `json:"user,omitempty"`            // Git user.name for this repo
	Email          string               `json:"email,omitempty"`           // Git user.email for this repo
	SigningEnabled bool                 `json:"signing_enabled,omitempty"` // Whether commit signing is configured
	UserSource     UserSource           `json:"user_source,omitempty"`     // Where the user config comes from
	Worktrees      []Worktree           `json:"worktrees,omitempty"`       // Git worktrees for this repo
}

// GetConfigPath returns the path to the gws config file
func GetConfigPath() (string, error) {
	return xdg.ConfigFile()
}

// GetConfigDir returns the directory containing the config file
func GetConfigDir() (string, error) {
	return xdg.ConfigDir()
}

// Load reads the configuration from the XDG config location, migrating a
// pre-XDG ~/.gws/config.json on first use.
func Load() (*Config, error) {
	ensureMigrated()

	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Migration is best-effort, so the config may still be at the legacy
		// path. Reading it there keeps a working workspace working.
		data, err = readLegacyConfig()
		if err != nil {
			return nil, err
		}
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to ~/.gws/config.json
func Save(cfg *Config) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file with secure permissions (user read/write only)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// New creates a new configuration with the specified workspace path
func New(workspacePath string) *Config {
	return &Config{
		Version:      ConfigVersion,
		Workspace:    workspacePath,
		Repositories: []Repository{},
	}
}

// Exists checks if a configuration file exists, in either the current location
// or the pre-XDG one, so a workspace pending migration is not reported missing.
func Exists() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(configPath); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	legacyPath, err := xdg.LegacyConfigFile()
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(legacyPath); err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return false, nil
}

// readLegacyConfig reads the pre-XDG config, translating its absence into the
// standard "not initialized" guidance.
func readLegacyConfig() ([]byte, error) {
	legacyPath, err := xdg.LegacyConfigFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workspace not initialized: run 'gws init <directory>' first")
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return data, nil
}
