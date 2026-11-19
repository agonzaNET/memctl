// Package gitlog runs the git command for memctl: the repository test now,
// and the committer of tool writes in M2 (SPEC-0005 R7, SPEC-0008).
package gitlog

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ErrGitMissing is returned when the git command cannot be run.
var ErrGitMissing = errors.New("gitlog: the git command cannot be run")

// RepoTimeout bounds the repository test (SPEC-0005 R7).
const RepoTimeout = 5 * time.Second

// IsRepo reports whether root is inside a git work tree (SPEC-0005 R7).
func IsRepo(ctx context.Context, root string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, RepoTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--is-inside-work-tree")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return strings.TrimSpace(out.String()) == "true", nil
	case errors.As(err, &ee) && ctx.Err() == nil:
		return false, nil // git ran and said this is not a repository
	default:
		return false, ErrGitMissing
	}
}
