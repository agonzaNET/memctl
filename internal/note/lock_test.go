package note_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/agonzaNET/memctl/internal/note"
)

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

func TestLock_SPEC0006_R2_memoryFolder(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	l, err := note.AcquireLock(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	mem := filepath.Join(root, ".memory")
	if m := mode(t, mem); m != 0o700 {
		t.Fatalf(".memory mode %v, want 0700", m)
	}
	gi := filepath.Join(mem, ".gitignore")
	if b, _ := os.ReadFile(gi); string(b) != "*\n" {
		t.Fatalf(".gitignore %q", b)
	}
	if m := mode(t, gi); m != 0o600 {
		t.Fatalf(".gitignore mode %v, want 0600", m)
	}
}

func TestLock_SPEC0006_R2_existingGitignoreUnchanged(t *testing.T) {
	root := tree(t, map[string]string{".memory/.gitignore": "keep-me\n"})
	l, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if b, _ := os.ReadFile(filepath.Join(root, ".memory", ".gitignore")); string(b) != "keep-me\n" {
		t.Fatalf(".gitignore changed to %q", b)
	}
}

func TestLock_SPEC0006_R2_gitStatusClean(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	l, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	out, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=all").CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Fatalf("git status: %v %q", err, out)
	}
}

func TestLock_SPEC0006_R3_R4_fileModePIDAndCloexec(t *testing.T) {
	root := t.TempDir()
	l, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	lf := filepath.Join(root, ".memory", "memctl.lock")
	if m := mode(t, lf); m != 0o600 {
		t.Fatalf("lock file mode %v, want 0600", m)
	}
	if b, _ := os.ReadFile(lf); string(b) != strconv.Itoa(os.Getpid())+"\n" {
		t.Fatalf("lock file content %q", b)
	}
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, note.LockFd(l), syscall.F_GETFD, 0)
	if errno != 0 || flags&syscall.FD_CLOEXEC == 0 {
		t.Fatalf("close-on-exec not set: flags %d errno %v", flags, errno)
	}
}

func TestLock_SPEC0006_R4_truncatesOldContent(t *testing.T) {
	root := tree(t, map[string]string{".memory/memctl.lock": "999999999999\nleftover\n"})
	l, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if b, _ := os.ReadFile(filepath.Join(root, ".memory", "memctl.lock")); string(b) != strconv.Itoa(os.Getpid())+"\n" {
		t.Fatalf("lock file content %q", b)
	}
}

func TestLock_SPEC0006_R3_R5_heldByAnother(t *testing.T) {
	root := t.TempDir()
	first, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	_, err = note.AcquireLock(root, logger(new(bytes.Buffer)))
	var held *note.HeldError
	if !errors.As(err, &held) {
		t.Fatalf("second lock: %v, want HeldError", err)
	}
	want := "root " + root + " is in use by memctl (pid " + strconv.Itoa(os.Getpid()) + "), and it did not accept a shared session within 10 s"
	if held.PID != os.Getpid() || err.Error() != want {
		t.Fatalf("got %q (pid %d), want %q", err.Error(), held.PID, want)
	}

	// The holder's PID cannot be read: the message names no PID.
	if err := os.WriteFile(filepath.Join(root, ".memory", "memctl.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = note.AcquireLock(root, logger(new(bytes.Buffer)))
	if !errors.As(err, &held) || held.PID != 0 || err.Error() != "root "+root+" is in use by another memctl process, and it did not accept a shared session within 10 s" {
		t.Fatalf("empty lock file: %v", err)
	}
}

func TestLock_SPEC0006_R6_cannotCreate(t *testing.T) {
	skipIfRoot(t)
	cases := []struct {
		name  string
		setup func(root string)
		path  string
		op    string
	}{
		{"read-only root", func(root string) {}, ".memory", "mkdir"},
		{"read-only .memory", func(root string) {
			os.Chmod(root, 0o755)
			os.Mkdir(filepath.Join(root, ".memory"), 0o700)
			os.WriteFile(filepath.Join(root, ".memory", ".gitignore"), []byte("*\n"), 0o600)
			os.Chmod(filepath.Join(root, ".memory"), 0o500)
		}, "memctl.lock", "open"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			os.Chmod(root, 0o555)
			c.setup(root)
			t.Cleanup(func() {
				os.Chmod(filepath.Join(root, ".memory"), 0o700)
				os.Chmod(root, 0o755)
			})
			_, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
			if err == nil {
				t.Fatal("lock taken on a read-only root")
			}
			var held *note.HeldError
			if errors.As(err, &held) {
				t.Fatalf("got HeldError: %v", err)
			}
			if !strings.Contains(err.Error(), c.path) || !strings.Contains(err.Error(), c.op) {
				t.Fatalf("error %q does not name %q and %q", err, c.path, c.op)
			}
		})
	}
}

func TestLock_SPEC0006_R7_R8_closeKeepsFileAndReleases(t *testing.T) {
	root := t.TempDir()
	l, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	lf := filepath.Join(root, ".memory", "memctl.lock")
	if _, err := os.Stat(lf); err != nil {
		t.Fatalf("lock file gone after release: %v", err)
	}
	l2, err := note.AcquireLock(root, logger(new(bytes.Buffer)))
	if err != nil {
		t.Fatalf("lock not free after release: %v", err)
	}
	l2.Close()
	if _, err := os.Stat(lf); err != nil {
		t.Fatalf("lock file gone: %v", err)
	}
}
