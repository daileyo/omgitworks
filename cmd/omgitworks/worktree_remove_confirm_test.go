package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/xdg"
)

// forceStdinTTY overrides the terminal check for the duration of a test.
func forceStdinTTY(t *testing.T, isTTY bool) {
	t.Helper()
	orig := stdinIsTerminalFunc
	stdinIsTerminalFunc = func() bool { return isTTY }
	t.Cleanup(func() { stdinIsTerminalFunc = orig })
}

// removeWithInput runs a tag-scoped removal with the given stdin content.
func removeWithInput(t *testing.T, tag, branch string, opts removeOptions, input string) (string, error) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeTargets(cfg, "", tag)
	if err != nil {
		t.Fatalf("selection failed: %v", err)
	}
	var buf bytes.Buffer
	runErr := runWorktreeRemove(cfg, repos, branch, opts, false, &buf, strings.NewReader(input))
	return buf.String(), runErr
}

// twoWorktreeFixture gives two tagged repositories each holding the branch.
func twoWorktreeFixture(t *testing.T, branch string) map[string]string {
	t.Helper()
	paths := setupTaggedFixture(t,
		taggedRepo{Name: "svc-a", Tags: []string{"backend"}},
		taggedRepo{Name: "svc-b", Tags: []string{"backend"}},
	)
	addWorktreeTo(t, "svc-a", branch)
	addWorktreeTo(t, "svc-b", branch)
	return paths
}

// configBytes reads config.json for before/after comparison.
func configBytes(t *testing.T) []byte {
	t.Helper()
	p, err := xdg.ConfigFile()
	if err != nil {
		t.Fatalf("failed to locate config: %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	return b
}

func TestWorktreeRemove_DryRunChangesNothing(t *testing.T) {
	resetRemoveFlags(t)
	twoWorktreeFixture(t, "feat-x")
	before := configBytes(t)

	out, err := removeWithInput(t, "backend", "feat-x", removeOptions{DryRun: true}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "Dry run") || !strings.Contains(out, "Would remove") {
		t.Errorf("dry run did not announce itself:\n%s", out)
	}
	for _, name := range []string{"svc-a", "svc-b"} {
		if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); statErr != nil {
			t.Errorf("dry run removed %s's worktree: %v", name, statErr)
		}
	}
	if string(before) != string(configBytes(t)) {
		t.Error("dry run modified config.json")
	}
}

func TestWorktreeRemove_DryRunAnnotations(t *testing.T) {
	resetRemoveFlags(t)
	paths := twoWorktreeFixture(t, "feat-x")

	// svc-a locked, svc-b dirty.
	lockWorktree(t, paths["svc-a"], projectsPath(t, "svc-a", "feat-x"), "in review")
	if err := os.WriteFile(filepath.Join(projectsPath(t, "svc-b", "feat-x"), "wip.txt"), []byte("x\n"), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	out, err := removeWithInput(t, "backend", "feat-x", removeOptions{DryRun: true}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "locked") || !strings.Contains(out, "will be skipped") {
		t.Errorf("preview did not mark the locked worktree:\n%s", out)
	}
	if !strings.Contains(out, "will fail without --force") {
		t.Errorf("preview did not mark the dirty worktree:\n%s", out)
	}
	if !strings.Contains(out, "--force is not set") {
		t.Errorf("preview did not state the force posture:\n%s", out)
	}
}

func TestWorktreeRemove_ConfirmationGate(t *testing.T) {
	t.Run("declining removes nothing and returns nil", func(t *testing.T) {
		resetRemoveFlags(t)
		twoWorktreeFixture(t, "feat-x")
		forceStdinTTY(t, true)

		out, err := removeWithInput(t, "backend", "feat-x", removeOptions{}, "n\n")
		if err != nil {
			t.Fatalf("declining is an answer, not an error: %v", err)
		}
		if !strings.Contains(out, "Aborted") {
			t.Errorf("abort not reported:\n%s", out)
		}
		for _, name := range []string{"svc-a", "svc-b"} {
			if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); statErr != nil {
				t.Errorf("%s was removed despite declining: %v", name, statErr)
			}
		}
	})

	t.Run("accepting proceeds", func(t *testing.T) {
		resetRemoveFlags(t)
		twoWorktreeFixture(t, "feat-x")
		forceStdinTTY(t, true)

		if _, err := removeWithInput(t, "backend", "feat-x", removeOptions{}, "y\n"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, name := range []string{"svc-a", "svc-b"} {
			if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); !os.IsNotExist(statErr) {
				t.Errorf("%s was not removed after confirming", name)
			}
		}
	})
}

func TestWorktreeRemove_YesSkipsPrompt(t *testing.T) {
	resetRemoveFlags(t)
	twoWorktreeFixture(t, "feat-x")
	// A non-TTY stdin and an empty reader: --yes must make both irrelevant.
	forceStdinTTY(t, false)

	out, err := removeWithInput(t, "backend", "feat-x", removeOptions{Yes: true}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "[y/N]") {
		t.Errorf("--yes still prompted:\n%s", out)
	}
	for _, name := range []string{"svc-a", "svc-b"} {
		if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); !os.IsNotExist(statErr) {
			t.Errorf("%s was not removed with --yes", name)
		}
	}
}

