package note_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

// benchRoot writes 5,000 notes of about 300 words (SPEC-0003 R16).
func benchRoot(b testing.TB) string {
	return benchRootN(b, 5000)
}

func benchRootN(b testing.TB, n int) string {
	b.Helper()
	root := b.TempDir()
	r := rand.New(rand.NewSource(1))
	for i := 0; i < n; i++ {
		var s strings.Builder
		fmt.Fprintf(&s, "---\ntitle: Note %d\n---\n", i)
		for w := 0; w < 300; w++ {
			fmt.Fprintf(&s, "w%d ", r.Intn(3000))
		}
		p := filepath.Join(root, fmt.Sprintf("f%d", i%50), fmt.Sprintf("n%d.md", i))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s.String()), 0o644); err != nil {
			b.Fatal(err)
		}
		// A settled notes root: files saved well before the check, so the
		// check needs no rehash of racily clean notes (SPEC-0003 R17).
		old := time.Now().Add(-time.Hour)
		os.Chtimes(p, old, old)
	}
	return root
}

// BenchmarkCheckRacy is a check right after a bulk change: every note was
// saved within the last 2 s, so every note is hashed again (SPEC-0003 R17).
func BenchmarkCheckRacy(b *testing.B) {
	root := benchRoot(b)
	now := time.Now()
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			os.Chtimes(p, now, now)
		}
		return nil
	})
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		s.Call(func(*index.Index, note.Stats) error { return nil })
	}
}

func BenchmarkOpen(b *testing.B) {
	root := benchRoot(b)
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if _, _, err := note.Open(root, logger(&buf)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheck(b *testing.B) {
	root := benchRoot(b)
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		s.Call(func(*index.Index, note.Stats) error { return nil })
	}
}

// TestPerf_SPEC0003_R16_benchmarksRun loads and checks a small root the way
// the benchmarks do; the 5,000-note results are recorded in the stage plan.
func TestPerf_SPEC0003_R16_benchmarksRun(t *testing.T) {
	root := benchRootN(t, 100)
	var buf bytes.Buffer
	s, _, err := note.Open(root, logger(&buf))
	if err != nil || s.Index().Len() != 100 {
		t.Fatalf("open: %v", err)
	}
	if err := s.Call(func(*index.Index, note.Stats) error { return nil }); err != nil {
		t.Fatal(err)
	}
}
