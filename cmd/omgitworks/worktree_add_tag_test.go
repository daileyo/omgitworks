package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/xdg"
)

// taggedRepo describes one repository in a selection fixture.
type taggedRepo struct {
	Name string
	Tags []string
}

// setupTaggedFixture creates a workspace of real git repositories with tags and
// saves a configuration tracking them.
func setupTaggedFixture(t *testing.T, repos ...taggedRepo) map[string]string {
	t.Helper()
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv(xdg.EnvConfigHome, "")
	t.Setenv(xdg.EnvDataHome, "")

	workspaceDir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(workspaceDir)
	if err != nil {
		t.Fatalf("EvalSymlinks failed: %v", err)
	}
	workspaceDir = resolved

	cfg := config.New(workspaceDir)
	paths := make(map[string]string, len(repos))

	for _, r := range repos {
		dir := filepath.Join(workspaceDir, r.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create repo dir: %v", err)
		}
		for _, args := range [][]string{
			{"git", "init"},
			{"git", "config", "user.email", "test@test.com"},
			{"git", "config", "user.name", "Test"},
			{"git", "commit", "--allow-empty", "-m", "init"},
		} {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%v failed: %s\n%s", args, err, out)
			}
		}
		cfg.Repositories = append(cfg.Repositories, config.Repository{
			Name: r.Name, Path: dir, Tags: r.Tags,
		})
		paths[r.Name] = dir
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// --tag is package state; keep cases independent.
	t.Cleanup(func() { flagWorktreeAddTags = nil })

	return paths
}

// selectNames runs selection and returns the chosen repository names.
func selectNames(t *testing.T, pattern, tag string) ([]string, error) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeAddTargets(cfg, pattern, tag)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(repos))
	for _, r := range repos {
		names = append(names, r.Name)
	}
	slices.Sort(names)
	return names, nil
}

// standardFixture: two backend repos, one frontend, one untagged. The "api"
// substring spans a tagged and an untagged repo so patterns can cross groups.
func standardFixture(t *testing.T) map[string]string {
	t.Helper()
	return setupTaggedFixture(t,
		taggedRepo{Name: "api-core", Tags: []string{"backend"}},
		taggedRepo{Name: "api-edge", Tags: []string{"backend"}},
		taggedRepo{Name: "web-ui", Tags: []string{"frontend"}},
		taggedRepo{Name: "api-legacy", Tags: nil},
	)
}

