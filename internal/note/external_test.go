package note_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

func TestFresh_SPEC0003_R18_externalPaths(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "a\n", "c.md": "c\n", "d.md": "d\n"})
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))
	call := func() {
		t.Helper()
		if err := s.Call(func(*index.Index, note.Stats) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	call()
	if ext, _ := rec.externals(); len(ext) != 0 {
		t.Fatalf("the startup load and an unchanged check hand over %v", ext)
	}

	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("edited in an editor\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "c.md")); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(root, "d.md"), later, later); err != nil {
		t.Fatal(err)
	}
	call()
	ext, _ := rec.externals()
	if !reflect.DeepEqual(ext, [][]string{{"a.md", "b.md", "c.md"}}) {
		t.Fatalf("external %v", ext)
	}

	// A tool write is not an external change, and it waits for a pending
	// external change of its own path first (SPEC-0008 R16).
	if _, err := s.Write(note.NewNote{Title: "N", Body: "x"}); err != nil {
		t.Fatal(err)
	}
	call()
	ext, waited := rec.externals()
	if len(ext) != 1 {
		t.Fatalf("a tool write was handed over as external: %v", ext)
	}
	if !reflect.DeepEqual(waited, []string{"N.md"}) {
		t.Fatalf("waited %v", waited)
	}
}
