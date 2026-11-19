package note_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/agonzaNET/memctl/internal/note"
)

func TestSafePath_SPEC0003_R14_R15(t *testing.T) {
	root := tree(t, map[string]string{"ops/a.md": "a", "elsewhere/x.md": "x"})
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "ops"), filepath.Join(root, "inner")); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)

	ok := map[string]string{
		"a.md":                       "a.md",
		"ops/a.md":                   "ops/a.md",
		"new/folder/b.md":            "new/folder/b.md",
		"runbooks/Upgrade PAN-OS.md": "runbooks/Upgrade PAN-OS.md",
	}
	for rel, want := range ok {
		got, err := s.SafePath(rel)
		if err != nil || got != filepath.Join(s.Root(), filepath.FromSlash(want)) {
			t.Errorf("SafePath(%q) = %q, %v", rel, got, err)
		}
	}
	bad := []string{
		"", "/abs.md", `a\b.md`, "a\x00.md", "./a.md", "../a.md", "ops/../a.md",
		"ops//a.md", ".hidden.md", "ops/.d/a.md", "a.txt", "a", "ops/", "link/x.md", "inner/a.md",
	}
	for _, rel := range bad {
		if got, err := s.SafePath(rel); !errors.Is(err, note.ErrBadPath) {
			t.Errorf("SafePath(%q) = %q, %v; want ErrBadPath", rel, got, err)
		}
	}
	if after := snapshot(t, root); after != before {
		t.Fatalf("root changed:\n%s\n%s", before, after)
	}
}

// snapshot lists every entry under root.
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b bytes.Buffer
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			b.WriteString(p + "\n")
		}
		return nil
	})
	return b.String()
}
