package index_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
)

func graphFixture() *index.Index {
	x := index.New()
	put(x, "s.md", "---\ntitle: Start\n---\nmentions [[b]] and [[s]] and [[nowhere]] and [[dup]]\n\n## Relations\n- depends_on [[a]]\n")
	put(x, "a.md", "---\ntitle: A\n---\n\n## Relations\n- runs_on [[c]]\n")
	put(x, "b.md", "---\ntitle: B\n---\n")
	put(x, "c.md", "---\ntitle: C\n---\nnext [[d]]\n")
	put(x, "d.md", "---\ntitle: D\n---\n")
	put(x, "in.md", "---\ntitle: In\n---\n\n## Relations\n- see_also [[s]]\n")
	put(x, "x/dup.md", "---\ntitle: X\n---\n")
	put(x, "y/dup.md", "---\ntitle: Y\n---\n")
	return x
}

type nd struct {
	p string
	d int
}

func nodes(g index.Graph) []nd {
	var out []nd
	for _, n := range g.Nodes {
		out = append(out, nd{n.Path, n.Distance})
	}
	return out
}

func edges(g index.Graph) []string {
	var out []string
	for _, e := range g.Edges {
		out = append(out, fmt.Sprintf("%s:%d %s %s %s", e.Source, e.Line, e.Type, e.Target, e.Resolution.Status))
	}
	return out
}

func TestWalk_SPEC0002_R15_R16_R17_depths(t *testing.T) {
	x := graphFixture()
	g, err := x.Walk("s.md", 0, nil, index.EdgesAll)
	if err != nil {
		t.Fatal(err)
	}
	wantNodes := []nd{{"s.md", 0}, {"a.md", 1}, {"b.md", 1}, {"in.md", 1}}
	if !reflect.DeepEqual(nodes(g), wantNodes) {
		t.Fatalf("depth 1 nodes %v", nodes(g))
	}
	wantEdges := []string{
		"in.md:6 see_also s resolved",
		"s.md:4 links_to b resolved",
		"s.md:4 links_to dup ambiguous",
		"s.md:4 links_to nowhere dangling",
		"s.md:7 depends_on a resolved",
	}
	if !reflect.DeepEqual(edges(g), wantEdges) {
		t.Fatalf("depth 1 edges %v", edges(g))
	}

	g, _ = x.Walk("s.md", 3, nil, index.EdgesAll)
	if !reflect.DeepEqual(nodes(g), []nd{{"s.md", 0}, {"a.md", 1}, {"b.md", 1}, {"in.md", 1}, {"c.md", 2}, {"d.md", 3}}) {
		t.Fatalf("depth 3 nodes %v", nodes(g))
	}

	g, _ = x.Walk("s.md", 2, []string{"depends_on", "runs_on"}, index.EdgesAll)
	if !reflect.DeepEqual(nodes(g), []nd{{"s.md", 0}, {"a.md", 1}, {"c.md", 2}}) {
		t.Fatalf("relTypes nodes %v", nodes(g))
	}
	if !reflect.DeepEqual(edges(g), []string{"a.md:6 runs_on c resolved", "s.md:7 depends_on a resolved"}) {
		t.Fatalf("relTypes edges %v", edges(g))
	}
}

func TestWalk_SPEC0002_R17_edgesAmongListed(t *testing.T) {
	x := index.New()
	put(x, "s.md", "---\ntitle: S\n---\n[[a]] [[b]] [[s]]\n")
	put(x, "a.md", "---\ntitle: A\n---\n[[c]]\n")
	put(x, "b.md", "---\ntitle: B\n---\n[[d]]\n")
	put(x, "c.md", "---\ntitle: C\n---\nsee [[d]]\nagain [[d]]\n\n## Relations\n- depends_on [[d]]\n")
	put(x, "d.md", "---\ntitle: D\n---\n")
	g, err := x.Walk("s.md", 2, nil, index.EdgesAll)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nodes(g), []nd{{"s.md", 0}, {"a.md", 1}, {"b.md", 1}, {"c.md", 2}, {"d.md", 2}}) {
		t.Fatalf("nodes %v", nodes(g))
	}
	want := []string{
		"a.md:4 links_to c resolved",
		"b.md:4 links_to d resolved",
		"c.md:4 links_to d resolved", // the link on line 5 is the same edge
		"c.md:8 depends_on d resolved",
		"s.md:4 links_to a resolved",
		"s.md:4 links_to b resolved",
	}
	if !reflect.DeepEqual(edges(g), want) {
		t.Fatalf("edges %v", edges(g))
	}
	g, _ = x.Walk("s.md", 2, []string{"links_to"}, index.EdgesAll)
	for _, e := range edges(g) {
		if e == "c.md:8 depends_on d resolved" {
			t.Fatalf("relTypes let another type through: %v", edges(g))
		}
	}
	if len(g.Edges) != 5 {
		t.Fatalf("relTypes edges %v", edges(g))
	}
}

