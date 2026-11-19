package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeTransport serves over any SDK transport, for in-memory tests.
func (s *Server) ServeTransport(ctx context.Context, t mcp.Transport) error {
	return s.serveTransport(ctx, t)
}

// SetBeforeCall installs a hook that runs inside every tool call, before the
// operation, for tests of in-flight calls.
func (s *Server) SetBeforeCall(f func()) { s.beforeCall = f }
