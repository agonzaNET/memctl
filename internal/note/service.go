// Package note is the note service: it owns the notes root, keeps the
// in-memory index equal to the files on disk, and serialises every call
// (SPEC-0003).
package note

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	mu      sync.Mutex
	root    string
	idx     *index.Index
	log     *slog.Logger
	skipped map[string]string // rel → reason, from the last load or check
}

// Open resolves the notes root and loads every note into a new index
// (SPEC-0003 R1–R9).
func Open(root string, log *slog.Logger) (*Service, Stats, error) {
	resolved, err := resolveRoot(root)
	if err != nil {
		return nil, Stats{}, err
	}
	s := &Service{root: resolved, idx: index.New(), log: log}
	start := time.Now()
	st := s.refresh(true)
	log.Info("notes loaded",
		slog.Int("notes", s.idx.Len()),
		slog.Int("skipped", st.Skipped),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()))
	return s, st, nil
}

// resolveRoot makes the root absolute with symlinks evaluated, and checks that
// it is a readable directory (SPEC-0003 R1).
func resolveRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("notes root %s: %w", root, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("notes root %s: %w", root, err)
	}
	fi, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("notes root %s: %w", root, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("notes root %s: not a directory", root)
	}
	if _, err := os.ReadDir(resolved); err != nil {
		return "", fmt.Errorf("notes root %s: %w", root, err)
	}
	return resolved, nil
}

// Root returns the resolved, absolute notes root.
func (s *Service) Root() string { return s.root }

// Index returns the index. Callers outside the service must hold the service
// mutex (see Call); tests may use it directly.
func (s *Service) Index() *index.Index { return s.idx }