func TestWalk_SPEC0002_R16_R19_errors(t *testing.T) {
	x := graphFixture()
	for _, d := range []int{-1, 4} {
		if _, err := x.Walk("s.md", d, nil, index.EdgesAll); !errors.Is(err, index.ErrBadDepth) {
			t.Fatalf("depth %d: %v", d, err)
		}
	}
	if _, err := x.Walk("missing.md", 1, nil, index.EdgesAll); !errors.Is(err, index.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestWalk_SPEC0002_R18_truncated(t *testing.T) {
	x := index.New()
	src := "---\ntitle: hub\n---\n"
	for i := 0; i < 250; i++ {
		src += fmt.Sprintf("[[n%03d]]\n", i)
		put(x, fmt.Sprintf("n%03d.md", i), "x\n")
	}
	put(x, "hub.md", src)
	g, err := x.Walk("hub.md", 1, nil, index.EdgesAll)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 200 || !g.Truncated {
		t.Fatalf("nodes %d truncated %v", len(g.Nodes), g.Truncated)
	}
}

func TestList_SPEC0002_R20(t *testing.T) {
	x := index.New()
	put(x, "ops/b.md", "---\ntitle: B\ntype: Runbook\ntags: [fw]\nupdated: 2026-11-19\n---\n")
	put(x, "ops/a.md", "---\ntitle: A\ntype: host\n---\n")
	put(x, "other/c.md", "---\ntitle: C\ntype: runbook\ntags: [FW, x]\n---\n")
	list := func(f index.Filter) []string {
		var out []string
		for _, e := range x.List(f) {
			out = append(out, e.Path)
		}
		return out
	}
	if got := list(index.Filter{}); !reflect.DeepEqual(got, []string{"ops/a.md", "ops/b.md", "other/c.md"}) {
		t.Fatalf("all %v", got)
	}
	if got := list(index.Filter{Folder: "ops"}); !reflect.DeepEqual(got, []string{"ops/a.md", "ops/b.md"}) {
		t.Fatalf("folder %v", got)
	}
	if got := list(index.Filter{Type: "RUNBOOK", Tag: "fw"}); !reflect.DeepEqual(got, []string{"ops/b.md", "other/c.md"}) {
		t.Fatalf("type+tag %v", got)
	}
	e := x.List(index.Filter{Folder: "ops", Type: "runbook"})[0]
	if e.Title != "B" || !reflect.DeepEqual(e.Tags, []string{"fw"}) || !e.Updated.Equal(time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("entry %+v", e)
	}
}

func TestLinks_SPEC0002_R21_report(t *testing.T) {
	x := graphFixture()
	var got []string
	for _, l := range x.Links() {
		got = append(got, fmt.Sprintf("%s:%d %s %s %s %s", l.Source, l.Line, l.Kind, l.Type, l.Target, l.Resolution.Status))
	}
	want := []string{
		"a.md:6 relation runs_on c resolved",
		"c.md:4 link links_to d resolved",
		"in.md:6 relation see_also s resolved",
		"s.md:4 link links_to b resolved",
		"s.md:4 link links_to dup ambiguous",
		"s.md:4 link links_to nowhere dangling",
		"s.md:4 link links_to s resolved",
		"s.md:7 relation depends_on a resolved",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("links\n%v\nwant\n%v", got, want)
	}
}

func TestList_SPEC0002_R26_modTime(t *testing.T) {
	x := index.New()
	mt := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	x.Put("a.md", note("a.md", "---\ntitle: A\n---\n"), index.Meta{ModTime: mt})
	es := x.List(index.Filter{})
	if len(es) != 1 || !es[0].ModTime.Equal(mt) || !es[0].Updated.IsZero() {
		t.Fatalf("entries %+v", es)
	}
}

func TestWalk_SPEC0002_R28_edgeScope(t *testing.T) {
	x := graphFixture()
	all, err := x.Walk("s.md", 2, nil, index.EdgesAll)
	if err != nil {
		t.Fatal(err)
	}
	start, err := x.Walk("s.md", 2, nil, index.EdgesStart)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nodes(start), nodes(all)) || start.Truncated != all.Truncated {
		t.Fatalf("scope changed the notes: %v vs %v", nodes(start), nodes(all))
	}
	var want []string
	for _, e := range all.Edges {
		if e.Source == "s.md" || (e.Resolution.Status == index.Resolved && e.Resolution.Path == "s.md") {
			want = append(want, fmt.Sprintf("%s:%d %s %s %s", e.Source, e.Line, e.Type, e.Target, e.Resolution.Status))
		}
	}
	if !reflect.DeepEqual(edges(start), want) {
		t.Fatalf("start edges %v, want %v", edges(start), want)
	}
	for _, must := range []string{
		"s.md:4 links_to nowhere dangling",
		"s.md:4 links_to dup ambiguous",
		"in.md:6 see_also s resolved",
		"s.md:7 depends_on a resolved",
	} {
		if !contains(edges(start), must) {
			t.Fatalf("missing %q in %v", must, edges(start))
		}
	}
	if contains(edges(start), "a.md:6 runs_on c resolved") || len(edges(start)) >= len(edges(all)) {
		t.Fatalf("edges not of the start: %v", edges(start))
	}

	g, _ := x.Walk("s.md", 2, []string{"depends_on", "runs_on"}, index.EdgesStart)
	if !reflect.DeepEqual(edges(g), []string{"s.md:7 depends_on a resolved"}) {
		t.Fatalf("relTypes with EdgesStart: %v", edges(g))
	}

	hub := index.New()
	src := "---\ntitle: hub\n---\n"
	for i := 0; i < 250; i++ {
		src += fmt.Sprintf("[[n%03d]]\n", i)
		put(hub, fmt.Sprintf("n%03d.md", i), "x\n")
	}
	put(hub, "hub.md", src)
	g, _ = hub.Walk("hub.md", 1, nil, index.EdgesStart)
	if len(g.Nodes) != 200 || !g.Truncated {
		t.Fatalf("truncation with EdgesStart: %d %v", len(g.Nodes), g.Truncated)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
