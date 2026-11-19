package mcpserver

import (
	"context"
	"io"
	"log/slog"

	"github.com/agonzaNET/memctl/internal/note"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Options configures the server.
type Options struct {
	Version string
	Notes   *note.Service
	Log     *slog.Logger
}

// Server is the memctl MCP server.
type Server struct {
	opts       Options
	sdk        *mcp.Server
	beforeCall func()
}

// New returns a server with every tool registered.
func New(o Options) *Server { return &Server{opts: o} }

// Serve runs the server over in and out until in ends or ctx is cancelled,
// then waits for the tool call in progress (SPEC-0004 R6).
func (s *Server) Serve(ctx context.Context, in io.ReadCloser, out io.WriteCloser) error {
	return nil
}

func (s *Server) serveTransport(ctx context.Context, t mcp.Transport) error { return nil }
