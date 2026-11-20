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
	mu        sync.Mutex
	root      string
	idx       *index.Index
	log       *slog.Logger
	skipped   map[string]string // rel → reason, from the last load or check
	now       func() time.Time
	committer Committer
	hooks     hooks
}

// Change is one successful tool write, handed to the committer (SPEC-0007
// R5): the tool, the note title and the changed paths relative to the root
// (old and new for a move).
type Change struct {
	Tool, Title string
	Paths       []string
}

// Committer receives every successful tool write, and the changes made
// outside memctl. Enqueue and External must not block. WaitExternal returns
// once a pending external change of path is committed, or at once when
// there is none (SPEC-0008 R14, R16).
type Committer interface {
	Enqueue(Change)
	External(paths []string)
	WaitExternal(path string) error
}

type nopCommitter struct{}

func (nopCommitter) Enqueue(Change)            {}
func (nopCommitter) External([]string)         {}
func (nopCommitter) WaitExternal(string) error { return nil }

// Option configures a Service.
type Option func(*Service)

// WithClock sets the clock of created and updated stamps (SPEC-0007 R7).
func WithClock(now func() time.Time) Option { return func(s *Service) { s.now = now } }

// WithCommitter sets the receiver of tool writes (SPEC-0007 R5).
func WithCommitter(c Committer) Option { return func(s *Service) { s.committer = c } }

// hooks inject faults into a write in tests (SPEC-0007 R2, R3).
type hooks struct {
	beforeRehash func(abs string)
	beforeRename func(abs string) error
}

// Open resolves the notes root and loads every note into a new index
// (SPEC-0003 R1–R9).
func Open(root string, log *slog.Logger, opts ...Option) (*Service, Stats, error) {
	resolved, err := resolveRoot(root)
	if err != nil {
		return nil, Stats{}, err
	}
	s := &Service{root: resolved, idx: index.New(), log: log, now: time.Now, committer: nopCommitter{}}
	for _, o := range opts {
		o(s)
	}
	start := time.Now()
	st, _ := s.refresh(true)
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

// Skipped returns the notes the last load or check skipped, with the reason
// (too_large or unreadable), keyed by relative path.
func (s *Service) Skipped() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.skipped))
	for k, v := range s.skipped {
		out[k] = v
	}
	return out
}
