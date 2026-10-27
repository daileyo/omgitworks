package main

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

// completeAssign runs the assign completion function and returns the sorted
// candidates with the directive.
func completeAssign(t *testing.T, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	t.Helper()
	names, directive := completeUserAssignArgs(userAssignCmd, args, toComplete)
	slices.Sort(names)
	return names, directive
}

func TestCompleteUserAssign_ProfilesWhenResolved(t *testing.T) {
	paths := setupAssignFixture(t, "alpha", "beta")
	chdirForTest(t, paths["alpha"])

	names, directive := completeAssign(t, nil, "")
	if want := []string{"personal", "work"}; !slices.Equal(names, want) {
		t.Errorf("inside a tracked repository the first argument completed to %v, want %v", names, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteUserAssign_ReposWhenNotResolved(t *testing.T) {
	setupAssignFixture(t, "alpha", "beta")
	chdirForTest(t, t.TempDir())

	names, directive := completeAssign(t, nil, "")
	if want := []string{"alpha", "beta"}; !slices.Equal(names, want) {
		t.Errorf("outside a tracked repository the first argument completed to %v, want %v", names, want)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}

func TestCompleteUserAssign_ProfilesForSecondArg(t *testing.T) {
	setupAssignFixture(t, "alpha", "beta")
	chdirForTest(t, t.TempDir())

	names, _ := completeAssign(t, []string{"alpha"}, "")
	if want := []string{"personal", "work"}; !slices.Equal(names, want) {
		t.Errorf("second argument completed to %v, want %v", names, want)
	}

	// The prefix filter still applies.
	names, _ = completeAssign(t, []string{"alpha"}, "w")
	if want := []string{"work"}; !slices.Equal(names, want) {
		t.Errorf("second argument with prefix 'w' completed to %v, want %v", names, want)
	}
}

func TestCompleteUserAssign_NothingAfterTwoArgs(t *testing.T) {
	paths := setupAssignFixture(t, "alpha")
	chdirForTest(t, paths["alpha"])

	names, directive := completeAssign(t, []string{"alpha", "work"}, "")
	if len(names) != 0 {
		t.Errorf("third position completed to %v, want nothing", names)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
}
