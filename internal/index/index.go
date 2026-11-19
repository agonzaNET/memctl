// Package index holds every parsed note of one notes root in memory and
// answers link resolution, search, graph and listing questions (SPEC-0002).
// It is not safe for concurrent use; the note service serialises every call.
package index

import (
	"time"

	"github.com/agonzaNET/memctl/internal/parse"
)

// Meta is the file metadata of a note, supplied by the caller.
type Meta struct {
	ModTime time.Time
	Size    int64
	SHA256  string
}

// Index is the in-memory index of one notes root.
type Index struct{}

// New returns an empty index.
func New() *Index { return &Index{} }

// Put adds or replaces the note at path.
func (x *Index) Put(path string, n parse.Note, m Meta) {}

// Remove removes the note at path; an unknown path is a no-op.
func (x *Index) Remove(path string) {}

// Get returns the note and meta stored for path.
func (x *Index) Get(path string) (parse.Note, Meta, bool) { return parse.Note{}, Meta{}, false }

// Len returns the number of notes.
func (x *Index) Len() int { return 0 }

// Paths returns every note path, sorted.
func (x *Index) Paths() []string { return nil }

// Titled returns every path whose title equals title, compared like a title
// link (trimmed, case-folded), sorted.
func (x *Index) Titled(title string) []string { return nil }
