package index_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

func note(rel, src string) parse.Note { return parse.Parse(rel, []byte(src)) }

func put(x *index.Index, rel, src string) {
	x.Put(rel, note(rel, src), index.Meta{Size: int64(len(src)), SHA256: rel})
}

func TestIndex_SPEC0002_R1_putGetRemove(t *testing.T) {
	x := index.New()
	m := index.Meta{ModTime: time.Unix(10, 0), Size: 3, SHA256: "abc"}
	// The file stem (note) differs from both titles, so only the title step
	// can match "B" or "C".
	n := note("a/note.md", "---\ntitle: B\n---\nx\n")
	x.Put("a/note.md", n, m)
	gn, gm, ok := x.Get("a/note.md")
	if !ok || gn.Title != "B" || gm != m || x.Len() != 1 {
		t.Fatalf("get: %v %q %+v len=%d", ok, gn.Title, gm, x.Len())
	}
	if r := x.Resolve("B"); r.Status != index.Resolved || r.Path != "a/note.md" {
		t.Fatalf("resolve after put: %+v", r)
	}

	x.Put("a/note.md", note("a/note.md", "---\ntitle: C\n---\n"), m)
	if r := x.Resolve("B"); r.Status != index.Dangling {
		t.Fatalf("old title still resolves after replace: %+v", r)
	}
	if r := x.Resolve("C"); r.Path != "a/note.md" || x.Len() != 1 {
		t.Fatalf("replace: %+v len=%d", r, x.Len())
	}

	x.Remove("a/note.md")
	x.Remove("unknown.md")
	if _, _, ok := x.Get("a/note.md"); ok || x.Len() != 0 || len(x.Paths()) != 0 {
		t.Fatalf("remove: ok=%v len=%d", ok, x.Len())
	}
	if r := x.Resolve("a/note"); r.Status != index.Dangling {
		t.Fatalf("resolve after remove: %+v", r)
	}
}

func fixture() *index.Index {
	x := index.New()
	put(x, "network/intpxy.md", "---\ntitle: Internal proxy\n---\n")
	put(x, "hosts/intpxy.md", "---\ntitle: intpxy host\n---\n")
	put(x, "services/intpxy.md", "---\ntitle: intpxy service\n---\n")
	put(x, "a/shared.md", "---\ntitle: Shared\n---\n")
	put(x, "b/shared title.md", "---\ntitle: Shared\n---\n")
	put(x, "c/Shared.md", "---\ntitle: Other\n---\n")
	put(x, "ops/firewall.md", "---\ntitle: Firewall\naliases: [Main FW, old/place, legacy/fw.md]\n---\n")
	put(x, "ops/twins.md", "---\ntitle: Twins\naliases: [Twin, twin, ' TWIN ']\n---\n")
	return x
}

func TestResolve_SPEC0002_R3_R4_steps(t *testing.T) {
	x := fixture()
	cases := []struct {
		target string
		status index.Status
		path   string
		cands  []string
	}{
		{"network/intpxy", index.Resolved, "network/intpxy.md", nil},
		{"network/intpxy.md", index.Resolved, "network/intpxy.md", nil},
		{"./network/intpxy", index.Resolved, "network/intpxy.md", nil},
		{"/network/intpxy.MD", index.Resolved, "network/intpxy.md", nil},
		{"  NETWORK/INTPXY  ", index.Resolved, "network/intpxy.md", nil},
		{"Internal proxy", index.Resolved, "network/intpxy.md", nil},
		{"internal PROXY", index.Resolved, "network/intpxy.md", nil},
		// Title step matches two notes: ambiguous, even though c/Shared.md's
		// stem would match in a later step.
		{"Shared", index.Ambiguous, "", []string{"a/shared.md", "b/shared title.md"}},
		{"main fw", index.Resolved, "ops/firewall.md", nil},
		// Stem step: three notes named intpxy.
		{"intpxy", index.Ambiguous, "", []string{"hosts/intpxy.md", "network/intpxy.md", "services/intpxy.md"}},
		{"firewall", index.Resolved, "ops/firewall.md", nil},
		{"nothing", index.Dangling, "", nil},
		{"", index.Dangling, "", nil},
	}
	for _, c := range cases {
		t.Run(c.target, func(t *testing.T) {
			r := x.Resolve(c.target)
			if r.Status != c.status || r.Path != c.path || !reflect.DeepEqual(r.Candidates, c.cands) {
				t.Fatalf("Resolve(%q) = %+v, want %s %q %v", c.target, r, c.status, c.path, c.cands)
			}
		})
	}
}

