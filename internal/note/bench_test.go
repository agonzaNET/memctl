package note_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

// benchRoot writes 5,000 notes of about 300 words (SPEC-0003 R16).
func benchRoot(b *testing.B) string {
	b.Helper()
	root := b.TempDir()
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 5000; i++ {
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
	}
	return root
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
