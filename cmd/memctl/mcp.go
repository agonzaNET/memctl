package main

import (
	"io"
	"log/slog"
	"os"

	"github.com/agonzaNET/memctl/internal/mcpserver"
	"github.com/agonzaNET/memctl/internal/note"
)

// afterLock, when set by a test, runs after the lock is taken and before the
// notes are loaded.
var afterLock func()

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

	srv := mcpserver.New(mcpserver.Options{Version: version, Notes: svc, Log: log})
	if err := srv.Serve(ctx, io.NopCloser(stdin), nopWriteCloser{stdout}); err != nil {
		log.Warn("session ended with an error", slog.String("class", "transport"))
	}

	log.Info("stopped")
	return exitOK
}

// nopWriteCloser keeps the server from closing stdout.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
