package main

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/agonzaNET/memctl/internal/note"
)

// runMCP runs memctl mcp: load the root, serve until stdin ends or a signal
// arrives, and stop (SPEC-0004 R1–R6).
func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseMCP(args, os.Getenv)
	if err != nil {
		return usageError(stderr, err.Error())
	}
	log := newLogger(stderr, cfg.logLevel)

	ctx, cancel := shutdownContext()
	defer cancel()

	svc, _, err := note.Open(cfg.root, log)
	if err != nil {
		log.Error("cannot open the notes root", slog.String("err", err.Error()))
		return exitError
	}
	log.Info("started",
		slog.String("version", version),
		slog.String("log_level", cfg.logLevel),
		slog.Int("notes", svc.Index().Len()))

	serve(ctx, cancel, stdin, stdout)

	log.Info("stopped")
	return exitOK
}

// serve runs until the shutdown context ends. Until the MCP server lands, it
// only watches stdin for EOF.
func serve(ctx context.Context, cancel context.CancelFunc, stdin io.Reader, _ io.Writer) {
	go func() {
		_, _ = io.Copy(io.Discard, stdin)
		cancel()
	}()
	<-ctx.Done()
}
