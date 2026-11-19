package gitlog_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

var defaultAuthor = gitlog.Author{Name: "memctl", Email: "memctl@localhost"}

func TestCommitter_SPEC0008_R1_notARepository(t *testing.T) {
	isolateGit(t)
	calls := fakeGit(t, "")
	root := t.TempDir()
	h := startCommitter(t, root, defaultAuthor)
	warns := h.log.lines(t, "")
	n := 0
	for _, l := range warns {
		if l["level"] == "WARN" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("want one WARN line at start, got %v", warns)
	}
	before := len(readCalls(t, calls))
	write(t, root, "a.md", "a\n")
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	h.clock.Advance(3 * time.Second)
	h.idle(t)
	if after := len(readCalls(t, calls)); after != before {
		t.Fatalf("git ran %d times for a root outside git", after-before)
	}
}

func TestCommitter_SPEC0008_R1_gitMissing(t *testing.T) {
	isolateGit(t)
	t.Setenv("PATH", t.TempDir())
	h := startCommitter(t, t.TempDir(), defaultAuthor)
	if l := h.log.lines(t, ""); len(l) != 1 || l[0]["level"] != "WARN" {
		t.Fatalf("want one WARN line, got %v", l)
	}
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	h.clock.Advance(3 * time.Second)
	h.idle(t)
}

func TestCommitter_SPEC0008_R1_subfolderOfRepository(t *testing.T) {
	repo := newRepo(t)
	root := repo + "/notes/deep"
	write(t, root, "a.md", "a\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	if got := git(t, repo, "show", "--name-only", "--format=%s", "HEAD"); got != "memory: note_write A\n\nnotes/deep/a.md\n" {
		t.Fatalf("commit %q", got)
	}
}

