package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/agonzaNET/memctl/internal/mcpserver"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/share"
)

// socketPath is the owner's socket in a resolved notes root (SPEC-0009 R1).
func socketPath(root string) string {
	return filepath.Join(root, note.MemoryDir, share.SocketName)
}

// listenShared opens the owner's socket after the load. On failure it logs
// that sharing is off and returns nil (SPEC-0009 R1, R2).
func listenShared(root string, log *slog.Logger) *share.Listener {
	path := socketPath(root)
	ln, err := share.Listen(path)
	if err != nil {
		reason := err.Error()
		if errors.Is(err, share.ErrPathTooLong) {
			reason = "the socket path is longer than this platform allows"
		}
		log.Warn("sharing is off", slog.String("reason", reason), slog.String("socket", path))
		return nil
	}
	return ln
}

// acceptShared serves every connection that passes the handshake as a
// forwarded session, until the listener is closed (SPEC-0009 R6, R7, R12).
func acceptShared(ctx context.Context, ln *share.Listener, srv *mcpserver.Server, owner share.Owner, sessions *sessionSet, log *slog.Logger) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func() {
			h, err := share.Greet(conn, owner)
			if err != nil {
				log.Warn("forwarded session refused", slog.String("reason", err.Error()))
				return
			}
			if h.Version != owner.Version {
				log.Warn("version differs", slog.String("forwarder_version", h.Version))
			}
			name, ok := sessions.add()
			if !ok {
				conn.Close()
				return
			}
			defer sessions.done()
			log.Info("session opened", slog.String("session", name), slog.Int("forwarder_pid", h.PID))
			if err := srv.ServeConn(ctx, conn, name); err != nil {
				log.Warn("session ended with an error", slog.String("session", name), slog.String("class", "transport"))
			}
			conn.Close()
			log.Info("session closed", slog.String("session", name), slog.Int("forwarder_pid", h.PID))
		}()
	}
}

// ownerInfo is this process as an owner in the handshake.
func ownerInfo(cfg mcpConfig) share.Owner {
	return share.Owner{PID: os.Getpid(), Version: version, GitAuthor: cfg.gitAuthor.String()}
}
