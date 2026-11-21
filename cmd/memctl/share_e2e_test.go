package main

import (
	"context"
	"io"
	"net"
	"os"
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
