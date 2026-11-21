package main

import (
	"context"
	"strconv"
	"sync"
)

// sessionSet tracks the owner's forwarded sessions (SPEC-0009 R12–R14).
type sessionSet struct {
	mu     sync.Mutex
	n      int           // forwarded sessions accepted so far
	open   int           // forwarded sessions not yet ended
	closed bool          // no session is added after close
	idle   chan struct{} // closed when open drops to 0, if someone waits
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
	s.open++
	s.wg.Add(1)
	return "forward-" + strconv.Itoa(s.n), true
}

// done marks a session added by add as ended.
func (s *sessionSet) done() {
	s.mu.Lock()
	s.open--
	if s.open == 0 && s.idle != nil {
		close(s.idle)
		s.idle = nil
	}
	s.mu.Unlock()
	s.wg.Done()
}

// closeWhenIdle closes the set as soon as no forwarded session is open, or at
// once when ctx is done (SPEC-0009 R14, R15). Checking and closing happen
// under one lock, so no session is accepted after the last one ended.
func (s *sessionSet) closeWhenIdle(ctx context.Context) {
	for {
		s.mu.Lock()
		if s.open == 0 || ctx.Err() != nil {
			s.closed = true
			s.mu.Unlock()
			return
		}
		idle := make(chan struct{})
		s.idle = idle
		s.mu.Unlock()
		select {
		case <-idle:
		case <-ctx.Done():
		}
	}
}

// closeAndWait stops new sessions and waits for the open ones to end.
func (s *sessionSet) closeAndWait() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.wg.Wait()
}
