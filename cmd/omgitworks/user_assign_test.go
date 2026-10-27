package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/repocontext"
	"github.com/daileyo/omgitworks/internal/user"
)

// setupAssignFixture creates a workspace tracking one initialized repository
// per name, each with a seed local identity, and two stored profiles whose
// names never collide with a repository name. It returns each repository's
// path by name; the workspace is the parent of any path.
func setupAssignFixture(t *testing.T, repoNames ...string) map[string]string {
	t.Helper()
	withTempHome(t)

	workspace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}

	paths := make(map[string]string, len(repoNames))
	var repos []config.Repository
	for _, name := range repoNames {
		dir := filepath.Join(workspace, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create repo dir: %v", err)
		}
		runGitIn(t, dir, "init")
		runGitIn(t, dir, "config", "user.email", "seed@example.com")
		runGitIn(t, dir, "config", "user.name", "Seed")
		runGitIn(t, dir, "commit", "--allow-empty", "-m", "init")
		paths[name] = dir
		repos = append(repos, config.Repository{Name: name, Path: dir})
	}

	cfg := config.New(workspace)
	cfg.Repositories = repos
	cfg.Profiles = []config.Profile{
		{Name: "work", GitName: "Worker", Email: "work@example.com"},
		{Name: "personal", GitName: "Person", Email: "personal@example.com"},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	return paths
}

// localGitValue reads one key from the repository's local git config.
func localGitValue(t *testing.T, repoDir, key string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repoDir, "config", "--local", key)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --local %s failed in %s: %v", key, repoDir, err)
	}
	return strings.TrimSpace(string(out))
}

// gitConfigBytes snapshots the repository's .git/config.
func gitConfigBytes(t *testing.T, repoDir string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoDir, ".git", "config"))
	if err != nil {
		t.Fatalf("failed to read .git/config: %v", err)
	}
	return b
}

// assignOut runs runUserAssign with captured stdout and no prompt input.
func assignOut(t *testing.T, args []string, opts userAssignOptions) (string, error) {
	t.Helper()
	return assignWithInput(t, args, opts, "")
}

// assignWithInput is assignOut with a scripted answer for the prompt.
func assignWithInput(t *testing.T, args []string, opts userAssignOptions, input string) (string, error) {
	t.Helper()
	var err error
	out := captureStdoutStr(func() {
		err = runUserAssign(args, opts, strings.NewReader(input))
	})
	return out, err
}

