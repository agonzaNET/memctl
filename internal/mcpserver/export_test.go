package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServeTransport serves over any SDK transport, for in-memory tests.
func (s *Server) ServeTransport(ctx context.Context, t mcp.Transport) error {
	return s.serveTransport(ctx, t, "stdio")
}

// SetBeforeCall installs a hook that runs inside every tool call, before the
// operation, for tests of in-flight calls.
func (s *Server) SetBeforeCall(f func()) { s.beforeCall = f }

// Instructions is the fixed server instructions, for SPEC-0011 tests.
const Instructions = instructions

// ScopePrefix is the fixed part of the scope line, for SPEC-0011 tests.
const ScopePrefix = scopePrefix

// Descriptions is the fixed description of each tool, for SPEC-0011 tests.
var Descriptions = map[string]string{
	"note_read":   descNoteRead,
	"search":      descSearch,
	"context":     descContext,
	"list":        descList,
	"note_write":  descNoteWrite,
	"note_edit":   descNoteEdit,
	"note_move":   descNoteMove,
	"note_delete": descNoteDelete,
}
