package repocontext

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/xdg"
)

// runGit runs a git command in dir and fails the test if it errors.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
}

// initRepo creates a git repository with one empty commit at dir.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "commit", "--allow-empty", "-m", "init")
}

// newWorkspace returns a symlink-resolved temp workspace directory.
func newWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	return resolved
}

// chdir moves into dir for the duration of the test, following the pattern in
// cmd/omgitworks/init_test.go. Tests using it must not call t.Parallel.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

// singleRepoFixture creates one tracked repository named "my-repo" and returns
// the workspace, the repository path, and a config tracking it.
func singleRepoFixture(t *testing.T) (workspace, repoPath string, cfg *config.Config) {
	t.Helper()
	workspace = newWorkspace(t)
	repoPath = filepath.Join(workspace, "my-repo")
	initRepo(t, repoPath)

	cfg = config.New(workspace)
	cfg.Repositories = []config.Repository{{Name: "my-repo", Path: repoPath}}
	return workspace, repoPath, cfg
}

func TestResolve(t *testing.T) {
	_, repoPath, cfg := singleRepoFixture(t)

	repo, err := Resolve(cfg, repoPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Errorf("resolved %q, want my-repo", repo.Name)
	}
}

func TestResolve_Subdirectory(t *testing.T) {
	_, repoPath, cfg := singleRepoFixture(t)

	nested := filepath.Join(repoPath, "internal", "deep", "deeper")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	repo, err := Resolve(cfg, nested)
	if err != nil {
		t.Fatalf("unexpected error from a nested directory: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Errorf("resolved %q, want my-repo", repo.Name)
	}
}

func TestResolve_InsideWorktree(t *testing.T) {
	workspace, repoPath, cfg := singleRepoFixture(t)

	wtPath := filepath.Join(workspace, "worktrees", "feature-x")
	runGit(t, repoPath, "worktree", "add", "-b", "feature-x", wtPath)
	cfg.Repositories[0].Worktrees = []config.Worktree{{Path: wtPath, Branch: "feature-x"}}

	repo, err := Resolve(cfg, wtPath)
	if err != nil {
		t.Fatalf("unexpected error from inside a worktree: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Errorf("resolved %q, want my-repo", repo.Name)
	}
	// The owning repository is returned, never the worktree itself.
	if repo.Path != repoPath {
		t.Errorf("resolved path %q, want the owning repository %q", repo.Path, repoPath)
	}
}

func TestResolve_InsideWorktreeSubdirectory(t *testing.T) {
	workspace, repoPath, cfg := singleRepoFixture(t)

	wtPath := filepath.Join(workspace, "worktrees", "feature-x")
	runGit(t, repoPath, "worktree", "add", "-b", "feature-x", wtPath)
	cfg.Repositories[0].Worktrees = []config.Worktree{{Path: wtPath, Branch: "feature-x"}}

	nested := filepath.Join(wtPath, "pkg", "sub")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	repo, err := Resolve(cfg, nested)
	if err != nil {
		t.Fatalf("unexpected error from inside a worktree subdirectory: %v", err)
	}
	if repo.Path != repoPath {
		t.Errorf("resolved path %q, want the owning repository %q", repo.Path, repoPath)
	}
}

func TestResolve_SymlinkedPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevation on Windows")
	}

	workspace, repoPath, cfg := singleRepoFixture(t)

	// Track the repository through a symlinked path while the working
	// directory is the real one. Both sides must be resolved to match.
	linkDir := filepath.Join(workspace, "linked")
	if err := os.Symlink(repoPath, linkDir); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}
	cfg.Repositories[0].Path = linkDir

	repo, err := Resolve(cfg, repoPath)
	if err != nil {
		t.Fatalf("unexpected error with a symlinked stored path: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Errorf("resolved %q, want my-repo", repo.Name)
	}
}

