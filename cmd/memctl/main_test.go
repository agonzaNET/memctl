package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var (
	binVersioned string // built with -ldflags -X main.version=9.8.7-test
	binPlain     string // built without -ldflags
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "memctl-bin")
	if err != nil {
		panic(err)
	}
	binVersioned = filepath.Join(dir, "memctl")
	binPlain = filepath.Join(dir, "memctl-plain")
	for _, b := range []struct {
		out   string
		flags []string
	}{
		{binVersioned, []string{"-ldflags", "-X main.version=9.8.7-test"}},
		{binPlain, nil},
	} {
		args := append([]string{"build", "-o", b.out}, b.flags...)
		args = append(args, ".")
		cmd := exec.Command("go", args...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			panic(err)
		}
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	code           int
	stdout, stderr string
}

func runBin(t *testing.T, bin string, env []string, args ...string) result {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = strings.NewReader("")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return result{code, out.String(), errb.String()}
}

// tempRoot returns a new folder under /tmp, removed after the test. The folder
// of t.TempDir can make the socket path of a notes root too long to share.
func tempRoot(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "mct")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

func notesRoot(t *testing.T) string {
	t.Helper()
	root := tempRoot(t)
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestCLI_SPEC0004_R1_usageErrors(t *testing.T) {
	root := notesRoot(t)
	cases := []struct {
		name    string
		args    []string
		mention string
	}{
		{"no arguments", nil, "usage"},
		{"unknown command", []string{"frobnicate"}, "usage"},
		{"unknown flag", []string{"mcp", "--root", root, "--nope"}, "nope"},
		{"bad log level", []string{"mcp", "--root", root, "--log-level", "loud"}, "--log-level"},
		{"positional argument", []string{"mcp", "--root", root, "extra"}, "usage"},
		{"version with flags", []string{"version", "--x"}, "usage"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := runBin(t, binVersioned, nil, c.args...)
			if r.code != 2 || r.stdout != "" {
				t.Fatalf("code %d stdout %q, want 2 and empty", r.code, r.stdout)
			}
			if !strings.Contains(strings.ToLower(r.stderr), strings.ToLower(c.mention)) {
				t.Fatalf("stderr %q does not mention %q", r.stderr, c.mention)
			}
			if strings.Contains(r.stderr, "loud") {
				t.Fatalf("stderr echoes the bad value: %q", r.stderr)
			}
		})
	}
}

func TestCLI_SPEC0004_R2_rootErrors(t *testing.T) {
	r := runBin(t, binVersioned, []string{"MEMCTL_ROOT="}, "mcp")
	if r.code != 2 || !strings.Contains(r.stderr, "--root") {
		t.Fatalf("no root: %+v", r)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	r = runBin(t, binVersioned, nil, "mcp", "--root", missing)
	if r.code != 1 {
		t.Fatalf("missing root: %+v", r)
	}
	lines := jsonLines(t, r.stderr)
	if len(lines) != 1 || lines[0]["level"] != "ERROR" {
		t.Fatalf("want one ERROR line, got %v", lines)
	}
}

func jsonLines(t *testing.T, s string) []map[string]any {
	t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		if sc.Text() == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("stderr line is not JSON: %q", sc.Text())
		}
		out = append(out, m)
	}
	return out
}

func byMsg(lines []map[string]any, msg string) map[string]any {
	for _, l := range lines {
		if l["msg"] == msg {
			return l
		}
	}
	return nil
}

func TestCLI_SPEC0004_R3_R4_startedAndCleanStdout(t *testing.T) {
	root := notesRoot(t)
	r := runBin(t, binVersioned, nil, "mcp", "--root", root, "--log-level", "debug")
	if r.code != 0 || r.stdout != "" {
		t.Fatalf("code %d stdout %q", r.code, r.stdout)
	}
	lines := jsonLines(t, r.stderr)
	started := byMsg(lines, "started")
	if started == nil || started["version"] != "9.8.7-test" || started["log_level"] != "debug" || started["notes"] != 1.0 || started["level"] != "INFO" {
		t.Fatalf("started line %v in %v", started, lines)
	}
	if byMsg(lines, "notes loaded") == nil {
		t.Fatalf("no notes loaded line: %v", lines)
	}
}

func TestCLI_SPEC0004_R3_envAndFlagPrecedence(t *testing.T) {
	root := notesRoot(t)
	r := runBin(t, binVersioned, []string{"MEMCTL_ROOT=" + root, "MEMCTL_LOG_LEVEL=warn"}, "mcp")
	if r.code != 0 || byMsg(jsonLines(t, r.stderr), "started") != nil {
		t.Fatalf("env root and warn level: %+v", r)
	}
	r = runBin(t, binVersioned, []string{"MEMCTL_ROOT=/does/not/exist", "MEMCTL_LOG_LEVEL=warn"}, "mcp", "--root", root, "--log-level", "info")
	if r.code != 0 || byMsg(jsonLines(t, r.stderr), "started") == nil {
		t.Fatalf("flags must win over env: %+v", r)
	}
	r = runBin(t, binVersioned, []string{"MEMCTL_LOG_LEVEL=loud"}, "mcp", "--root", root)
	if r.code != 2 || !strings.Contains(r.stderr, "MEMCTL_LOG_LEVEL") {
		t.Fatalf("bad env level: %+v", r)
	}
}

func TestCLI_SPEC0004_R6_shutdown(t *testing.T) {
	root := notesRoot(t)
	for _, how := range []string{"stdin EOF", "SIGTERM", "SIGINT"} {
		t.Run(how, func(t *testing.T) {
			cmd := exec.Command(binVersioned, "mcp", "--root", root)
			stdin, _ := cmd.StdinPipe()
			var errb syncBuffer
			cmd.Stderr = &errb
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			waitFor(t, &errb, `"msg":"started"`)
			switch how {
			case "stdin EOF":
				stdin.Close()
			case "SIGTERM":
				cmd.Process.Signal(syscall.SIGTERM)
			case "SIGINT":
				cmd.Process.Signal(syscall.SIGINT)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("exit: %v, stderr %s", err, errb.String())
				}
			case <-time.After(5 * time.Second):
				cmd.Process.Kill()
				t.Fatal("did not stop")
			}
			if byMsg(jsonLines(t, errb.String()), "stopped") == nil {
				t.Fatalf("no stopped line: %s", errb.String())
			}
		})
	}
}

// syncBuffer is a bytes.Buffer that a running process and the test can share.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitFor(t *testing.T, buf *syncBuffer, s string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), s) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %q in %q", s, buf.String())
}

func TestCLI_SPEC0004_R26_version(t *testing.T) {
	r := runBin(t, binVersioned, nil, "version")
	if r.code != 0 || r.stdout != "memctl 9.8.7-test\n" || r.stderr != "" {
		t.Fatalf("versioned: %+v", r)
	}
	r = runBin(t, binPlain, nil, "version")
	if r.code != 0 || r.stdout != "memctl dev\n" {
		t.Fatalf("plain: %+v", r)
	}
}
