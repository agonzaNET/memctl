package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type session struct {
	cmd    *exec.Cmd
	cs     *mcp.ClientSession
	stderr *syncBuffer
	done   chan error
}

// connect starts memctl mcp on root and connects an MCP client over pipes.
func connect(t *testing.T, root string) *session {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "0"}, nil).
		Connect(ctx, &mcp.IOTransport{Reader: stdout, Writer: stdin}, nil)
	if err != nil {
		t.Fatalf("connect: %v; stderr %s", err, s.stderr.String())
	}
	s.cs = cs
	return s
}

func (s *session) write(t *testing.T, title string) {
	t.Helper()
	res, err := s.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_write", Arguments: map[string]any{"title": title, "body": "text"}})
	if err != nil || res.IsError {
		t.Fatalf("note_write: %v %+v", err, res)
	}
}

// exit waits for the process and returns its exit status.
func (s *session) exit(t *testing.T, within time.Duration) int {
	t.Helper()
	select {
	case err := <-s.done:
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		if err != nil {
			t.Fatal(err)
		}
		return 0
	case <-time.After(within):
		t.Fatalf("did not exit within %v; stderr %s", within, s.stderr.String())
		return -1
	}
}

func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	root := tempRoot(t)
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	return root
}

func gitLog(t *testing.T, root string) string {
	t.Helper()
	out, _ := exec.Command("git", "-C", root, "log", "--format=%s").CombinedOutput()
	return string(out)
}

func TestE2E_SPEC0008_R12_flushOnStdinEOF(t *testing.T) {
	root := gitRepo(t)
	s := connect(t, root)
	s.write(t, "Last words")
	s.cs.Close() // closes stdin at once, inside the quiet window
	if code := s.exit(t, 20*time.Second); code != 0 {
		t.Fatalf("status %d; stderr %s", code, s.stderr.String())
	}
	if got := gitLog(t, root); got != "memory: note_write Last words.md\n" {
		t.Fatalf("git log %q; stderr %s", got, s.stderr.String())
	}
	if byMsg(jsonLines(t, s.stderr.String()), "stopped") == nil {
		t.Fatalf("no stopped line: %s", s.stderr.String())
	}
}

func TestE2E_SPEC0008_R12_flushOnSIGTERM(t *testing.T) {
	root := gitRepo(t)
	s := connect(t, root)
	s.write(t, "Signalled")
	s.cmd.Process.Signal(syscall.SIGTERM)
	if code := s.exit(t, 20*time.Second); code != 0 {
		t.Fatalf("status %d; stderr %s", code, s.stderr.String())
	}
	if got := gitLog(t, root); got != "memory: note_write Signalled.md\n" {
		t.Fatalf("git log %q; stderr %s", got, s.stderr.String())
	}
}

func TestE2E_SPEC0008_R13_secondSignalExitsAtOnce(t *testing.T) {
	root := gitRepo(t)
	// A full identity, so the only WARN is the second signal's (R17).
	gitIn(t, root, "config", "user.name", "P")
	gitIn(t, root, "config", "user.email", "p@example.org")
	real, _ := exec.LookPath("git")
	dir := t.TempDir()
	marker := filepath.Join(dir, "commit-started")
	script := "#!/bin/sh\ncase \" $* \" in *\" commit \"*) touch \"" + marker + "\"; sleep 10; exit 1 ;; esac\nexec \"" + real + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	s := connect(t, root)
	s.write(t, "Slow")
	s.cmd.Process.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the final commit never started; stderr %s", s.stderr.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	start := time.Now()
	s.cmd.Process.Signal(syscall.SIGTERM)
	if code := s.exit(t, 3*time.Second); code != 1 {
		t.Fatalf("status %d, want 1; stderr %s", code, s.stderr.String())
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("exit took %v after the second signal", d)
	}
	var warns []map[string]any
	for _, l := range jsonLines(t, s.stderr.String()) {
		if l["level"] == "WARN" {
			warns = append(warns, l)
		}
	}
	if len(warns) != 1 || !strings.Contains(warns[0]["msg"].(string), "signal") {
		t.Fatalf("WARN lines %v", warns)
	}
}

func TestE2E_SPEC0008_R4_burstIsOneCommit(t *testing.T) {
	root := gitRepo(t)
	s := connect(t, root)
	for i := 0; i < 10; i++ {
		s.write(t, "Burst "+string(rune('A'+i)))
	}
	time.Sleep(3 * time.Second) // past the quiet window, before shutdown
	if got := gitLog(t, root); got != "memory: 10 changes\n" {
		t.Fatalf("git log %q; stderr %s", got, s.stderr.String())
	}
	s.cs.Close()
	if code := s.exit(t, 20*time.Second); code != 0 {
		t.Fatalf("status %d", code)
	}
	if got := gitLog(t, root); got != "memory: 10 changes\n" {
		t.Fatalf("shutdown added a commit: %q", got)
	}
}

