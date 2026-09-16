package git

import "strings"

// Toplevel returns the root of the git checkout enclosing dir, as reported by
// "git rev-parse --show-toplevel". The directory need not be the checkout root:
// git performs the upward walk itself, so subdirectories resolve to the same
// answer, and git's own boundary rules decide where a nested repository or a
// submodule starts.
//
// The returned path comes straight from git and has not been passed through
// ResolvePath. Callers comparing it against stored paths must resolve both
// sides. The error is git's own; callers that need a friendlier message are
// expected to replace it rather than wrap it.
func Toplevel(dir string) (string, error) {
	return gitCommand(dir, "rev-parse", "--show-toplevel")
}

// ListBranches returns the local branch names of the repository at repoPath,
// in git's own ordering. A repository with no commits yet has no branches and
// yields an empty slice rather than an error.
func ListBranches(repoPath string) ([]string, error) {
	output, err := gitCommand(repoPath, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	var branches []string
	for _, line := range strings.Split(output, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			branches = append(branches, name)
		}
	}
	return branches, nil
}
