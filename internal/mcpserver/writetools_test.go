package mcpserver_test

import (
	"context"
	"strings"
	"testing"
)

// TestTools_SPEC0007_R25_annotationTable checks the hints of every tool
// against the design's tool table (SPEC-0004 R8, SPEC-0007 R25).
func TestTools_SPEC0007_R25_annotationTable(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	type hints struct{ readOnly, destructive, idempotent bool }
	want := map[string]hints{
		"note_read":   {true, false, true},
		"search":      {true, false, true},
		"context":     {true, false, true},
		"list":        {true, false, true},
		"note_write":  {false, false, false},
		"note_edit":   {false, false, false},
		"note_move":   {false, false, false},
		"note_delete": {false, true, true},
	}
	if len(res.Tools) != len(want) {
		t.Fatalf("%d tools, want %d", len(res.Tools), len(want))
	}
	for _, tool := range res.Tools {
		w, ok := want[tool.Name]
		a := tool.Annotations
		if !ok || a == nil || a.DestructiveHint == nil || a.OpenWorldHint == nil {
			t.Fatalf("%s: %+v", tool.Name, a)
		}
		got := hints{a.ReadOnlyHint, *a.DestructiveHint, a.IdempotentHint}
		if got != w || *a.OpenWorldHint {
			t.Errorf("%s: %+v open world %v, want %+v and no open world", tool.Name, got, *a.OpenWorldHint, w)
		}
	}
}

// TestLog_SPEC0007_R27_countsOnly calls every write tool with arguments
// that are easy to spot, and checks that the tool-call lines carry the
// counts and none of the values.
func TestLog_SPEC0007_R27_countsOnly(t *testing.T) {
	e := start(t, map[string]string{"zz-linker.md": "[[Qx title]]\n"})
	e.call(t, "note_write", map[string]any{"title": "Qx title", "folder": "qxfolder", "body": "qxbody", "type": "qxtype", "tags": []any{"qxtag"}})
	sha := e.readSha(t, "qxfolder/Qx title.md")
	e.call(t, "note_edit", map[string]any{"ref": "Qx title", "expected_sha": sha, "op": "find_replace", "find": "qxbody", "replace": "qxreplace"})
	e.call(t, "note_move", map[string]any{"ref": "qxfolder/Qx title", "new_path": "qxnew/Qx2.md", "new_title": "Qx2 title"})
	e.call(t, "note_delete", map[string]any{"ref": "qxnew/Qx2"})

	want := map[string][]string{
		"note_write":  {"bytes", "paths"},
		"note_edit":   {"bytes", "paths"},
		"note_move":   {"bytes", "paths", "linkers"},
		"note_delete": {"bytes", "paths", "linkers"},
	}
	seen := 0
	for _, l := range e.logLines(t, "tool call") {
		counts, ok := want[l["tool"].(string)]
		if !ok {
			continue
		}
		seen++
		for _, c := range counts {
			if _, ok := l[c].(float64); !ok {
				t.Errorf("%s line lacks the count %s: %v", l["tool"], c, l)
			}
		}
		if l["error"] != nil {
			t.Errorf("%s failed: %v", l["tool"], l)
		}
	}
	if seen != 4 {
		t.Fatalf("%d write tool lines, want 4: %s", seen, e.log.String())
	}
	if s := strings.ToLower(e.log.String()); strings.Contains(s, "qx") {
		t.Fatalf("log holds an argument value: %s", e.log.String())
	}
}

// TestServer_SPEC0007_R26_instructionsNameTheWriteFlow checks that the
// server instructions tell the agent how to change a note safely.
func TestServer_SPEC0007_R26_instructionsNameTheWriteFlow(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	ins := e.cs.InitializeResult().Instructions
	for _, s := range []string{"note_edit", "sha256", "git"} {
		if !strings.Contains(ins, s) {
			t.Errorf("instructions do not mention %q:\n%s", s, ins)
		}
	}
}
