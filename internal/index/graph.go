package index

import (
	"errors"
	"time"
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
	return Graph{}, nil
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
func (x *Index) List(f Filter) []Entry { return nil }

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
func (x *Index) Links() []LinkReport { return nil }
