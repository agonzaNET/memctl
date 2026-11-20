package mcpserver_test

import (
	"fmt"
	"context"
	"strings"
	"testing"
)

func TestDelete_SPEC0007_R24_result(t *testing.T) {
	e := start(t, map[string]string{"hosts/a.md": "---\ntitle: A\n---\n", "l.md": "[[A]]\n", "m.md": "[[hosts/a]]\n"})
	got := text(t, e.call(t, "note_delete", map[string]any{"ref": "hosts/a"}))
	if !strings.HasPrefix(got, "Deleted \"A\" at hosts/a.md.\n\n") || !strings.Contains(got, "2 notes") || !strings.Contains(got, "git") {
		t.Fatalf("result:\n%s", got)
	}
	again := e.call(t, "note_delete", map[string]any{"ref": "hosts/a"})
	if !again.IsError || !strings.HasPrefix(text(t, again), "not_found: ") {
		t.Fatalf("second delete: %s", text(t, again))
	}
	lone := text(t, e.call(t, "note_delete", map[string]any{"ref": "l.md"}))
	if !strings.HasPrefix(lone, "Deleted \"l\" at l.md.\n\n") || !strings.Contains(lone, "No other note") {
		t.Fatalf("lone: %s", lone)
	}
}

func TestDelete_SPEC0007_R25_R27_annotationsAndLog(t *testing.T) {
	e := start(t, map[string]string{"secret/a.md": "---\ntitle: Secret\n---\n", "l.md": "[[Secret]]\n"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range res.Tools {
		if tool.Name == "note_delete" {
			a := tool.Annotations
			if a == nil || a.ReadOnlyHint || a.DestructiveHint == nil || !*a.DestructiveHint || !a.IdempotentHint || a.OpenWorldHint == nil || *a.OpenWorldHint {
				t.Fatalf("annotations %+v", a)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("note_delete not listed")
	}
	e.call(t, "note_delete", map[string]any{"ref": "Secret"})
	lines := e.logLines(t, "tool call")
	l := lines[len(lines)-1]
	if l["tool"] != "note_delete" || l["bytes"] != 0.0 || l["paths"] != 1.0 || l["linkers"] != 1.0 {
		t.Fatalf("log line %v", l)
	}
	if s := e.log.String(); strings.Contains(s, "ecret") {
		t.Fatalf("log holds argument text: %s", s)
	}
}

func TestDeleteMove_SPEC0007_R23_R24_linkerNames(t *testing.T) {
	files := map[string]string{"t.md": "---\ntitle: T\n---\n", "u.md": "---\ntitle: U\n---\n"}
	for i := 0; i < 3; i++ {
		files[fmt.Sprintf("l%d.md", i)] = "[[t]]\n"
	}
	for i := 0; i < 12; i++ {
		files[fmt.Sprintf("m%02d.md", i)] = "[[u]]\n"
	}
	e := start(t, files)
	got := text(t, e.call(t, "note_move", map[string]any{"ref": "t", "new_path": "x/t.md"}))
	if !strings.Contains(got, "3 notes link to it: l0.md, l1.md, l2.md.") {
		t.Fatalf("move:\n%s", got)
	}
	got = text(t, e.call(t, "note_delete", map[string]any{"ref": "u"}))
	if !strings.Contains(got, "12 notes link to it: m00.md, m01.md, m02.md, m03.md, m04.md, m05.md, m06.md, m07.md, m08.md, m09.md and 2 more.") {
		t.Fatalf("delete:\n%s", got)
	}
}
