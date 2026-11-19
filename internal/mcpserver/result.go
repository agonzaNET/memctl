package mcpserver

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Code is a tool error code from the fixed list (SPEC-0004, "Error
// handling"). Tools can only emit the constants below.
type Code string

const (
	CodeInvalidArgument Code = "invalid_argument"
	CodeNotFound        Code = "not_found"
	CodeAmbiguous       Code = "ambiguous"
	CodeInternalError   Code = "internal_error"
)

// Codes lists every code a tool may emit.
var Codes = []Code{CodeInvalidArgument, CodeNotFound, CodeAmbiguous, CodeInternalError}

// toolError is an error a tool returns to the agent as "<code>: <message>".
type toolError struct {
	code Code
	msg  string
}

func (e *toolError) Error() string { return string(e.code) + ": " + e.msg }

func errorf(code Code, format string, args ...any) error {
	return &toolError{code: code, msg: fmt.Sprintf(format, args...)}
}

// internalMessage is the fixed sentence of internal_error.
const internalMessage = "memctl could not complete the call; try again, and if it fails again, tell the person."

// success builds a success result: first line, a blank line, the body
// (ADR-0015, SPEC-0004 R10).
func success(first, body string) *mcp.CallToolResult {
	t := first
	if body != "" {
		t += "\n\n" + strings.TrimRight(body, "\n")
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: t}}}
}

// failure builds an error result "<code>: <message>" (SPEC-0004 R11).
func failure(e *toolError) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: e.Error()}},
	}
}
