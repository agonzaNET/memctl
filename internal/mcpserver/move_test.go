package mcpserver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMove_SPEC0007_R20_R23_result(t *testing.T) {
	e := start(t, map[string]string{
		"hosts/a.md": "---\ntitle: A\n---\n",
		"p.md":       "see [[hosts/a]] and [[A]]\n",
		"q.md":       "[[A]]\n",
		"taken.md":   "t\n",
	})
	got := text(t, e.call(t, "note_move", map[string]any{"ref": "hosts/a", "new_path": "servers/a.md"}))
	b, err := os.ReadFile(filepath.Join(e.root, "servers", "a.md"))
	if err != nil {
		t.Fatalf("%v; result %s", err, got)
	}
	first := "Moved \"A\" from hosts/a.md to servers/a.md.\n\nsha256: " + sum(string(b)) + "\n\n"
	if !strings.HasPrefix(got, first) || !strings.Contains(got, "2 notes link") || !strings.Contains(got, "aliases") || !strings.Contains(got, "memctl check") {
		t.Fatalf("result:\n%s", got)
	}
	same := text(t, e.call(t, "note_move", map[string]any{"ref": "servers/a.md", "new_path": "servers/a.md"}))
	if same != "\"A\" is already at servers/a.md." {
		t.Fatalf("same: %q", same)
	}
	for _, c := range []struct {
		args map[string]any
		code string
	}{
		{map[string]any{"ref": "servers/a.md", "new_path": "taken.md"}, "exists"},
		{map[string]any{"ref": "servers/a.md"}, "invalid_argument"},
		{map[string]any{"ref": "servers/a.md", "new_path": "x.txt"}, "invalid_argument"},
		{map[string]any{"ref": "nothing", "new_path": "n.md"}, "not_found"},
	} {
		if got := text(t, e.call(t, "note_move", c.args)); !strings.HasPrefix(got, c.code+": ") {
			t.Errorf("%v: %q, want %s", c.args, got, c.code)
		}
	}
}

func TestMove_SPEC0007_R25_R27_annotationsAndLog(t *testing.T) {
	e := start(t, map[string]string{"secret/a.md": "---\ntitle: Secret\n---\n", "l.md": "[[Secret]]\n"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range res.Tools {
		if tool.Name == "note_move" {
			a := tool.Annotations
			if a == nil || a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint || a.IdempotentHint || a.OpenWorldHint == nil || *a.OpenWorldHint {
				t.Fatalf("annotations %+v", a)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("note_move not listed")
	}
	e.call(t, "note_move", map[string]any{"ref": "Secret", "new_title": "Hidden name"})
	lines := e.logLines(t, "tool call")
	l := lines[len(lines)-1]
	b, _ := os.ReadFile(filepath.Join(e.root, "secret", "Hidden name.md"))
	if l["tool"] != "note_move" || l["bytes"] != float64(len(b)) || l["paths"] != 2.0 || l["linkers"] != 1.0 {
		t.Fatalf("log line %v", l)
	}
	if s := e.log.String(); strings.Contains(s, "ecret") || strings.Contains(s, "Hidden") {
		t.Fatalf("log holds argument text: %s", s)
	}
}