func (s *session) call(t *testing.T, tool string, args map[string]any) string {
	t.Helper()
	res, err := s.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil || res.IsError {
		t.Fatalf("%s: %v %+v", tool, err, res)
	}
	return res.Content[0].(*mcp.TextContent).Text
}

func gitIn(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=P", "GIT_AUTHOR_EMAIL=p@example.org", "GIT_COMMITTER_NAME=P", "GIT_COMMITTER_EMAIL=p@example.org")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return string(out)
}

func TestE2E_SPEC0008_R12_R16_externalChange(t *testing.T) {
	root := gitRepo(t)
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\nfirst\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, root, "add", "a.md")
	gitIn(t, root, "commit", "-q", "-m", "baseline")
	s := connect(t, root)
	got := s.call(t, "note_read", map[string]any{"ref": "a.md"})
	_ = got
	// A person edits the note; the next call finds it, and an edit by the
	// tool on the same note waits for the external commit.
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\nfirst\nperson line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = s.call(t, "note_read", map[string]any{"ref": "a.md"})
	sha := got[strings.Index(got, "- sha256: ")+10:][:64]
	s.call(t, "note_edit", map[string]any{"ref": "a.md", "expected_sha": sha, "op": "append", "text": "tool line"})
	s.cs.Close()
	if code := s.exit(t, 20*time.Second); code != 0 {
		t.Fatalf("status %d; stderr %s", code, s.stderr.String())
	}
	if got := gitLog(t, root); got != "memory: note_edit a.md\nmemory: external change a.md\nbaseline\n" {
		t.Fatalf("git log %q; stderr %s", got, s.stderr.String())
	}
	ext := gitIn(t, root, "show", "--format=", "HEAD~1")
	tool := gitIn(t, root, "show", "--format=", "HEAD")
	if !strings.Contains(ext, "+person line") || strings.Contains(ext, "tool line") {
		t.Fatalf("external commit:\n%s", ext)
	}
	if !strings.Contains(tool, "+tool line") || strings.Contains(tool, "+person line") {
		t.Fatalf("tool commit:\n%s", tool)
	}

	// An editor change found by a check, then stdin closed at once.
	s = connect(t, root)
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\ntitle: A\n---\nlast edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.call(t, "list", map[string]any{})
	s.cs.Close()
	if code := s.exit(t, 20*time.Second); code != 0 {
		t.Fatalf("status %d; stderr %s", code, s.stderr.String())
	}
	if got := gitLog(t, root); !strings.HasPrefix(got, "memory: external change a.md\nmemory: note_edit a.md\n") {
		t.Fatalf("git log %q", got)
	}
}

func TestE2E_SPEC0008_R15_startupScan(t *testing.T) {
	root := gitRepo(t)
	files := map[string]string{".gitignore": "ignored/\n", "a.md": "a\n", "c.md": "c\n"}
	for p, c := range files {
		if err := os.WriteFile(filepath.Join(root, p), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, root, "add", ".")
	gitIn(t, root, "commit", "-q", "-m", "baseline")
	// While memctl is not running:
	os.WriteFile(filepath.Join(root, "a.md"), []byte("edited\n"), 0o644)
	os.WriteFile(filepath.Join(root, "b.md"), []byte("new\n"), 0o644)
	os.Remove(filepath.Join(root, "c.md"))
	os.WriteFile(filepath.Join(root, "x.txt"), []byte("x\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "ignored"), 0o755)
	os.WriteFile(filepath.Join(root, "ignored", "d.md"), []byte("d\n"), 0o644)

	s := connect(t, root)
	s.cs.Close()
	if code := s.exit(t, 20*time.Second); code != 0 {
		t.Fatalf("status %d; stderr %s", code, s.stderr.String())
	}
	if got := gitIn(t, root, "log", "-1", "--format=%B"); got != "memory: 3 external changes\n\n- a.md\n- b.md\n- c.md\n\n" {
		t.Fatalf("message %q; stderr %s", got, s.stderr.String())
	}
	if left := gitIn(t, root, "status", "--porcelain", "--untracked-files=all"); left != "?? x.txt\n" {
		t.Fatalf("left %q", left)
	}
}
