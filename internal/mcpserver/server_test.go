package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServer_SPEC0004_R5_infoInstructionsCapabilities(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	init := e.cs.InitializeResult()
	if init.ServerInfo.Name != "memctl" || init.ServerInfo.Version != "1.2.3" {
		t.Fatalf("server info %+v", init.ServerInfo)
	}
	for _, want := range []string{"Markdown", "path", "reference data"} {
		if !strings.Contains(init.Instructions, want) {
			t.Fatalf("instructions miss %q: %q", want, init.Instructions)
		}
	}
	c := init.Capabilities
	if c.Tools == nil || c.Logging != nil || c.Prompts != nil || c.Resources != nil {
		t.Fatalf("capabilities %+v", c)
	}
}

func TestTools_SPEC0004_R8_R9_annotationsAndSchemas(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, tool := range res.Tools {
		a := tool.Annotations
		if a == nil || !a.ReadOnlyHint || !a.IdempotentHint || a.DestructiveHint == nil || *a.DestructiveHint || a.OpenWorldHint == nil || *a.OpenWorldHint {
			t.Fatalf("%s annotations %+v", tool.Name, a)
		}
		raw, _ := json.Marshal(tool.InputSchema)
		var schema map[string]any
		json.Unmarshal(raw, &schema)
		if schema["additionalProperties"] != false || schema["type"] != "object" {
			t.Fatalf("%s schema %s", tool.Name, raw)
		}
		if tool.OutputSchema != nil {
			t.Fatalf("%s has an output schema", tool.Name)
		}
		if tool.Name == "note_read" {
			found = true
		}
	}
	if !found {
		t.Fatal("note_read not listed")
	}

	calls := 0
	e.srv.SetBeforeCall(func() { calls++ })
	for _, args := range []map[string]any{
		{"ref": "a", "nope": 1},
		{},
		{"ref": ""},
		{"ref": 3},
	} {
		r := e.call(t, "note_read", args)
		if !r.IsError {
			t.Fatalf("args %v accepted", args)
		}
	}
	if calls != 0 {
		t.Fatalf("handler ran %d times for schema-breaking input", calls)
	}
}

func TestResult_SPEC0004_R10_R11_shape(t *testing.T) {
	e := start(t, map[string]string{"a.md": "---\ntitle: A\n---\nbody\n"})
	ok := e.call(t, "note_read", map[string]any{"ref": "a.md"})
	if ok.IsError || ok.StructuredContent != nil {
		t.Fatalf("success result %+v", ok)
	}
	first, rest, _ := strings.Cut(text(t, ok), "\n")
	if !strings.HasSuffix(first, ".") || !strings.HasPrefix(rest, "\n") {
		t.Fatalf("first line %q, rest %q", first, rest)
	}

	bad := e.call(t, "note_read", map[string]any{"ref": "missing"})
	code, msg, found := strings.Cut(text(t, bad), ": ")
	if !bad.IsError || !found || !strings.HasSuffix(msg, ".") {
		t.Fatalf("error result %q", text(t, bad))
	}
	known := false
	for _, c := range mcpserver.Codes {
		if string(c) == code {
			known = true
		}
	}
	if !known {
		t.Fatalf("code %q not in the fixed list", code)
	}
}

func TestResult_SPEC0004_R11_fixedCodeList(t *testing.T) {
	want := []string{"invalid_argument", "not_found", "ambiguous", "internal_error"}
	if len(mcpserver.Codes) != len(want) {
		t.Fatalf("codes %v", mcpserver.Codes)
	}
	for i, c := range mcpserver.Codes {
		if string(c) != want[i] {
			t.Fatalf("codes %v, want %v", mcpserver.Codes, want)
		}
	}
}

func TestLog_SPEC0004_R12_toolCallLine(t *testing.T) {
	e := start(t, map[string]string{"secret/a.md": "---\ntitle: Secret title\n---\nbody text\n"})
	e.call(t, "note_read", map[string]any{"ref": "Secret title"})
	e.call(t, "note_read", map[string]any{"ref": "missing secret"})
	lines := e.logLines(t, "tool call")
	if len(lines) != 2 {
		t.Fatalf("lines %v", lines)
	}
	ok, bad := lines[0], lines[1]
	if ok["level"] != "INFO" || ok["tool"] != "note_read" || ok["duration_ms"] == nil || ok["bytes"] == nil || ok["links"] == nil || ok["error"] != nil {
		t.Fatalf("success line %v", ok)
	}
	if bad["error"] != "not_found" {
		t.Fatalf("error line %v", bad)
	}
	if strings.Contains(e.log.String(), "ecret") || strings.Contains(e.log.String(), "body text") {
		t.Fatalf("log holds argument or note text: %s", e.log.String())
	}
}

func TestServer_SPEC0004_R6_inFlightCallFinishes(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	release := make(chan struct{})
	entered := make(chan struct{})
	e.srv.SetBeforeCall(func() {
		close(entered)
		<-release
	})
	result := make(chan *mcp.CallToolResult, 1)
	go func() {
		r, _ := e.cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "note_read", Arguments: map[string]any{"ref": "a"}})
		result <- r
	}()
	<-entered
	e.cancel()
	select {
	case <-e.done:
		t.Fatal("Serve returned while a call was in progress")
	case <-time.After(200 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-e.done:
		e.done <- err // let the cleanup see that the server stopped
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the call finished")
	}
	if r := <-result; r != nil && r.IsError {
		t.Fatalf("in-flight call failed: %+v", r)
	}
}
