package mcpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

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
	inFlight   sync.WaitGroup
	beforeCall func()
}

// New returns a server with every tool registered (SPEC-0004 R5, SPEC-0007).
func New(o Options) *Server {
	s := &Server{opts: o}
	s.sdk = mcp.NewServer(
		&mcp.Implementation{Name: "memctl", Version: o.Version},
		&mcp.ServerOptions{
			Instructions: instructions,
			Capabilities: &mcp.ServerCapabilities{Tools: &mcp.ToolCapabilities{}},
			Logger:       slog.New(slog.DiscardHandler),
		})
	s.registerRead()
	s.registerSearch()
	s.registerGraph()
	s.registerWrite()
	return s
}

// Serve runs the server over in and out until in ends or ctx is cancelled,
// then waits for the tool call in progress (SPEC-0004 R6).
func (s *Server) Serve(ctx context.Context, in io.ReadCloser, out io.WriteCloser) error {
	return s.serveTransport(ctx, &mcp.IOTransport{Reader: in, Writer: out})
}

func (s *Server) serveTransport(ctx context.Context, t mcp.Transport) error {
	err := s.sdk.Run(ctx, t)
	s.inFlight.Wait()
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func readOnly() *mcp.ToolAnnotations {
	no := false
	return &mcp.ToolAnnotations{ReadOnlyHint: true, DestructiveHint: &no, IdempotentHint: true, OpenWorldHint: &no}
}

// handle wraps a tool function: it counts the call as in flight, runs the
// test hook, turns a toolError into an error result, maps any other error to
// internal_error, and writes the tool-call log line (SPEC-0004 R10–R12).
func handle[In any](s *Server, tool string, fn func(in In, log *callLog) (*mcp.CallToolResult, error)) mcp.ToolHandlerFor[In, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		s.inFlight.Add(1)
		defer s.inFlight.Done()
		start := time.Now()
		if s.beforeCall != nil {
			s.beforeCall()
		}
		var cl callLog
		res, err := fn(in, &cl)
		var code Code
		if err != nil {
			var te *toolError
			if !errors.As(err, &te) {
				s.opts.Log.Error("tool failed", slog.String("tool", tool), slog.String("class", errorClass(err)))
				msg := internalMessage
				if errors.As(err, new(unchangedError)) {
					msg = unchangedMessage
				}
				te = &toolError{code: CodeInternalError, msg: msg}
			}
			code = te.code
			res = failure(te)
		}
		cl.write(s.opts.Log, tool, start, code)
		return res, nil, nil
	}
}

// errorClass names the kind of an unexpected error without its text, which
// may hold a path.
func errorClass(err error) string {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "cancelled"
	case errors.Is(err, io.ErrUnexpectedEOF):
		return "io"
	default:
		return "unexpected"
	}
}
