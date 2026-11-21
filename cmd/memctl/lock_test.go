package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/note"
)

// splitStderr separates the plain message lines of stderr from the JSON log
// lines.
func splitStderr(t *testing.T, s string) (plain []string, logs []map[string]any) {
	t.Helper()
	var js strings.Builder
	for _, l := range strings.Split(s, "\n") {
		switch {
		case l == "":
		case strings.HasPrefix(l, "{"):
			js.WriteString(l + "\n")
		default:
			plain = append(plain, l)
		}
	}
	return plain, jsonLines(t, js.String())
}

func errorLines(lines []map[string]any) []map[string]any {
	var out []map[string]any
	for _, l := range lines {
		if l["level"] == "ERROR" {
			out = append(out, l)
		}
	}
	return out
}

// assertRefused checks the refusal of SPEC-0006 R5: status 1, the message on
// stderr, one ERROR line with the same text, and no load.
func assertRefused(t *testing.T, r result, want string) {
	t.Helper()
	if r.code != 1 || r.stdout != "" {
		t.Fatalf("code %d stdout %q, want 1 and empty; stderr %s", r.code, r.stdout, r.stderr)
	}
	// stderr is a pipe, not a terminal: the message is in the ERROR line
	// only, so a client's log holds it once (SPEC-0006 R5).
	plain, logs := splitStderr(t, r.stderr)
	if len(plain) != 0 {
		t.Fatalf("plain lines %q on a pipe", plain)
	}
	errs := errorLines(logs)
	if len(errs) != 1 || errs[0]["msg"] != want {
		t.Fatalf("ERROR lines %v, want one with %q", errs, want)
	}
	if byMsg(logs, "notes loaded") != nil || byMsg(logs, "started") != nil {
		t.Fatalf("refused process loaded the notes: %v", logs)
	}
}

// label is the root as a refusal names it: as given, with the resolved root
// in parentheses when it differs (SPEC-0006 R5).
func label(t *testing.T, given string) string {
	t.Helper()
	if r := resolved(t, given); r != given {
		return given + " (" + r + ")"
	}
	return given
}

func resolved(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// startForwarder starts memctl mcp on a root another process owns, with stdin
// held open.
func startForwarder(t *testing.T, root string) (*exec.Cmd, io.WriteCloser, *syncBuffer) {
	t.Helper()
	cmd := exec.Command(binVersioned, "mcp", "--root", root)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	errb := new(syncBuffer)
	cmd.Stderr = errb
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})
	return cmd, stdin, errb
}

// startServer starts memctl mcp on root with stdin held open and waits for
// the started line.
func startServer(t *testing.T, root string) (*exec.Cmd, io.WriteCloser, *syncBuffer) {
	t.Helper()
	cmd := exec.Command(binVersioned, "mcp", "--root", root)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	errb := new(syncBuffer)
	cmd.Stderr = errb
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})
	waitFor(t, errb, `"msg":"started"`)
	return cmd, stdin, errb
}

// heldSuffix ends the refusal message of SPEC-0006 R5.
const heldSuffix = ", and it did not accept a shared session within 10 s"

// TestCLI_SPEC0006_R1_SPEC0009_R5_lockHeldBeforeLoad: the lock is taken before
// the load, and a second process started during the load forwards once the
// first listens.
func TestCLI_SPEC0006_R1_SPEC0009_R5_lockHeldBeforeLoad(t *testing.T) {
	root := notesRoot(t)
	var stderrAtLock string
	var errb syncBuffer
	var fwdErr *syncBuffer
	var fwdIn io.WriteCloser
	afterLock = func() {
		stderrAtLock = errb.String()
		_, fwdIn, fwdErr = startForwarder(t, root)
		time.Sleep(500 * time.Millisecond) // the forwarder retries meanwhile
	}
	t.Cleanup(func() { afterLock = nil })

	in, inW := io.Pipe()
	code := make(chan int, 1)
	go func() { code <- run([]string{"mcp", "--root", root}, in, io.Discard, &errb) }()
	waitFor(t, &errb, `"msg":"started"`)
	if strings.Contains(stderrAtLock, "notes loaded") || strings.Contains(stderrAtLock, `"started"`) {
		t.Fatalf("lock taken after the load: %s", stderrAtLock)
	}
	waitFor(t, fwdErr, `"msg":"forwarding"`)
	if strings.Contains(fwdErr.String(), "notes loaded") {
		t.Fatalf("the second process loaded the notes: %s", fwdErr.String())
	}
	fwdIn.Close()
	waitFor(t, fwdErr, `"msg":"stopped"`)
	inW.Close()
	if c := <-code; c != 0 {
		t.Fatalf("first process: code %d, stderr %s", c, errb.String())
	}
}

func TestCLI_SPEC0006_R3_R4_secondProcessTakesNoLock(t *testing.T) {
	root := notesRoot(t)
	first, _, _ := startServer(t, root)
	second, _, errb := startForwarder(t, root)
	waitFor(t, errb, `"msg":"forwarding"`)
	b, _ := os.ReadFile(filepath.Join(root, ".memory", "memctl.lock"))
	if string(b) != strconv.Itoa(first.Process.Pid)+"\n" {
		t.Fatalf("lock file %q, want the first PID", b)
	}
	if strings.Contains(errb.String(), "notes loaded") {
		t.Fatalf("second process (pid %d) loaded the notes: %s", second.Process.Pid, errb.String())
	}
}

