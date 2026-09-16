package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const dryRunHeader = "Dry run — no changes will be made:"

// refreshDryRun runs the command entry point with --dry-run set.
func refreshDryRun(t *testing.T, args []string, tags ...string) (string, error) {
	t.Helper()
	resetRefreshFlags(t)
	flagWorktreeRefreshTags = tags
	flagWorktreeRefreshDryRun = true
	defer func() { flagWorktreeRefreshDryRun = false }()
	var buf bytes.Buffer
	err := runWorktreeRefreshCommand(args, &buf)
	return buf.String(), err
}

// backdateTree sets every entry under the given roots to a fixed time long in
// the past. File timestamps come from a coarse kernel clock, so a write landing
// in the same tick as the fixture's last write would otherwise leave the
// modification time unchanged and go unnoticed.
func backdateTree(t *testing.T, roots ...string) {
	t.Helper()
	past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			return os.Chtimes(path, past, past)
		})
		if err != nil {
			t.Fatalf("backdating %s failed: %v", root, err)
		}
	}
}

// treeSnapshot records every entry under the given roots with its size, mode,
// modification time, and, for regular files, a hash of the content, so any
// write, creation, or deletion shows up as a difference. Call backdateTree
// first so modification times can reveal a write.
func treeSnapshot(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			digest := ""
			if info.Mode().IsRegular() {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				digest = fmt.Sprintf("%x", sha256.Sum256(data))
			}
			snap[path] = fmt.Sprintf("%d %v %d %s", info.Size(), info.Mode(), info.ModTime().UnixNano(), digest)
			return nil
		})
		if err != nil {
			t.Fatalf("snapshot of %s failed: %v", root, err)
		}
	}
	return snap
}

// dryRunFixture tracks svc-a with pending changes of every detectable kind and
// svc-b as a repository that will fail. It returns the paths involved.
func dryRunFixture(t *testing.T) (paths map[string]string, ext, gone, mislinked string) {
	t.Helper()
	paths = setupTaggedFixture(t, taggedRepo{Name: "svc-a"}, taggedRepo{Name: "svc-b"})
	addWorktreeTo(t, "svc-a", "feat-gone")
	addWorktreeTo(t, "svc-a", "feat-stale")
	addWorktreeTo(t, "svc-a", "feat-switch")

	gone = projectsPath(t, "svc-a", "feat-gone")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	ext = filepath.Join(outside, "ext")
	gitWorktreeAdd(t, paths["svc-a"], "feat-ext", ext)
	mislinked = filepath.Join(outside, "mislinked")
	gitWorktreeAdd(t, paths["svc-a"], "feat-mislinked", mislinked)
	breakWorktreeLink(t, mislinked)
	setStoredAligned(t, "svc-a", "feat-stale", false)
	runGitIn(t, projectsPath(t, "svc-a", "feat-switch"), "checkout", "-q", "-b", "feat-switched")
	breakRepo(t, paths["svc-b"])
	return paths, ext, gone, mislinked
}

func TestWorktreeRefreshDryRun_ConfigByteIdentical(t *testing.T) {
	dryRunFixture(t)
	before := configBytes(t)

	out, err := refreshDryRun(t, nil)

	if !errors.Is(err, errPartialFailure) {
		t.Fatalf("expected the fixture's failing repository to be reported, got %v", err)
	}
	if !strings.Contains(out, "added") || !strings.Contains(out, "removed") {
		t.Fatalf("fixture sanity: the dry run should report pending changes:\n%s", out)
	}
	if after := configBytes(t); !bytes.Equal(before, after) {
		t.Errorf("config.json changed during a dry run\nbefore: %s\nafter:  %s", before, after)
	}
}

func TestWorktreeRefreshDryRun_DoesNotRepair(t *testing.T) {
	_, _, _, mislinked := dryRunFixture(t)

	if _, err := refreshDryRun(t, nil); !errors.Is(err, errPartialFailure) {
		t.Fatalf("unexpected result: %v", err)
	}

	if _, err := os.Stat(filepath.Join(mislinked, ".git")); !os.IsNotExist(err) {
		t.Errorf("the broken .git link was restored, so repair ran during a dry run (stat err: %v)", err)
	}
}

