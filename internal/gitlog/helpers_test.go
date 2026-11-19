package gitlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

// fakeClock is a Clock whose time moves only on Advance.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []waiter
}

type waiter struct {
	at time.Time
	ch chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *fakeClock) After(d time.Duration) <-chan time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- f.now
		return ch
	}
	f.waiters = append(f.waiters, waiter{f.now.Add(d), ch})
	return ch
}

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
	kept := f.waiters[:0]
	for _, w := range f.waiters {
		if !w.at.After(f.now) {
			w.ch <- f.now
		} else {
			kept = append(kept, w)
		}
	}
	f.waiters = kept
}

// lockedBuffer collects log output from several goroutines.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func (l *lockedBuffer) lines(t *testing.T, msg string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, s := range strings.Split(strings.TrimSpace(l.String()), "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(s), &m) == nil && (msg == "" || m["msg"] == msg) {
			out = append(out, m)
		}
	}
	return out
}

// isolateGit keeps the user's and the system's git config out of the test,
// so no identity is configured.
func isolateGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("GIT_COMMITTER_NAME", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")
	os.Unsetenv("GIT_AUTHOR_NAME")
	os.Unsetenv("GIT_AUTHOR_EMAIL")
	os.Unsetenv("GIT_COMMITTER_NAME")
	os.Unsetenv("GIT_COMMITTER_EMAIL")
}

// git runs a git command in dir for the test, with a test identity.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Person", "GIT_AUTHOR_EMAIL=person@example.org",
		"GIT_COMMITTER_NAME=Person", "GIT_COMMITTER_EMAIL=person@example.org")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newRepo makes a git repository with one commit and returns its path.
func newRepo(t *testing.T) string {
	t.Helper()
	isolateGit(t)
	dir := t.TempDir()
	git(t, dir, "init", "-q")
	write(t, dir, "README.md", "readme\n")
	git(t, dir, "add", "README.md")
	git(t, dir, "commit", "-q", "-m", "init")
	return dir
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commits(t *testing.T, dir string) int {
	t.Helper()
	n := strings.TrimSpace(git(t, dir, "rev-list", "--count", "HEAD"))
	var c int
	for _, r := range n {
		c = c*10 + int(r-'0')
	}
	return c
}

// harness runs a committer on a fake clock and lets the test wait until its
// loop is idle (blocked waiting for an entry, a timer or shutdown).
type harness struct {
	c      *gitlog.Committer
	clock  *fakeClock
	log    *lockedBuffer
	idleCh chan struct{}
	cancel context.CancelFunc
	done   chan struct{}
}

func startCommitter(t *testing.T, root string, author gitlog.Author) *harness {
	t.Helper()
	h := &harness{clock: newFakeClock(), log: &lockedBuffer{}, idleCh: make(chan struct{}, 100), done: make(chan struct{})}
	log := slog.New(slog.NewJSONHandler(h.log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h.c = gitlog.New(context.Background(), gitlog.Config{Root: root, Author: author, Log: log, Clock: h.clock})
	gitlog.SetIdle(h.c, func() { h.idleCh <- struct{}{} })
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	go func() { h.c.Run(ctx); close(h.done) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-h.done:
		case <-time.After(10 * time.Second):
			t.Error("committer did not stop")
		}
	})
	h.idle(t)
	return h
}

func (h *harness) idle(t *testing.T) {
	t.Helper()
	select {
	case <-h.idleCh:
	case <-time.After(10 * time.Second):
		t.Fatal("committer loop never became idle")
	}
}

// add queues an entry and waits until the loop has taken it in.
func (h *harness) add(t *testing.T, e gitlog.Entry) {
	t.Helper()
	h.c.Add(e)
	h.idle(t)
}

// fakeGit puts a git wrapper first on PATH that appends one line per call
// to the returned log: the environment flag and the arguments. With
// sleep > 0, a commit call sleeps that long first.
func fakeGit(t *testing.T, sleep string) string {
	t.Helper()
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	logFile := filepath.Join(dir, "calls.log")
	script := `#!/bin/sh
printf 'prompt=%s|%s\n' "$GIT_TERMINAL_PROMPT" "$*" >> "` + logFile + `"
case " $* " in *" commit "*) [ -n "` + sleep + `" ] && sleep ` + sleep + ` ;; esac
exec "` + real + `" "$@"
`
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logFile
}

func readCalls(t *testing.T, logFile string) []string {
	t.Helper()
	b, err := os.ReadFile(logFile)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}
