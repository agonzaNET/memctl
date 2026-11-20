package index_test

import (
	"reflect"
	"sort"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
)

func TestSearch_SPEC0002_R29_nearNames(t *testing.T) {
	x := index.New()
	put(x, "a.md", "---\ntitle: Alpha\n---\nrouter cisco1\n")
	put(x, "b.md", "---\ntitle: Bravo\n---\nrouter cisco12\n")
	put(x, "c.md", "---\ntitle: Charlie\n---\nrouter ciscoasa\n")
	put(x, "d.md", "---\ntitle: Delta\n---\nsite eu1\n")
	put(x, "e.md", "---\ntitle: Echo\n---\ncell c1\n")
	put(x, "h.md", "---\ntitle: Hotel\n---\nbox cisco1a\n")

	find := func(text string, near bool) []string {
		t.Helper()
		r, err := x.Search(index.Query{Text: text, NearNames: near})
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		var out []string
		for _, h := range r.Hits {
			out = append(out, h.Path)
		}
		sort.Strings(out)
		return out
	}
	tests := []struct {
		query string
		near  bool
		want  []string
	}{
		{"cisco", true, []string{"a.md", "b.md"}},
		{"CISCO", true, []string{"a.md", "b.md"}},
		{"eu", true, []string{"d.md"}},
		{"c", true, nil},          // one letter is too short
		{"cisco-core", true, nil}, // a compound is not expanded
		{"cisco1", true, []string{"a.md"}},
		{"cisco", false, nil},
		{"eu", false, nil},
	}
	for _, tt := range tests {
		if got := find(tt.query, tt.near); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q near=%v: %v, want %v", tt.query, tt.near, got, tt.want)
		}
	}

	y := index.New()
	put(y, "both.md", "---\ntitle: P\n---\ncisco cisco1\n")
	put(y, "near.md", "---\ntitle: Q\n---\nintro\ncisco1 other\n")
	r, err := y.Search(index.Query{Text: "cisco", NearNames: true})
	if err != nil || len(r.Hits) != 2 || r.Hits[0].Path != "both.md" || !(r.Hits[0].Score > r.Hits[1].Score) {
		t.Fatalf("ranking %+v, %v", r.Hits, err)
	}
	if r.Hits[1].Snippet != "cisco1 other" {
		t.Fatalf("near name not a query term for the snippet: %q", r.Hits[1].Snippet)
	}
}
