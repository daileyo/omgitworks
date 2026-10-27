package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
)

const testSigningKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITESTKEY"

// isolateGitForSync gives the test an empty HOME with no system or global git
// config, so only the config the test writes is visible to git.
func isolateGitForSync(t *testing.T) string {
	t.Helper()
	withTempHome(t)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// An empty GIT_CONFIG_GLOBAL disables global config entirely, so unset it.
	t.Setenv("GIT_CONFIG_GLOBAL", "")
	os.Unsetenv("GIT_CONFIG_GLOBAL")
	return os.Getenv("HOME")
}

// writeGlobalGitConfig writes ~/.gitconfig in the isolated home.
func writeGlobalGitConfig(t *testing.T, home, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write global git config: %v", err)
	}
}

const signedGlobalIdentity = "[user]\n\tname = Robert D\n\temail = robert@example.com\n\tsigningkey = " + testSigningKey + "\n[commit]\n\tgpgsign = true\n[gpg]\n\tformat = ssh\n"

func TestSyncProfilesFromRepos_GlobalDefaultCarriesSigning(t *testing.T) {
	home := isolateGitForSync(t)
	writeGlobalGitConfig(t, home, signedGlobalIdentity)

	cfg := &config.Config{}
	syncProfilesFromRepos(cfg)

	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected one profile from the global identity, got %+v", cfg.Profiles)
	}
	p := cfg.Profiles[0]
	if p.Name != "robert-d" || p.GitName != "Robert D" || p.Email != "robert@example.com" {
		t.Errorf("unexpected identity on profile: %+v", p)
	}
	if p.SigningKey != testSigningKey {
		t.Errorf("expected signing key %q, got %q", testSigningKey, p.SigningKey)
	}
	if !p.SignCommits {
		t.Errorf("expected SignCommits to be true, got %+v", p)
	}
	if p.SigningFormat != "ssh" {
		t.Errorf("expected signing format ssh, got %q", p.SigningFormat)
	}
}

func TestSyncProfilesFromRepos_RepoLocalIdentityCarriesSigning(t *testing.T) {
	isolateGitForSync(t)

	repoPath := filepath.Join(t.TempDir(), "signed-repo")
	createInitTestRepo(t, repoPath)
	local := "[user]\n\tname = Local Dev\n\temail = local@example.com\n\tsigningkey = " + testSigningKey + "\n[commit]\n\tgpgsign = true\n"
	if err := os.WriteFile(filepath.Join(repoPath, ".git", "config"), []byte(local), 0644); err != nil {
		t.Fatalf("failed to write local git config: %v", err)
	}

	cfg := &config.Config{Repositories: []config.Repository{{Name: "signed-repo", Path: repoPath}}}
	syncProfilesFromRepos(cfg)

	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected one profile from the local identity, got %+v", cfg.Profiles)
	}
	p := cfg.Profiles[0]
	if p.Email != "local@example.com" || p.SigningKey != testSigningKey || !p.SignCommits {
		t.Errorf("expected signed profile for local@example.com, got %+v", p)
	}
}

func TestSyncProfilesFromRepos_StoredRepoIdentityReadsSigningKeyFromGit(t *testing.T) {
	home := isolateGitForSync(t)
	writeGlobalGitConfig(t, home, signedGlobalIdentity)

	repoPath := filepath.Join(t.TempDir(), "global-repo")
	createInitTestRepo(t, repoPath)

	// A repo record as detectUserForRepos stores it: identity and sign flag,
	// but no key.
	cfg := &config.Config{Repositories: []config.Repository{{
		Name: "global-repo", Path: repoPath,
		User: "Robert D", Email: "robert@example.com", SigningEnabled: true, UserSource: config.UserSourceGlobal,
	}}}
	syncProfilesFromRepos(cfg)

	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected a single profile, got %+v", cfg.Profiles)
	}
	if p := cfg.Profiles[0]; p.SigningKey != testSigningKey || !p.SignCommits {
		t.Errorf("expected signing key and sign flag on profile, got %+v", p)
	}
}

