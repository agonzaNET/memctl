package main

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// startWith starts bin as memctl mcp on root with extra arguments and stdin
// held open.
func startWith(t *testing.T, bin, root string, args ...string) (*exec.Cmd, io.WriteCloser, *syncBuffer, chan int) {
	t.Helper()
	cmd := exec.Command(bin, append([]string{"mcp", "--root", root}, args...)...)
	stdin, _ := cmd.StdinPipe()
	errb := new(syncBuffer)
	cmd.Stderr = errb
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan int, 1)
	go func() {
		err := cmd.Wait()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		exited <- code
	}()
	t.Cleanup(func() { stdin.Close(); cmd.Process.Kill() })
	return cmd, stdin, errb, exited
}

func countMsg(lines []map[string]any, msg string) int {
	n := 0
	for _, l := range lines {
		if l["msg"] == msg {
			n++
		}
	}
	return n
}

func TestE2E_SPEC0009_R7_otherGitAuthorRefused(t *testing.T) {
	root := notesRoot(t)
	owner, _, ownerErr := startServer(t, root)
	_, _, errb, exited := startWith(t, binVersioned, root, "--git-author", "Ann <ann@example.org>")
	var code int
	select {
	case code = <-exited:
	case <-time.After(10 * time.Second):
		t.Fatal("refused forwarder did not exit")
	}
	want := "git author differs: this process has Ann <ann@example.org>; memctl (pid " +
		strconv.Itoa(owner.Process.Pid) + ") serving the root has memctl <memctl@localhost>"
	errs := errorLines(jsonLines(t, errb.String()))
	if code != 1 || len(errs) != 1 || errs[0]["msg"] != want {
		t.Fatalf("code %d, ERROR lines %v; want 1 and %q", code, errs, want)
	}
	// One attempt only: the owner refused one connection.
	if n := countMsg(jsonLines(t, ownerErr.String()), "forwarded session refused"); n != 1 {
		t.Fatalf("owner refused %d connections, want 1: %s", n, ownerErr.String())
	}
}

func TestE2E_SPEC0009_R8_versionDiffersWarns(t *testing.T) {
	root := gitRepo(t)
	owner := connect(t, root) // version 9.8.7-test
	waitFor(t, owner.stderr, `"msg":"started"`)
	_, _, errb, _ := startWith(t, binPlain, root) // version dev
	waitFor(t, errb, `"msg":"forwarding"`)
	waitFor(t, owner.stderr, `"msg":"version differs"`)
	fw := byMsg(jsonLines(t, errb.String()), "version differs")
	if fw == nil || fw["level"] != "WARN" || fw["owner_version"] != "9.8.7-test" {
		t.Fatalf("forwarder WARN %v", fw)
	}
	ow := byMsg(jsonLines(t, owner.stderr.String()), "version differs")
	if ow["level"] != "WARN" || ow["forwarder_version"] != "dev" {
		t.Fatalf("owner WARN %v", ow)
	}
	owner.write(t, "Still works")
}

func waitExit(t *testing.T, exited chan int, within time.Duration, errb *syncBuffer) int {
	t.Helper()
	select {
	case c := <-exited:
		return c
	case <-time.After(within):
		t.Fatalf("did not exit within %v: %s", within, errb.String())
		return -1
	}
}

func TestE2E_SPEC0009_R10_forwarderStdinEOF(t *testing.T) {
	root := notesRoot(t)
	_, _, ownerErr := startServer(t, root)
	_, stdin, errb, exited := startWith(t, binVersioned, root)
	waitFor(t, errb, `"msg":"forwarding"`)
	stdin.Close()
	if c := waitExit(t, exited, 5*time.Second, errb); c != 0 || byMsg(jsonLines(t, errb.String()), "stopped") == nil {
		t.Fatalf("code %d, stderr %s; want 0 and a stopped line", c, errb.String())
	}
	// A normal end is no error for the owner (SPEC-0009 R13).
	waitFor(t, ownerErr, `"msg":"session closed"`)
	if strings.Contains(ownerErr.String(), "session ended with an error") {
		t.Fatalf("owner warned about a normal end: %s", ownerErr.String())
	}
}

func TestE2E_SPEC0009_R10_forwarderSIGTERM(t *testing.T) {
	root := notesRoot(t)
	startServer(t, root)
	cmd, _, errb, exited := startWith(t, binVersioned, root)
	waitFor(t, errb, `"msg":"forwarding"`)
	cmd.Process.Signal(syscall.SIGTERM)
	if c := waitExit(t, exited, 5*time.Second, errb); c != 0 {
		t.Fatalf("code %d after SIGTERM: %s", c, errb.String())
	}
}

