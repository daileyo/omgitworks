package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"

	"github.com/daileyo/omgitworks/internal/config"
)

// UserConfig represents the git user configuration for a repository
type UserConfig struct {
	Name          string            // user.name value
	Email         string            // user.email value
	SigningKey    string            // user.signingkey value
	SignCommits   bool              // commit.gpgsign setting
	SigningFormat string            // gpg.format value; empty means git's default (openpgp)
	Source        config.UserSource // Where the config comes from
}

// GlobalUserConfig represents the user identity git uses outside any repository
type GlobalUserConfig struct {
	Name          string
	Email         string
	SigningKey    string
	SignCommits   bool
	SigningFormat string // gpg.format value; empty means git's default (openpgp)
}

// IncludeIfDirective is an [includeIf "<condition>"] path entry, with the path
// resolved the way git resolves it.
type IncludeIfDirective struct {
	Condition string // e.g. "gitdir:~/work/"
	Path      string // absolute path of the included file
}

// userKeysPattern matches every key that contributes to a user identity.
const userKeysPattern = `^(user\.(name|email|signingkey)|commit\.gpgsign|gpg\.format)$`

// configEntry is one value reported by git config, with where it came from.
type configEntry struct {
	scope  string // system, global, local, worktree or command
	origin string // e.g. "file:/home/me/.gitconfig"
	key    string // lowercased by git, e.g. "user.email"
	value  string
	bare   bool // key present with no "= value", which git reads as boolean true
}

// Reading identity through git itself, rather than parsing config files, means
// every location git honors is respected: ~/.gitconfig, $XDG_CONFIG_HOME/git/config,
// $GIT_CONFIG_GLOBAL, the system config, relative [include] paths and every
// [includeIf] condition type.

// GetUserConfig reads the effective git user configuration for a repository.
// Source is local when the repository's own config supplies the name or email,
// includeIf when a conditional include does, and global otherwise.
func GetUserConfig(repoPath string) (*UserConfig, error) {
	if _, err := git.PlainOpen(repoPath); err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	entries, err := readConfigEntries(repoPath, "--get-regexp", userKeysPattern)
	if err != nil {
		return nil, err
	}

	return resolveUserConfig(entries)
}

// GetNonLocalUserConfig reads the git user configuration for a repository,
// skipping the repository's own config. This returns the underlying global or
// includeIf configuration, which is used when local config should not be persisted.
func GetNonLocalUserConfig(repoPath string) (*UserConfig, error) {
	entries, err := readConfigEntries(repoPath, "--get-regexp", userKeysPattern)
	if err != nil {
		return nil, err
	}

	var nonLocal []configEntry
	for _, e := range entries {
		if !isRepoScope(e.scope) {
			nonLocal = append(nonLocal, e)
		}
	}

	return resolveUserConfig(nonLocal)
}

// GetGlobalDefaultUser reads the user identity git uses outside any repository:
// the system and global config plus their unconditional includes. Returns nil
// when no identity is configured.
func GetGlobalDefaultUser() (*GlobalUserConfig, error) {
	entries, err := readConfigEntries(neutralDir(), "--get-regexp", userKeysPattern)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	eff := effectiveEntries(entries)
	return &GlobalUserConfig{
		Name:          eff["user.name"].value,
		Email:         eff["user.email"].value,
		SigningKey:    eff["user.signingkey"].value,
		SignCommits:   entryBool(eff["commit.gpgsign"]),
		SigningFormat: eff["gpg.format"].value,
	}, nil
}

// ListIncludeIfDirectives returns every includeIf path directive git sees
// outside a repository, in the order git reads them.
func ListIncludeIfDirectives() ([]IncludeIfDirective, error) {
	entries, err := readConfigEntries(neutralDir(), "--get-regexp", `^includeif\..*\.path$`)
	if err != nil {
		return nil, err
	}

	var directives []IncludeIfDirective
	for _, e := range entries {
		// git lowercases the section and variable but preserves the condition
		condition := strings.TrimSuffix(strings.TrimPrefix(e.key, "includeif."), ".path")
		path := resolveIncludePath(e.value, e.origin)
		if condition == "" || path == "" {
			continue
		}
		directives = append(directives, IncludeIfDirective{Condition: condition, Path: path})
	}
	return directives, nil
}

// ReadUserConfigFile reads the user identity defined in a single config file,
// following any includes inside it.
func ReadUserConfigFile(path string) (*GlobalUserConfig, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	entries, err := readConfigEntries(neutralDir(), "--file", path, "--includes", "--get-regexp", userKeysPattern)
	if err != nil {
		return nil, err
	}

	eff := effectiveEntries(entries)
	return &GlobalUserConfig{
		Name:          eff["user.name"].value,
		Email:         eff["user.email"].value,
		SigningKey:    eff["user.signingkey"].value,
		SignCommits:   entryBool(eff["commit.gpgsign"]),
		SigningFormat: eff["gpg.format"].value,
	}, nil
}

