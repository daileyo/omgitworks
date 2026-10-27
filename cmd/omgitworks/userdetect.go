package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/git"
	"github.com/daileyo/omgitworks/internal/user"
)

// detectUserForRepos iterates over repos, calls git.GetUserConfig() for each,
// and populates User/Email/SigningEnabled/UserSource fields. When a repo has
// local .git/config user settings, the underlying global/includeIf config is
// stored instead (local config is detected at display time only). When profiles
// are provided and a repo's user is detected via includeIf, it attempts to
// auto-link the repo to a matching stored profile. Returns the count of repos
// with detected user info.
func detectUserForRepos(repos []config.Repository, profiles ...[]config.Profile) int {
	var profileList []config.Profile
	if len(profiles) > 0 {
		profileList = profiles[0]
	}

	userDetectedCount := 0
	for i := range repos {
		repo := &repos[i]
		userCfg, err := git.GetUserConfig(repo.Path)
		if err != nil || userCfg == nil {
			continue
		}

		// If local config was detected, don't persist it - get the underlying
		// global/includeIf config instead. Local config is read at display time.
		if userCfg.Source == config.UserSourceLocal {
			nonLocalCfg, err := git.GetNonLocalUserConfig(repo.Path)
			if err == nil && nonLocalCfg != nil && (nonLocalCfg.Name != "" || nonLocalCfg.Email != "") {
				userCfg = nonLocalCfg
			} else {
				// No underlying config found; still count as detected but store empty
				userDetectedCount++
				continue
			}
		}

		repo.User = userCfg.Name
		repo.Email = userCfg.Email
		repo.SigningEnabled = userCfg.SignCommits
		repo.UserSource = userCfg.Source
		if userCfg.Name != "" || userCfg.Email != "" {
			userDetectedCount++
		}

		// Auto-link to stored profile if detected via includeIf
		if userCfg.Source == config.UserSourceIncludeIf && len(profileList) > 0 {
			if matched := user.MatchProfileByUser(profileList, userCfg.Name, userCfg.Email); matched != nil {
				_ = matched // Match exists, linking is implicit via matching user/email
			}
		}
	}
	return userDetectedCount
}

// syncProfilesFromRepos scans repos for unique user identities and adds any
// that don't already exist as profiles to cfg.Profiles. This ensures the
// profile list reflects all users that are in use across repos.
//
// Identities are collected from three places, each carrying the signing
// configuration git reports for it: includeIf config files, the global default
// identity, and the identity in effect for each repository. A profile that
// already exists for an identity's email is kept, but its signing key and
// sign-commits setting are filled in when the profile has none and the
// detected identity does.
func syncProfilesFromRepos(cfg *config.Config) {
	// Collect from auto-detected profiles (includeIf); these keep their
	// derived profile names.
	detected, _ := user.DetectProfiles()
	for _, p := range detected {
		if existing := findProfileByEmail(cfg.Profiles, p.Email); existing != nil {
			backfillProfileSigning(existing, p.SigningKey, p.SignCommits, p.SigningFormat)
			continue
		}
		cfg.Profiles = append(cfg.Profiles, p)
	}

	type userIdentity struct {
		name          string
		email         string
		signingKey    string
		signCommits   bool
		signingFormat string
	}
	var identities []userIdentity

	// The identity git uses outside any repository.
	if global, err := git.GetGlobalDefaultUser(); err == nil && global != nil && global.Email != "" {
		identities = append(identities, userIdentity{global.Name, global.Email, global.SigningKey, global.SignCommits, global.SigningFormat})
	}

	// Collect unique users from repos (stored config + live local config)
	for _, repo := range cfg.Repositories {
		liveCfg, err := git.GetUserConfig(repo.Path)
		if err != nil {
			liveCfg = nil
		}

		// Add the stored (global/includeIf) identity. The signing key is not
		// stored on the repo, so take it from the live config when that is the
		// same identity.
		if repo.Email != "" {
			id := userIdentity{name: repo.User, email: repo.Email, signCommits: repo.SigningEnabled}
			if liveCfg != nil && liveCfg.Source != config.UserSourceLocal && strings.EqualFold(liveCfg.Email, repo.Email) {
				id.signingKey = liveCfg.SigningKey
				id.signCommits = id.signCommits || liveCfg.SignCommits
				id.signingFormat = liveCfg.SigningFormat
			}
			identities = append(identities, id)
		}

		// Also check live local config (detectUserForRepos skips persisting these)
		if liveCfg != nil && liveCfg.Source == config.UserSourceLocal && liveCfg.Email != "" {
			identities = append(identities, userIdentity{liveCfg.Name, liveCfg.Email, liveCfg.SigningKey, liveCfg.SignCommits, liveCfg.SigningFormat})
		}
	}

	for _, id := range identities {
		if existing := findProfileByEmail(cfg.Profiles, id.email); existing != nil {
			backfillProfileSigning(existing, id.signingKey, id.signCommits, id.signingFormat)
			continue
		}

		// Derive profile name from git name (lowercase, spaces to hyphens)
		name := strings.ToLower(id.name)
		name = strings.ReplaceAll(name, " ", "-")
		if name == "" {
			// Fall back to email prefix
			name = strings.SplitN(id.email, "@", 2)[0]
		}

		// Ensure unique profile name
		baseName := name
		suffix := 1
		for profileNameExists(cfg.Profiles, name) {
			suffix++
			name = fmt.Sprintf("%s-%d", baseName, suffix)
		}

		cfg.Profiles = append(cfg.Profiles, config.Profile{
			Name:          name,
			GitName:       id.name,
			Email:         id.email,
			SigningKey:    id.signingKey,
			SignCommits:   id.signCommits,
			SigningFormat: id.signingFormat,
		})
	}
}

// findProfileByEmail returns the stored profile with the given email, ignoring
// case, or nil when there is none.
func findProfileByEmail(profiles []config.Profile, email string) *config.Profile {
	for i := range profiles {
		if strings.EqualFold(profiles[i].Email, email) {
			return &profiles[i]
		}
	}
	return nil
}

// backfillProfileSigning fills in signing settings a profile is missing from
// what git reports for the same identity. Values already on the profile are
// never replaced, so explicit choices made with user update are kept.
func backfillProfileSigning(profile *config.Profile, signingKey string, signCommits bool, signingFormat string) bool {
	changed := false
	if profile.SigningKey == "" && signingKey != "" {
		profile.SigningKey = signingKey
		changed = true
	}
	if !profile.SignCommits && signCommits {
		profile.SignCommits = true
		changed = true
	}
	if profile.SigningFormat == "" && signingFormat != "" {
		profile.SigningFormat = signingFormat
		changed = true
	}
	return changed
}

// warnMissingEmail reports repositories whose git identity has a name but no
// email. Profiles are keyed by email, so no profile can be created for them.
func warnMissingEmail(w io.Writer, repos []config.Repository) {
	count := 0
	for _, repo := range repos {
		if repo.User != "" && repo.Email == "" {
			count++
		}
	}
	if count == 0 {
		return
	}
	fmt.Fprintf(w, "Warning: %d %s user.name but no user.email; no profile can be created until user.email is set.\n",
		count, pluralize(count, "repository has", "repositories have"))
}

// profileNameExists checks if a profile with the given name already exists
func profileNameExists(profiles []config.Profile, name string) bool {
	for _, p := range profiles {
		if strings.EqualFold(p.Name, name) {
			return true
		}
	}
	return false
}
