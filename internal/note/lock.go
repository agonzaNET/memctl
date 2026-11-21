package note

import (
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Lock is the instance lock of a notes root: an exclusive flock on
// <root>/.memory/memctl.lock (SPEC-0006). The kernel releases it when the
// process ends; the file is never deleted (R7).
type Lock struct{ f *os.File }

// HeldError reports that another process holds the lock of Root and that no
// shared session with it started (SPEC-0006 R5, SPEC-0009 R5). PID is 0
// when the holder's PID cannot be read. Given is the root as the user gave
// it; when set and different from Root, the message names both
// (SPEC-0006 R5).
type HeldError struct {
	Root  string
	Given string
	PID   int
}

func (e *HeldError) Error() string {
	root := e.Root
	if e.Given != "" && e.Given != e.Root {
		root = e.Given + " (" + e.Root + ")"
	}
	const suffix = ", and it did not accept a shared session within 10 s"
	if e.PID <= 0 {
		return "root " + root + " is in use by another memctl process" + suffix
	}
	return "root " + root + " is in use by memctl (pid " + strconv.Itoa(e.PID) + ")" + suffix
}

// MemoryDir is the folder of memctl's own files in a notes root.
const MemoryDir = ".memory"

// AcquireLock takes the instance lock of the resolved root without blocking
// (SPEC-0006 R2–R5). It creates .memory/ and .memory/.gitignore when they are
// missing. When another process holds the lock it returns a *HeldError; any
// other error names the path and the operation (R6). The caller keeps the
// returned Lock reachable until the process exits.
func AcquireLock(root string, log *slog.Logger) (*Lock, error) {
	dir := filepath.Join(root, MemoryDir)
	if err := ensureMemoryDir(dir); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "memctl.lock")
	// os.OpenFile sets O_CLOEXEC on every file it opens (R3).
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, &HeldError{Root: root, PID: readPID(path)}
		}
		return nil, &fs.PathError{Op: "flock", Path: path, Err: err}
	}
	if err := writePID(f); err != nil {
		log.Warn("cannot write the PID to the lock file", slog.String("err", err.Error()))
	}
	return &Lock{f: f}, nil
}

// ensureMemoryDir creates dir with mode 0700 and its .gitignore with "*" and
// mode 0600 when they are missing; an existing .gitignore is left as it is
// (SPEC-0006 R2).
func ensureMemoryDir(dir string) error {
	err := os.Mkdir(dir, 0o700)
	switch {
	case err == nil:
		if err := os.Chmod(dir, 0o700); err != nil { // the umask may have narrowed it
			return err
		}
	case !errors.Is(err, fs.ErrExist):
		return err
	}
	gi := filepath.Join(dir, ".gitignore")
	f, err := os.OpenFile(gi, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	_, werr := f.WriteString("*\n")
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return &fs.PathError{Op: "write", Path: gi, Err: werr}
	}
	return nil
}

// writePID replaces the content of the lock file with this process's PID and
// a newline (SPEC-0006 R4).
func writePID(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	return err
}

// readPID returns the PID in the lock file, or 0 when it cannot be read.
func readPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// Close releases the lock and keeps the lock file (SPEC-0006 R7).
func (l *Lock) Close() error {
	if l == nil || l.f == nil {
		return nil
	}
	return l.f.Close()
}

// ResolveRoot makes root absolute with symlinks evaluated and checks that it
// is a readable directory (SPEC-0003 R1).
func ResolveRoot(root string) (string, error) { return resolveRoot(root) }
