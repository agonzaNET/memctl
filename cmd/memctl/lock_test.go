package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
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

func TestCLI_SPEC0006_R1_lockHeldBeforeLoad(t *testing.T) {
	root := notesRoot(t)
	var during result
	var stderrAtLock string
	var errb bytes.Buffer
	afterLock = func() {
		stderrAtLock = errb.String()
		during = runBin(t, binVersioned, nil, "mcp", "--root", root)
	}
	t.Cleanup(func() { afterLock = nil })

	code := run([]string{"mcp", "--root", root}, strings.NewReader(""), io.Discard, &errb)
	if code != 0 {
		t.Fatalf("first process: code %d, stderr %s", code, errb.String())
	}
	if strings.Contains(stderrAtLock, "notes loaded") || strings.Contains(stderrAtLock, `"started"`) {
		t.Fatalf("lock taken after the load: %s", stderrAtLock)
	}
	assertRefused(t, during, "root "+label(t, root)+" is in use by memctl (pid "+strconv.Itoa(os.Getpid())+")")
}

func TestCLI_SPEC0006_R3_R4_R5_secondProcessRefused(t *testing.T) {
	root := notesRoot(t)
	first, _, _ := startServer(t, root)
	r := runBin(t, binVersioned, nil, "mcp", "--root", root)
	assertRefused(t, r, "root "+label(t, root)+" is in use by memctl (pid "+strconv.Itoa(first.Process.Pid)+")")

	// A root given through a symlink names the given path and the resolved one.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	r = runBin(t, binVersioned, nil, "mcp", "--root", link)
	assertRefused(t, r, "root "+link+" ("+resolved(t, root)+") is in use by memctl (pid "+strconv.Itoa(first.Process.Pid)+")")

	// An empty lock file: the message names no PID.
	if err := os.WriteFile(filepath.Join(root, ".memory", "memctl.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r = runBin(t, binVersioned, nil, "mcp", "--root", root)
	assertRefused(t, r, "root "+label(t, root)+" is in use by another memctl process")
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
