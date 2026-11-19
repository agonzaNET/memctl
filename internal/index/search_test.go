package index_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
)

func TestTokenize_SPEC0002_R7(t *testing.T) {
	cases := map[string][]string{
		"PA-850":                   {"pa", "850"},
		"C#":                       {"c"},
		"10.11.0.51":               {"10", "11", "0", "51"},
		"Größe der Datei":          {"größe", "der", "datei"},
		"Upgrade the FIREWALLS!":   {"upgrade", "the", "firewalls"},
		"":                         nil,
		"--- !!":                   nil,
		"snake_case and tab\there": {"snake", "case", "and", "tab", "here"},
	}
	for in, want := range cases {
		if got := index.Tokenize(in); !reflect.DeepEqual(got, want) {
			t.Errorf("Tokenize(%q) = %q, want %q", in, got, want)
		}
	}
}

// bm25f computes one term's contribution, written out from the spec, for
// the hand-checked fixture below.
func bm25f(n, df float64, fields []struct{ w, tf, len, avg float64 }) float64 {
	const k1, b = 1.2, 0.75
	idf := math.Log(1 + (n-df+0.5)/(df+0.5))
	wtf := 0.0
	for _, f := range fields {
		if f.tf == 0 {
			continue
		}
		wtf += f.w * f.tf / (1 - b + b*f.len/f.avg)
	}
	return idf * wtf / (k1 + wtf)
}