// loadRepo reloads configuration and returns the named repository.
func loadRepo(t *testing.T, name string) config.Repository {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	for _, r := range cfg.Repositories {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("repository %q not found after reload", name)
	return config.Repository{}
}

// assertAssignedWork checks the observable outcome of assigning the work
// profile to the named repository: git's local config and the stored fields.
func assertAssignedWork(t *testing.T, name, repoDir, out string) {
	t.Helper()
	if got := localGitValue(t, repoDir, "user.email"); got != "work@example.com" {
		t.Errorf("user.email = %q, want work@example.com", got)
	}
	if got := localGitValue(t, repoDir, "user.name"); got != "Worker" {
		t.Errorf("user.name = %q, want Worker", got)
	}
	repo := loadRepo(t, name)
	if repo.User != "Worker" || repo.Email != "work@example.com" {
		t.Errorf("stored identity = %q <%s>, want Worker <work@example.com>", repo.User, repo.Email)
	}
	if repo.SigningEnabled {
		t.Error("SigningEnabled should be false for a profile without signing")
	}
	if repo.UserSource != config.UserSourceLocal {
		t.Errorf("UserSource = %q, want %q", repo.UserSource, config.UserSourceLocal)
	}
	for _, want := range []string{
		"Assigned profile 'work' to repository '" + name + "'",
		"  user.name:  Worker",
		"  user.email: work@example.com",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestUserAssign_TwoArgs_AssignsNamedRepo(t *testing.T) {
	paths := setupAssignFixture(t, "alpha")

	out, err := assignOut(t, []string{"alpha", "work"}, userAssignOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertAssignedWork(t, "alpha", paths["alpha"], out)
}

func TestUserAssign_TwoArgs_MultiMatchError(t *testing.T) {
	paths := setupAssignFixture(t, "api-core", "api-web")
	beforeCore := gitConfigBytes(t, paths["api-core"])
	beforeWeb := gitConfigBytes(t, paths["api-web"])

	out, err := assignOut(t, []string{"api", "work"}, userAssignOptions{})
	if err == nil {
		t.Fatal("expected an error for an ambiguous identifier")
	}
	if got := err.Error(); got != "please specify a more specific repository identifier" {
		t.Errorf("error = %q", got)
	}
	for _, want := range []string{
		"Multiple repositories match 'api':",
		"  - api-core (" + paths["api-core"] + ")",
		"  - api-web (" + paths["api-web"] + ")",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if !bytes.Equal(beforeCore, gitConfigBytes(t, paths["api-core"])) {
		t.Error("api-core's .git/config changed on an ambiguous match")
	}
	if !bytes.Equal(beforeWeb, gitConfigBytes(t, paths["api-web"])) {
		t.Error("api-web's .git/config changed on an ambiguous match")
	}
}

func TestUserAssign_TwoArgs_UnknownProfileError(t *testing.T) {
	paths := setupAssignFixture(t, "alpha")
	before := gitConfigBytes(t, paths["alpha"])

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	_, want := user.GetProfile(cfg, "nope")
	if want == nil {
		t.Fatal("fixture bug: profile 'nope' should not exist")
	}

	_, err = assignOut(t, []string{"alpha", "nope"}, userAssignOptions{})
	if err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
	if err.Error() != want.Error() {
		t.Errorf("error = %q, want the profile lookup error %q", err, want)
	}
	if errors.Is(err, repocontext.ErrNotTracked) {
		t.Error("a named repository must never produce the resolver error")
	}
	if !bytes.Equal(before, gitConfigBytes(t, paths["alpha"])) {
		t.Error(".git/config changed although the profile was unknown")
	}
}

func TestUserAssign_TwoArgs_DryRunOutput(t *testing.T) {
	paths := setupAssignFixture(t, "alpha")
	beforeGit := gitConfigBytes(t, paths["alpha"])
	beforeCfg := configBytes(t)

	out, err := assignOut(t, []string{"alpha", "work"}, userAssignOptions{dryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		"Dry run - would assign profile 'work' to repository 'alpha':",
		`user.name: "Seed" -> "Worker"`,
		`user.email: "seed@example.com" -> "work@example.com"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if !bytes.Equal(beforeGit, gitConfigBytes(t, paths["alpha"])) {
		t.Error("dry run changed .git/config")
	}
	if !bytes.Equal(beforeCfg, configBytes(t)) {
		t.Error("dry run changed config.json")
	}
}

func TestUserAssign_TwoArgs_UseSubdirsStillWorks(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		paths := setupAssignFixture(t, "alpha")
		oldPath := paths["alpha"]
		workspace := filepath.Dir(oldPath)
		newPath := filepath.Join(workspace, "work", "alpha")

		out, err := assignWithInput(t, []string{"alpha", "work"}, userAssignOptions{useSubdirs: true}, "y\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(newPath); err != nil {
			t.Errorf("repository should have moved to %s: %v", newPath, err)
		}
		if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
			t.Errorf("old path %s should be gone, stat err = %v", oldPath, err)
		}
		repo := loadRepo(t, "alpha")
		if repo.Path != newPath {
			t.Errorf("stored Path = %q, want %q", repo.Path, newPath)
		}
		if repo.UserSource != config.UserSourceIncludeIf {
			t.Errorf("UserSource = %q, want %q", repo.UserSource, config.UserSourceIncludeIf)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("UserHomeDir failed: %v", err)
		}
		gitconfig, err := os.ReadFile(filepath.Join(home, ".gitconfig"))
		if err != nil {
			t.Fatalf("expected a global gitconfig to be written: %v", err)
		}
		if !strings.Contains(string(gitconfig), "[includeIf") {
			t.Errorf("global gitconfig lacks an includeIf section:\n%s", gitconfig)
		}
		if !strings.Contains(out, "Moved repository to:") {
			t.Errorf("output missing move confirmation:\n%s", out)
		}
	})

	t.Run("canceled", func(t *testing.T) {
		paths := setupAssignFixture(t, "alpha")
		oldPath := paths["alpha"]
		before := gitConfigBytes(t, oldPath)

		out, err := assignWithInput(t, []string{"alpha", "work"}, userAssignOptions{useSubdirs: true}, "n\n")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(out, "Canceled.") {
			t.Errorf("output missing cancellation:\n%s", out)
		}
		if _, err := os.Stat(oldPath); err != nil {
			t.Errorf("repository should not have moved: %v", err)
		}
		if !bytes.Equal(before, gitConfigBytes(t, oldPath)) {
			t.Error(".git/config changed after canceling")
		}
	})
}