func TestResolve_SPEC0002_R5_stale(t *testing.T) {
	x := fixture()
	for _, target := range []string{"old/place", "legacy/fw", "legacy/fw.md", "./OLD/place.md"} {
		r := x.Resolve(target)
		if r.Status != index.Resolved || r.Path != "ops/firewall.md" || !r.Stale {
			t.Fatalf("Resolve(%q) = %+v, want stale ops/firewall.md", target, r)
		}
	}
	for _, target := range []string{"Main FW", "Firewall", "ops/firewall"} {
		if r := x.Resolve(target); r.Stale {
			t.Fatalf("Resolve(%q) = %+v, want not stale", target, r)
		}
	}
}

func TestResolve_SPEC0002_R6_countOnce(t *testing.T) {
	x := fixture()
	if r := x.Resolve("twin"); r.Status != index.Resolved || r.Path != "ops/twins.md" {
		t.Fatalf("Resolve(twin) = %+v", r)
	}
	// A note whose title and stem match the same name counts once in each step.
	y := index.New()
	put(y, "x/Same.md", "---\ntitle: Same\naliases: [same]\n---\n")
	if r := y.Resolve("same"); r.Status != index.Resolved {
		t.Fatalf("Resolve(same) = %+v", r)
	}
}

func TestIndex_SPEC0002_R1_titled(t *testing.T) {
	x := fixture()
	if got := x.Titled(" shared "); !reflect.DeepEqual(got, []string{"a/shared.md", "b/shared title.md"}) {
		t.Fatalf("Titled = %v", got)
	}
	if got := x.Paths(); len(got) != 8 || got[0] != "a/shared.md" {
		t.Fatalf("Paths = %v", got)
	}
}

func TestResolve_SPEC0002_R23_stepAndAlso(t *testing.T) {
	x := fixture()
	put(x, "x/onlystem.md", "---\ntitle: Q\n---\n")
	put(x, "alpha.md", "---\ntitle: Root\n---\n")
	put(x, "notes/alpha.md", "---\ntitle: First\n---\n")
	put(x, "archive/alpha.md", "---\ntitle: Alpha\n---\n")
	cases := []struct {
		target string
		status index.Status
		step   index.Step
		also   []string
	}{
		{"network/intpxy", index.Resolved, index.StepPath, nil},
		{"Internal proxy", index.Resolved, index.StepTitle, nil},
		{"main fw", index.Resolved, index.StepAlias, nil},
		{"old/place", index.Resolved, index.StepAlias, nil},
		{"onlystem", index.Resolved, index.StepStem, nil},
		{"firewall", index.Resolved, index.StepTitle, nil},
		// The path wins; the title and the stems name two other notes.
		{"alpha", index.Resolved, index.StepPath, []string{"archive/alpha.md", "notes/alpha.md"}},
		{"Shared", index.Ambiguous, "", nil},
		{"nothing", index.Dangling, "", nil},
	}
	for _, c := range cases {
		r := x.Resolve(c.target)
		if r.Status != c.status || r.Step != c.step || !reflect.DeepEqual(r.Also, c.also) {
			t.Errorf("Resolve(%q) = %+v, want %s %s %v", c.target, r, c.status, c.step, c.also)
		}
	}
}
