package gitlog_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScan_SPEC0008_R15_uncommittedNotes(t *testing.T) {
	for _, sub := range []string{"", "notes"} {
		repo := newRepo(t)
		root := filepath.Join(repo, sub)
		write(t, root, ".gitignore", "ignored/\n")
		write(t, root, "a.md", "a\n")
		write(t, root, "c.md", "c\n")
		write(t, root, "old.md", "o\n")
		git(t, repo, "add", ".")
		git(t, repo, "commit", "-q", "-m", "notes")
		// While memctl is not running:
		write(t, root, "a.md", "edited\n")
		write(t, root, "b.md", "new\n")
		if err := os.Remove(filepath.Join(root, "c.md")); err != nil {
			t.Fatal(err)
		}
		git(t, repo, "mv", filepath.Join(sub, "old.md"), filepath.Join(sub, "renamed.md"))
		write(t, root, "x.txt", "not a note\n")
		write(t, root, "ignored/d.md", "ignored\n")
		if sub != "" {
			write(t, repo, "outside.md", "outside the notes root\n")
		}

		h := startCommitter(t, root, defaultAuthor)
		isNote := func(rel string) bool { return strings.HasSuffix(rel, ".md") && !strings.HasPrefix(rel, "ignored/") }
		if err := h.c.Scan(context.Background(), isNote); err != nil {
			t.Fatal(err)
		}
		h.idle(t)
		h.clock.Advance(2 * time.Second)
		h.idle(t)
		want := "memory: 5 external changes\n\n- a.md\n- b.md\n- c.md\n- old.md\n- renamed.md\n\n"
		if got := git(t, repo, "log", "-1", "--format=%B"); got != want {
			t.Fatalf("root %q: message %q", sub, got)
		}
		left := git(t, repo, "status", "--porcelain", "--untracked-files=all")
		for _, w := range []string{"x.txt", "outside.md"} {
			if w == "outside.md" && sub == "" {
				continue
			}
			if !strings.Contains(left, w) {
				t.Fatalf("root %q: %s was committed:\n%s", sub, w, left)
			}
		}
		if strings.Contains(left, ".md") && !strings.Contains(left, "outside.md") {
			t.Fatalf("root %q: notes left uncommitted:\n%s", sub, left)
		}
	}
}
