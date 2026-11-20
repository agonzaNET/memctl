package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
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
	want := `Found 3 notes within 1 hop of "Start", plus the note itself.

Notes:
- 0 · s.md — Start
- 1 · a.md — A
- 1 · b.md — B
- 1 · in.md — In

Edges (5):
- in.md —see_also→ s.md
- s.md —links_to→ b.md
- s.md —links_to→ dup: ambiguous (x/dup.md, y/dup.md)
- s.md —links_to→ nowhere: dangling
- s.md —depends_on→ a.md`
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}

	got = text(t, e.call(t, "context", map[string]any{"ref": "s.md", "depth": 2, "rel_types": []string{"depends_on", "runs_on"}}))
	if !strings.HasPrefix(got, `Found 2 notes within 2 hops of "Start", plus the note itself.`) || !strings.Contains(got, "- 2 · c.md — C") || strings.Contains(got, "links_to") {
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
	mt := time.Date(2026, 3, 4, 12, 0, 0, 0, time.Local)
	for _, p := range []string{"ops/a.md", "ops/b.md", "other/c.md"} {
		if err := os.Chtimes(filepath.Join(e.root, p), mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	got := text(t, e.call(t, "list", map[string]any{}))
	want := "Listed 3 notes.\n\n- ops/a.md — A · file changed 2026-03-04\n- ops/b.md — B (runbook) · tags fw · updated 2026-11-19\n- other/c.md — C (runbook) · file changed 2026-03-04"
	if got != want {
		t.Fatalf("got:\n%s", got)
	}
	got = text(t, e.call(t, "list", map[string]any{"folder": "ops/", "type": "runbook", "tag": "FW"}))
	if got != "Listed 1 note.\n\n- ops/b.md — B (runbook) · tags fw · updated 2026-11-19" {
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
	want = append(want, "note_write", "note_edit", "note_move", "note_delete")
	sort.Strings(want)
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("tools %v, want %v", names, want)
	}
}

func TestContext_SPEC0004_R22_descriptionNamesEdgeRule(t *testing.T) {
	if d := toolDescription(t, "context"); !strings.Contains(d, "every connection between the listed notes") {
		t.Fatalf("description:\n%s", d)
	}
}

func resolutionFiles() map[string]string {
	return map[string]string{
		"notes/alpha.md":   "---\ntitle: First\n---\nx\n",
		"archive/alpha.md": "---\ntitle: Alpha\n---\nx\n",
		"x/b.md":           "---\ntitle: Bee\naliases: [bee-alias]\n---\nx\n",
	}
}

func TestRead_SPEC0004_R27_resolution(t *testing.T) {
	e := start(t, resolutionFiles())
	cases := []struct {
		ref        string
		want, none []string
	}{
		{"Bee", []string{"\n- resolved by: title \"Bee\"\n"}, []string{"other notes"}},
		{"bee-alias", []string{"\n- resolved by: alias \"bee-alias\"\n"}, nil},
		{"notes/alpha", nil, []string{"resolved by", "other notes"}},
		{"alpha", []string{"\n- resolved by: title \"alpha\"\n", "\n- other notes with this name: notes/alpha.md\n"}, nil},
	}
	for _, c := range cases {
		got := text(t, e.call(t, "note_read", map[string]any{"ref": c.ref}))
		for _, w := range c.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: missing %q in:\n%s", c.ref, w, got)
			}
		}
		for _, w := range c.none {
			if strings.Contains(got, w) {
				t.Errorf("%s: unexpected %q in:\n%s", c.ref, w, got)
			}
		}
		if strings.Contains(got, "resolved by") && strings.Index(got, "resolved by") < strings.Index(got, "- sha256: ") {
			t.Errorf("%s: resolution item must follow sha256:\n%s", c.ref, got)
		}
	}

	f := resolutionFiles()
	f["alpha.md"] = "---\ntitle: Root\n---\nx\n"
	e = start(t, f)
	got := text(t, e.call(t, "note_read", map[string]any{"ref": "alpha"}))
	if !strings.Contains(got, "\n- other notes with this name: archive/alpha.md, notes/alpha.md\n") || strings.Contains(got, "resolved by") {
		t.Fatalf("path ref with other names:\n%s", got)
	}
}

func TestContext_SPEC0004_R27_resolution(t *testing.T) {
	e := start(t, resolutionFiles())
	got := text(t, e.call(t, "context", map[string]any{"ref": "alpha"}))
	want := "Found no notes within 1 hop of \"Alpha\", plus the note itself.\n\nResolved by title \"alpha\".\nOther notes with this name: notes/alpha.md.\n\nNotes:\n"
	if !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s", got)
	}
	got = text(t, e.call(t, "context", map[string]any{"ref": "notes/alpha"}))
	if strings.Contains(got, "Resolved by") || strings.Contains(got, "Other notes") {
		t.Fatalf("path ref:\n%s", got)
	}
}