func TestResolve_NestedRepoNotPrefixMatched(t *testing.T) {
	_, repoPath, cfg := singleRepoFixture(t)

	// An untracked repository inside the tracked one's tree. Prefix matching
	// would wrongly resolve it to the outer repository.
	inner := filepath.Join(repoPath, "vendor", "inner-repo")
	initRepo(t, inner)

	repo, err := Resolve(cfg, inner)
	if !errors.Is(err, ErrNotTracked) {
		t.Fatalf("got (%v, %v), want ErrNotTracked", repo, err)
	}
}

func TestResolve_NotTracked(t *testing.T) {
	_, _, cfg := singleRepoFixture(t)

	// A git repository that exists but is not tracked in config.
	other := filepath.Join(newWorkspace(t), "untracked")
	initRepo(t, other)

	_, err := Resolve(cfg, other)
	if !errors.Is(err, ErrNotTracked) {
		t.Fatalf("got %v, want ErrNotTracked", err)
	}
	assertNamesBothRemedies(t, err)
}

func TestResolve_NotAGitRepo(t *testing.T) {
	_, _, cfg := singleRepoFixture(t)

	_, err := Resolve(cfg, newWorkspace(t))
	if !errors.Is(err, ErrNotTracked) {
		t.Fatalf("got %v, want ErrNotTracked", err)
	}
	assertNamesBothRemedies(t, err)

	// git's own wording must not leak through.
	for _, leak := range []string{"rev-parse", "not a git repository", "fatal"} {
		if strings.Contains(strings.ToLower(err.Error()), leak) {
			t.Errorf("error leaks git's message %q: %s", leak, err)
		}
	}
}

// assertNamesBothRemedies checks the error contract: state the problem, then
// name both ways out.
func assertNamesBothRemedies(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	if !strings.Contains(msg, "not inside a tracked repository") {
		t.Errorf("error does not state the problem: %s", msg)
	}
	if !strings.Contains(msg, "repository argument") {
		t.Errorf("error does not name the explicit-argument remedy: %s", msg)
	}
	if !strings.Contains(msg, "omgw add") {
		t.Errorf("error does not name the 'omgw add' remedy: %s", msg)
	}
}

func TestResolveCurrent(t *testing.T) {
	_, repoPath, cfg := singleRepoFixture(t)
	chdir(t, repoPath)

	repo, err := ResolveCurrent(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Errorf("resolved %q, want my-repo", repo.Name)
	}
}

func TestResolve_DoesNotMutateConfig(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(xdg.EnvConfigHome, "")
	t.Setenv(xdg.EnvDataHome, "")

	_, repoPath, cfg := singleRepoFixture(t)
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	cfgPath, err := xdg.ConfigFile()
	if err != nil {
		t.Fatalf("failed to locate config file: %v", err)
	}
	before, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	// One successful resolution and one failure.
	if _, err := Resolve(cfg, repoPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := Resolve(cfg, newWorkspace(t)); !errors.Is(err, ErrNotTracked) {
		t.Fatalf("got %v, want ErrNotTracked", err)
	}

	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to re-read config: %v", err)
	}
	if string(before) != string(after) {
		t.Error("resolution modified config.json; it must be read-only")
	}
}

func TestResolve_WindowsDriveLetterCase(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only path casing")
	}

	_, repoPath, cfg := singleRepoFixture(t)

	// Store the drive letter in the opposite case from what git reports. The
	// filesystem is case-insensitive, so resolution must still match.
	if len(repoPath) < 2 || repoPath[1] != ':' {
		t.Skipf("unexpected path shape for a drive-letter test: %q", repoPath)
	}
	drive := repoPath[:1]
	flipped := strings.ToLower(drive)
	if flipped == drive {
		flipped = strings.ToUpper(drive)
	}
	cfg.Repositories[0].Path = flipped + repoPath[1:]

	repo, err := Resolve(cfg, repoPath)
	if err != nil {
		t.Fatalf("drive-letter case difference defeated resolution: %v", err)
	}
	if repo.Name != "my-repo" {
		t.Errorf("resolved %q, want my-repo", repo.Name)
	}
}

func TestResolve_NilConfig(t *testing.T) {
	if _, err := Resolve(nil, t.TempDir()); !errors.Is(err, ErrNotTracked) {
		t.Errorf("got %v, want ErrNotTracked", err)
	}
}
