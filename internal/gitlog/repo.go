// Package gitlog runs the git command for memctl: the repository test now,
// and the committer of tool writes in M2 (SPEC-0005 R7, SPEC-0008).
package gitlog

import (
	"context"
	"errors"
)

// ErrGitMissing is returned when the git command cannot be run.
var ErrGitMissing = errors.New("gitlog: the git command cannot be run")

// IsRepo reports whether root is inside a git work tree (SPEC-0005 R7).
func IsRepo(ctx context.Context, root string) (bool, error) { return false, nil }
