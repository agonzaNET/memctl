package main

import (
	"strconv"
	"sync"
)

// sessionSet tracks the owner's forwarded sessions (SPEC-0009 R12–R14).
type sessionSet struct {
	mu     sync.Mutex
	n      int  // forwarded sessions accepted so far
	closed bool // no session is added after close
	wg     sync.WaitGroup
}

// add registers a new forwarded session and returns its name, forward-<n>;
// ok is false once the set is closed.
func (s *sessionSet) add() (name string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return "", false
	}
	s.n++
	s.wg.Add(1)
	return "forward-" + strconv.Itoa(s.n), true
}

// done marks a session added by add as ended.
func (s *sessionSet) done() { s.wg.Done() }

// closeAndWait stops new sessions and waits for the open ones to end.
func (s *sessionSet) closeAndWait() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.wg.Wait()
}
