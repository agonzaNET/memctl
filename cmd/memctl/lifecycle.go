package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// shutdown is the one shutdown path of memctl mcp (ADR-0014).
type shutdown struct {
	ctx    context.Context // cancelled on SIGINT, SIGTERM or begin
	cancel context.CancelFunc
	second chan struct{} // closed on a signal after shutdown began
	stop   func()
}

// newShutdown catches SIGINT and SIGTERM. The first one cancels ctx; one
// that arrives after shutdown began (a signal or stdin EOF) closes second,
// so the process exits without waiting for the final commit (SPEC-0004 R6,
// SPEC-0008 R13).
//
// SIGPIPE is ignored, so a write to a stdout or stderr that its reader closed
// fails with an error instead of ending the process; a log line that cannot
// be written is dropped (SPEC-0009 R16).
func newShutdown() *shutdown {
	signal.Ignore(syscall.SIGPIPE)
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	s := &shutdown{ctx: ctx, cancel: cancel, second: make(chan struct{})}
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-sigs:
				if ctx.Err() != nil {
					close(s.second)
					return
				}
				cancel()
			case <-quit:
				return
			}
		}
	}()
	s.stop = func() {
		signal.Stop(sigs)
		close(quit)
		cancel()
	}
	return s
}

// begin marks the start of shutdown, also when it came from stdin EOF.
func (s *shutdown) begin() { s.cancel() }