// resolveUserConfig collapses config entries into the effective identity and
// classifies where the name and email came from.
func resolveUserConfig(entries []configEntry) (*UserConfig, error) {
	eff := effectiveEntries(entries)
	userConfig := &UserConfig{
		Name:          eff["user.name"].value,
		Email:         eff["user.email"].value,
		SigningKey:    eff["user.signingkey"].value,
		SignCommits:   entryBool(eff["commit.gpgsign"]),
		SigningFormat: eff["gpg.format"].value,
		Source:        config.UserSourceUnknown,
	}

	identity := make([]configEntry, 0, 2)
	for _, key := range []string{"user.name", "user.email"} {
		if e, ok := eff[key]; ok {
			identity = append(identity, e)
		}
	}
	if len(identity) == 0 {
		return userConfig, nil
	}

	for _, e := range identity {
		if isRepoScope(e.scope) {
			userConfig.Source = config.UserSourceLocal
			return userConfig, nil
		}
	}

	// A value whose origin is not read outside the repository can only have
	// arrived through a conditional include that matched this repository.
	unconditional, err := neutralOrigins()
	if err != nil {
		return nil, err
	}
	userConfig.Source = config.UserSourceGlobal
	for _, e := range identity {
		if !unconditional[e.origin] {
			userConfig.Source = config.UserSourceIncludeIf
			break
		}
	}

	return userConfig, nil
}

// neutralOrigins returns the set of config origins git reads outside any repository.
func neutralOrigins() (map[string]bool, error) {
	entries, err := readConfigEntries(neutralDir(), "--get-regexp", userKeysPattern)
	if err != nil {
		return nil, err
	}
	origins := make(map[string]bool, len(entries))
	for _, e := range entries {
		origins[e.origin] = true
	}
	return origins, nil
}

// effectiveEntries returns the last value git reports for each key, which is
// the one that takes effect.
func effectiveEntries(entries []configEntry) map[string]configEntry {
	eff := make(map[string]configEntry, len(entries))
	for _, e := range entries {
		eff[e.key] = e
	}
	return eff
}

// isRepoScope reports whether a scope belongs to the repository itself.
func isRepoScope(scope string) bool {
	return scope == "local" || scope == "worktree"
}

// entryBool interprets a config value the way git reads booleans.
func entryBool(e configEntry) bool {
	if e.bare {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(e.value)) {
	case "true", "yes", "on", "1":
		return true
	}
	return false
}

// resolveIncludePath resolves an include path the way git does: ~/ expands to
// the home directory and relative paths are relative to the including file.
func resolveIncludePath(path, origin string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, path[2:])
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	originFile, ok := strings.CutPrefix(origin, "file:")
	if !ok || !filepath.IsAbs(originFile) {
		return ""
	}
	return filepath.Join(filepath.Dir(originFile), path)
}

// neutralDir returns a directory that is not inside any repository, so git
// evaluates config without repository context and no gitdir condition matches.
func neutralDir() string {
	tmp := os.TempDir()
	return filepath.VolumeName(tmp) + string(filepath.Separator)
}

// readConfigEntries runs git config with scope and origin reporting in dir and
// parses the NUL-delimited output. No matching keys is not an error.
func readConfigEntries(dir string, args ...string) ([]configEntry, error) {
	fullArgs := append([]string{"config", "--null", "--show-scope", "--show-origin"}, args...)
	cmd := exec.CommandContext(context.Background(), "git", fullArgs...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		// git config exits 1 when no key matches
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stderr.Len() == 0 {
			return nil, nil
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("git config failed: %s", msg)
		}
		return nil, fmt.Errorf("git config failed: %w", err)
	}

	return parseConfigEntries(out), nil
}

// parseConfigEntries parses `git config --null --show-scope --show-origin`
// output: each entry is scope NUL origin NUL key [LF value] NUL.
func parseConfigEntries(out []byte) []configEntry {
	fields := strings.Split(string(out), "\x00")

	var entries []configEntry
	for i := 0; i+2 < len(fields); i += 3 {
		e := configEntry{scope: fields[i], origin: fields[i+1]}
		key, value, hasValue := strings.Cut(fields[i+2], "\n")
		e.key = key
		e.value = value
		e.bare = !hasValue
		entries = append(entries, e)
	}
	return entries
}