func TestE2E_SPEC0009_R11_ownerLost(t *testing.T) {
	root := notesRoot(t)
	owner, _, _ := startServer(t, root)
	_, _, errb, exited := startWith(t, binVersioned, root)
	waitFor(t, errb, `"msg":"forwarding"`)
	owner.Process.Signal(syscall.SIGKILL)
	c := waitExit(t, exited, 5*time.Second, errb)
	want := "memctl (pid " + strconv.Itoa(owner.Process.Pid) + ") serving the root stopped"
	errs := errorLines(jsonLines(t, errb.String()))
	if c != 1 || len(errs) != 1 || errs[0]["msg"] != want {
		t.Fatalf("code %d, ERROR lines %v; want 1 and %q", c, errs, want)
	}
	// The client starts it again: now it is the owner.
	_, _, again := startServer(t, root)
	if byMsg(jsonLines(t, again.String()), "started") == nil {
		t.Fatalf("restarted process is not the owner: %s", again.String())
	}
}

// TestE2E_SPEC0009_R14_SPEC0004_R6_SPEC0008_R12_ownerOutlivesItsClient: the
// owner's stdin closed while a forwarder is connected leaves the forwarder
// served; when the forwarder ends, the owner commits, removes the socket and
// exits with status 0.
func TestE2E_SPEC0009_R14_SPEC0004_R6_SPEC0008_R12_ownerOutlivesItsClient(t *testing.T) {
	root := gitRepo(t)
	owner := connect(t, root)
	waitFor(t, owner.stderr, `"msg":"started"`)
	fwd := connect(t, root)
	waitFor(t, fwd.stderr, `"msg":"forwarding"`)

	owner.cs.Close()
	select {
	case <-owner.done:
		t.Fatalf("owner exited while a forwarded session was open: %s", owner.stderr.String())
	case <-time.After(300 * time.Millisecond):
	}
	fwd.write(t, "After the owner's client left")

	fwd.cs.Close()
	if c := fwd.exit(t, 5*time.Second); c != 0 {
		t.Fatalf("forwarder exit %d", c)
	}
	if c := owner.exit(t, 10*time.Second); c != 0 {
		t.Fatalf("owner exit %d: %s", c, owner.stderr.String())
	}
	log := gitIn(t, root, "log", "--format=%an <%ae>|%s")
	if !strings.Contains(log, "memctl <memctl@localhost>|memory: note_write After the owner's client left.md") {
		t.Fatalf("final commit missing or with another author:\n%s", log)
	}
	if _, err := os.Lstat(socketOf(root)); !os.IsNotExist(err) {
		t.Fatalf("socket file after exit: %v", err)
	}
}

func TestE2E_SPEC0009_R15_SIGTERMToOwner(t *testing.T) {
	root := gitRepo(t)
	owner := connect(t, root)
	waitFor(t, owner.stderr, `"msg":"started"`)
	fwd := connect(t, root)
	waitFor(t, fwd.stderr, `"msg":"forwarding"`)
	fwd.write(t, "Before SIGTERM")
	owner.cmd.Process.Signal(syscall.SIGTERM)
	if c := fwd.exit(t, 5*time.Second); c != 1 {
		t.Fatalf("forwarder exit %d, want 1: %s", c, fwd.stderr.String())
	}
	if c := owner.exit(t, 10*time.Second); c != 0 {
		t.Fatalf("owner exit %d: %s", c, owner.stderr.String())
	}
	if log := gitLog(t, root); !strings.Contains(log, "Before SIGTERM") {
		t.Fatalf("final commit missing:\n%s", log)
	}
}

