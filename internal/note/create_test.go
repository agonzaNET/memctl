package note_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/parse"
)

var fixedClock = note.WithClock(func() time.Time {
	return time.Date(2026, 11, 19, 10, 12, 0, 0, time.FixedZone("CET", 3600))
})

func TestFileName_SPEC0007_R10(t *testing.T) {
	cases := map[string]string{
		"PAN-OS upgrade procedure":      "PAN-OS upgrade procedure.md",
		"a/b: c":                        "a-b- c.md",
		`x*?"<>|\`:                      "x-------.md",
		".hidden":                       "-hidden.md",
		"..x":                           "--x.md",
		"a.b":                           "a.b.md",
		"  a   b  ":                     "a b.md",
		"a\tb\x00c\x7fd":                "a-b-c-d.md",
		"Größe & Ümlaut":                "Größe & Ümlaut.md",
		strings.Repeat("é", 150):        strings.Repeat("é", 100) + ".md",
		strings.Repeat("a", 199) + "é":  strings.Repeat("a", 199) + ".md",
		strings.Repeat("a", 199) + " b": strings.Repeat("a", 199) + ".md",
	}
	for title, want := range cases {
		if got, err := note.FileName(title); err != nil || got != want {
			t.Errorf("FileName(%q) = %q, %v; want %q", title, got, err, want)
		}
	}
	if _, err := note.FileName("   "); err == nil {
		t.Error("blank title accepted")
	}
}

func TestCreate_SPEC0007_R13_file(t *testing.T) {
	root := t.TempDir()
	rec := &recCommitter{}
	s := openWith(t, root, fixedClock, note.WithCommitter(rec))
	w, err := s.Write(note.NewNote{
		Title:  "PAN-OS upgrade: HA",
		Folder: "runbooks",
		Type:   "runbook",
		Tags:   []string{"palo-alto", "firewall", "a,b", "true", "123", "#x", "plain tag"},
		Body:   "# H\n\ntext\n\n\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	const wantPath = "runbooks/PAN-OS upgrade- HA.md"
	want := `---
title: "PAN-OS upgrade: HA"
type: runbook
tags: [palo-alto, firewall, "a,b", "true", "123", "#x", plain tag]
created: 2026-11-19T10:12:00+01:00
updated: 2026-11-19T10:12:00+01:00
---

# H

text
`
	b, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(wantPath)))
	if string(b) != want {
		t.Fatalf("file:\n%s\nwant:\n%s", b, want)
	}
	if w.Path != wantPath || w.Title != "PAN-OS upgrade: HA" || w.SHA256 != sum(b) || w.Bytes != len(b) || len(w.SameTitle) != 0 {
		t.Fatalf("result %+v", w)
	}
	n := parse.Parse(wantPath, b)
	if n.Title != "PAN-OS upgrade: HA" || n.Type != "runbook" || !reflect.DeepEqual(n.Tags, []string{"palo-alto", "firewall", "a,b", "true", "123", "#x", "plain tag"}) || len(n.Problems) != 0 {
		t.Fatalf("parsed back %+v", n)
	}
	if got := rec.changes(); !reflect.DeepEqual(got, []note.Change{{Tool: "note_write", Title: "PAN-OS upgrade: HA", Paths: []string{wantPath}}}) {
		t.Fatalf("committer got %+v", got)
	}
	assertIndexFresh(t, s)
}

func TestCreate_SPEC0007_R13_minimal(t *testing.T) {
	root := t.TempDir()
	s := openWith(t, root, fixedClock)
	if _, err := s.Write(note.NewNote{Title: "Empty", Body: ""}); err != nil {
		t.Fatal(err)
	}
	want := "---\ntitle: Empty\ncreated: 2026-11-19T10:12:00+01:00\nupdated: 2026-11-19T10:12:00+01:00\n---\n"
	if b, _ := os.ReadFile(filepath.Join(root, "Empty.md")); string(b) != want {
		t.Fatalf("file %q, want %q", b, want)
	}
	if _, err := s.Write(note.NewNote{Title: "One line", Body: "no newline"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "One line.md")); !strings.HasSuffix(string(b), "---\n\nno newline\n") {
		t.Fatalf("file %q", b)
	}
}

func TestCreate_SPEC0007_R11_exists(t *testing.T) {
	root := tree(t, map[string]string{"notes/Alpha.md": "a\n", "Taken.md": "t\n"})
	rec := &recCommitter{}
	s := openWith(t, root, note.WithCommitter(rec))
	for _, c := range []struct{ title, folder, path string }{
		{"Taken", "", "Taken.md"},
		{"alpha", "notes", "notes/Alpha.md"},
		{"ALPHA", "notes/", "notes/Alpha.md"},
	} {
		_, err := s.Write(note.NewNote{Title: c.title, Folder: c.folder, Body: "x"})
		var ex *note.ExistsError
		if !errors.As(err, &ex) || ex.Path != c.path {
			t.Errorf("%s in %q: err %v, want ExistsError at %s", c.title, c.folder, err, c.path)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(root, "Taken.md")); string(b) != "t\n" {
		t.Fatalf("existing note changed: %q", b)
	}
	if len(rec.changes()) != 0 {
		t.Fatalf("refused write handed %+v", rec.changes())
	}
}

func TestCreate_SPEC0007_R12_frontmatterInBody(t *testing.T) {
	s := openWith(t, t.TempDir())
	for _, body := range []string{"---\ntitle: X\n---\ntext\n", "---", "---   \nx: 1\n---\n", "\ufeff---\nx\n---\n", "---\r\nx: 1\r\n---\r\n"} {
		if _, err := s.Write(note.NewNote{Title: "X", Body: body}); !errors.Is(err, note.ErrFrontmatterInBody) {
			t.Errorf("body %q: err %v, want ErrFrontmatterInBody", body, err)
		}
	}
	if _, err := s.Write(note.NewNote{Title: "Y", Body: "text\n---\nmore\n"}); err != nil {
		t.Errorf("a later --- line refused: %v", err)
	}
}

func TestCreate_SPEC0007_R14_sameTitle(t *testing.T) {
	root := tree(t, map[string]string{
		"hosts/intpxy.md":    "---\ntitle: intpxy\n---\n",
		"services/intpxy.md": "---\ntitle: intpxy\n---\n",
	})
	s := openWith(t, root)
	w, err := s.Write(note.NewNote{Title: "intpxy", Folder: "runbooks", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.SameTitle, []string{"hosts/intpxy.md", "services/intpxy.md"}) {
		t.Fatalf("same title %v", w.SameTitle)
	}
}

func TestCreate_SPEC0007_R28_folderNotANote(t *testing.T) {
	root := tree(t, map[string]string{".gitignore": "drafts/\n"})
	s := openWith(t, root)
	for _, folder := range []string{"drafts", ".hidden", "a/../b", "/abs"} {
		_, err := s.Write(note.NewNote{Title: "X", Folder: folder, Body: "x"})
		var pe *note.PathError
		if !errors.As(err, &pe) {
			t.Errorf("folder %q: err %v, want PathError", folder, err)
		}
	}
}

func TestCreate_SPEC0007_inputRules(t *testing.T) {
	s := openWith(t, t.TempDir())
	cases := []struct {
		name string
		n    note.NewNote
		arg  string
	}{
		{"title line break", note.NewNote{Title: "a\nb"}, "title"},
		{"title too long", note.NewNote{Title: strings.Repeat("x", 201)}, "title"},
		{"empty title", note.NewNote{Title: ""}, "title"},
		{"type line break", note.NewNote{Title: "t", Type: "a\rb"}, "type"},
		{"type too long", note.NewNote{Title: "t", Type: strings.Repeat("x", 65)}, "type"},
		{"too many tags", note.NewNote{Title: "t", Tags: make21()}, "tags"},
		{"empty tag", note.NewNote{Title: "t", Tags: []string{""}}, "tags"},
		{"tag line break", note.NewNote{Title: "t", Tags: []string{"a\nb"}}, "tags"},
		{"body too large", note.NewNote{Title: "t", Body: strings.Repeat("x", 1<<20+1)}, "body"},
	}
	for _, c := range cases {
		_, err := s.Write(c.n)
		var ae *note.ArgError
		if !errors.As(err, &ae) || ae.Arg != c.arg {
			t.Errorf("%s: err %v, want ArgError for %s", c.name, err, c.arg)
		}
	}
	if _, err := s.Write(note.NewNote{Title: strings.Repeat("x", 200), Body: strings.Repeat("y", 1<<20)}); err != nil {
		t.Errorf("limits refused: %v", err)
	}
}

func make21() []string {
	out := make([]string, 21)
	for i := range out {
		out[i] = string(rune('a' + i))
	}
	return out
}