func TestWorktreeAddSelection_PrecedenceTable(t *testing.T) {
	paths := standardFixture(t)

	tests := []struct {
		name    string
		pattern string
		tag     string
		cwd     string
		want    []string
	}{
		{"1 positional, no tag -> current repo", "", "", paths["web-ui"], []string{"web-ui"}},
		{"1 positional with tag -> all tagged", "", "backend", paths["web-ui"], []string{"api-core", "api-edge"}},
		{"2 positionals, no tag -> name pattern", "web", "", paths["api-core"], []string{"web-ui"}},
		{"2 positionals with tag -> name AND tag", "api", "backend", paths["web-ui"], []string{"api-core", "api-edge"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chdirForTest(t, tt.cwd)
			got, err := selectNames(t, tt.pattern, tt.tag)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("selected %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorktreeAddSelection_TagMatching(t *testing.T) {
	standardFixture(t)

	tests := []struct {
		name string
		tag  string
		want []string
	}{
		{"exact", "backend", []string{"api-core", "api-edge"}},
		{"case-insensitive", "BACKEND", []string{"api-core", "api-edge"}},
		{"wildcard", "back*", []string{"api-core", "api-edge"}},
		{"wildcard single char", "backen?", []string{"api-core", "api-edge"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectNames(t, "", tt.tag)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("tag %q selected %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestWorktreeAddSelection_CombinedAnd(t *testing.T) {
	standardFixture(t)

	// "api" matches api-core, api-edge, api-legacy. Only the first two carry
	// the backend tag, so api-legacy must drop out.
	got, err := selectNames(t, "api", "backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"api-core", "api-edge"}
	if !slices.Equal(got, want) {
		t.Errorf("selected %v, want %v", got, want)
	}

	// A repo carrying the tag but not matching the name is excluded too.
	got, err = selectNames(t, "edge", "backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, []string{"api-edge"}) {
		t.Errorf("selected %v, want [api-edge]", got)
	}
}

func TestWorktreeAddSelection_ResolverNotConsulted(t *testing.T) {
	paths := standardFixture(t)

	t.Run("tag ignores the working directory", func(t *testing.T) {
		chdirForTest(t, paths["web-ui"])
		got, err := selectNames(t, "", "backend")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if slices.Contains(got, "web-ui") {
			t.Errorf("selected %v; the current repo must not leak in when -t is given", got)
		}
	})

	t.Run("pattern ignores the working directory", func(t *testing.T) {
		chdirForTest(t, paths["web-ui"])
		got, err := selectNames(t, "api-core", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !slices.Equal(got, []string{"api-core"}) {
			t.Errorf("selected %v, want [api-core]", got)
		}
	})

	t.Run("detection applies only when neither is given", func(t *testing.T) {
		chdirForTest(t, paths["web-ui"])
		got, err := selectNames(t, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !slices.Equal(got, []string{"web-ui"}) {
			t.Errorf("selected %v, want [web-ui]", got)
		}
	})
}

func TestWorktreeAddSelection_AmbiguityAfterTagFilter(t *testing.T) {
	standardFixture(t)

	// Without a tag, an ambiguous pattern is still rejected, exactly as before.
	_, err := selectNames(t, "api", "")
	if err == nil {
		t.Fatal("expected an ambiguity error for a pattern matching three repos")
	}
	if !strings.Contains(err.Error(), "multiple repositories match 'api'") {
		t.Errorf("error = %q, want the existing ambiguity wording", err)
	}

	// With a tag, the same pattern is a bulk run over the intersection.
	got, tagErr := selectNames(t, "api", "backend")
	if tagErr != nil {
		t.Fatalf("pattern combined with a tag should not be ambiguous: %v", tagErr)
	}
	if len(got) != 2 {
		t.Errorf("selected %v, want both tagged repos", got)
	}
}

func TestWorktreeAddSelection_EmptyMatches(t *testing.T) {
	standardFixture(t)

	tests := []struct {
		name     string
		pattern  string
		tag      string
		contains []string
	}{
		{"tag alone", "", "nosuchtag", []string{"nosuchtag"}},
		{"pattern alone", "nosuchrepo", "", []string{"nosuchrepo"}},
		{"combined names both", "web", "backend", []string{"web", "backend"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := selectNames(t, tt.pattern, tt.tag)
			if err == nil {
				t.Fatal("expected an error for an empty selection")
			}
			for _, want := range tt.contains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not name %q", err, want)
				}
			}
		})
	}
}

func TestWorktreeAddCmd_TagFlag(t *testing.T) {
	t.Cleanup(func() { flagWorktreeAddTags = nil })

	t.Run("single value accepted", func(t *testing.T) {
		flagWorktreeAddTags = []string{"backend"}
		got, err := worktreeAddTag()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "backend" {
			t.Errorf("tag = %q, want backend", got)
		}
	})

	t.Run("absent yields empty", func(t *testing.T) {
		flagWorktreeAddTags = nil
		got, err := worktreeAddTag()
		if err != nil || got != "" {
			t.Errorf("got (%q, %v), want (\"\", nil)", got, err)
		}
	})

	t.Run("repeated rejected", func(t *testing.T) {
		flagWorktreeAddTags = []string{"backend", "frontend"}
		_, err := worktreeAddTag()
		if err == nil {
			t.Fatal("expected an error when --tag is given twice")
		}
		if !strings.Contains(err.Error(), "--tag") {
			t.Errorf("error %q should name the --tag flag", err)
		}
	})
}