func TestCommitter_SPEC0008_R2_onlyAllowedCalls(t *testing.T) {
	root := newRepo(t)
	calls := fakeGit(t, "")
	write(t, root, "a.md", "a\n")
	write(t, root, "b.md", "b\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "B", Paths: []string{"b.md"}})
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	got := readCalls(t, calls)
	if len(got) < 4 {
		t.Fatalf("calls %v", got)
	}
	allowed := map[string]bool{"rev-parse": true, "check-ignore": true, "add": true, "commit": true}
	prefix := "prompt=0|-C " + root + " -c commit.gpgSign=false "
	for _, c := range got {
		if !strings.HasPrefix(c, prefix) {
			t.Fatalf("call %q lacks %q", c, prefix)
		}
		sub := strings.Fields(strings.TrimPrefix(c, prefix))[0]
		if !allowed[sub] {
			t.Fatalf("call %q uses %s", c, sub)
		}
	}
	if commits(t, root) != 2 {
		t.Fatalf("want one tool commit, got %d commits", commits(t, root))
	}
}

func TestCommitter_SPEC0008_R3_R4_quietWindow(t *testing.T) {
	root := newRepo(t)
	write(t, root, "a.md", "a\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	h.clock.Advance(1900 * time.Millisecond)
	if commits(t, root) != 1 {
		t.Fatal("committed before the quiet window ended")
	}
	h.clock.Advance(100 * time.Millisecond)
	h.idle(t)
	if commits(t, root) != 2 {
		t.Fatal("no commit 2 s after the write")
	}
}

func TestCommitter_SPEC0008_R4_maxDelay(t *testing.T) {
	root := newRepo(t)
	h := startCommitter(t, root, defaultAuthor)
	for i := 0; i < 32; i++ {
		rel := "n" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + ".md"
		write(t, root, rel, "x\n")
		h.add(t, gitlog.Entry{Tool: "note_write", Title: rel, Paths: []string{rel}})
		h.clock.Advance(time.Second)
		if i == 29 {
			h.idle(t) // the batch's first entry is 30 s old
		}
		want := 1
		if i >= 29 {
			want = 2
		}
		if got := commits(t, root); got != want {
			t.Fatalf("after %d s: %d commits, want %d", i+1, got, want)
		}
	}
	if subj := git(t, root, "log", "-1", "--format=%s"); subj != "memory: 30 changes\n" {
		t.Fatalf("subject %q", subj)
	}
}

func TestCommitter_SPEC0008_R3_addDoesNotBlock(t *testing.T) {
	root := newRepo(t)
	fakeGit(t, "1")
	write(t, root, "a.md", "a\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	h.clock.Advance(2 * time.Second) // the commit starts and takes 1 s
	time.Sleep(100 * time.Millisecond)
	start := time.Now()
	for i := 0; i < 50; i++ {
		h.c.Add(gitlog.Entry{Tool: "note_write", Title: "B", Paths: []string{"b.md"}})
	}
	if d := time.Since(start); d > 50*time.Millisecond {
		t.Fatalf("Add blocked for %v during a commit", d)
	}
	h.idle(t)
}

func TestMessage_SPEC0008_R5(t *testing.T) {
	one := gitlog.Message([]gitlog.Entry{{Tool: "note_write", Title: "PAN-OS upgrade", Paths: []string{"runbooks/PAN-OS upgrade.md"}}})
	if one != "memory: note_write PAN-OS upgrade\n" {
		t.Fatalf("one entry: %q", one)
	}
	three := gitlog.Message([]gitlog.Entry{
		{Tool: "note_write", Title: "A", Paths: []string{"a.md"}},
		{Tool: "note_move", Title: "B\nsecond\r\nline", Paths: []string{"old/b.md", "new/b.md"}},
		{Tool: "note_delete", Title: "C", Paths: []string{"c.md"}},
	})
	want := "memory: 3 changes\n\n- note_write A (a.md)\n- note_move B second line (new/b.md)\n- note_delete C (c.md)\n"
	if three != want {
		t.Fatalf("three entries:\n%q\nwant\n%q", three, want)
	}
	if got := gitlog.Message([]gitlog.Entry{{Tool: "note_edit", Title: "x\ny", Paths: []string{"x.md"}}}); got != "memory: note_edit x y\n" {
		t.Fatalf("line break in a single title: %q", got)
	}
}

func TestCommitter_SPEC0008_R6_onlyToolPaths(t *testing.T) {
	root := newRepo(t)
	write(t, root, "keep.md", "k\n")
	write(t, root, "gone.md", "g\n")
	write(t, root, "old.md", "o\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", "notes")
	write(t, root, ".gitignore", "ignored/\n")
	git(t, root, "add", ".gitignore")
	git(t, root, "commit", "-q", "-m", "ignore")

	// A person's staged change and an unstaged one in other files.
	write(t, root, "keep.md", "person staged\n")
	git(t, root, "add", "keep.md")
	write(t, root, "README.md", "person unstaged\n")

	// Tool changes: a new note, a delete, a move, an ignored path, and a
	// note created and deleted within the batch.
	write(t, root, "new.md", "n\n")
	os.Remove(root + "/gone.md")
	os.Rename(root+"/old.md", root+"/moved.md")
	write(t, root, "ignored/x.md", "i\n")

	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "N", Paths: []string{"new.md"}})
	h.add(t, gitlog.Entry{Tool: "note_delete", Title: "G", Paths: []string{"gone.md"}})
	h.add(t, gitlog.Entry{Tool: "note_move", Title: "O", Paths: []string{"old.md", "moved.md"}})
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "I", Paths: []string{"ignored/x.md"}})
	h.add(t, gitlog.Entry{Tool: "note_delete", Title: "T", Paths: []string{"transient.md"}})
	h.clock.Advance(2 * time.Second)
	h.idle(t)

	got := git(t, root, "show", "-M", "--name-status", "--format=", "HEAD")
	want := "D\tgone.md\nR100\told.md\tmoved.md\nA\tnew.md\n"
	if got != want {
		t.Fatalf("commit holds:\n%s\nwant:\n%s\nlog %s", got, want, h.log.String())
	}
	status := git(t, root, "status", "--porcelain")
	if !strings.Contains(status, "M  keep.md") || !strings.Contains(status, " M README.md") {
		t.Fatalf("the person's changes were touched:\n%s", status)
	}
	if l := h.log.lines(t, "commit failed"); len(l) != 0 {
		t.Fatalf("commit failed: %v", l)
	}
}

