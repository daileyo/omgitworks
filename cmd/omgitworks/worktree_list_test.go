package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/xdg"
)

func saveConfigForWorktreeTests(t *testing.T, repos []config.Repository) {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(xdg.EnvConfigHome, "")
	t.Setenv(xdg.EnvDataHome, "")

	cfg := config.New("/workspace")
	cfg.Repositories = repos
	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
}

func TestRunWorktreeList_AllWorktrees(t *testing.T) {
	repos := []config.Repository{
		{
			Name: "repo-a",
			Path: "/workspace/repo-a",
			Worktrees: []config.Worktree{
				{Path: "/workspace/repo-a.wt/feat", Branch: "feat", Aligned: true},
				{Path: "/tmp/hotfix", Branch: "hotfix", Aligned: false},
			},
		},
		{
			Name: "repo-b",
			Path: "/workspace/repo-b",
			Worktrees: []config.Worktree{
				{Path: "/workspace/repo-b.wt/dev", Branch: "dev", Aligned: true},
			},
		},
	}
	saveConfigForWorktreeTests(t, repos)

	var buf bytes.Buffer
	err := runWorktreeList(worktreeScope{}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Should show header
	if !strings.Contains(output, "REPO") || !strings.Contains(output, "BRANCH") {
		t.Errorf("output should contain header, got:\n%s", output)
	}

	// Should show all 3 worktrees
	if !strings.Contains(output, "repo-a") || !strings.Contains(output, "feat") {
		t.Errorf("should show repo-a feat worktree, got:\n%s", output)
	}
	if !strings.Contains(output, "hotfix") {
		t.Errorf("should show hotfix worktree, got:\n%s", output)
	}
	if !strings.Contains(output, "repo-b") || !strings.Contains(output, "dev") {
		t.Errorf("should show repo-b dev worktree, got:\n%s", output)
	}

	// Unaligned indicator
	if !strings.Contains(output, "(unaligned)") {
		t.Errorf("should show (unaligned) for hotfix worktree, got:\n%s", output)
	}

	// Count lines (header + separator + 3 data rows)
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines (header + sep + 3 rows), got %d:\n%s", len(lines), output)
	}
}

func TestRunWorktreeList_FilterByRepo(t *testing.T) {
	repos := []config.Repository{
		{
			Name: "repo-a",
			Path: "/workspace/repo-a",
			Worktrees: []config.Worktree{
				{Path: "/workspace/repo-a.wt/feat", Branch: "feat", Aligned: true},
			},
		},
		{
			Name: "repo-b",
			Path: "/workspace/repo-b",
			Worktrees: []config.Worktree{
				{Path: "/workspace/repo-b.wt/dev", Branch: "dev", Aligned: true},
			},
		},
	}
	saveConfigForWorktreeTests(t, repos)

	var buf bytes.Buffer
	err := runWorktreeList(worktreeScope{NamePattern: "repo-a", Label: "repo-a"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "repo-a") {
		t.Errorf("should show repo-a, got:\n%s", output)
	}
	if strings.Contains(output, "repo-b") {
		t.Errorf("should NOT show repo-b when filtered, got:\n%s", output)
	}
}

func TestRunWorktreeList_NoWorktrees(t *testing.T) {
	repos := []config.Repository{
		{Name: "plain-repo", Path: "/workspace/plain-repo"},
	}
	saveConfigForWorktreeTests(t, repos)

	var buf bytes.Buffer
	err := runWorktreeList(worktreeScope{}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No worktrees found") {
		t.Errorf("expected 'No worktrees found' message, got:\n%s", output)
	}
}

func TestRunWorktreeList_NoWorktreesForRepo(t *testing.T) {
	repos := []config.Repository{
		{Name: "plain-repo", Path: "/workspace/plain-repo"},
	}
	saveConfigForWorktreeTests(t, repos)

	var buf bytes.Buffer
	err := runWorktreeList(worktreeScope{NamePattern: "plain-repo", Label: "plain-repo"}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No worktrees found for 'plain-repo'") {
		t.Errorf("expected repo-specific no-worktrees message, got:\n%s", output)
	}
}

func TestRunWorktreeList_UnalignedIndicator(t *testing.T) {
	repos := []config.Repository{
		{
			Name: "my-repo",
			Path: "/workspace/my-repo",
			Worktrees: []config.Worktree{
				{Path: "/workspace/my-repo.wt/feat", Branch: "feat", Aligned: true},
				{Path: "/tmp/external", Branch: "external", Aligned: false},
			},
		},
	}
	saveConfigForWorktreeTests(t, repos)

	var buf bytes.Buffer
	err := runWorktreeList(worktreeScope{}, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Find the aligned and unaligned lines
	var alignedLine, unalignedLine string
	for _, line := range lines {
		if strings.Contains(line, "feat") {
			alignedLine = line
		}
		if strings.Contains(line, "external") {
			unalignedLine = line
		}
	}

	if !strings.Contains(alignedLine, "aligned") || strings.Contains(alignedLine, "(unaligned)") {
		t.Errorf("aligned worktree should show 'aligned', got: %s", alignedLine)
	}
	if !strings.Contains(unalignedLine, "(unaligned)") {
		t.Errorf("unaligned worktree should show '(unaligned)', got: %s", unalignedLine)
	}
}
