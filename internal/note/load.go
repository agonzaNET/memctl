package note

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

// Skip reasons (SPEC-0003 R6, R7).
const (
	reasonTooLarge   = "too_large"
	reasonUnreadable = "unreadable"
)

var errTooLarge = errors.New("note too large")

// loaded is one note read from disk.
type loaded struct {
	rel  string
	note parse.Note
	meta index.Meta
	err  error
}

// readNote reads, hashes and parses one note. The size and modification time
// come from the open file, so they match the bytes read.
func readNote(c candidate) loaded {
	f, err := os.Open(c.abs)
	if err != nil {
		return loaded{rel: c.rel, err: err}
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return loaded{rel: c.rel, err: err}
	}
	if fi.Size() > MaxNoteSize {
		return loaded{rel: c.rel, err: errTooLarge}
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxNoteSize+1))
	if err != nil {
		return loaded{rel: c.rel, err: err}
	}
	if len(data) > MaxNoteSize {
		return loaded{rel: c.rel, err: errTooLarge}
	}
	sum := sha256.Sum256(data)
	return loaded{
		rel:  c.rel,
		note: parse.Parse(c.rel, data),
		meta: index.Meta{ModTime: fi.ModTime(), Size: int64(len(data)), SHA256: hex.EncodeToString(sum[:])},
	}
}

// readAll reads the candidates with at most GOMAXPROCS workers and returns the
// results in the order of cands (SPEC-0003 R8).
func readAll(cands []candidate) []loaded {
	out := make([]loaded, len(cands))
	jobs := make(chan int)
	var wg sync.WaitGroup
	workers := runtime.GOMAXPROCS(0)
	if workers > len(cands) {
		workers = len(cands)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				out[i] = readNote(cands[i])
			}
		}()
	}
	for i := range cands {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return out
}

// load fills the empty index from the root (SPEC-0003 R8).
func (s *Service) load() Stats {
	var st Stats
	skipped := map[string]int{}
	w := walk(s.root)
	for _, rel := range w.unreadable {
		skipped[reasonUnreadable]++
		s.log.Debug("note skipped", slog.String("path", rel), slog.String("reason", reasonUnreadable))
	}
	var toRead []candidate
	for _, c := range w.notes {
		if c.size > MaxNoteSize {
			skipped[reasonTooLarge]++
			s.log.Debug("note skipped", slog.String("path", c.rel), slog.String("reason", reasonTooLarge))
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
			skipped[reason]++
			s.log.Debug("note skipped", slog.String("path", l.rel), slog.String("reason", reason))
			continue
		}
		s.idx.Put(l.rel, l.note, l.meta)
		st.Added++
	}
	st.Skipped = s.reportSkipped(skipped)
	return st
}

// reportSkipped logs one WARN line per reason and returns the total
// (SPEC-0003 R7).
func (s *Service) reportSkipped(skipped map[string]int) int {
	total := 0
	for _, reason := range []string{reasonTooLarge, reasonUnreadable} {
		if n := skipped[reason]; n > 0 {
			s.log.Warn("notes skipped", slog.String("reason", reason), slog.Int("count", n))
			total += n
		}
	}
	return total
}
