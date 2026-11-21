package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/share"
)

// runForward runs memctl mcp as a forwarder: the lock of root is held by
// another process, so this one passes its client's bytes to that owner over
// the root's socket and loads nothing itself (SPEC-0009 R4, R9–R11).
func runForward(cfg mcpConfig, root string, held *note.HeldError, stdin io.Reader, stdout, stderr io.Writer, log *slog.Logger) int {
	hello := share.NewHello(version, os.Getpid(), cfg.gitAuthor.String())
	conn, ans, err := share.Dial(socketPath(root), hello)
	if err != nil {
		reportHeld(held, cfg, stderr, log)
		return exitError
	}
	if conn == nil {
		log.Error(ans.Error)
		return exitError
	}
	log.Info("forwarding", slog.Int("owner_pid", ans.PID), slog.String("owner_version", ans.Version))
	if share.Pipe(stdin, stdout, conn) {
		log.Error(fmt.Sprintf("memctl (pid %d) serving the root stopped", ans.PID))
		return exitError
	}
	log.Info("stopped")
	return exitOK
}

// reportHeld logs the refusal of SPEC-0006 R5 once in a client's log, and
// also as a plain line for a person at a terminal.
func reportHeld(held *note.HeldError, cfg mcpConfig, stderr io.Writer, log *slog.Logger) {
	held.Given = cfg.root
	if isTerminal(stderr) {
		fmt.Fprintln(stderr, held.Error())
	}
	log.Error(held.Error())
}
