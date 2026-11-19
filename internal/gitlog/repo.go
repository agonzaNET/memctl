// Package gitlog runs the git command for memctl: the repository test and
// the committer of tool writes (SPEC-0005 R7, SPEC-0008).
package gitlog

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ErrGitMissing is returned when the git command cannot be run.
var ErrGitMissing = errors.New("gitlog: the git command cannot be run")

// RepoTimeout bounds the repository test (SPEC-0005 R7, SPEC-0008 R1).
const RepoTimeout = 5 * time.Second

// IsRepo reports whether root is inside a git work tree (SPEC-0005 R7,
// SPEC-0008 R1).
func IsRepo(ctx context.Context, root string) (bool, error) {
	out, err := runGit(ctx, root, RepoTimeout, nil, nil, "rev-parse", "--is-inside-work-tree")
	var ge *gitError
	switch {
	case err == nil:
		return strings.TrimSpace(string(out)) == "true", nil
	case errors.As(err, &ge) && ge.class != classTimeout:
		return false, nil // git ran and said this is not a repository
	default:
		return false, ErrGitMissing
	}
}
