package note_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/note"
)

func TestDelete_SPEC0007_R24(t *testing.T) {
	root := tree(t, map[string]string{
		"hosts/a.md":   "---\ntitle: A\n---\n",
		"b.md":         "[[hosts/a]] and [[A]]\n",
		"c.md":         "## Relations\n- runs_on [[A]]\n",
		"d.md":         "[[hosts/a|label]]\n",
		"unrelated.md": "x\n",
	})
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))
	d, err := s.Delete(resolvePath("hosts/a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if d.Path != "hosts/a.md" || d.Title != "A" || d.Dangling != 3 {
		t.Fatalf("result %+v", d)
	}
	if _, err := os.Stat(filepath.Join(root, "hosts", "a.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still there: %v", err)
	}
	if fi, err := os.Stat(filepath.Join(root, "hosts")); err != nil || !fi.IsDir() {
		t.Fatal("empty folder removed")
	}
	if got := rec.changes(); !reflect.DeepEqual(got, []note.Change{{Tool: "note_delete", Title: "A", Paths: []string{"hosts/a.md"}}}) {
		t.Fatalf("committer got %+v", got)
	}
	assertIndexFresh(t, s)
	if _, err := s.Delete(resolvePath("hosts/a.md")); err == nil {
		t.Fatal("second delete succeeded")
	}
}