func TestContext_SPEC0004_R22_edgeAtMaxDepth(t *testing.T) {
	e := start(t, map[string]string{
		"s.md": "---\ntitle: S\n---\n[[a]] [[b]]\n",
		"a.md": "---\ntitle: A\n---\n[[c]]\n",
		"b.md": "---\ntitle: B\n---\n[[d]]\n",
		"c.md": "---\ntitle: C\n---\n[[d]]\nagain [[d]]\n",
		"d.md": "---\ntitle: D\n---\n",
	})
	got := text(t, e.call(t, "context", map[string]any{"ref": "s", "depth": 2}))
	if !strings.HasPrefix(got, `Found 4 notes within 2 hops of "S", plus the note itself.`) || strings.Count(got, "- c.md —links_to→ d.md") != 1 {
		t.Fatalf("got:\n%s", got)
	}
}

func TestList_SPEC0004_R30_descriptionNamesDates(t *testing.T) {
	d := toolDescription(t, "list")
	for _, w := range []string{"updated", "file changed", "tags"} {
		if !strings.Contains(d, w) {
			t.Fatalf("description lacks %q:\n%s", w, d)
		}
	}
}

func TestContext_SPEC0004_R31_edgesStart(t *testing.T) {
	e := start(t, graphFiles())
	got := text(t, e.call(t, "context", map[string]any{"ref": "s", "depth": 2, "edges": "start"}))
	want := `Found 4 notes within 2 hops of "Start", plus the note itself.

Notes:
- 0 · s.md — Start
- 1 · a.md — A
- 1 · b.md — B
- 1 · in.md — In
- 2 · c.md — C (via a.md)

Edges of the start note (5):
- in.md —see_also→ s.md
- s.md —links_to→ b.md
- s.md —links_to→ dup: ambiguous (x/dup.md, y/dup.md)
- s.md —links_to→ nowhere: dangling
- s.md —depends_on→ a.md`
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}

	def := text(t, e.call(t, "context", map[string]any{"ref": "s", "depth": 2}))
	all := text(t, e.call(t, "context", map[string]any{"ref": "s", "depth": 2, "edges": "all"}))
	if def != all || !strings.Contains(all, "\nEdges (6):\n") || !strings.Contains(all, "- a.md —runs_on→ c.md") || strings.Contains(all, "(via ") {
		t.Fatalf("default and all differ or miss an edge:\n%s\n\n%s", def, all)
	}
	if r := e.call(t, "context", map[string]any{"ref": "s", "edges": "other"}); !r.IsError || strings.HasPrefix(text(t, r), "invalid_argument") {
		t.Fatalf("edges other must be refused by the schema: %s", text(t, r))
	}
}

func TestContext_SPEC0004_R31_schemaAndDescription(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "context" {
			continue
		}
		raw, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]struct {
				Type    string   `json:"type"`
				Enum    []string `json:"enum"`
				Default any      `json:"default"`
			} `json:"properties"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		p := schema.Properties["edges"]
		if p.Type != "string" || !reflect.DeepEqual(p.Enum, []string{"all", "start"}) || p.Default != "all" {
			t.Fatalf("edges schema %s", raw)
		}
	}
	d := toolDescription(t, "context")
	for _, s := range []string{
		"every connection between the listed notes",
		"unless edges is start",
		"edges start to get the same notes with only the start note's connections",
		"names the note it was reached through",
		"edges all, the default",
		"dozens of notes and hundreds of connections",
		"depth 1 lists fewer notes, and edges start fewer connections",
	} {
		if !strings.Contains(d, s) {
			t.Fatalf("description lacks %q:\n%s", s, d)
		}
	}
}

func TestContext_SPEC0004_R22_R31_noEdges(t *testing.T) {
	e := start(t, map[string]string{"lone.md": "---\ntitle: Lone\n---\nalone\n"})
	if got := text(t, e.call(t, "context", map[string]any{"ref": "lone"})); !strings.HasSuffix(got, "\n\nEdges: none.") {
		t.Fatalf("all:\n%s", got)
	}
	if got := text(t, e.call(t, "context", map[string]any{"ref": "lone", "edges": "start"})); !strings.HasSuffix(got, "\n\nEdges of the start note: none.") {
		t.Fatalf("start:\n%s", got)
	}
}
