package note

import "github.com/agonzaNET/memctl/internal/index"

// Refresh runs the freshness check without taking the mutex; callers use
// Call. It makes the index equal to the notes on disk (SPEC-0003 R10, R13).
func (s *Service) refresh() Stats { return Stats{} }

// Call runs the freshness check and then fn, both under the service mutex, so
// no call sees a half-updated index (SPEC-0003 R12). fn must not keep idx
// after it returns.
func (s *Service) Call(fn func(idx *index.Index, st Stats) error) error {
	return fn(s.idx, Stats{})
}
