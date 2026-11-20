package gitlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GitTimeout bounds every git call of the committer (SPEC-0008 R2).
const GitTimeout = 30 * time.Second

// allowedSubcommands are the only git subcommands memctl runs (SPEC-0008 R2).
var allowedSubcommands = map[string]bool{"rev-parse": true, "check-ignore": true, "add": true, "commit": true, "status": true, "config": true}

// Failure classes of a git call (SPEC-0008 R11).
const (
	classIndexLocked = "index_locked"
	classHookFailed  = "hook_failed"
	classTimeout     = "timeout"
	classGitError    = "git_error"
)

// gitError is a git call that ran and failed. Its text never reaches a log
// line: only the exit status and the class do (SPEC-0008 R11).
type gitError struct {
	sub    string
	status int // exit status, -1 when git did not exit normally
	class  string
	out    string // combined output, for classification only
}

func (e *gitError) Error() string {
	return fmt.Sprintf("git %s failed (%s, status %d)", e.sub, e.class, e.status)
}

// runGit runs "git -C <root> -c commit.gpgSign=false <sub> <args...>" with
// GIT_TERMINAL_PROMPT=0, GIT_LITERAL_PATHSPECS=1 except for check-ignore
// (a note path is never a pattern), the C locale, extra environment and an optional stdin, bounded
// by timeout (SPEC-0008 R2). It returns stdout. A git that
// cannot be run gives ErrGitMissing.
func runGit(ctx context.Context, root string, timeout time.Duration, env []string, stdin io.Reader, sub string, args ...string) ([]byte, error) {
	if !allowedSubcommands[sub] || (sub == "config" && (len(args) == 0 || args[0] != "--get")) {
		return nil, fmt.Errorf("gitlog: git %s is not allowed", sub)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	argv := append([]string{"-C", root, "-c", "commit.gpgSign=false", sub}, args...)
	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C", "LANGUAGE=C")
	if sub != "check-ignore" { // it reads paths literally and refuses the setting
		cmd.Env = append(cmd.Env, "GIT_LITERAL_PATHSPECS=1")
	}
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdin = stdin
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err == nil {
		return out.Bytes(), nil
	}
	var ee *exec.ExitError
	switch {
	case ctx.Err() == context.DeadlineExceeded:
		return nil, &gitError{sub: sub, status: -1, class: classTimeout}
	case errors.As(err, &ee):
		combined := out.String() + errb.String()
		return out.Bytes(), &gitError{sub: sub, status: ee.ExitCode(), class: classify(sub, combined), out: combined}
	default:
		return nil, ErrGitMissing
	}
}

// classify names the kind of a failed git call from its output (in the C
// locale): a held index.lock, a failing hook (a commit that fails without a
// message of git's own), or any other error.
func classify(sub, out string) string {
	if strings.Contains(out, "index.lock") {
		return classIndexLocked
	}
	if sub == "commit" && !gitMessage(out) {
		return classHookFailed
	}
	return classGitError
}

// gitMessage reports whether out holds a line that git itself writes on an
// error.
func gitMessage(out string) bool {
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "fatal: ") || strings.HasPrefix(l, "error: ") {
			return true
		}
	}
	return false
}
