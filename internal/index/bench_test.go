package index_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

// syntheticNotes returns n notes of about 300 words, with links between
// them, for the performance targets of SPEC-0002 R22.
func syntheticNotes(n int) map[string]string {
	r := rand.New(rand.NewSource(42))
	vocab := make([]string, 3000)
	for i := range vocab {
		vocab[i] = fmt.Sprintf("w%d", i)
	}
	out := make(map[string]string, n)
	for i := 0; i < n; i++ {
		var b strings.Builder
		fmt.Fprintf(&b, "---\ntitle: Note %d\ntags: [t%d, u%d]\naliases: [N%d]\n---\n# Note %d\n\n", i, i%20, i%7, i, i)
		for w := 0; w < 300; w++ {
			b.WriteString(vocab[r.Intn(len(vocab))])
			if w%15 == 14 {
				fmt.Fprintf(&b, " [[Note %d]]\n", r.Intn(n))
			} else {
				b.WriteByte(' ')
			}
		}
		fmt.Fprintf(&b, "\n\n## Observations\n- [fact] w%d and w%d matter\n\n## Relations\n- depends_on [[f%d/n%d]]\n", r.Intn(3000), r.Intn(3000), r.Intn(50), r.Intn(n))
		out[fmt.Sprintf("f%d/n%d.md", i%50, i)] = b.String()
	}
	return out
}

func build(notes map[string]string) *index.Index {
	x := index.New()
	for p, src := range notes {
		x.Put(p, parse.Parse(p, []byte(src)), index.Meta{})
	}
	return x
}

func BenchmarkBuild(b *testing.B) {
	notes := syntheticNotes(5000)
	parsed := make(map[string]parse.Note, len(notes))
	for p, src := range notes {
		parsed[p] = parse.Parse(p, []byte(src))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := index.New()
		for p, n := range parsed {
			x.Put(p, n, index.Meta{})
		}
	}
}

func BenchmarkSearch(b *testing.B) {
	x := build(syntheticNotes(5000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Search(index.Query{Text: fmt.Sprintf("w%d w%d w%d", i%3000, (i*7)%3000, (i*13)%3000)}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWalk(b *testing.B) {
	x := build(syntheticNotes(5000))
	paths := x.Paths()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := x.Walk(paths[i%len(paths)], 3, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkWalkCold walks right after a change, when the resolved edges are
// computed again.
func BenchmarkWalkCold(b *testing.B) {
	notes := syntheticNotes(5000)
	x := build(notes)
	paths := x.Paths()
	n := parse.Parse(paths[0], []byte(notes[paths[0]]))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x.Put(paths[0], n, index.Meta{})
		if _, err := x.Walk(paths[i%len(paths)], 3, nil); err != nil {
			b.Fatal(err)
		}
	}
}
