package index_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
)

// randomNote returns a note source with random titles, aliases, tags, words
// and links, from small vocabularies so that names collide often.
func randomNote(r *rand.Rand) string {
	words := []string{"alpha", "beta", "gamma", "delta", "firewall", "proxy", "host"}
	s := fmt.Sprintf("---\ntitle: T%d\naliases: [A%d, old/p%d]\ntags: [t%d]\n---\n", r.Intn(6), r.Intn(6), r.Intn(4), r.Intn(3))
	for i := 0; i < 1+r.Intn(4); i++ {
		s += words[r.Intn(len(words))] + " [[T" + fmt.Sprint(r.Intn(6)) + "]] [[n" + fmt.Sprint(r.Intn(8)) + "]]\n"
	}
	s += "\n## Relations\n- depends_on [[A" + fmt.Sprint(r.Intn(6)) + "]]\n"
	return s
}

// TestIndex_SPEC0002_R1_R2_orderIndependent applies random Put and Remove
// sequences and compares every operation with an index built fresh from the
// final set of notes.
func TestIndex_SPEC0002_R1_R2_orderIndependent(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for round := 0; round < 30; round++ {
		x := index.New()
		final := map[string]string{}
		for step := 0; step < 60; step++ {
			p := fmt.Sprintf("d%d/n%d.md", r.Intn(2), r.Intn(8))
			if r.Intn(4) == 0 {
				x.Remove(p)
				delete(final, p)
				continue
			}
			src := randomNote(r)
			put(x, p, src)
			final[p] = src
		}
		fresh := index.New()
		for p, src := range final {
			put(fresh, p, src)
		}
		compare(t, round, x, fresh)
	}
}

func compare(t *testing.T, round int, a, b *index.Index) {
	t.Helper()
	if !reflect.DeepEqual(a.Paths(), b.Paths()) {
		t.Fatalf("round %d: paths %v vs %v", round, a.Paths(), b.Paths())
	}
	for _, target := range []string{"T0", "T3", "A1", "old/p2", "n4", "d0/n1", "missing"} {
		if ra, rb := a.Resolve(target), b.Resolve(target); !reflect.DeepEqual(ra, rb) {
			t.Fatalf("round %d: Resolve(%s) %+v vs %+v", round, target, ra, rb)
		}
	}
	for _, q := range []string{"alpha", "firewall proxy", "t1"} {
		ha, _ := a.Search(index.Query{Text: q, Limit: 50})
		hb, _ := b.Search(index.Query{Text: q, Limit: 50})
		if !reflect.DeepEqual(ha, hb) {
			t.Fatalf("round %d: Search(%s) differs", round, q)
		}
	}
	for _, p := range b.Paths() {
		ga, _ := a.Walk(p, 2, nil)
		gb, _ := b.Walk(p, 2, nil)
		if !reflect.DeepEqual(ga, gb) {
			t.Fatalf("round %d: Walk(%s) differs", round, p)
		}
	}
	if !reflect.DeepEqual(a.Links(), b.Links()) || !reflect.DeepEqual(a.List(index.Filter{}), b.List(index.Filter{})) {
		t.Fatalf("round %d: Links or List differ", round)
	}
}
