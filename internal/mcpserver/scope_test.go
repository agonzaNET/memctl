package mcpserver_test

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/agonzaNET/memctl/internal/mcpserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const testScope = "Alberto's personal notes: family, homelab, hobbies. No appliedAI work."

var scopeFiles = map[string]string{"a.md": "---\ntitle: A\n---\nbody\n"}

func listTools(t *testing.T, e *env) map[string]*mcp.Tool {
	t.Helper()
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		out[tool.Name] = tool
	}
	if len(out) != len(mcpserver.Descriptions) {
		t.Fatalf("listed %d tools, want %d", len(out), len(mcpserver.Descriptions))
	}
	return out
}

func TestScope_SPEC0011_R5_instructions(t *testing.T) {
	e := startScoped(t, scopeFiles, testScope)
	want := mcpserver.Instructions + "\n" + mcpserver.ScopePrefix + testScope
	if got := e.cs.InitializeResult().Instructions; got != want {
		t.Fatalf("instructions\n%q\nwant\n%q", got, want)
	}
	if mcpserver.ScopePrefix != "Scope of this memory, set by the person who runs it: " {
		t.Fatalf("scope prefix %q", mcpserver.ScopePrefix)
	}
}

func TestScope_SPEC0011_R6_everyDescription(t *testing.T) {
	e := startScoped(t, scopeFiles, testScope)
	for name, tool := range listTools(t, e) {
		want := mcpserver.Descriptions[name] + "\n" + mcpserver.ScopePrefix + testScope
		if tool.Description != want {
			t.Errorf("%s description\n%q\nwant\n%q", name, tool.Description, want)
		}
	}
}

func TestScope_SPEC0011_R7_noScopeLeavesTextAsIs(t *testing.T) {
	e := startScoped(t, scopeFiles, "")
	if got := e.cs.InitializeResult().Instructions; got != mcpserver.Instructions {
		t.Fatalf("instructions changed without a scope: %q", got)
	}
	for name, tool := range listTools(t, e) {
		if tool.Description != mcpserver.Descriptions[name] {
			t.Errorf("%s description changed without a scope", name)
		}
	}
}

// durationField hides the only field of a tool call line that may differ.
var durationField = regexp.MustCompile(`"duration_ms":[0-9.]+`)

func TestScope_SPEC0011_R8_nothingElseChanges(t *testing.T) {
	plain := startScoped(t, scopeFiles, "")
	scoped := startScoped(t, scopeFiles, testScope)
	pt, st := listTools(t, plain), listTools(t, scoped)
	for name := range pt {
		a, b := *pt[name], *st[name]
		a.Description, b.Description = "", ""
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if string(ja) != string(jb) {
			t.Errorf("%s differs beyond its description:\n%s\n%s", name, ja, jb)
		}
	}
	calls := []struct {
		tool string
		args map[string]any
	}{
		{"note_read", map[string]any{"ref": "a"}},
		{"note_write", map[string]any{"title": "B", "body": "b"}},
		{"note_read", map[string]any{"ref": "missing"}},
	}
	for _, c := range calls {
		ra, rb := plain.call(t, c.tool, c.args), scoped.call(t, c.tool, c.args)
		if ra.IsError != rb.IsError || text(t, ra) != text(t, rb) {
			t.Errorf("%s result differs:\n%q\n%q", c.tool, text(t, ra), text(t, rb))
		}
	}
	la, lb := plain.logLines(t, "tool call"), scoped.logLines(t, "tool call")
	if len(la) != len(calls) || len(la) != len(lb) {
		t.Fatalf("tool call lines %d and %d, want %d", len(la), len(lb), len(calls))
	}
	for i := range la {
		delete(la[i], "time")
		delete(lb[i], "time")
		ja, _ := json.Marshal(la[i])
		jb, _ := json.Marshal(lb[i])
		if durationField.ReplaceAllString(string(ja), "") != durationField.ReplaceAllString(string(jb), "") {
			t.Errorf("tool call line differs:\n%s\n%s", ja, jb)
		}
	}
}
