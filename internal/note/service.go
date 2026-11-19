// Package note is the note service: it owns the notes root, keeps the
// in-memory index equal to the files on disk, and serialises every call
// (SPEC-0003).
package note

import (
	"log/slog"

	"github.com/agonzaNET/memctl/internal/index"
)

// MaxNoteSize is the largest note that is loaded (SPEC-0003 R6).
const MaxNoteSize = 4 << 20

// Stats counts what a load or freshness check did.
type Stats struct {
	Added, Changed, Removed, Unchanged, Skipped int
}

// Service is the note service of one notes root.
type Service struct {
	root string
	idx  *index.Index
	log  *slog.Logger
}

// Open resolves the notes root and loads every note into a new index
// (SPEC-0003 R1–R9).
func Open(root string, log *slog.Logger) (*Service, Stats, error) {
	return nil, Stats{}, nil
}

// Root returns the resolved, absolute notes root.
func (s *Service) Root() string { return s.root }

// Index returns the index. Callers outside the service must hold the service
// mutex (see Call); tests may use it directly.
func (s *Service) Index() *index.Index { return s.idx }
