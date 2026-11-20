package note_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

// recCommitter records every change handed to it.
type recCommitter struct {
	mu       sync.Mutex
	got      []note.Change
	external [][]string
	waited   []string
}

func (r *recCommitter) External(paths []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.external = append(r.external, append([]string(nil), paths...))
}

func (r *recCommitter) WaitExternal(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.waited = append(r.waited, path)
	return nil
}

func (r *recCommitter) externals() ([][]string, []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([][]string(nil), r.external...), append([]string(nil), r.waited...)
}

func (r *recCommitter) Enqueue(c note.Change) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, c)
}

func (r *recCommitter) changes() []note.Change {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]note.Change(nil), r.got...)
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func openWith(t *testing.T, root string, opts ...note.Option) *note.Service {
	t.Helper()
	s, _, err := note.Open(root, logger(new(bytes.Buffer)), opts...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// dotFiles lists the names starting with "." in dir (temporary files).
func dotFiles(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") && e.Name() != ".memory" {
			out = append(out, e.Name())
		}
	}
	return out
}

// assertIndexFresh checks that the index equals a fresh load of the root
// (SPEC-0007 R4) and that the next freshness check changes nothing.
func assertIndexFresh(t *testing.T, s *note.Service) {
	t.Helper()
	var st note.Stats
	s.Call(func(idx *index.Index, got note.Stats) error { st = got; return nil })
	if st.Added+st.Changed+st.Removed != 0 {
		t.Fatalf("freshness check after the write changed the index: %+v", st)
	}
	assertIndexMatchesDisk(t, s)
}

// assertIndexMatchesDisk runs a freshness check, which picks up changes
// made outside the service, and compares the index with a fresh load.
func assertIndexMatchesDisk(t *testing.T, s *note.Service) {
	t.Helper()
	s.Call(func(*index.Index, note.Stats) error { return nil })
	fresh := openWith(t, s.Root())
	if got, want := s.Index().Paths(), fresh.Index().Paths(); !reflect.DeepEqual(got, want) {
		t.Fatalf("paths %v, fresh load %v", got, want)
	}
	for _, p := range fresh.Index().Paths() {
		gn, gm, _ := s.Index().Get(p)
		fn, fm, _ := fresh.Index().Get(p)
		if !reflect.DeepEqual(gn, fn) || gm.SHA256 != fm.SHA256 || gm.Size != fm.Size || !gm.ModTime.Equal(fm.ModTime) {
			t.Fatalf("%s differs from a fresh load:\n%+v %+v\n%+v %+v", p, gn, gm, fn, fm)
		}
	}
}

func TestWrite_SPEC0007_R1_freshnessCheckFirst(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "---\ntitle: A\n---\n"})
	s := openWith(t, root)
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("editor\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var seen bool
	err := note.Mutate(s, func(idx *index.Index) (note.Change, error) {
		_, _, seen = idx.Get("b.md")
		return note.Change{}, nil
	})
	if err != nil || !seen {
		t.Fatalf("write did not see the editor's file: %v %v", seen, err)
	}
}

func TestWrite_SPEC0007_R2_R4_R5_newAndReplace(t *testing.T) {
	root := tree(t, map[string]string{"old.md": "old\n"})
	if err := os.Chmod(filepath.Join(root, "old.md"), 0o640); err != nil {
		t.Fatal(err)
	}
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))

	err := note.Mutate(s, func(idx *index.Index) (note.Change, error) {
		if _, err := note.PutFile(s, "new.md", []byte("---\ntitle: New\n---\nhello\n"), ""); err != nil {
			return note.Change{}, err
		}
		_, m, _ := idx.Get("old.md")
		if _, err := note.PutFile(s, "old.md", []byte("replaced\n"), m.SHA256); err != nil {
			return note.Change{}, err
		}
		return note.Change{Tool: "test", Title: "New", Paths: []string{"new.md", "old.md"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "new.md")); string(b) != "---\ntitle: New\n---\nhello\n" {
		t.Fatalf("new.md %q", b)
	}
	if m := mode(t, filepath.Join(root, "new.md")); m != 0o644 {
		t.Fatalf("new file mode %v, want 0644", m)
	}
	if m := mode(t, filepath.Join(root, "old.md")); m != 0o640 {
		t.Fatalf("replaced file mode %v, want 0640 kept", m)
	}
	if d := dotFiles(t, root); len(d) != 0 {
		t.Fatalf("temporary files left: %v", d)
	}
	n, m, ok := s.Index().Get("new.md")
	if !ok || n.Title != "New" || m.SHA256 != sum([]byte("---\ntitle: New\n---\nhello\n")) {
		t.Fatalf("index entry %+v %+v %v", n, m, ok)
	}
	assertIndexFresh(t, s)
	want := []note.Change{{Tool: "test", Title: "New", Paths: []string{"new.md", "old.md"}}}
	if got := rec.changes(); !reflect.DeepEqual(got, want) {
		t.Fatalf("committer got %+v, want %+v", got, want)
	}
}

func TestWrite_SPEC0007_R2_R5_interruptedBeforeRename(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "original\n"})
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))
	boom := errors.New("injected")
	note.SetHooks(s, nil, func(string) error { return boom })

	for _, rel := range []string{"a.md", "b.md"} {
		err := note.Mutate(s, func(idx *index.Index) (note.Change, error) {
			prev := ""
			if _, m, ok := idx.Get(rel); ok {
				prev = m.SHA256
			}
			if _, err := note.PutFile(s, rel, []byte("new\n"), prev); err != nil {
				return note.Change{}, err
			}
			return note.Change{Tool: "test", Paths: []string{rel}}, nil
		})
		if !errors.Is(err, boom) {
			t.Fatalf("%s: err %v, want the injected fault", rel, err)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.md")); string(b) != "original\n" {
		t.Fatalf("target changed: %q", b)
	}
	if _, err := os.Stat(filepath.Join(root, "b.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new target exists after a failed write: %v", err)
	}
	if d := dotFiles(t, root); len(d) != 0 {
		t.Fatalf("temporary files left: %v", d)
	}
	if got := rec.changes(); len(got) != 0 {
		t.Fatalf("failed write handed %+v to the committer", got)
	}
	assertIndexFresh(t, s)
}

