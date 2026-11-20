package gitlog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/gitlog"
)

// errorLines returns the ERROR lines of the log.
func (h *harness) errorLines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range h.log.lines(t, "") {
		if l["level"] == "ERROR" {
			out = append(out, l)
		}
	}
	return out
}

// assertNoContent fails when the log holds a title, a path, a commit
// message or git's output (SPEC-0008 R11).
func assertNoContent(t *testing.T, h *harness, words ...string) {
	t.Helper()
	s := h.log.String()
	for _, w := range append(words, "memory:", "fatal", "hook said") {
		if strings.Contains(s, w) {
			t.Fatalf("log holds %q: %s", w, s)
		}
	}
}

func TestCommitter_SPEC0008_R8_R10_R11_hookFails(t *testing.T) {
	root := newRepo(t)
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'hook said no' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "secret-note.md", "x\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "Secret title", Paths: []string{"secret-note.md"}})
	h.clock.Advance(2 * time.Second)
	h.idle(t)

	errs := h.errorLines(t)
	if len(errs) != 1 || errs[0]["msg"] != "commit failed" || errs[0]["class"] != "hook_failed" || errs[0]["status"] != 1.0 {
		t.Fatalf("ERROR lines %v", errs)
	}
	if commits(t, root) != 1 {
		t.Fatal("a commit was made although the hook failed")
	}
	if st := git(t, root, "status", "--porcelain"); !strings.Contains(st, "secret-note.md") {
		t.Fatalf("the file is not in git status: %q", st)
	}
	assertNoContent(t, h, "Secret title", "secret-note")

	// The batch was dropped, and the committer goes on: the next write is
	// committed once the hook passes.
	os.Remove(hook)
	write(t, root, "next.md", "n\n")
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "Next", Paths: []string{"next.md"}})
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	if got := git(t, root, "show", "--name-only", "--format=%s", "HEAD"); got != "memory: note_write next.md\n\nnext.md\n" {
		t.Fatalf("next commit %q", got)
	}
}

func TestCommitter_SPEC0008_R10_indexLockReleased(t *testing.T) {
	root := newRepo(t)
	lock := filepath.Join(root, ".git", "index.lock")
	os.WriteFile(lock, nil, 0o644)
	write(t, root, "a.md", "a\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "A", Paths: []string{"a.md"}})
	time.AfterFunc(1500*time.Millisecond, func() { os.Remove(lock) })
	start := time.Now()
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	if commits(t, root) != 2 {
		t.Fatalf("no commit after the lock was released; log %s", h.log.String())
	}
	if d := time.Since(start); d < time.Second {
		t.Fatalf("committed after %v, before the lock was released", d)
	}
	if errs := h.errorLines(t); len(errs) != 0 {
		t.Fatalf("ERROR lines %v", errs)
	}
}

func TestCommitter_SPEC0008_R10_R11_indexLockHeld(t *testing.T) {
	root := newRepo(t)
	lock := filepath.Join(root, ".git", "index.lock")
	os.WriteFile(lock, nil, 0o644)
	t.Cleanup(func() { os.Remove(lock) })
	write(t, root, "a.md", "a\n")
	h := startCommitter(t, root, defaultAuthor)
	h.add(t, gitlog.Entry{Tool: "note_write", Title: "Locked title", Paths: []string{"a.md"}})
	start := time.Now()
	h.clock.Advance(2 * time.Second)
	h.idle(t)
	d := time.Since(start)
	if d < 3*time.Second || d > 8*time.Second {
		t.Fatalf("gave up after %v, want about 3 retries 1 s apart", d)
	}
	errs := h.errorLines(t)
	if len(errs) != 1 || errs[0]["class"] != "index_locked" {
		t.Fatalf("ERROR lines %v", errs)
	}
	assertNoContent(t, h, "Locked title", "a.md", "index.lock")
}

func TestCommitter_SPEC0008_R10_R11_timeoutAndGitError(t *testing.T) {
	for _, c := range []struct {
		name, script, class string
	}{
		{"timeout", `case " $* " in *" commit "*) sleep 3 ;; esac`, "timeout"},
		{"git error", `case " $* " in *" add "*) echo "fatal: a git problem" >&2; exit 128 ;; esac`, "git_error"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := newRepo(t)
			wrapGit(t, c.script)
			write(t, root, "a.md", "a\n")
			h := startCommitter(t, root, defaultAuthor)
			gitlog.SetTimeout(h.c, 500*time.Millisecond)
			h.add(t, gitlog.Entry{Tool: "note_write", Title: "T", Paths: []string{"a.md"}})
			h.clock.Advance(2 * time.Second)
			h.idle(t)
			errs := h.errorLines(t)
			if len(errs) != 1 || errs[0]["msg"] != "commit failed" || errs[0]["class"] != c.class {
				t.Fatalf("ERROR lines %v", errs)
			}
			assertNoContent(t, h, "a.md", "a git problem")
		})
	}
}

// wrapGit puts a git wrapper first on PATH that runs the shell snippet
// before the real git.
func wrapGit(t *testing.T, snippet string) {
	t.Helper()
	real, _ := lookGit(t)
	dir := t.TempDir()
	script := "#!/bin/sh\n" + snippet + "\nexec \"" + real + "\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
