package note_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

// bump sets the modification time of a file to a new, later value, so a
// change is seen even on file systems with coarse timestamps.
var clock = time.Now().Add(-time.Hour)

func bump(t *testing.T, path string) {
	t.Helper()
	clock = clock.Add(time.Second)
	if err := os.Chtimes(path, clock, clock); err != nil {
		t.Fatal(err)
	}
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
	bump(t, p)
}

func check(t *testing.T, s *note.Service) note.Stats {
	t.Helper()
	var got note.Stats
	if err := s.Call(func(_ *index.Index, st note.Stats) error { got = st; return nil }); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestFresh_SPEC0003_R10_cases(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "---\ntitle: A\n---\n", "b.md": "b", "c.md": "c"})
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	if st := check(t, s); st != (note.Stats{Unchanged: 3}) {
		t.Fatalf("no change: %+v", st)
	}

	write(t, root, "a.md", "---\ntitle: A2\n---\n") // edit
	bump(t, filepath.Join(root, "b.md"))            // touch, same content
	write(t, root, "d.md", "d")                     // add
	os.Remove(filepath.Join(root, "c.md"))          // delete
	st := check(t, s)
	// a edited, b touched with the same content, c deleted, d added.
	if st != (note.Stats{Added: 1, Changed: 1, Removed: 1, Unchanged: 1}) {
		t.Fatalf("stats %+v", st)
	}
	n, m, _ := s.Index().Get("a.md")
	if n.Title != "A2" {
		t.Fatalf("edit not seen: %q", n.Title)
	}
	_, mb, _ := s.Index().Get("b.md")
	fi, _ := os.Stat(filepath.Join(root, "b.md"))
	if !mb.ModTime.Equal(fi.ModTime()) {
		t.Fatalf("touched note meta not updated: %v vs %v", mb.ModTime, fi.ModTime())
	}
	_ = m

	// Rename: one removed, one added.
	if err := os.Rename(filepath.Join(root, "d.md"), filepath.Join(root, "e.md")); err != nil {
		t.Fatal(err)
	}
	if st := check(t, s); st.Added != 1 || st.Removed != 1 {
		t.Fatalf("rename stats %+v", st)
	}
	if !reflect.DeepEqual(s.Index().Paths(), []string{"a.md", "b.md", "e.md"}) {
		t.Fatalf("paths %v", s.Index().Paths())
	}

	// A note that grows past the limit is removed and skipped.
	write(t, root, "b.md", string(make([]byte, note.MaxNoteSize+1)))
	if st := check(t, s); st.Skipped != 1 || st.Removed != 1 {
		t.Fatalf("too large stats %+v", st)
	}
}

func TestFresh_SPEC0003_R7_unreadableRemoved(t *testing.T) {
	skipIfRoot(t)
	root := tree(t, map[string]string{"a.md": "a", "b.md": "b"})
	var buf bytes.Buffer
	s, _, _ := note.Open(root, logger(&buf))
	os.Chmod(filepath.Join(root, "b.md"), 0)
	bump(t, filepath.Join(root, "b.md"))
	defer os.Chmod(filepath.Join(root, "b.md"), 0o644)
	buf.Reset()
	if st := check(t, s); st.Skipped != 1 || st.Removed != 1 {
		t.Fatalf("stats %+v", st)
	}
	if len(find(records(t, &buf), "notes skipped")) != 1 {
		t.Fatalf("want one WARN for the newly skipped note: %s", buf.String())
	}
	buf.Reset()
	check(t, s)
	if len(find(records(t, &buf), "notes skipped")) != 0 {
		t.Fatalf("a note skipped before must not warn again: %s", buf.String())
	}
}

func TestFresh_SPEC0003_R12_serialised(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "a"})
	var buf bytes.Buffer
	s, _, _ := note.Open(root, logger(&buf))
	var wg sync.WaitGroup
	inside := 0
	var mu sync.Mutex
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				p := filepath.Join(root, fmt.Sprintf("g%d-%d.md", g, i))
				os.WriteFile(p, []byte("x"), 0o644)
				s.Call(func(idx *index.Index, _ note.Stats) error {
					mu.Lock()
					inside++
					if inside != 1 {
						t.Errorf("two calls inside the mutex")
					}
					mu.Unlock()
					_ = idx.Paths()
					mu.Lock()
					inside--
					mu.Unlock()
					return nil
				})
			}
		}(g)
	}
	wg.Wait()
	if n := s.Index().Len(); n != 161 {
		t.Fatalf("len %d", n)
	}
}

// TestFresh_SPEC0003_R13_equalsFreshOpen applies random file operations and
// compares the checked index with a fresh Open after each step.
func TestFresh_SPEC0003_R13_equalsFreshOpen(t *testing.T) {
	root := t.TempDir()
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil {
		t.Fatal(err)
	}
	r := rand.New(rand.NewSource(7))
	names := []string{"a.md", "b.md", "x/c.md", "x/y/d.md", ".h/e.md", "f.txt"}
	for step := 0; step < 150; step++ {
		rel := names[r.Intn(len(names))]
		p := filepath.Join(root, filepath.FromSlash(rel))
		switch r.Intn(4) {
		case 0, 1:
			write(t, root, rel, fmt.Sprintf("---\ntitle: T%d\n---\n%s\n", r.Intn(5), rel))
		case 2:
			os.Remove(p)
		case 3:
			other := names[r.Intn(len(names))]
			if _, err := os.Stat(p); err == nil {
				os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(other))), 0o755)
				os.Rename(p, filepath.Join(root, filepath.FromSlash(other)))
			}
		}
		check(t, s)
		fresh, _, err := note.Open(root, logger(&buf))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(s.Index().Paths(), fresh.Index().Paths()) {
			t.Fatalf("step %d: paths %v, fresh %v", step, s.Index().Paths(), fresh.Index().Paths())
		}
		for _, p := range fresh.Index().Paths() {
			n1, m1, _ := s.Index().Get(p)
			n2, m2, _ := fresh.Index().Get(p)
			if !reflect.DeepEqual(n1, n2) || m1 != m2 {
				t.Fatalf("step %d: %s differs", step, p)
			}
		}
	}
}

func TestFresh_SPEC0003_R10_debugLine(t *testing.T) {
	root := tree(t, map[string]string{"a.md": "a"})
	var buf bytes.Buffer
	s, _, _ := note.Open(root, logger(&buf))
	buf.Reset()
	check(t, s)
	l := find(records(t, &buf), "freshness check")
	if len(l) != 1 || l[0]["level"] != "DEBUG" || l[0]["unchanged"] != 1.0 || l[0]["duration_ms"] == nil {
		t.Fatalf("debug line %v", l)
	}
}