// holdLock takes the instance lock of root from the test process, so a
// memctl started on root finds it held by a process without a socket.
func holdLock(t *testing.T, root string) {
	t.Helper()
	l, err := note.AcquireLock(resolved(t, root), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
}

// runIn runs memctl in this process with a short forward timeout.
func runIn(t *testing.T, timeout time.Duration, args ...string) result {
	t.Helper()
	old := forwardTimeout
	forwardTimeout = timeout
	t.Cleanup(func() { forwardTimeout = old })
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(""), &out, &errb)
	return result{code, out.String(), errb.String()}
}

func TestCLI_SPEC0006_R5_SPEC0009_R5_refusedWithoutSharedSession(t *testing.T) {
	root := notesRoot(t)
	holdLock(t, root)
	start := time.Now()
	r := runIn(t, time.Second, "mcp", "--root", root)
	if d := time.Since(start); d < time.Second {
		t.Fatalf("refused after %v, before the forward timeout", d)
	}
	assertRefused(t, r, "root "+label(t, root)+" is in use by memctl (pid "+strconv.Itoa(os.Getpid())+")"+heldSuffix)

	// A root given through a symlink names the given path and the resolved one.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	r = runIn(t, 200*time.Millisecond, "mcp", "--root", link)
	assertRefused(t, r, "root "+link+" ("+resolved(t, root)+") is in use by memctl (pid "+strconv.Itoa(os.Getpid())+")"+heldSuffix)

	// An empty lock file: the message names no PID.
	if err := os.WriteFile(filepath.Join(root, ".memory", "memctl.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r = runIn(t, 200*time.Millisecond, "mcp", "--root", root)
	assertRefused(t, r, "root "+label(t, root)+" is in use by another memctl process"+heldSuffix)
}

// TestCLI_SPEC0009_R5_realTimeout: a spawned process waits the full 10 s.
func TestCLI_SPEC0009_R5_realTimeout(t *testing.T) {
	t.Parallel()
	root := notesRoot(t)
	holdLock(t, root)
	start := time.Now()
	r := runBin(t, binVersioned, nil, "mcp", "--root", root)
	if d := time.Since(start); d < 10*time.Second || d > 13*time.Second {
		t.Fatalf("refused after %v, want about 10 s", d)
	}
	assertRefused(t, r, "root "+label(t, root)+" is in use by memctl (pid "+strconv.Itoa(os.Getpid())+")"+heldSuffix)
}

// TestCLI_SPEC0009_R5_takeOverFreedLock: a forwarder whose owner goes away
// during the retry takes the lock and serves as the owner.
func TestCLI_SPEC0009_R5_takeOverFreedLock(t *testing.T) {
	root := notesRoot(t)
	l, err := note.AcquireLock(resolved(t, root), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	old := forwardTimeout
	forwardTimeout = 5 * time.Second
	t.Cleanup(func() { forwardTimeout = old })
	var errb syncBuffer
	in, inW := io.Pipe()
	code := make(chan int, 1)
	go func() { code <- run([]string{"mcp", "--root", root}, in, io.Discard, &errb) }()
	time.Sleep(300 * time.Millisecond)
	if strings.Contains(errb.String(), `"started"`) {
		t.Fatalf("started while the lock was held: %s", errb.String())
	}
	l.Close()
	waitFor(t, &errb, `"msg":"started"`)
	inW.Close()
	if c := <-code; c != 0 {
		t.Fatalf("code %d: %s", c, errb.String())
	}
}

func TestCLI_SPEC0006_R6_readOnlyRoot(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	root := notesRoot(t)
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0o755) })
	r := runBin(t, binVersioned, nil, "mcp", "--root", root)
	if r.code != 1 {
		t.Fatalf("code %d, want 1; stderr %s", r.code, r.stderr)
	}
	errs := errorLines(jsonLines(t, r.stderr))
	if len(errs) != 1 {
		t.Fatalf("want one ERROR line: %s", r.stderr)
	}
	line := r.stderr
	if !strings.Contains(line, filepath.Join(resolved(t, root), ".memory")) || !strings.Contains(line, "mkdir") {
		t.Fatalf("ERROR line does not name the path and the operation: %s", line)
	}
}

func TestCLI_SPEC0006_R7_R8_retakeAfterKill(t *testing.T) {
	root := notesRoot(t)
	lockFile := filepath.Join(root, ".memory", "memctl.lock")

	first, _, _ := startServer(t, root)
	if err := first.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	first.Wait()
	if _, err := os.Stat(lockFile); err != nil {
		t.Fatalf("lock file gone after SIGKILL: %v", err)
	}

	start := time.Now()
	second, stdin, errb := startServer(t, root)
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("lock taken after %v", d)
	}
	if b, _ := os.ReadFile(lockFile); string(b) != strconv.Itoa(second.Process.Pid)+"\n" {
		t.Fatalf("lock file %q, want the new PID", b)
	}
	stdin.Close()
	if err := second.Wait(); err != nil {
		t.Fatalf("second exit: %v; %s", err, errb.String())
	}
	if _, err := os.Stat(lockFile); err != nil {
		t.Fatalf("lock file gone after a clean exit: %v", err)
	}
	r := runBin(t, binVersioned, nil, "mcp", "--root", root)
	if r.code != 0 {
		t.Fatalf("third start after clean exit: %+v", r)
	}
	if _, err := os.Stat(lockFile); err != nil {
		t.Fatalf("lock file gone: %v", err)
	}
}