func TestE2E_SPEC0009_R16_closedStderrDoesNotStopTheOwner(t *testing.T) {
	root := notesRoot(t)
	cmd := exec.Command(binVersioned, "mcp", "--root", root)
	stdin, _ := cmd.StdinPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	t.Cleanup(func() { stdin.Close(); cmd.Process.Kill() })
	buf := make([]byte, 4096)
	var seen string
	for !strings.Contains(seen, `"msg":"started"`) {
		n, err := stderr.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		seen += string(buf[:n])
	}
	stderr.Close() // the reader goes away

	fwd := connect(t, root)
	if got := fwd.call(t, "note_read", map[string]any{"ref": "A"}); !strings.Contains(got, "Read \"A\"") {
		t.Fatalf("note_read: %s", got)
	}
	fwd.call(t, "search", map[string]any{"query": "body"})
	select {
	case err := <-exited:
		t.Fatalf("owner stopped after its stderr closed: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestE2E_SPEC0009_R13_sessionsInTheOwnerLog(t *testing.T) {
	root := gitRepo(t)
	owner := connect(t, root)
	waitFor(t, owner.stderr, `"msg":"started"`)
	fwd := connect(t, root)
	waitFor(t, fwd.stderr, `"msg":"forwarding"`)
	owner.write(t, "Zebra from the owner")
	fwd.write(t, "Zebra from the forwarder")
	fwd.cs.Close()
	fwd.exit(t, 5*time.Second)
	waitFor(t, owner.stderr, `"msg":"session closed"`)

	lines := jsonLines(t, owner.stderr.String())
	var sessions []any
	for _, l := range lines {
		if l["msg"] == "tool call" {
			sessions = append(sessions, l["session"])
		}
	}
	if len(sessions) != 2 || sessions[0] != "stdio" || sessions[1] != "forward-1" {
		t.Fatalf("tool call sessions %v, want stdio and forward-1", sessions)
	}
	pid := float64(fwd.cmd.Process.Pid)
	for _, msg := range []string{"session opened", "session closed"} {
		l := byMsg(lines, msg)
		if l == nil || l["level"] != "INFO" || l["session"] != "forward-1" || l["forwarder_pid"] != pid {
			t.Fatalf("%s line %v, want session forward-1 and forwarder_pid %v", msg, l, pid)
		}
	}
	if strings.Contains(owner.stderr.String(), "Zebra") {
		t.Fatalf("owner log holds note text: %s", owner.stderr.String())
	}
}

// editResult calls note_edit and returns "ok" or the error code.
func editResult(s *session, sha, text string) string {
	res, err := s.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_edit",
		Arguments: map[string]any{"ref": "a.md", "expected_sha": sha, "op": "append", "text": text}})
	if err != nil {
		return "transport: " + err.Error()
	}
	if !res.IsError {
		return "ok"
	}
	msg := res.Content[0].(*mcp.TextContent).Text
	code, _, _ := strings.Cut(msg, ":")
	return code
}

func TestE2E_SPEC0009_R12_concurrentEditsFromTwoSessions(t *testing.T) {
	root := gitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\nfirst\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "add", "a.md")
	gitIn(t, root, "-c", "user.name=Base", "-c", "user.email=base@example.org", "commit", "-q", "-m", "baseline")
	owner := connect(t, root)
	waitFor(t, owner.stderr, `"msg":"started"`)
	fwd := connect(t, root)
	waitFor(t, fwd.stderr, `"msg":"forwarding"`)

	for round := 0; round < 20; round++ {
		got := owner.call(t, "note_read", map[string]any{"ref": "a.md"})
		sha := got[strings.Index(got, "- sha256: ")+10:][:64]
		results := make(chan string, 2)
		go func() { results <- editResult(owner, sha, "owner "+strconv.Itoa(round)) }()
		go func() { results <- editResult(fwd, sha, "forwarder "+strconv.Itoa(round)) }()
		a, b := <-results, <-results
		if !((a == "ok" && b == "conflict") || (a == "conflict" && b == "ok")) {
			t.Fatalf("round %d: results %q and %q, want one ok and one conflict", round, a, b)
		}
	}
	fwd.cs.Close()
	fwd.exit(t, 5*time.Second)
	owner.cs.Close()
	owner.exit(t, 20*time.Second)
	authors := gitIn(t, root, "log", "--format=%an <%ae>|%s")
	for _, l := range strings.Split(strings.TrimSpace(authors), "\n") {
		if strings.HasSuffix(l, "|baseline") {
			continue
		}
		if !strings.HasPrefix(l, "memctl <memctl@localhost>|") {
			t.Fatalf("commit by another author: %q\n%s", l, authors)
		}
	}
}

// TestE2E_SPEC0009_R17_childrenKeepDefaultSIGPIPE: every git process that
// memctl starts for a tool write starts with SIGPIPE not ignored. A git
// wrapper first on PATH records its own SigIgn and runs the real git. The
// repository's hooks are not checked: git 2.54 and later ignores SIGPIPE
// while it runs hooks, so they inherit that from git, not from memctl.
func TestE2E_SPEC0009_R17_childrenKeepDefaultSIGPIPE(t *testing.T) {
	if _, err := os.Stat("/proc/self/status"); err != nil {
		t.Skip("needs /proc (Linux)")
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	root := gitRepo(t)
	bin := t.TempDir()
	out := filepath.Join(t.TempDir(), "sigign")
	wrapper := "#!/bin/sh\ngrep '^SigIgn:' /proc/$$/status >> \"$MEMCTL_TEST_GIT_OUT\"\nexec " + strconv.Quote(realGit) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEMCTL_TEST_GIT_OUT", out)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	s := connect(t, root)
	s.write(t, "Git check")
	s.cs.Close()
	s.exit(t, 20*time.Second)
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("memctl ran no git: %v; %s", err, s.stderr.String())
	}
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		mask, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(l, "SigIgn:")), 16, 64)
		if err != nil {
			t.Fatalf("SigIgn line %q: %v", l, err)
		}
		if mask&(1<<(uint(syscall.SIGPIPE)-1)) != 0 {
			t.Fatalf("SIGPIPE is ignored in a git that memctl started (SigIgn %s)", strings.TrimSpace(l))
		}
	}
}
