// Package index holds every parsed note of one notes root in memory and
// answers link resolution, search, graph and listing questions (SPEC-0002).
// It is not safe for concurrent use; the note service serialises every call.
package index

import (
	"path"
	"sort"
	"strings"
	"time"

	"github.com/agonzaNET/memctl/internal/parse"
)

// Meta is the file metadata of a note, supplied by the caller.
type Meta struct {
	ModTime time.Time
	Size    int64
	SHA256  string
}

type entry struct {
	note parse.Note
	meta Meta
}

// nameMap maps a normalised name to the set of note paths that carry it.
type nameMap map[string]map[string]struct{}

func (m nameMap) add(name, p string) {
	if name == "" {
		return
	}
	set := m[name]
	if set == nil {
		set = map[string]struct{}{}
		m[name] = set
	}
	set[p] = struct{}{}
}

func (m nameMap) remove(name, p string) {
	if set := m[name]; set != nil {
		delete(set, p)
		if len(set) == 0 {
			delete(m, name)
		}
	}
}

// Index is the in-memory index of one notes root.
type Index struct {
	notes map[string]*entry

	// The four resolution steps of design D23, keyed by normalised name.
	paths       nameMap // note path without .md
	titles      nameMap // title
	aliases     nameMap // plain aliases
	pathAliases nameMap // path-like aliases, normalised as paths
	stems       nameMap // file name without folder and .md
}

// New returns an empty index.
func New() *Index {
	return &Index{
		notes:       map[string]*entry{},
		paths:       nameMap{},
		titles:      nameMap{},
		aliases:     nameMap{},
		pathAliases: nameMap{},
		stems:       nameMap{},
	}
}

// Put adds or replaces the note at path (SPEC-0002 R1).
func (x *Index) Put(p string, n parse.Note, m Meta) {
	x.Remove(p)
	x.notes[p] = &entry{note: n, meta: m}
	x.eachName(p, n, func(mp nameMap, name string) { mp.add(name, p) })
}

// Remove removes the note at path; an unknown path is a no-op (SPEC-0002 R1).
func (x *Index) Remove(p string) {
	e, ok := x.notes[p]
	if !ok {
		return
	}
	x.eachName(p, e.note, func(mp nameMap, name string) { mp.remove(name, p) })
	delete(x.notes, p)
}

// eachName calls fn for every name under which the note can be resolved.
func (x *Index) eachName(p string, n parse.Note, fn func(nameMap, string)) {
	fn(x.paths, normPath(p))
	fn(x.titles, normName(n.Title))
	for _, a := range n.Aliases {
		if isPathLike(a) {
			fn(x.pathAliases, normPath(a))
		} else {
			fn(x.aliases, normName(a))
		}
	}
	fn(x.stems, normName(strings.TrimSuffix(path.Base(p), ".md")))
}

// Get returns the note and meta stored for path.
func (x *Index) Get(p string) (parse.Note, Meta, bool) {
	e, ok := x.notes[p]
	if !ok {
		return parse.Note{}, Meta{}, false
	}
	return e.note, e.meta, true
}

// Len returns the number of notes.
func (x *Index) Len() int { return len(x.notes) }

// Paths returns every note path, sorted.
func (x *Index) Paths() []string {
	out := make([]string, 0, len(x.notes))
	for p := range x.notes {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// Titled returns every path whose title equals title, compared like a title
// link (trimmed, case-folded), sorted.
func (x *Index) Titled(title string) []string {
	return sortedSet(x.titles[normName(title)])
}

func sortedSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
