package index

import (
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/agonzaNET/memctl/internal/parse"
)

// Graph walk errors (SPEC-0002 R16, R19).
var (
	ErrBadDepth = errors.New("index: depth must be 1 to 3")
	ErrNotFound = errors.New("index: no such note")
)

// MaxWalkNotes caps a walk (SPEC-0002 R18).
const MaxWalkNotes = 200

// Node is a note reached by a walk.
type Node struct {
	Path     string
	Title    string
	Distance int
}

// Edge is a relation or body link (type "links_to") from Source at Line to
// Target, with the target's resolution.
type Edge struct {
	Source     string
	Line       int
	Type       string
	Target     string
	Resolution Resolution
}

// Graph is the result of a walk.
type Graph struct {
	Nodes     []Node
	Edges     []Edge
	Truncated bool
}

// Walk walks relations, body links and backlinks from start (SPEC-0002
// R15–R19).
func (x *Index) Walk(start string, depth int, relTypes []string) (Graph, error) {
	if depth == 0 {
		depth = 1
	}
	if depth < 1 || depth > 3 {
		return Graph{}, ErrBadDepth
	}
	if _, ok := x.notes[start]; !ok {
		return Graph{}, ErrNotFound
	}
	allowed := func(typ string) bool {
		if len(relTypes) == 0 {
			return true
		}
		for _, t := range relTypes {
			if t == typ {
				return true
			}
		}
		return false
	}

	em := x.edges()
	out, in := em.out, em.in

	var g Graph
	dist := map[string]int{start: 0}
	g.Nodes = append(g.Nodes, Node{Path: start, Title: x.notes[start].note.Title})
	seenEdge := map[string]bool{}
	addEdge := func(e Edge) {
		key := e.Source + "\x00" + strconv.Itoa(e.Line) + "\x00" + e.Type + "\x00" + e.Target
		if !seenEdge[key] {
			seenEdge[key] = true
			g.Edges = append(g.Edges, e)
		}
	}
	// The start's unresolved edges are listed, never followed (R17).
	for _, e := range out[start] {
		if e.Resolution.Status != Resolved && allowed(e.Type) {
			addEdge(e)
		}
	}
	frontier := []string{start}
	for d := 0; d < depth && len(frontier) > 0 && !g.Truncated; d++ {
		var next []string
		visit := func(e Edge, other string) {
			if g.Truncated {
				return
			}
			if _, ok := dist[other]; !ok {
				if len(g.Nodes) >= MaxWalkNotes {
					g.Truncated = true
					return
				}
				dist[other] = d + 1
				g.Nodes = append(g.Nodes, Node{Path: other, Title: x.notes[other].note.Title, Distance: d + 1})
				next = append(next, other)
			}
			addEdge(e)
		}
		for _, p := range frontier {
			for _, e := range out[p] {
				if e.Resolution.Status == Resolved && e.Resolution.Path != p && allowed(e.Type) {
					visit(e, e.Resolution.Path)
				}
			}
			for _, e := range in[p] {
				if allowed(e.Type) {
					visit(e, e.Source)
				}
			}
		}
		frontier = next
	}
	sort.SliceStable(g.Nodes, func(i, j int) bool {
		a, b := g.Nodes[i], g.Nodes[j]
		if a.Distance != b.Distance {
			return a.Distance < b.Distance
		}
		return a.Path < b.Path
	})
	sortEdges(g.Edges)
	return g, nil
}

// outgoing returns the relations and body links of a note, resolved, in
// file order (SPEC-0002 R15).
func (x *Index) outgoing(p string) []Edge {
	n := x.notes[p].note
	var es []Edge
	for _, r := range n.Relations {
		es = append(es, Edge{Source: p, Line: r.Line, Type: r.Type, Target: r.Target, Resolution: x.Resolve(r.Target)})
	}
	for _, l := range n.Links {
		es = append(es, Edge{Source: p, Line: l.Line, Type: "links_to", Target: l.Target, Resolution: x.Resolve(l.Target)})
	}
	return es
}

func sortEdges(es []Edge) {
	sort.SliceStable(es, func(i, j int) bool {
		a, b := es[i], es[j]
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Target < b.Target
	})
}

// Filter selects notes for List (SPEC-0002 R20).
type Filter struct {
	Folder string
	Type   string
	Tag    string
}

// Entry is one listed note.
type Entry struct {
	Path    string
	Title   string
	Type    string
	Tags    []string
	Updated time.Time
}

// List returns the notes that pass f, sorted by path (SPEC-0002 R20).
func (x *Index) List(f Filter) []Entry {
	q := Query{Type: f.Type, Folder: f.Folder}
	if f.Tag != "" {
		q.Tags = []string{f.Tag}
	}
	var out []Entry
	for _, p := range x.Paths() {
		n := x.notes[p].note
		if !matches(p, n, q) {
			continue
		}
		out = append(out, Entry{Path: p, Title: n.Title, Type: n.Type, Tags: n.Tags, Updated: n.Updated})
	}
	return out
}

// LinkReport is one relation or link of a note with its resolution.
type LinkReport struct {
	Source     string
	Line       int
	Kind       string // "relation" or "link"
	Type       string
	Target     string
	Resolution Resolution
}

// Links returns every relation and link of every note (SPEC-0002 R21).
func (x *Index) Links() []LinkReport {
	var es []Edge
	for _, p := range x.Paths() {
		es = append(es, x.outgoing(p)...)
	}
	sortEdges(es)
	out := make([]LinkReport, len(es))
	for i, e := range es {
		kind := "relation"
		if e.Type == "links_to" && isLink(x.notes[e.Source].note.Links, e.Line, e.Target) {
			kind = "link"
		}
		out[i] = LinkReport{Source: e.Source, Line: e.Line, Kind: kind, Type: e.Type, Target: e.Target, Resolution: e.Resolution}
	}
	return out
}

func isLink(ls []parse.Link, line int, target string) bool {
	for _, l := range ls {
		if l.Line == line && l.Target == target {
			return true
		}
	}
	return false
}

// edgeMaps holds every edge resolved against the current notes, by source
// and by resolved target.
type edgeMaps struct {
	out map[string][]Edge
	in  map[string][]Edge
}

// edges returns the resolved edges, computed once per change of the index:
// Put and Remove drop the cache, because one note can change how the links
// of every other note resolve (SPEC-0002 R15, R22).
func (x *Index) edges() *edgeMaps {
	if x.edgeCache != nil {
		return x.edgeCache
	}
	em := &edgeMaps{out: map[string][]Edge{}, in: map[string][]Edge{}}
	for p := range x.notes {
		for _, e := range x.outgoing(p) {
			em.out[p] = append(em.out[p], e)
			if e.Resolution.Status == Resolved && e.Resolution.Path != p {
				em.in[e.Resolution.Path] = append(em.in[e.Resolution.Path], e)
			}
		}
	}
	for _, es := range em.in {
		sortEdges(es)
	}
	x.edgeCache = em
	return em
}

// Linkers returns how many other notes have a relation or body link that
// resolves to p (SPEC-0007 R23, R24).
func (x *Index) Linkers(p string) int {
	seen := map[string]bool{}
	for _, e := range x.edges().in[p] {
		if e.Source != p {
			seen[e.Source] = true
		}
	}
	return len(seen)
}