func TestWrite_SPEC0007_R3_conflictBeforeRename(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "original\n"})
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))
	note.SetHooks(s, func(abs string) {
		os.WriteFile(abs, []byte("human edit\n"), 0o644)
	}, nil)

	err := note.Mutate(s, func(idx *index.Index) (note.Change, error) {
		_, m, _ := idx.Get("a.md")
		if _, err := note.PutFile(s, "a.md", []byte("tool\n"), m.SHA256); err != nil {
			return note.Change{}, err
		}
		return note.Change{Tool: "test", Paths: []string{"a.md"}}, nil
	})
	if !errors.Is(err, note.ErrConflict) {
		t.Fatalf("err %v, want ErrConflict", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.md")); string(b) != "human edit\n" {
		t.Fatalf("human edit lost: %q", b)
	}
	if d := dotFiles(t, root); len(d) != 0 {
		t.Fatalf("temporary files left: %v", d)
	}
	if len(rec.changes()) != 0 {
		t.Fatal("conflict handed a change to the committer")
	}

	// A new note whose path appeared in the meantime is a conflict too.
	note.SetHooks(s, func(abs string) { os.WriteFile(abs, []byte("raced\n"), 0o644) }, nil)
	err = note.Mutate(s, func(idx *index.Index) (note.Change, error) {
		_, err := note.PutFile(s, "c.md", []byte("tool\n"), "")
		return note.Change{}, err
	})
	if !errors.Is(err, note.ErrConflict) {
		t.Fatalf("new path raced: err %v, want ErrConflict", err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "c.md")); string(b) != "raced\n" {
		t.Fatalf("raced file overwritten: %q", b)
	}
}

func TestWrite_SPEC0007_R4_R5_remove(t *testing.T) {
	root := tree(t, map[string]string{"dir/a.md": "a\n", "b.md": "[[dir/a]]\n"})
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))
	err := note.Mutate(s, func(idx *index.Index) (note.Change, error) {
		_, m, _ := idx.Get("dir/a.md")
		if err := note.RemoveFile(s, "dir/a.md", m.SHA256); err != nil {
			return note.Change{}, err
		}
		return note.Change{Tool: "test", Title: "a", Paths: []string{"dir/a.md"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "dir", "a.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file still there: %v", err)
	}
	// R6: the empty folder stays.
	if fi, err := os.Stat(filepath.Join(root, "dir")); err != nil || !fi.IsDir() {
		t.Fatalf("empty folder removed: %v", err)
	}
	assertIndexFresh(t, s)
	if len(rec.changes()) != 1 {
		t.Fatalf("committer got %+v", rec.changes())
	}
}

func TestWrite_SPEC0007_R6_nestedFolders(t *testing.T) {
	root := t.TempDir()
	s := openWith(t, root)
	err := note.Mutate(s, func(idx *index.Index) (note.Change, error) {
		_, err := note.PutFile(s, "a/b/c.md", []byte("x\n"), "")
		return note.Change{}, err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"a", "a/b"} {
		if m := mode(t, filepath.Join(root, d)); m != 0o755 {
			t.Fatalf("%s mode %v, want 0755", d, m)
		}
	}
	assertIndexFresh(t, s)
}

func TestWrite_SPEC0007_R7_timestamps(t *testing.T) {
	zone := time.FixedZone("X", 5*3600+30*60)
	clock := func() time.Time { return time.Date(2026, 11, 19, 8, 7, 6, 999, zone) }
	s := openWith(t, t.TempDir(), note.WithClock(clock))
	if got := note.Stamp(s); got != "2026-11-19T08:07:06+05:30" {
		t.Fatalf("stamp %q", got)
	}
	utc := openWith(t, t.TempDir(), note.WithClock(func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }))
	if got := note.Stamp(utc); got != "2026-01-02T03:04:05+00:00" {
		t.Fatalf("UTC stamp %q", got)
	}
}

func TestWrite_SPEC0007_R28_targetMustBeANote(t *testing.T) {
	root := tree(t, map[string]string{
		".gitignore":      "drafts/\n*.tmp.md\n",
		"keep/.gitignore": "secret.md\n",
	})
	s := openWith(t, root)
	cases := []struct{ rel, reason string }{
		{"drafts/x.md", ".gitignore"},
		{"a.tmp.md", ".gitignore"},
		{"keep/secret.md", ".gitignore"},
		{".hidden/x.md", `"."`},
		{"ok/.x.md", `"."`},
		{"../x.md", "inside the notes root"},
		{"x.txt", ".md"},
	}
	for _, c := range cases {
		_, err := note.TargetPath(s, c.rel)
		var pe *note.PathError
		if !errors.As(err, &pe) || !errors.Is(err, note.ErrBadPath) {
			t.Errorf("%s: err %v, want a PathError", c.rel, err)
			continue
		}
		if !strings.Contains(pe.Reason, c.reason) {
			t.Errorf("%s: reason %q does not mention %q", c.rel, pe.Reason, c.reason)
		}
	}
	if abs, err := note.TargetPath(s, "keep/fine.md"); err != nil || abs != filepath.Join(s.Root(), "keep", "fine.md") {
		t.Fatalf("fine path: %q %v", abs, err)
	}
}
