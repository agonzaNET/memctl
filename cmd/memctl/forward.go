package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/share"
)

// forwardTimeout is how long a process that finds the lock held tries to get
// a shared session before it stops (SPEC-0009 R5). Tests shorten it.
var forwardTimeout = 10 * time.Second

// retryEvery is the pause between two attempts (SPEC-0009 R5).
const retryEvery = 100 * time.Millisecond

// acquireOrForward takes the instance lock, or forwards this process's client
// to the owner of root (SPEC-0009 R4, R5). It returns the lock when this
// process is the owner; otherwise done is true and code is the exit status.
// Each attempt tries the lock first, so a process whose owner went away takes
// over as owner.
func acquireOrForward(ctx context.Context, cfg mcpConfig, root string, stdin io.Reader, stdout, stderr io.Writer, log *slog.Logger) (lock *note.Lock, code int, done bool) {
	deadline := time.Now().Add(forwardTimeout)
	hello := share.NewHello(version, os.Getpid(), cfg.gitAuthor.String())
	for {
		lock, err := note.AcquireLock(root, log)
		if err == nil {
			return lock, exitOK, false
		}
		var held *note.HeldError
		if !errors.As(err, &held) {
			log.Error("cannot take the instance lock", slog.String("err", err.Error()))
			return nil, exitError, true
		}
		conn, ans, err := share.Dial(socketPath(root), hello)
		if err == nil {
			if conn == nil {
				// Refused by the owner: no retry (SPEC-0009 R7).
				log.Error(ans.Error)
				return nil, exitError, true
			}
			return nil, forward(conn, ans, stdin, stdout, log), true
		}
		if !time.Now().Before(deadline) {
			reportHeld(held, cfg, stderr, log)
			return nil, exitError, true
		}
		select {
		case <-ctx.Done():
			log.Info("stopped")
			return nil, exitOK, true
		case <-time.After(retryEvery):
		}
	}
}

// forward passes the client's bytes to the owner and back until either side
// ends (SPEC-0009 R8–R11).
func forward(conn *net.UnixConn, ans share.Answer, stdin io.Reader, stdout io.Writer, log *slog.Logger) int {
	log.Info("forwarding", slog.Int("owner_pid", ans.PID), slog.String("owner_version", ans.Version))
	if ans.Version != version {
		log.Warn("version differs", slog.String("owner_version", ans.Version))
	}
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
