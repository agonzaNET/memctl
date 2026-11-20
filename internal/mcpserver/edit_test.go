package mcpserver_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var shaLine = regexp.MustCompile(`(?m)^- sha256: ([0-9a-f]{64})$`)

// readSha reads a note through note_read and returns its sha256.
func (e *env) readSha(t *testing.T, ref string) string {
	t.Helper()
	m := shaLine.FindStringSubmatch(text(t, e.call(t, "note_read", map[string]any{"ref": ref})))
	if m == nil {
		t.Fatalf("no sha256 in note_read of %s", ref)
	}
	return m[1]
}

const editNote = "---\ntitle: Runbook\n---\n# Steps\nold step\n# Notes\nn\n"

func TestEdit_SPEC0007_R15_R16_readThenEdit(t *testing.T) {
	e := start(t, map[string]string{"r.md": editNote})
	sha := e.readSha(t, "r.md")
	res := e.call(t, "note_edit", map[string]any{"ref": "Runbook", "expected_sha": sha, "op": "append", "text": "appended"})
	got := text(t, res)
	b, _ := os.ReadFile(filepath.Join(e.root, "r.md"))
	if res.IsError || got != "Edited \"Runbook\" at r.md.\n\nsha256: "+sum(string(b)) {
		t.Fatalf("result %q", got)
	}
	if !strings.HasSuffix(string(b), "n\n\nappended\n") || !strings.Contains(string(b), "updated: ") {
		t.Fatalf("file %q", b)
	}
	// The returned sha is good for the next edit.
	next := e.call(t, "note_edit", map[string]any{"ref": "r.md", "expected_sha": sum(string(b)), "op": "replace_section", "section": "steps", "text": "new step"})
	if next.IsError {
		t.Fatalf("second edit: %s", text(t, next))
	}
}

func TestEdit_SPEC0007_R15_conflictCarriesCurrentState(t *testing.T) {
	e := start(t, map[string]string{"r.md": editNote})
	sha := e.readSha(t, "r.md")
	human := editNote + "edited in an editor\n"
	os.WriteFile(filepath.Join(e.root, "r.md"), []byte(human), 0o644)
	res := e.call(t, "note_edit", map[string]any{"ref": "r.md", "expected_sha": sha, "op": "append", "text": "x"})
	got := text(t, res)
	if !res.IsError || !strings.HasPrefix(got, "conflict: ") || !strings.Contains(got, sum(human)) || !strings.Contains(got, "````markdown\n"+human+"````") {
		t.Fatalf("conflict result:\n%s", got)
	}
	if b, _ := os.ReadFile(filepath.Join(e.root, "r.md")); string(b) != human {
		t.Fatalf("human edit overwritten: %q", b)
	}
}

func TestEdit_SPEC0007_R17_R18_R19_codes(t *testing.T) {
	e := start(t, map[string]string{"r.md": editNote + "dup dup\n"})
	sha := e.readSha(t, "r.md")
	cases := []struct {
		args map[string]any
		code string
		says []string
	}{
		{map[string]any{"op": "replace_section", "section": "Missing", "text": "x"}, "section_not_found", []string{"Steps", "Notes"}},
		{map[string]any{"op": "find_replace", "find": "absent", "replace": "x"}, "no_match", nil},
		{map[string]any{"op": "find_replace", "find": "dup", "replace": "x"}, "not_unique", []string{"2"}},
		{map[string]any{"op": "append"}, "invalid_argument", []string{"text"}},
		{map[string]any{"op": "replace_section", "text": "x"}, "invalid_argument", []string{"section"}},
	}
	for _, c := range cases {
		c.args["ref"] = "r.md"
		c.args["expected_sha"] = sha
		got := text(t, e.call(t, "note_edit", c.args))
		if !strings.HasPrefix(got, c.code+": ") {
			t.Errorf("%v: %q, want %s", c.args, got, c.code)
		}
		for _, s := range c.says {
			if !strings.Contains(got, s) {
				t.Errorf("%v: %q does not mention %q", c.args, got, s)
			}
		}
	}
	missing := text(t, e.call(t, "note_edit", map[string]any{"ref": "nothing", "expected_sha": sha, "op": "append", "text": "x"}))
	if !strings.HasPrefix(missing, "not_found: ") {
		t.Fatalf("unknown ref: %q", missing)
	}
	same := e.call(t, "note_edit", map[string]any{"ref": "r.md", "expected_sha": sha, "op": "find_replace", "find": "old step", "replace": "old step"})
	if got := text(t, same); same.IsError || got != "\"Runbook\" at r.md is unchanged.\n\nsha256: "+sha {
		t.Fatalf("unchanged: %q", got)
	}
}

func TestEdit_SPEC0007_R25_R27_annotationsAndLog(t *testing.T) {
	e := start(t, map[string]string{"r.md": editNote})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range res.Tools {
		if tool.Name == "note_edit" {
			a := tool.Annotations
			if a == nil || a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint || a.IdempotentHint || a.OpenWorldHint == nil || *a.OpenWorldHint {
				t.Fatalf("annotations %+v", a)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("note_edit not listed")
	}
	sha := e.readSha(t, "r.md")
	e.call(t, "note_edit", map[string]any{"ref": "r.md", "expected_sha": sha, "op": "find_replace", "find": "old step", "replace": "secret replacement"})
	lines := e.logLines(t, "tool call")
	l := lines[len(lines)-1]
	b, _ := os.ReadFile(filepath.Join(e.root, "r.md"))
	if l["tool"] != "note_edit" || l["bytes"] != float64(len(b)) || l["paths"] != 1.0 {
		t.Fatalf("log line %v", l)
	}
	if s := e.log.String(); strings.Contains(s, "secret") || strings.Contains(s, "old step") || strings.Contains(s, "Runbook") {
		t.Fatalf("log holds argument text: %s", s)
	}
}

func TestEdit_SPEC0007_R16_R17_text(t *testing.T) {
	d := toolDescription(t, "note_edit")
	if !strings.Contains(d, "new paragraph") {
		t.Fatalf("description:\n%s", d)
	}
	e := start(t, map[string]string{"a.md": "---\ntitle: A\n---\n## Steps\nx\n"})
	got := text(t, e.call(t, "note_read", map[string]any{"ref": "a"}))
	sha := got[strings.Index(got, "- sha256: ")+10:][:64]
	r := e.call(t, "note_edit", map[string]any{"ref": "a", "expected_sha": sha, "op": "replace_section", "section": "Missing", "text": "x"})
	if !r.IsError || !strings.HasPrefix(text(t, r), "section_not_found: ") || !strings.Contains(text(t, r), "Steps") || !strings.Contains(text(t, r), "append") {
		t.Fatalf("got %s", text(t, r))
	}
}
