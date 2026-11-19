package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/agonzaNET/memctl/internal/gitlog"
	"github.com/agonzaNET/memctl/internal/mcpserver"
	"github.com/agonzaNET/memctl/internal/note"
)

// afterLock, when set by a test, runs after the lock is taken and before the
// notes are loaded.
var afterLock func()

// runMCP runs memctl mcp: load the root, serve until stdin ends or a signal
// arrives, and stop (SPEC-0004 R1–R6, SPEC-0006 R1, SPEC-0008 R12, R13).
func runMCP(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	cfg, err := parseMCP(args, os.Getenv)
	if err != nil {
		return usageError(stderr, err.Error())
	}
	log := newLogger(stderr, cfg.logLevel)

	sd := newShutdown()
	defer sd.stop()
	ctx := sd.ctx

	root, err := note.ResolveRoot(cfg.root)
	if err != nil {
		log.Error("cannot open the notes root", slog.String("err", err.Error()))
		return exitError
	}
	// One process per root (SPEC-0006 R1): the lock is taken before the load
	// and held until the process exits. The deferred Close also keeps the
	// Lock reachable, so its file is never closed by a finalizer early.
	lock, err := note.AcquireLock(root, log)
	if err != nil {
		var held *note.HeldError
		if errors.As(err, &held) {
			fmt.Fprintln(stderr, held.Error())
			log.Error(held.Error())
		} else {
			log.Error("cannot take the instance lock", slog.String("err", err.Error()))
		}
		return exitError
	}
	defer lock.Close()
	if afterLock != nil {
		afterLock()
	}

	// The repository test runs at start (SPEC-0008 R1); the committer takes
	// every successful tool write and commits it in the background.
	// The committer has its own context: it is stopped only after the last
	// tool call has finished, so that call's write is in the final commit
	// (SPEC-0008 R12).
	committer := gitlog.New(ctx, gitlog.Config{Root: root, Author: cfg.gitAuthor, Log: log})
	commitCtx, stopCommitter := context.WithCancel(context.Background())
	defer stopCommitter()
	committed := make(chan struct{})
	go func() { committer.Run(commitCtx); close(committed) }()

	svc, _, err := note.Open(root, log, note.WithCommitter(gitCommitter{committer}))
	if err != nil {
		log.Error("cannot open the notes root", slog.String("err", err.Error()))
		return exitError
	}
	log.Info("started",
		slog.String("version", version),
		slog.String("log_level", cfg.logLevel),
		slog.String("git_author", cfg.gitAuthor.String()),
		slog.Int("notes", svc.Index().Len()))

	srv := mcpserver.New(mcpserver.Options{Version: version, Notes: svc, Log: log})
	if err := srv.Serve(ctx, io.NopCloser(stdin), nopWriteCloser{stdout}); err != nil {
		log.Warn("session ended with an error", slog.String("class", "transport"))
	}

	// Serve has returned after the call in progress finished: commit what is
	// queued at once, and exit when that commit returns, or at once on
	// another signal (ADR-0014, SPEC-0008 R12, R13).
	sd.begin()
	stopCommitter()
	select {
	case <-committed:
	case <-sd.second:
		log.Warn("second signal during shutdown; exiting without the final commit")
		return exitError
	}

	log.Info("stopped")
	return exitOK
}

// nopWriteCloser keeps the server from closing stdout.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// gitCommitter hands the note service's changes to the git committer.
type gitCommitter struct{ c *gitlog.Committer }

func (g gitCommitter) Enqueue(ch note.Change) {
	g.c.Add(gitlog.Entry{Tool: ch.Tool, Title: ch.Title, Paths: ch.Paths})
}
