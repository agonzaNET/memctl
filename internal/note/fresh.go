package note

import (
	"errors"
	"log/slog"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
)

// Call runs the freshness check and then fn, both under the service mutex, so
// no call sees a half-updated index (SPEC-0003 R12). fn must not keep idx
// after it returns.
func (s *Service) Call(fn func(idx *index.Index, st Stats) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := time.Now()
	st := s.refresh(false)
	s.log.Debug("freshness check",
		slog.Int("added", st.Added),
		slog.Int("changed", st.Changed),
		slog.Int("removed", st.Removed),
		slog.Int("unchanged", st.Unchanged),
		slog.Int("skipped", st.Skipped),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()))
	return fn(s.idx, st)
}

// refresh makes the index equal to the notes on disk (SPEC-0003 R8, R10,
// R13). A note whose size and modification time match its entry is not read;
// one whose hash did not change only gets new meta. initial is true for the
// startup load, which reports every skipped note.
func (s *Service) refresh(initial bool) Stats {
	var st Stats
	w := walk(s.root)
	skipped := map[string]string{} // rel → reason
	for _, rel := range w.unreadable {
		skipped[rel] = reasonUnreadable
	}
	keep := map[string]bool{}
	var toRead []candidate
	for _, c := range w.notes {
		if c.size > MaxNoteSize {
			skipped[c.rel] = reasonTooLarge
			continue
		}
		if _, m, ok := s.idx.Get(c.rel); ok && m.Size == c.size && m.ModTime.Equal(c.modTime) {
			keep[c.rel] = true
			st.Unchanged++
			continue
		}
		toRead = append(toRead, c)
	}
	for _, l := range readAll(toRead) {
		if l.err != nil {
			reason := reasonUnreadable
			if errors.Is(l.err, errTooLarge) {
				reason = reasonTooLarge
			}
			skipped[l.rel] = reason
			continue
		}
		keep[l.rel] = true
		old, m, ok := s.idx.Get(l.rel)
		switch {
		case !ok:
			s.idx.Put(l.rel, l.note, l.meta)
			st.Added++
		case m.SHA256 == l.meta.SHA256:
			s.idx.Put(l.rel, old, l.meta)
			st.Unchanged++
		default:
			s.idx.Put(l.rel, l.note, l.meta)
			st.Changed++
		}
	}
	for _, p := range s.idx.Paths() {
		if !keep[p] {
			s.idx.Remove(p)
			st.Removed++
		}
	}

	newly := map[string]int{}
	for rel, reason := range skipped {
		if initial || s.skipped[rel] != reason {
			newly[reason]++
			s.log.Debug("note skipped", slog.String("path", rel), slog.String("reason", reason))
		}
	}
	s.reportSkipped(newly)
	s.skipped = skipped
	st.Skipped = len(skipped)
	return st
}

// reportSkipped logs one WARN line per reason with its count (SPEC-0003 R7).
func (s *Service) reportSkipped(counts map[string]int) {
	for _, reason := range []string{reasonTooLarge, reasonUnreadable} {
		if n := counts[reason]; n > 0 {
			s.log.Warn("notes skipped", slog.String("reason", reason), slog.Int("count", n))
		}
	}
}