func TestSearch_SPEC0002_R8_R9_scores(t *testing.T) {
	x := index.New()
	// Note A: title "alpha beta" (2), body "gamma" (1).
	put(x, "a.md", "---\ntitle: alpha beta\n---\ngamma\n")
	// Note B: title "delta" (1), body "alpha alpha gamma" (3).
	put(x, "b.md", "---\ntitle: delta\n---\nalpha alpha gamma\n")
	// Note C: title "epsilon" (1), tags "alpha" (1), body empty.
	put(x, "c.md", "---\ntitle: epsilon\ntags: [alpha]\n---\n")

	hits, err := x.Search(index.Query{Text: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	// Field averages over N=3: title (2+1+1)/3, tags (0+0+1)/3, body (1+3+0)/3.
	type f = struct{ w, tf, len, avg float64 }
	want := map[string]float64{
		"a.md": bm25f(3, 3, []f{{3, 1, 2, 4.0 / 3}}),
		"b.md": bm25f(3, 3, []f{{1, 2, 3, 4.0 / 3}}),
		"c.md": bm25f(3, 3, []f{{2, 1, 1, 1.0 / 3}}),
	}
	if len(hits) != 3 {
		t.Fatalf("hits %+v", hits)
	}
	for _, h := range hits {
		if math.Abs(h.Score-want[h.Path]) > 1e-9 {
			t.Errorf("%s score %.12f, want %.12f", h.Path, h.Score, want[h.Path])
		}
	}
	for i := 1; i < len(hits); i++ {
		if hits[i-1].Score < hits[i].Score {
			t.Fatalf("not ordered by score: %+v", hits)
		}
	}

	// A title match outranks the same term only in the body.
	y := index.New()
	put(y, "t.md", "---\ntitle: zeta\n---\nsome words here\n")
	put(y, "u.md", "---\ntitle: other\n---\nzeta some words\n")
	hits, _ = y.Search(index.Query{Text: "zeta"})
	if len(hits) != 2 || hits[0].Path != "t.md" {
		t.Fatalf("title should outrank body: %+v", hits)
	}
}

func TestSearch_SPEC0002_R10_orSemanticsAndOrder(t *testing.T) {
	x := index.New()
	put(x, "one.md", "---\ntitle: n1\n---\nred\n")
	put(x, "two.md", "---\ntitle: n2\n---\nred green\n")
	put(x, "zz.md", "---\ntitle: n3\n---\nred\n")
	put(x, "none.md", "---\ntitle: n4\n---\nblue\n")
	hits, _ := x.Search(index.Query{Text: "red green yellow"})
	var paths []string
	for _, h := range hits {
		paths = append(paths, h.Path)
	}
	// two.md has two terms; one.md and zz.md tie and are ordered by path.
	if !reflect.DeepEqual(paths, []string{"two.md", "one.md", "zz.md"}) {
		t.Fatalf("paths %v", paths)
	}
}

func TestSearch_SPEC0002_R11_R12_filtersAndLimit(t *testing.T) {
	x := index.New()
	put(x, "ops/a.md", "---\ntype: Runbook\ntags: [fw, ha]\n---\nword\n")
	put(x, "ops/x/b.md", "---\ntype: runbook\ntags: [FW]\n---\nword\n")
	put(x, "opsx/c.md", "---\ntype: runbook\ntags: [fw, ha]\n---\nword\n")
	put(x, "d.md", "---\ntype: host\n---\nword\n")
	paths := func(q index.Query) []string {
		hits, err := x.Search(q)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, h := range hits {
			out = append(out, h.Path)
		}
		return out
	}
	if got := paths(index.Query{Text: "word", Type: "RUNBOOK"}); len(got) != 3 {
		t.Fatalf("type: %v", got)
	}
	if got := paths(index.Query{Text: "word", Tags: []string{"fw", "HA"}}); !reflect.DeepEqual(got, []string{"ops/a.md", "opsx/c.md"}) {
		t.Fatalf("tags: %v", got)
	}
	if got := paths(index.Query{Text: "word", Folder: "ops"}); !reflect.DeepEqual(got, []string{"ops/a.md", "ops/x/b.md"}) {
		t.Fatalf("folder: %v", got)
	}

	y := index.New()
	for i := 0; i < 60; i++ {
		put(y, "n"+strings.Repeat("x", i)+".md", "word\n")
	}
	if got := len(paths2(t, y, index.Query{Text: "word"})); got != 10 {
		t.Fatalf("default limit gave %d", got)
	}
	if got := len(paths2(t, y, index.Query{Text: "word", Limit: 500})); got != 50 {
		t.Fatalf("cap gave %d", got)
	}
	if got := len(paths2(t, y, index.Query{Text: "word", Limit: 3})); got != 3 {
		t.Fatalf("limit 3 gave %d", got)
	}
}

func paths2(t *testing.T, x *index.Index, q index.Query) []index.Hit {
	t.Helper()
	h, err := x.Search(q)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestSearch_SPEC0002_R13_emptyQuery(t *testing.T) {
	x := fixture()
	for _, q := range []string{"--- !!", "", "   "} {
		if hits, err := x.Search(index.Query{Text: q}); !errors.Is(err, index.ErrEmptyQuery) || hits != nil {
			t.Fatalf("query %q: %v %v", q, hits, err)
		}
	}
}

func TestSearch_SPEC0002_R14_snippets(t *testing.T) {
	x := index.New()
	put(x, "obs.md", "---\ntitle: o\n---\nintro about firewall\n\n## Observations\n- [risk] firewall upgrade must be passive first\n- [note] firewall\n")
	put(x, "body.md", "---\ntitle: b\n---\nfirst line\nthe upgrade of the firewall today\nfirewall only\n")
	put(x, "title.md", "---\ntitle: firewall upgrade\n---\n\n\nFirst real line\n")
	put(x, "empty.md", "---\ntitle: upgrade firewall plan\n---\n")
	put(x, "long.md", "---\ntitle: l\n---\nfirewall "+strings.Repeat("é", 300)+"\n")

	hits, _ := x.Search(index.Query{Text: "firewall upgrade", Limit: 50})
	got := map[string]index.Hit{}
	for _, h := range hits {
		got[h.Path] = h
	}
	check := func(p, snippet string, line int) {
		t.Helper()
		if h := got[p]; h.Snippet != snippet || h.SnippetLine != line {
			t.Errorf("%s snippet %q line %d, want %q line %d", p, h.Snippet, h.SnippetLine, snippet, line)
		}
	}
	check("obs.md", "- [risk] firewall upgrade must be passive first", 7)
	check("body.md", "the upgrade of the firewall today", 5)
	check("title.md", "First real line", 6)
	check("empty.md", "", 0)
	long := got["long.md"]
	if n := len([]rune(long.Snippet)); n != 201 || !strings.HasSuffix(long.Snippet, "…") || long.SnippetLine != 4 {
		t.Errorf("long snippet: %d runes, %q", n, long.Snippet[:20])
	}
}
