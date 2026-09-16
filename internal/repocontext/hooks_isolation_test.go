package repocontext

import (
	"os"
	"testing"
)

// isolateGitHooks stops a developer's global git hooks from running against the
// throwaway repositories these tests create, mirroring the same helper in
// internal/git.
//
// Fixtures commit with scaffolding messages like "init". A global
// core.hooksPath — a conventional-commit validator, say — applies to every repo
// on the machine, including fixtures in a temp dir, and rejects those messages.
// Fixture setup then fails before a single assertion runs.
//
// GIT_CONFIG_* is used rather than a repo-local setting because it propagates
// to every git subprocess the tests spawn, wherever that repo lives.
func isolateGitHooks() {
	os.Setenv("GIT_CONFIG_COUNT", "1")
	os.Setenv("GIT_CONFIG_KEY_0", "core.hooksPath")
	os.Setenv("GIT_CONFIG_VALUE_0", "")
}

func TestMain(m *testing.M) {
	isolateGitHooks()
	os.Exit(m.Run())
}
