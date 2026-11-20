package mcpserver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite_SPEC0007_R13_result(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	res := e.call(t, "note_write", map[string]any{
		"title": "PAN-OS upgrade", "folder": "runbooks", "body": "Steps.\n",
		"type": "runbook", "tags": []any{"palo-alto"},
	})
	got := text(t, res)
	b, err := os.ReadFile(filepath.Join(e.root, "runbooks", "PAN-OS upgrade.md"))
	if res.IsError || err != nil {
		t.Fatalf("result %q, file %v", got, err)
	}
	want := "Created \"PAN-OS upgrade\" at runbooks/PAN-OS upgrade.md.\n\nsha256: " + sum(string(b))
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	// The new note is visible to the next call.
	read := text(t, e.call(t, "note_read", map[string]any{"ref": "runbooks/PAN-OS upgrade"}))
	if !strings.HasPrefix(read, `Read "PAN-OS upgrade" at runbooks/PAN-OS upgrade.md.`) {
		t.Fatalf("read after write: %s", read)
	}
}

func TestWrite_SPEC0007_R14_sameTitleNotice(t *testing.T) {
	e := start(t, map[string]string{
		"hosts/intpxy.md":    "---\ntitle: intpxy\n---\n",
		"services/intpxy.md": "---\ntitle: intpxy\n---\n",
	})
	got := text(t, e.call(t, "note_write", map[string]any{"title": "intpxy", "folder": "runbooks", "body": ""}))
	for _, s := range []string{"\nsha256: ", "hosts/intpxy.md", "services/intpxy.md", "[[runbooks/intpxy]]"} {
		if !strings.Contains(got, s) {
			t.Fatalf("result lacks %q:\n%s", s, got)
		}
	}
	if strings.Index(got, "hosts/intpxy.md") < strings.Index(got, "sha256: ") {
		t.Fatalf("notice before the sha256 line:\n%s", got)
	}
}

func TestWrite_SPEC0007_R11_R12_R28_errors(t *testing.T) {
	e := start(t, map[string]string{"Taken.md": "t\n", ".gitignore": "drafts/\n"})
	cases := []struct {
		name string
		args map[string]any
		code string
		says string
	}{
		{"exists", map[string]any{"title": "taken", "body": "x"}, "exists", "Taken.md"},
		{"frontmatter body", map[string]any{"title": "X", "body": "---\ntitle: X\n---\n"}, "invalid_argument", "title"},
		{"ignored folder", map[string]any{"title": "X", "folder": "drafts", "body": "x"}, "invalid_argument", ".gitignore"},
		{"hidden folder", map[string]any{"title": "X", "folder": ".x", "body": "x"}, "invalid_argument", `"."`},
		{"bad folder", map[string]any{"title": "X", "folder": "../up", "body": "x"}, "invalid_argument", "folder"},
		{"title line break", map[string]any{"title": "a\nb", "body": "x"}, "invalid_argument", "title"},
	}
	for _, c := range cases {
		res := e.call(t, "note_write", c.args)
		got := text(t, res)
		if !res.IsError || !strings.HasPrefix(got, c.code+": ") || !strings.Contains(got, c.says) {
			t.Errorf("%s: %q, want %s mentioning %q", c.name, got, c.code, c.says)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(e.root, "Taken.md")); string(b) != "t\n" {
		t.Fatalf("existing note changed: %q", b)
	}
}

func TestWrite_SPEC0007_internalErrorLeavesNoteUnchanged(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	e := start(t, map[string]string{"locked/a.md": "a\n"})
	os.Chmod(filepath.Join(e.root, "locked"), 0o555)
	t.Cleanup(func() { os.Chmod(filepath.Join(e.root, "locked"), 0o755) })
	res := e.call(t, "note_write", map[string]any{"title": "B", "folder": "locked", "body": "x"})
	got := text(t, res)
	if !res.IsError || !strings.HasPrefix(got, "internal_error: ") || !strings.Contains(got, "unchanged") {
		t.Fatalf("got %q", got)
	}
}

func TestWrite_SPEC0007_R25_annotations(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	res, err := e.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		if tool.Name != "note_write" {
			continue
		}
		a := tool.Annotations
		if a == nil || a.ReadOnlyHint || a.DestructiveHint == nil || *a.DestructiveHint || a.IdempotentHint || a.OpenWorldHint == nil || *a.OpenWorldHint {
			t.Fatalf("note_write annotations %+v", a)
		}
		return
	}
	t.Fatal("note_write not listed")
}

func TestWrite_SPEC0007_R27_logCounts(t *testing.T) {
	e := start(t, map[string]string{"a.md": "a"})
	e.call(t, "note_write", map[string]any{"title": "Secret title", "folder": "hidden-place", "body": "secret body", "tags": []any{"secret-tag"}})
	e.call(t, "note_write", map[string]any{"title": "Secret title", "folder": "hidden-place", "body": "secret body"})
	lines := e.logLines(t, "tool call")
	if len(lines) != 2 {
		t.Fatalf("lines %v", lines)
	}
	b, _ := os.ReadFile(filepath.Join(e.root, "hidden-place", "Secret title.md"))
	ok, bad := lines[0], lines[1]
	if ok["tool"] != "note_write" || ok["bytes"] != float64(len(b)) || ok["paths"] != 1.0 || ok["error"] != nil {
		t.Fatalf("success line %v", ok)
	}
	if bad["error"] != "exists" || bad["paths"] != 0.0 || bad["bytes"] != 0.0 {
		t.Fatalf("error line %v", bad)
	}
	if l := e.log.String(); strings.Contains(l, "ecret") || strings.Contains(l, "hidden-place") {
		t.Fatalf("log holds argument text: %s", l)
	}
}

func TestWriteEdit_SPEC0007_R29_unresolvedLinks(t *testing.T) {
	e := start(t, map[string]string{
		"x/dup.md": "---\ntitle: X\n---\n",
		"y/dup.md": "---\ntitle: Y\n---\n",
		"ok.md":    "---\ntitle: OK\n---\n",
	})
	got := text(t, e.call(t, "note_write", map[string]any{"title": "New", "body": "See [[nowhere]] and [[ok]].\n\n## Relations\n- depends_on [[dup]]\n"}))
	want := "\n\nUnresolved links:\n- links_to → nowhere: dangling\n- depends_on → dup: ambiguous (x/dup.md, y/dup.md)"
	if !strings.Contains(got, want) || strings.Contains(got, "→ ok") {
		t.Fatalf("write:\n%s", got)
	}
	l := e.logLines(t, "tool call")
	if l[len(l)-1]["unresolved"] != 2.0 {
		t.Fatalf("log %v", l[len(l)-1])
	}

	got = text(t, e.call(t, "note_write", map[string]any{"title": "Fine", "body": "See [[ok]].\n"}))
	if strings.Contains(got, "Unresolved") {
		t.Fatalf("resolved links listed:\n%s", got)
	}
	sha := e.readSha(t, "Fine.md")
	got = text(t, e.call(t, "note_edit", map[string]any{"ref": "Fine.md", "expected_sha": sha, "op": "append", "text": "Also [[ghost]]."}))
	if !strings.Contains(got, "\n\nUnresolved links:\n- links_to → ghost: dangling") {
		t.Fatalf("edit:\n%s", got)
	}
	l = e.logLines(t, "tool call")
	if l[len(l)-1]["unresolved"] != 1.0 {
		t.Fatalf("log %v", l[len(l)-1])
	}
}