func TestSyncProfilesFromRepos_BackfillsExistingProfileSigning(t *testing.T) {
	home := isolateGitForSync(t)
	writeGlobalGitConfig(t, home, signedGlobalIdentity)

	// A profile created by an earlier sync that dropped signing info.
	cfg := &config.Config{Profiles: []config.Profile{{Name: "robertd", GitName: "Robert D", Email: "Robert@Example.com"}}}
	syncProfilesFromRepos(cfg)

	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected the existing profile to be reused, got %+v", cfg.Profiles)
	}
	p := cfg.Profiles[0]
	if p.Name != "robertd" || p.Email != "Robert@Example.com" {
		t.Errorf("expected the stored profile to keep its name and email, got %+v", p)
	}
	if p.SigningKey != testSigningKey || !p.SignCommits || p.SigningFormat != "ssh" {
		t.Errorf("expected signing info to be backfilled, got %+v", p)
	}
}

func TestSyncProfilesFromRepos_BackfillKeepsExplicitSigningKey(t *testing.T) {
	home := isolateGitForSync(t)
	writeGlobalGitConfig(t, home, signedGlobalIdentity)

	cfg := &config.Config{Profiles: []config.Profile{{
		Name: "robertd", GitName: "Robert D", Email: "robert@example.com", SigningKey: "ABCDEF1234567890",
	}}}
	syncProfilesFromRepos(cfg)

	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected the existing profile to be reused, got %+v", cfg.Profiles)
	}
	p := cfg.Profiles[0]
	if p.SigningKey != "ABCDEF1234567890" {
		t.Errorf("expected the explicit signing key to be kept, got %q", p.SigningKey)
	}
	if !p.SignCommits {
		t.Errorf("expected the missing sign flag to be filled in, got %+v", p)
	}
}

func TestSyncProfilesFromRepos_IncludeIfProfileBackfillsExisting(t *testing.T) {
	home := isolateGitForSync(t)
	workDir := filepath.Join(home, "work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("failed to create work dir: %v", err)
	}
	workCfg := filepath.Join(home, ".gitconfig-work")
	if err := os.WriteFile(workCfg, []byte("[user]\n\tname = Work Me\n\temail = me@work.example\n\tsigningkey = "+testSigningKey+"\n[commit]\n\tgpgsign = true\n"), 0644); err != nil {
		t.Fatalf("failed to write work config: %v", err)
	}
	writeGlobalGitConfig(t, home, "[includeIf \"gitdir:"+workDir+"/\"]\n\tpath = "+workCfg+"\n")

	cfg := &config.Config{Profiles: []config.Profile{{Name: "work", GitName: "Work Me", Email: "me@work.example"}}}
	syncProfilesFromRepos(cfg)

	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected the existing profile to be reused, got %+v", cfg.Profiles)
	}
	if p := cfg.Profiles[0]; p.Name != "work" || p.SigningKey != testSigningKey || !p.SignCommits {
		t.Errorf("expected signing info backfilled onto the work profile, got %+v", p)
	}
}

func TestBackfillProfileSigning(t *testing.T) {
	tests := []struct {
		name        string
		profile     config.Profile
		key         string
		sign        bool
		format      string
		wantChanged bool
		want        config.Profile
	}{
		{"fills empty", config.Profile{}, "K", true, "ssh", true, config.Profile{SigningKey: "K", SignCommits: true, SigningFormat: "ssh"}},
		{"keeps existing key", config.Profile{SigningKey: "A"}, "K", false, "", false, config.Profile{SigningKey: "A"}},
		{"keeps existing format", config.Profile{SigningFormat: "openpgp"}, "", false, "ssh", false, config.Profile{SigningFormat: "openpgp"}},
		{"never clears sign", config.Profile{SignCommits: true}, "", false, "", false, config.Profile{SignCommits: true}},
		{"nothing to add", config.Profile{}, "", false, "", false, config.Profile{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.profile
			if got := backfillProfileSigning(&p, tt.key, tt.sign, tt.format); got != tt.wantChanged {
				t.Errorf("changed = %v, want %v", got, tt.wantChanged)
			}
			if p != tt.want {
				t.Errorf("profile = %+v, want %+v", p, tt.want)
			}
		})
	}
}

func TestSigningFormatLabel(t *testing.T) {
	tests := map[string]string{"": "gpg", "openpgp": "gpg", "OpenPGP": "gpg", "ssh": "ssh", " SSH ": "ssh", "x509": "x509"}
	for in, want := range tests {
		if got := signingFormatLabel(in); got != want {
			t.Errorf("signingFormatLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
