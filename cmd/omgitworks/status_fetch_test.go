package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/daileyo/omgitworks/internal/git"
)

// statusFetchRepo creates a repository with one commit on main and points
// HOME at a temp dir so the shared cache file lands there.
func statusFetchRepo(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	runGitIn(t, dir, "init", "-b", "main")
	runGitIn(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com",
		"commit", "--allow-empty", "-m", "init")
	return dir
}

// seedCache writes one entry to the shared cache file.
func seedCache(t *testing.T, path string, status *git.Status) {
	t.Helper()
	cachePath, err := git.GetCachePath()
	if err != nil {
		t.Fatalf("cache path: %v", err)
	}
	c := git.NewCache(git.DefaultTTL)
	c.Set(path, status)
	if err := c.Save(cachePath); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
}

// readCacheFile returns the entries in the shared cache file, or nil when the
// file does not exist.
func readCacheFile(t *testing.T) map[string]*git.Status {
	t.Helper()
	cachePath, err := git.GetCachePath()
	if err != nil {
		t.Fatalf("cache path: %v", err)
	}
	data, err := os.ReadFile(cachePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("read cache: %v", err)
	}
	var file struct {
		Statuses map[string]*git.Status `json:"statuses"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse cache: %v", err)
	}
	return file.Statuses
}

func TestFetchStatuses_FreshEntryNeedsNoGit(t *testing.T) {
	repo := statusFetchRepo(t)
	seedCache(t, repo, &git.Status{Branch: "cached-branch", IsClean: true, LastChecked: time.Now()})

	// With no git binary reachable, only the cache can answer.
	t.Setenv("PATH", t.TempDir())

	statuses, wait := fetchStatuses([]string{repo}, 2)
	wait()

	got := statuses[repo]
	if got == nil || got.Branch != "cached-branch" {
		t.Fatalf("expected the cached entry to be served, got %+v", got)
	}
}

func TestFetchStatuses_StaleServedThenRefreshed(t *testing.T) {
	repo := statusFetchRepo(t)
	staleTime := time.Now().Add(-2 * git.DefaultTTL)
	seedCache(t, repo, &git.Status{Branch: "stale-branch", IsClean: true, LastChecked: staleTime})

	statuses, wait := fetchStatuses([]string{repo}, 2)

	got := statuses[repo]
	if got == nil || got.Branch != "stale-branch" {
		t.Fatalf("expected the stale entry to be served immediately, got %+v", got)
	}
	if !got.LastChecked.Equal(staleTime) {
		t.Fatalf("expected the served entry to carry the stale timestamp, got %v", got.LastChecked)
	}

	wait()

	after := readCacheFile(t)[repo]
	if after == nil {
		t.Fatal("expected the cache file to still hold the path after refresh")
	}
	if after.Branch != "main" {
		t.Errorf("expected the background refresh to store the real branch, got %q", after.Branch)
	}
	if !after.LastChecked.After(staleTime) {
		t.Errorf("expected a newer LastChecked after refresh, got %v", after.LastChecked)
	}
}

func TestFetchStatuses_UncachedFetchedAndSavedBeforeBackground(t *testing.T) {
	repo := statusFetchRepo(t)
	if readCacheFile(t) != nil {
		t.Fatal("fixture: cache file should not exist yet")
	}

	statuses, wait := fetchStatuses([]string{repo}, 2)

	got := statuses[repo]
	if got == nil || got.Branch != "main" {
		t.Fatalf("expected a foreground fetch of the real status, got %+v", got)
	}
	// The save happens before the background phase, so the file must already
	// hold the entry before wait() is called.
	saved := readCacheFile(t)[repo]
	if saved == nil || saved.Branch != "main" {
		t.Fatalf("expected the cache file to hold the fetched entry before wait(), got %+v", saved)
	}

	wait()
}

func TestFetchStatuses_UnreadablePathHasNoEntry(t *testing.T) {
	repo := statusFetchRepo(t)
	missing := repo + "-missing"

	statuses, wait := fetchStatuses([]string{repo, missing}, 2)
	wait()

	if statuses[repo] == nil {
		t.Error("expected the real repository to have a status")
	}
	if statuses[missing] != nil {
		t.Errorf("expected no entry for a missing path, got %+v", statuses[missing])
	}
}
