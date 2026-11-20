package gitlog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

func TestCommitter_SPEC0008_R14_externalCommits(t *testing.T) {
	for _, withIdentity := range []bool{false, true} {
		root := newRepo(t)
		wantAuthor := "memctl <memctl@localhost>"
		if withIdentity {
			git(t, root, "config", "user.name", "Ann Person")
			git(t, root, "config", "user.email", "ann@example.org")
			wantAuthor = "Ann Person <ann@example.org>"
		}
		write(t, root, "x.md", "edited by a person\n")
		write(t, root, "y.md", "written by a tool\n")
		h := startCommitter(t, root, defaultAuthor)
		h.c.External([]string{"x.md"})
		h.idle(t)
		h.add(t, gitlog.Entry{Tool: "note_write", Title: "Y", Paths: []string{"y.md"}})
		h.clock.Advance(2 * time.Second)
		h.idle(t)
		got := git(t, root, "log", "-2", "--format=%s|%an <%ae>|%cn <%ce>", "--name-only")
		want := "memory: note_write y.md|memctl <memctl@localhost>|memctl <memctl@localhost>\n\ny.md\n" +
			"memory: external change x.md|" + wantAuthor + "|" + wantAuthor + "\n\nx.md\n"
		if got != want {
			t.Fatalf("identity %v: log\n%s\nwant\n%s", withIdentity, got, want)
		}
	}

	root := newRepo(t)
	for _, p := range []string{"c.md", "a.md", "b.md"} {
		write(t, root, p, p+"\n")
	}
	h := startCommitter(t, root, defaultAuthor)
	h.c.External([]string{"c.md", "a.md"})
	h.c.External([]string{"b.md", "a.md"})
	h.idle(t)
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	if got := git(t, root, "log", "-1", "--format=%B"); got != "memory: 3 external changes\n\n- a.md\n- b.md\n- c.md\n\n" {
		t.Fatalf("message %q", got)
	}
}

func TestCommitter_SPEC0008_R16_waitExternal(t *testing.T) {
	root := newRepo(t)
	write(t, root, "a.md", "person\n")
	h := startCommitter(t, root, defaultAuthor)
	before := commits(t, root)
	if err := h.c.WaitExternal("a.md"); err != nil {
		t.Fatal(err)
	}
	if commits(t, root) != before {
		t.Fatal("a wait without a pending external change committed")
	}
	h.c.External([]string{"a.md"})
	if err := h.c.WaitExternal("a.md"); err != nil {
		t.Fatal(err)
	}
	if got := git(t, root, "log", "-1", "--format=%s"); got != "memory: external change a.md\n" {
		t.Fatalf("not committed before the clock moved: %q", got)
	}

	// A failing commit logs the ERROR line and lets the write go on.
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "a.md", "person again\n")
	h.c.External([]string{"a.md"})
	done := make(chan struct{})
	go func() { h.c.WaitExternal("a.md"); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("WaitExternal did not return after a failed commit")
	}
	if errs := h.log.lines(t, "commit failed"); len(errs) != 1 || errs[0]["class"] != "hook_failed" {
		t.Fatalf("errors %v", errs)
	}
}

func TestCommitter_SPEC0008_R12_flushesExternal(t *testing.T) {
	root := newRepo(t)
	write(t, root, "a.md", "person\n")
	h := startCommitter(t, root, defaultAuthor)
	h.c.External([]string{"a.md"})
	h.idle(t)
	h.cancel()
	select {
	case <-h.done:
	case <-time.After(10 * time.Second):
		t.Fatal("committer did not stop")
	}
	if got := git(t, root, "log", "-1", "--format=%s"); !strings.HasPrefix(got, "memory: external change a.md") {
		t.Fatalf("not flushed: %q", got)
	}
}
