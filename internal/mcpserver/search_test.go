package mcpserver_test

import (
	"fmt"
	"strings"
	"testing"
)

func searchFixture(t *testing.T) *env {
	return start(t, map[string]string{
		"runbooks/Upgrade PAN-OS.md": "---\ntitle: Upgrade PAN-OS\ntype: runbook\ntags: [firewall]\n---\nSteps for the firewall upgrade.\n\n## Observations\n- [risk] Upgrade the passive firewall first\n",
		"hosts/PA-850.md":            "---\ntitle: PA-850\ntype: host\ntags: [firewall]\n---\nThe main firewall.\n",
		"notes/misc.md":              "---\ntitle: Misc\n---\nNothing here.\n",
	})
}

func TestSearch_SPEC0004_R19_render(t *testing.T) {
	e := searchFixture(t)
	r := e.call(t, "search", map[string]any{"query": "firewall upgrade"})
	got := text(t, r)
	want := `Found 2 notes for "firewall upgrade".

1. **Upgrade PAN-OS** — runbooks/Upgrade PAN-OS.md (runbook) · tags firewall · score `
	if r.IsError || !strings.HasPrefix(got, want) {
		t.Fatalf("got:\n%s", got)
	}
	for _, s := range []string{
		"   > L9: - [risk] Upgrade the passive firewall first\n",
		"2. **PA-850** — hosts/PA-850.md (host) · tags firewall · score ",
		"   > L6: The main firewall.",
	} {
		if !strings.Contains(got, s) {
			t.Fatalf("missing %q in:\n%s", s, got)
		}
	}

	r = e.call(t, "search", map[string]any{"query": "upgrade", "type": "host"})
	if r.IsError || text(t, r) != `Found no notes for "upgrade".` {
		t.Fatalf("no hits: %q", text(t, r))
	}
	r = e.call(t, "search", map[string]any{"query": "firewall", "tags": []string{"firewall"}, "folder": "./hosts/", "limit": 1})
	if !strings.HasPrefix(text(t, r), `Found 1 note for "firewall".`) || !strings.Contains(text(t, r), "hosts/PA-850.md") {
		t.Fatalf("filters: %s", text(t, r))
	}
	r = e.call(t, "search", map[string]any{"query": "nothing"})
	if !strings.Contains(text(t, r), "**Misc** — notes/misc.md · score") {
		t.Fatalf("no type: %s", text(t, r))
	}

	if strings.Contains(got, "showing the first") || strings.Contains(got, "up to 50") {
		t.Fatalf("all hits shown, yet the more line appears:\n%s", got)
	}

	r = e.call(t, "search", map[string]any{"query": "!!! ---"})
	if !r.IsError || !strings.HasPrefix(text(t, r), "invalid_argument: ") {
		t.Fatalf("empty query: %s", text(t, r))
	}
}

func TestSearch_SPEC0004_R20_folderRules(t *testing.T) {
	e := searchFixture(t)
	for _, f := range []string{"../x", "a/../b", `a\b`, "/abs"} {
		r := e.call(t, "search", map[string]any{"query": "firewall", "folder": f})
		if !r.IsError || !strings.HasPrefix(text(t, r), "invalid_argument: ") {
			t.Fatalf("folder %q: %s", f, text(t, r))
		}
	}
}

func TestSearch_SPEC0004_R21_counts(t *testing.T) {
	e := searchFixture(t)
	e.call(t, "search", map[string]any{"query": "firewall secret-word firewall"})
	l := e.logLines(t, "tool call")
	if len(l) != 1 || l[0]["tool"] != "search" || l[0]["results"] != 2.0 || l[0]["terms"] != 2.0 { // "secret-word" is one term (SPEC-0002 R24)
		t.Fatalf("log %v", l)
	}
	if strings.Contains(e.log.String(), "secret") {
		t.Fatalf("query in log: %s", e.log.String())
	}
}

func TestSearch_SPEC0004_R9_schemaLimits(t *testing.T) {
	e := searchFixture(t)
	for _, args := range []map[string]any{
		{"query": "x", "limit": 0},
		{"query": "x", "limit": 51},
		{"query": ""},
		{"query": "x", "tags": []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}},
	} {
		if r := e.call(t, "search", args); !r.IsError || strings.HasPrefix(text(t, r), "invalid_argument") {
			t.Fatalf("args %v should be refused by the schema: %s", args, text(t, r))
		}
	}
}

func TestSearch_SPEC0004_R29_more(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 25; i++ {
		files[fmt.Sprintf("n/n%02d.md", i)] = "---\ntitle: n\n---\nword\n"
	}
	e := start(t, files)
	r := e.call(t, "search", map[string]any{"query": "word"})
	got := text(t, r)
	if !strings.HasPrefix(got, `Found 25 notes for "word"; showing the first 10.`) {
		t.Fatalf("first line:\n%s", got)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	last := lines[len(lines)-1]
	for _, w := range []string{"limit", "50", "type", "tags", "folder"} {
		if !strings.Contains(last, w) {
			t.Fatalf("last line %q lacks %q", last, w)
		}
	}
	if strings.Count(got, "**n**") != 10 {
		t.Fatalf("want 10 hits:\n%s", got)
	}
}

func TestSearch_SPEC0004_R29_maximumLimit(t *testing.T) {
	files := map[string]string{}
	for i := 0; i < 55; i++ {
		files[fmt.Sprintf("n/n%02d.md", i)] = "---\ntitle: n\n---\nword\n"
	}
	e := start(t, files)
	got := text(t, e.call(t, "search", map[string]any{"query": "word", "limit": 50}))
	if !strings.HasPrefix(got, `Found 55 notes for "word"; showing the first 50.`) {
		t.Fatalf("first line:\n%s", got)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	last := lines[len(lines)-1]
	for _, w := range []string{"type", "tags", "folder"} {
		if !strings.Contains(last, w) {
			t.Fatalf("last line %q lacks %q", last, w)
		}
	}
	if strings.Contains(last, "limit") {
		t.Fatalf("last line %q offers a higher limit at the maximum", last)
	}
}

func TestSearch_SPEC0004_R19_descriptionNamesIdentifiers(t *testing.T) {
	d := toolDescription(t, "search")
	for _, w := range []string{"IT-104", "tags", "how many"} {
		if !strings.Contains(d, w) {
			t.Fatalf("description lacks %q:\n%s", w, d)
		}
	}
}
