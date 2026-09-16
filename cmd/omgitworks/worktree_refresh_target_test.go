package main

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/daileyo/omgitworks/internal/config"
	"github.com/daileyo/omgitworks/internal/repocontext"
)

// resetRefreshFlags clears refresh's package-level flag state after a test.
func resetRefreshFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		flagWorktreeRefreshTags = nil
		flagWorktreeRefreshDryRun = false
	})
}

// refreshTargetNames returns the sorted names selectWorktreeRefreshTargets picks.
func refreshTargetNames(t *testing.T, repoArg, tag string) ([]string, error) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	repos, err := selectWorktreeRefreshTargets(cfg, repoArg, tag)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.Name)
	}
	slices.Sort(names)
	return names, nil
}

// refreshWith runs the command entry point with a tag flag, as cobra would.
func refreshWith(t *testing.T, args []string, tags ...string) (string, error) {
	t.Helper()
	resetRefreshFlags(t)
	flagWorktreeRefreshTags = tags
	var buf bytes.Buffer
	err := runWorktreeRefreshCommand(args, &buf)
	return buf.String(), err
}

func TestWorktreeRefreshTargets_EachForm(t *testing.T) {
	paths := standardFixture(t)
	// Every case runs from inside web-ui. A no-argument refresh must still
	// select every repository: that is exactly where add and remove's precedence
	// table would wrongly resolve to the current repository instead.
	chdirForTest(t, paths["web-ui"])

	tests := []struct {
		name    string
		repoArg string
		tag     string
		want    []string
	}{
		{"no argument selects every repo, even inside one", "", "",
			[]string{"api-core", "api-edge", "api-legacy", "web-ui"}},
		{"name pattern selects every match without an ambiguity error", "api", "",
			[]string{"api-core", "api-edge", "api-legacy"}},
		{"dot selects the current repo", ".", "", []string{"web-ui"}},
		{"tag selects every tagged repo", "", "backend", []string{"api-core", "api-edge"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := refreshTargetNames(t, tt.repoArg, tt.tag)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("selected %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWorktreeRefreshTargets_NameAndTagAreAnded(t *testing.T) {
	standardFixture(t)

	got, err := refreshTargetNames(t, "api", "backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// api-legacy matches the name but carries no tag; OR semantics would include it.
	if want := []string{"api-core", "api-edge"}; !slices.Equal(got, want) {
		t.Errorf("selected %v, want %v", got, want)
	}

	// web-ui carries a tag, but not this one, and does not match the name either.
	if _, err := refreshTargetNames(t, "web", "backend"); err == nil {
		t.Error("a name and a tag with no common repository must not select anything")
	}
}

func TestWorktreeRefreshTargets_DotAndTagAreAnded(t *testing.T) {
	paths := standardFixture(t)
	chdirForTest(t, paths["api-core"])

	got, err := refreshTargetNames(t, ".", "backend")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"api-core"}; !slices.Equal(got, want) {
		t.Errorf("selected %v, want %v", got, want)
	}
}

// TestWorktreeRefreshTargets_SingleMatchTagIsBulk guards against deriving
// behavior from how many repositories matched. Specs 26 and 27 both shipped
// that bug: a tag matching one repository took a different path from a tag
// matching several. Here a failure must be reported identically either way.
func TestWorktreeRefreshTargets_SingleMatchTagIsBulk(t *testing.T) {
	paths := standardFixture(t)
	breakRepo(t, paths["web-ui"])
	breakRepo(t, paths["api-core"])

	single, singleErr := refreshWith(t, nil, "frontend") // matches web-ui only
	multi, multiErr := refreshWith(t, nil, "backend")    // matches api-core and api-edge

	for name, err := range map[string]error{"single-match": singleErr, "multi-match": multiErr} {
		if !errors.Is(err, errPartialFailure) {
			t.Errorf("%s tag: expected the bulk partial-failure result, got %v", name, err)
		}
	}
	// Same shape either way: summary line, then the error list.
	if !strings.Contains(single, "\nRefreshed 0 repositories, 0 changed, 1 failed\n\n1 error:\n  web-ui: ") {
		t.Errorf("single-match output is not in the bulk failure format:\n%q", single)
	}
	if !strings.Contains(multi, "\nRefreshed 1 repository, 0 changed, 1 failed\n\n1 error:\n  api-core: ") {
		t.Errorf("multi-match output is not in the bulk failure format:\n%q", multi)
	}
}

func TestWorktreeRefreshTargets_RepeatedTagRejected(t *testing.T) {
	standardFixture(t)

	_, err := refreshWith(t, nil, "backend", "frontend")

	if err == nil || !strings.Contains(err.Error(), "--tag accepts a single value, but was given 2 times") {
		t.Errorf("expected the single-value error, got %v", err)
	}
}

func TestWorktreeRefreshTargets_UnmatchedFilters(t *testing.T) {
	paths := standardFixture(t)

	tests := []struct {
		name     string
		cwd      string
		repoArg  string
		tag      string
		mentions []string
	}{
		{"unmatched name", "", "nope", "", []string{"'nope'"}},
		{"unmatched tag", "", "", "nope", []string{"'nope'"}},
		{"unmatched combination", "", "api", "frontend", []string{"'api'", "'frontend'"}},
		{"current repo lacks the tag", paths["web-ui"], ".", "backend", []string{"'web-ui'", "'backend'"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cwd != "" {
				chdirForTest(t, tt.cwd)
			}
			var args []string
			if tt.repoArg != "" {
				args = []string{tt.repoArg}
			}
			var tags []string
			if tt.tag != "" {
				tags = []string{tt.tag}
			}

			_, err := refreshWith(t, args, tags...)

			if err == nil {
				t.Fatal("expected an error so the command exits non-zero")
			}
			if errors.Is(err, errPartialFailure) {
				t.Fatal("an unmatched filter is a usage error, not a partial failure")
			}
			for _, want := range tt.mentions {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q should name %s", err, want)
				}
			}
		})
	}
}

func TestWorktreeRefreshTargets_DotOutsideRepo(t *testing.T) {
	standardFixture(t)
	chdirForTest(t, t.TempDir())

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	_, resolverErr := repocontext.ResolveCurrent(cfg)
	if resolverErr == nil {
		t.Fatal("fixture: the temp directory should not resolve to a tracked repository")
	}

	out, err := refreshWith(t, []string{"."})

	if err == nil {
		t.Fatal("'.' outside a tracked repository must fail, not refresh everything")
	}
	if err.Error() != resolverErr.Error() {
		t.Errorf("error = %q, want the resolver's error %q unchanged", err, resolverErr)
	}
	if out != "" {
		t.Errorf("nothing should run, got output:\n%s", out)
	}
}

func TestWorktreeRefreshCompletion_OffersDot(t *testing.T) {
	paths := standardFixture(t)
	complete := worktreeRefreshCmd.ValidArgsFunction
	if complete == nil {
		t.Fatal("refresh has no ValidArgsFunction registered")
	}

	t.Run("dot offered when resolved", func(t *testing.T) {
		chdirForTest(t, paths["web-ui"])
		got, _ := complete(worktreeRefreshCmd, nil, "")
		if !slices.Contains(got, ".") || !slices.Contains(got, "api-core") {
			t.Errorf("completions %v, want '.' alongside repository names", got)
		}
	})

	t.Run("dot withheld when unresolved", func(t *testing.T) {
		chdirForTest(t, t.TempDir())
		got, _ := complete(worktreeRefreshCmd, nil, "")
		if slices.Contains(got, ".") {
			t.Errorf("completions %v offer '.' where it would fail", got)
		}
	})

	t.Run("tag flag completes tags in use", func(t *testing.T) {
		completeTag, ok := worktreeRefreshCmd.GetFlagCompletionFunc("tag")
		if !ok {
			t.Fatal("--tag has no completion registered")
		}
		got, _ := completeTag(worktreeRefreshCmd, nil, "")
		slices.Sort(got)
		if want := []string{"backend", "frontend"}; !slices.Equal(got, want) {
			t.Errorf("tag completions %v, want %v", got, want)
		}
	})
}