func TestCommitter_SPEC0008_R7_author(t *testing.T) {
	for _, c := range []struct {
		author gitlog.Author
		want   string
	}{
		{defaultAuthor, "memctl <memctl@localhost>|memctl <memctl@localhost>"},
		{gitlog.Author{Name: "Ann", Email: "ann@example.org"}, "Ann <ann@example.org>|Ann <ann@example.org>"},
	} {
		root := newRepo(t)
		write(t, root, "a.md", "a\n")
		h := startCommitter(t, root, c.author)
		h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
		h.clock.Advance(2 * time.Second)
		h.idle(t)
		if got := git(t, root, "log", "-1", "--format=%an <%ae>|%cn <%ce>"); got != c.want+"\n" {
			t.Fatalf("identity %q, want %q; log %s", got, c.want, h.log.String())
		}
	}
}

func TestAuthor_SPEC0008_inputRule(t *testing.T) {
	good := map[string]gitlog.Author{
		"memctl <memctl@localhost>":         {"memctl", "memctl@localhost"},
		"Ann Lee <ann@example.org>":         {"Ann Lee", "ann@example.org"},
		strings.Repeat("n", 100) + " <a@b>": {strings.Repeat("n", 100), "a@b"},
	}
	for in, want := range good {
		if got, err := gitlog.ParseAuthor(in); err != nil || got != want {
			t.Errorf("ParseAuthor(%q) = %+v, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "Ann", "<a@b>", "Ann <ab>", "Ann <a@b@c>", "Ann <a@b", "Ann\n <a@b>", "Ann <a@b>\n", strings.Repeat("n", 101) + " <a@b>", "Ann <a b@c>"} {
		if _, err := gitlog.ParseAuthor(in); err == nil {
			t.Errorf("ParseAuthor(%q) accepted", in)
		}
	}
}

func TestCommitter_SPEC0008_R9_nothingToCommit(t *testing.T) {
	root := newRepo(t)
	write(t, root, "README.md", "edited\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_edit", Title: "R", Paths: []string{"README.md"}})
	write(t, root, "README.md", "readme\n") // reverted before the batch commits
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	if commits(t, root) != 1 {
		t.Fatal("a commit was made for an unchanged file")
	}
	for _, l := range h.log.lines(t, "") {
		if l["level"] == "ERROR" {
			t.Fatalf("ERROR line: %v", l)
		}
	}
}

func TestCommitter_SPEC0008_R11_committedLine(t *testing.T) {
	root := newRepo(t)
	write(t, root, "secret/a.md", "a\n")
	write(t, root, "secret/b.md", "b\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_move", Title: "Secret title", Paths: []string{"secret/a.md", "secret/b.md"}})
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	l := h.log.lines(t, "committed")
	short := strings.TrimSpace(git(t, root, "rev-parse", "--short", "HEAD"))
	if len(l) != 1 || l[0]["level"] != "INFO" || l[0]["paths"] != 2.0 || l[0]["commit"] != short {
		t.Fatalf("committed line %v, want paths 2 and commit %s", l, short)
	}
	if s := h.log.String(); strings.Contains(s, "ecret") || strings.Contains(s, "a.md") {
		t.Fatalf("log holds a title or path: %s", s)
	}
}

func TestIsRepo_SPEC0008_R2_noSigning(t *testing.T) {
	root := newRepo(t)
	calls := fakeGit(t, "")
	if ok, err := gitlog.IsRepo(context.Background(), root); !ok || err != nil {
		t.Fatalf("IsRepo: %v %v", ok, err)
	}
	got := readCalls(t, calls)
	if len(got) != 1 || got[0] != "prompt=0|-C "+root+" -c commit.gpgSign=false rev-parse --is-inside-work-tree" {
		t.Fatalf("calls %v", got)
	}
	_ = slog.Default
}
