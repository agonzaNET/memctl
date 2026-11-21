package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/share"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func socketOf(root string) string {
	r, err := filepath.EvalSymlinks(root)
	if err != nil {
		r = root
	}
	return filepath.Join(r, ".memory", "memctl.sock")
}

// rawSession does the forwarder handshake by hand on the owner's socket and
// connects an MCP client over the connection.
func rawSession(t *testing.T, root, author string) *mcp.ClientSession {
	t.Helper()
	conn, err := net.Dial("unix", socketOf(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := share.WriteLine(conn, share.NewHello("9.8.7-test", os.Getpid(), author)); err != nil {
		t.Fatal(err)
	}
	a, err := share.ReadAnswer(conn)
	if err != nil || !a.OK {
		t.Fatalf("handshake: %+v %v", a, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "raw", Version: "0"}, nil).
		Connect(ctx, &mcp.IOTransport{Reader: conn, Writer: conn}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestE2E_SPEC0009_R1_R3_socketAfterStarted(t *testing.T) {
	root := notesRoot(t)
	// A stale file at the socket path is replaced.
	if err := os.MkdirAll(filepath.Join(root, ".memory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socketOf(root), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, errb := startServer(t, root)
	fi, err := os.Lstat(socketOf(root))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode()&os.ModeSocket == 0 || fi.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode %v, want a socket with 0600", fi.Mode())
	}
	started := byMsg(jsonLines(t, errb.String()), "started")
	if started["sharing"] != true {
		t.Fatalf("started line %v, want sharing true", started)
	}
}

func TestE2E_SPEC0009_R1_noAnswerBeforeLoad(t *testing.T) {
	root := notesRoot(t)
	var dialErr error
	afterLock = func() {
		_, dialErr = net.DialTimeout("unix", socketOf(root), time.Second)
	}
	t.Cleanup(func() { afterLock = nil })
	if code := run([]string{"mcp", "--root", root}, strings.NewReader(""), io.Discard, io.Discard); code != 0 {
		t.Fatalf("code %d", code)
	}
	if dialErr == nil {
		t.Fatal("the socket accepted a connection before the load")
	}
}

func TestE2E_SPEC0009_R2_sharingOffForLongPath(t *testing.T) {
	base := t.TempDir()
	long := filepath.Join(base, strings.Repeat("d", share.MaxPathLen()))
	if err := os.MkdirAll(long, 0o755); err != nil {
		t.Skipf("cannot create a long path here: %v", err)
	}
	if err := os.WriteFile(filepath.Join(long, "a.md"), []byte("---\ntitle: A\n---\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := connect(t, long)
	if got := s.call(t, "note_read", map[string]any{"ref": "A"}); !strings.Contains(got, "Read \"A\"") {
		t.Fatalf("note_read over stdio: %s", got)
	}
	lines := jsonLines(t, s.stderr.String())
	warn := byMsg(lines, "sharing is off")
	if warn == nil || warn["level"] != "WARN" || warn["reason"] == nil || warn["socket"] != socketOf(long) {
		t.Fatalf("want the sharing WARN, got %s", s.stderr.String())
	}
	if byMsg(lines, "started")["sharing"] != false {
		t.Fatalf("started line without sharing false: %s", s.stderr.String())
	}
	s.cs.Close()
	if code := s.exit(t, 5*time.Second); code != 0 {
		t.Fatalf("exit %d", code)
	}
}

func TestE2E_SPEC0009_R12_ownerServesForwardedConnection(t *testing.T) {
	root := gitRepo(t)
	startServer(t, root)
	cs := rawSession(t, root, "memctl <memctl@localhost>")
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(res.Tools) == 0 {
		t.Fatalf("tools through the socket: %v %+v", err, res)
	}
}

// connectTee starts memctl mcp on root like connect, and also records its
// stdout.
func connectTee(t *testing.T, root string) (*session, *tee) {
	t.Helper()
	cmd := exec.Command(binVersioned, "mcp", "--root", root)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	s := &session{cmd: cmd, stderr: new(syncBuffer), done: make(chan error, 1)}
	cmd.Stderr = s.stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { s.done <- cmd.Wait() }()
	t.Cleanup(func() { cmd.Process.Kill() })
	out := &tee{r: stdout}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "0"}, nil).
		Connect(ctx, &mcp.IOTransport{Reader: out, Writer: stdin}, nil)
	if err != nil {
		t.Fatalf("connect: %v; stderr %s", err, s.stderr.String())
	}
	s.cs = cs
	return s, out
}

func TestE2E_SPEC0009_R4_R9_R12_secondProcessForwards(t *testing.T) {
	root := gitRepo(t)
	owner := connect(t, root)
	waitFor(t, owner.stderr, `"msg":"started"`)
	fwd, out := connectTee(t, root)

	tools, err := fwd.cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) == 0 {
		t.Fatalf("tools through the forwarder: %v", err)
	}
	fwd.write(t, "Written by the forwarder")
	if got := owner.call(t, "note_read", map[string]any{"ref": "Written by the forwarder"}); !strings.Contains(got, "Read \"Written by the forwarder\"") {
		t.Fatalf("note_read through the owner: %s", got)
	}

	logs := jsonLines(t, fwd.stderr.String())
	if byMsg(logs, "notes loaded") != nil || byMsg(logs, "started") != nil {
		t.Fatalf("the forwarder loaded the notes: %s", fwd.stderr.String())
	}
	f := byMsg(logs, "forwarding")
	if f == nil || f["owner_pid"] != float64(owner.cmd.Process.Pid) || f["owner_version"] != "9.8.7-test" {
		t.Fatalf("forwarding line %v, want the owner's PID and version", f)
	}
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if !json.Valid([]byte(l)) {
			t.Fatalf("forwarder stdout line is not protocol: %q", l)
		}
	}
}