func TestWorktreeRemove_NonInteractiveRefuses(t *testing.T) {
	resetRemoveFlags(t)
	twoWorktreeFixture(t, "feat-x")
	forceStdinTTY(t, false)

	_, err := removeWithInput(t, "backend", "feat-x", removeOptions{}, "y\n")
	if err == nil {
		t.Fatal("expected a refusal with a non-interactive stdin and no --yes")
	}
	if !strings.Contains(err.Error(), "--yes") {
		t.Errorf("error %q does not instruct the user to pass --yes", err)
	}
	// Nothing may be removed, and the stream must not be consumed as consent.
	for _, name := range []string{"svc-a", "svc-b"} {
		if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); statErr != nil {
			t.Errorf("%s was removed without confirmation: %v", name, statErr)
		}
	}
}

func TestWorktreeRemove_SingleNotPrompted(t *testing.T) {
	resetRemoveFlags(t)
	setupTaggedFixture(t, taggedRepo{Name: "svc-a", Tags: []string{"backend"}})
	addWorktreeTo(t, "svc-a", "feat-x")
	// Not a terminal and nothing to read: a single-worktree run must not ask.
	forceStdinTTY(t, false)

	out, err := removeWithInput(t, "backend", "feat-x", removeOptions{}, "")
	if err != nil {
		t.Fatalf("a single-worktree run must not require confirmation: %v", err)
	}
	if strings.Contains(out, "[y/N]") {
		t.Errorf("single-worktree run prompted:\n%s", out)
	}
	if _, statErr := os.Stat(projectsPath(t, "svc-a", "feat-x")); !os.IsNotExist(statErr) {
		t.Error("worktree was not removed")
	}
}

func TestWorktreeRemove_DryRunWithYes(t *testing.T) {
	resetRemoveFlags(t)
	twoWorktreeFixture(t, "feat-x")

	out, err := removeWithInput(t, "backend", "feat-x", removeOptions{DryRun: true, Yes: true}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Dry run") {
		t.Errorf("--dry-run with --yes should still preview:\n%s", out)
	}
	for _, name := range []string{"svc-a", "svc-b"} {
		if _, statErr := os.Stat(projectsPath(t, name, "feat-x")); statErr != nil {
			t.Errorf("%s was removed despite --dry-run: %v", name, statErr)
		}
	}
}

func TestWorktreeRemove_PreviewMatchesRun(t *testing.T) {
	resetRemoveFlags(t)
	twoWorktreeFixture(t, "feat-x")

	preview, err := removeWithInput(t, "backend", "feat-x", removeOptions{DryRun: true}, "")
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// Every path named in the preview must be exactly what the real run removes.
	var previewed []string
	for _, name := range []string{"svc-a", "svc-b"} {
		p := projectsPath(t, name, "feat-x")
		if strings.Contains(preview, p) {
			previewed = append(previewed, p)
		}
	}
	if len(previewed) != 2 {
		t.Fatalf("preview named %d paths, want 2:\n%s", len(previewed), preview)
	}

	if _, err := removeWithInput(t, "backend", "feat-x", removeOptions{Yes: true}, ""); err != nil {
		t.Fatalf("real run failed: %v", err)
	}
	for _, p := range previewed {
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Errorf("previewed path %s was not removed by the real run", p)
		}
	}
}

// The spec requires the confirmation listing to be textually the same as the
// --dry-run listing, so what a user approves is what they were shown.
func TestWorktreeRemove_ConfirmListingMatchesDryRun(t *testing.T) {
	resetRemoveFlags(t)
	twoWorktreeFixture(t, "feat-x")

	dry, err := removeWithInput(t, "backend", "feat-x", removeOptions{DryRun: true}, "")
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	resetRemoveFlags(t)
	forceStdinTTY(t, true)
	confirm, err := removeWithInput(t, "backend", "feat-x", removeOptions{}, "n\n")
	if err != nil {
		t.Fatalf("confirmation run failed: %v", err)
	}

	// Strip the dry-run banner and the prompt, then compare the listings.
	strip := func(s string) string {
		s = strings.ReplaceAll(s, "Dry run — no changes will be made:\n\n", "")
		if i := strings.Index(s, "Remove these worktrees?"); i >= 0 {
			s = s[:i]
		}
		s = strings.ReplaceAll(s, "Aborted; nothing was removed.\n", "")
		return strings.TrimSpace(s)
	}

	if strip(dry) != strip(confirm) {
		t.Errorf("confirmation listing differs from the dry-run listing:\n--- dry ---\n%s\n--- confirm ---\n%s",
			strip(dry), strip(confirm))
	}
}