func TestWorktreeRefreshDryRun_DoesNotPrune(t *testing.T) {
	paths, _, gone, _ := dryRunFixture(t)

	if _, err := refreshDryRun(t, nil); !errors.Is(err, errPartialFailure) {
		t.Fatalf("unexpected result: %v", err)
	}

	if !containsPath(gitWorktreePaths(t, paths["svc-a"]), gone) {
		t.Error("git no longer records the deleted worktree, so prune ran during a dry run")
	}
}

func TestWorktreeRefreshDryRun_FilesystemUntouched(t *testing.T) {
	paths, ext, _, _ := dryRunFixture(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	roots := []string{home, filepath.Dir(paths["svc-a"]), filepath.Dir(ext)}
	backdateTree(t, roots...)
	before := treeSnapshot(t, roots...)

	if _, err := refreshDryRun(t, nil); !errors.Is(err, errPartialFailure) {
		t.Fatalf("unexpected result: %v", err)
	}

	after := treeSnapshot(t, roots...)
	for path, was := range before {
		if now, ok := after[path]; !ok {
			t.Errorf("removed during dry run: %s", path)
		} else if now != was {
			t.Errorf("modified during dry run: %s", path)
		}
	}
	for path := range after {
		if _, ok := before[path]; !ok {
			t.Errorf("created during dry run: %s", path)
		}
	}
}

func TestWorktreeRefreshDryRun_StatesCaveat(t *testing.T) {
	setupTaggedFixture(t, taggedRepo{Name: "svc-a"})

	out, err := refreshDryRun(t, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(out, dryRunHeader+"\n") {
		t.Errorf("dry run should open with the shared header, got:\n%s", out)
	}
	if !strings.Contains(out, "git worktree repair and prune were not run") {
		t.Errorf("dry run must say repair and prune were not run, got:\n%s", out)
	}
	if !strings.Contains(out, "\nWould refresh 1 repository, 0 would change\n") {
		t.Errorf("dry-run summary should be framed as what would happen, got:\n%s", out)
	}
}

// TestWorktreeRefreshDryRun_OutputParity runs a dry run and then a real run
// on the same fixture, which holds every kind of reportable change plus a
// failing repository. The preview must predict the real run exactly: the same
// change lines and the same counts, differing only in the dry-run header and
// the summary's wording. The fixture includes a worktree whose .git link is
// broken, the case the caveat warns about.
func TestWorktreeRefreshDryRun_OutputParity(t *testing.T) {
	dryRunFixture(t)

	preview, previewErr := refreshDryRun(t, nil)
	actual, actualErr := refreshAll(t)

	if !errors.Is(previewErr, errPartialFailure) || !errors.Is(actualErr, errPartialFailure) {
		t.Fatalf("both runs should report the failing repository: dry=%v real=%v", previewErr, actualErr)
	}

	// Up to the error list, the preview should equal the real run once the
	// header is dropped and the summary reworded. The error detail itself
	// legitimately differs: the dry run's first git step is list, not repair.
	normalize := func(out string) string {
		body, _, _ := strings.Cut(out, "\n1 error:\n")
		body = strings.TrimPrefix(body, dryRunHeader+"\n")
		if caveat, rest, ok := strings.Cut(body, "\n\n"); ok && strings.Contains(caveat, "were not run") {
			body = rest
		}
		body = strings.Replace(body, "Would refresh", "Refreshed", 1)
		return strings.Replace(body, " would change", " changed", 1)
	}
	if normalize(preview) != normalize(actual) {
		t.Errorf("preview does not match the real run\n--- dry run ---\n%s\n--- real run ---\n%s", preview, actual)
	}
	for _, kind := range []string{"added", "removed", "realigned", "branch"} {
		if !strings.Contains(actual, "  "+kind+" ") {
			t.Errorf("fixture sanity: real run should include a %q line:\n%s", kind, actual)
		}
	}
	if !strings.Contains(preview, "  svc-b: ") || !strings.Contains(actual, "  svc-b: ") {
		t.Error("both runs should name the failing repository")
	}
}
