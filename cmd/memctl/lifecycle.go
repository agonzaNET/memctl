package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// shutdownContext returns the one shutdown context of memctl mcp: it is
// cancelled on SIGINT or SIGTERM (ADR-0014, SPEC-0004 R6). Stdin EOF cancels
// it through the returned cancel function, called by the server when its
// input ends.
func shutdownContext() (context.Context, context.CancelFunc) {
	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(ctx)
	return ctx, func() {
		cancel()
		stopSignals()
	}
}
