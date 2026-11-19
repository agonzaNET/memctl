package mcpserver_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func graphFiles() map[string]string {
	return map[string]string{
		"s.md":     "---\ntitle: Start\n---\nmentions [[b]] and [[nowhere]] and [[dup]]\n\n## Relations\n- depends_on [[a]]\n",
		"a.md":     "---\ntitle: A\n---\n\n## Relations\n- runs_on [[c]]\n",
		"b.md":     "---\ntitle: B\n---\n",
		"c.md":     "---\ntitle: C\n---\n",
		"in.md":    "---\ntitle: In\n---\n\n## Relations\n- see_also [[s]]\n",
		"x/dup.md": "---\ntitle: X\n---\n",
		"y/dup.md": "---\ntitle: Y\n---\n",
	}
}

func TestContext_SPEC0004_R22_render(t *testing.T) {
	e := start(t, graphFiles())
	got := text(t, e.call(t, "context", map[string]any{"ref": "s"}))
	want := `Found 3 notes within 1 hop of "Start".

Notes:
- 0 · s.md — Start
- 1 · a.md — A
- 1 · b.md — B
- 1 · in.md — In

Edges:
- in.md —see_also→ s.md
- s.md —links_to→ b.md
- s.md —links_to→ dup: ambiguous (x/dup.md, y/dup.md)
- s.md —links_to→ nowhere: dangling
- s.md —depends_on→ a.md`
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}

	got = text(t, e.call(t, "context", map[string]any{"ref": "s.md", "depth": 2, "rel_types": []string{"depends_on", "runs_on"}}))
	if !strings.HasPrefix(got, `Found 2 notes within 2 hops of "Start".`) || !strings.Contains(got, "- 2 · c.md — C") || strings.Contains(got, "links_to") {
		t.Fatalf("depth 2 with rel_types:\n%s", got)
	}

	r := e.call(t, "context", map[string]any{"ref": "dup"})
	if !r.IsError || !strings.HasPrefix(text(t, r), "ambiguous: ") {
		t.Fatalf("ambiguous ref: %s", text(t, r))
	}
	if r := e.call(t, "context", map[string]any{"ref": "s", "depth": 4}); !r.IsError || strings.HasPrefix(text(t, r), "invalid_argument") {
		t.Fatalf("depth 4 must be refused by the schema: %s", text(t, r))
	}
}

func TestContext_SPEC0004_R22_truncation(t *testing.T) {
	files := map[string]string{}
	hub := "---\ntitle: hub\n---\n"
	for i := 0; i < 250; i++ {
		hub += fmt.Sprintf("[[n%03d]]\n", i)
		files[fmt.Sprintf("n%03d.md", i)] = "x\n"
	}
	files["hub.md"] = hub
	e := start(t, files)
	got := text(t, e.call(t, "context", map[string]any{"ref": "hub"}))
	if !strings.HasSuffix(got, "The walk stopped at 200 notes; pass a smaller depth or rel_types to narrow it.") {
		t.Fatalf("truncation line missing:\n%s", got[len(got)-200:])
	}
}

func TestContext_SPEC0004_R23_counts(t *testing.T) {
	e := start(t, graphFiles())
	e.call(t, "context", map[string]any{"ref": "s", "depth": 2})
	l := e.logLines(t, "tool call")
	if l[0]["tool"] != "context" || l[0]["notes"] != 5.0 || l[0]["edges"] != 6.0 || l[0]["depth"] != 2.0 {
		t.Fatalf("log %v", l)
	}
}

func TestList_SPEC0004_R24_render(t *testing.T) {
	e := start(t, map[string]string{
		"ops/b.md":   "---\ntitle: B\ntype: runbook\ntags: [fw]\nupdated: 2026-11-19T10:00:00+01:00\n---\n",
		"ops/a.md":   "---\ntitle: A\n---\n",
		"other/c.md": "---\ntitle: C\ntype: runbook\n---\n",
	})
	got := text(t, e.call(t, "list", map[string]any{}))
	want := "Listed 3 notes.\n\n- ops/a.md — A\n- ops/b.md — B (runbook) · updated 2026-11-19\n- other/c.md — C (runbook)"
	if got != want {
		t.Fatalf("got:\n%s", got)
	}
	got = text(t, e.call(t, "list", map[string]any{"folder": "ops/", "type": "runbook", "tag": "FW"}))
	if got != "Listed 1 note.\n\n- ops/b.md — B (runbook) · updated 2026-11-19" {
		t.Fatalf("filtered:\n%s", got)
	}
	if got := text(t, e.call(t, "list", map[string]any{"type": "host"})); got != "Found no notes that match." {
		t.Fatalf("empty: %q", got)
	}
	if r := e.call(t, "list", map[string]any{"folder": "../x"}); !r.IsError || !strings.HasPrefix(text(t, r), "invalid_argument: ") {
		t.Fatalf("bad folder: %s", text(t, r))
	}
}

func TestList_SPEC0004_R24_R25_cap(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 501; i++ {
		files[fmt.Sprintf("n%04d.md", i)] = "x\n"
	}
	e := start(t, files)
	got := text(t, e.call(t, "list", map[string]any{}))
	lines := strings.Split(got, "\n")
	if !strings.HasPrefix(got, "Listed 500 of 501 notes.") || len(lines) != 504 ||
		lines[len(lines)-1] != "The list shows the first 500 of 501 notes; pass folder, type or tag to narrow it." {
		t.Fatalf("cap: first %q last %q len %d", lines[0], lines[len(lines)-1], len(lines))
	}
	l := e.logLines(t, "tool call")
	if l[0]["tool"] != "list" || l[0]["results"] != 501.0 {
		t.Fatalf("log %v", l)
	}
}

func TestServer_SPEC0004_R5_fourTools(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	// The four read tools of SPEC-0004 and the four write tools of SPEC-0007.
	want := []string{"context", "list", "note_read", "search"}
	for _, w := range []string{"note_write", "note_edit", "note_move", "note_delete"} {
		want = append(want, w)
	}
	sort.Strings(want)
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("tools %v, want %v", names, want)
	}
}
