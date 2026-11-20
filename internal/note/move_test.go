package note_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

const hostA = "---\ntitle: A\naliases: [x]\ntype: host\n---\nbody of A\n"

func moveRoot(t *testing.T) (string, map[string]string) {
	t.Helper()
	files := map[string]string{
		"hosts/a.md":   hostA,
		"by-path.md":   "---\ntitle: P\n---\nsee [[hosts/a]]\n",
		"by-title.md":  "---\ntitle: Q\n---\n## Relations\n- runs_on [[A]]\n",
		"unrelated.md": "---\ntitle: U\n---\nnothing\n",
		"servers/b.md": "---\ntitle: B\n---\n",
		".gitignore":   "drafts/\n",
	}
	return tree(t, files), files
}

func TestMove_SPEC0007_R20_R22_R23_byPath(t *testing.T) {
	root, files := moveRoot(t)
	rec := &recCommitter{}
	s := openWith(t, root, fixedClock, note.WithCommitter(rec))
	mv, err := s.Move(resolvePath("hosts/a.md"), note.MoveOp{NewPath: "servers/a2.md"})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\ntitle: A\naliases: [x, hosts/a]\ntype: host\nupdated: " + fixedStamp + "\n---\nbody of A\n"
	if got := readFile(t, root, "servers/a2.md"); got != want {
		t.Fatalf("moved file:\n%q\nwant\n%q", got, want)
	}
	if mv.OldPath != "hosts/a.md" || mv.Path != "servers/a2.md" || mv.Title != "A" || len(mv.Linkers) != 2 || mv.SHA256 != sum([]byte(want)) || mv.Bytes != len(want) || mv.Same {
		t.Fatalf("result %+v", mv)
	}
	if got := rec.changes(); !reflect.DeepEqual(got, []note.Change{{Tool: "note_move", Title: "A", Paths: []string{"hosts/a.md", "servers/a2.md"}}}) {
		t.Fatalf("committer got %+v", got)
	}
	// R23: no other note changes; links to the old path still resolve.
	for p, c := range files {
		if p != "hosts/a.md" && readFile(t, root, p) != c {
			t.Fatalf("%s changed", p)
		}
	}
	if r := s.Index().Resolve("hosts/a"); r.Status != index.Resolved || r.Path != "servers/a2.md" || !r.Stale {
		t.Fatalf("old path resolves to %+v", r)
	}
	assertIndexFresh(t, s)
}

func TestMove_SPEC0007_R20_R22_byTitleAndBoth(t *testing.T) {
	root, _ := moveRoot(t)
	s := openWith(t, root, fixedClock)
	mv, err := s.Move(resolvePath("hosts/a.md"), note.MoveOp{NewTitle: "Alpha: one"})
	if err != nil {
		t.Fatal(err)
	}
	want := "---\ntitle: \"Alpha: one\"\naliases: [x, hosts/a, A]\ntype: host\nupdated: " + fixedStamp + "\n---\nbody of A\n"
	if mv.Path != "hosts/Alpha- one.md" || readFile(t, root, "hosts/Alpha- one.md") != want {
		t.Fatalf("by title: %+v\n%q", mv, readFile(t, root, mv.Path))
	}
	mv, err = s.Move(resolvePath("hosts/Alpha- one.md"), note.MoveOp{NewPath: "x/y.md", NewTitle: "Why"})
	if err != nil {
		t.Fatal(err)
	}
	n, _, _ := s.Index().Get("x/y.md")
	if mv.Path != "x/y.md" || mv.Title != "Why" || n.Title != "Why" || !reflect.DeepEqual(n.Aliases, []string{"x", "hosts/a", "A", "hosts/Alpha- one", "Alpha: one"}) {
		t.Fatalf("both: %+v %+v", mv, n)
	}
	// Back to an old path: no alias is added twice.
	if _, err := s.Move(resolvePath("x/y.md"), note.MoveOp{NewPath: "hosts/a.md", NewTitle: "A"}); err != nil {
		t.Fatal(err)
	}
	n, _, _ = s.Index().Get("hosts/a.md")
	if !reflect.DeepEqual(n.Aliases, []string{"x", "hosts/a", "A", "hosts/Alpha- one", "Alpha: one", "x/y", "Why"}) {
		t.Fatalf("aliases after three moves %v", n.Aliases)
	}
	assertIndexFresh(t, s)
}

func TestMove_SPEC0007_R20_titleFromFileNameKept(t *testing.T) {
	root := tree(t, map[string]string{"old name.md": "---\ntype: t\n---\nb\n"})
	s := openWith(t, root, fixedClock)
	if _, err := s.Move(resolvePath("old name.md"), note.MoveOp{NewPath: "new name.md"}); err != nil {
		t.Fatal(err)
	}
	n, _, _ := s.Index().Get("new name.md")
	if n.Title != "old name" {
		t.Fatalf("title %q, want the old file stem kept", n.Title)
	}
}

func TestMove_SPEC0007_R21_sameAndExisting(t *testing.T) {
	root, _ := moveRoot(t)
	rec := &recCommitter{}
	s := openWith(t, root, fixedClock, note.WithCommitter(rec))
	for _, op := range []note.MoveOp{{NewPath: "hosts/a.md"}, {NewTitle: "A", NewPath: "hosts/a.md"}} {
		mv, err := s.Move(resolvePath("hosts/a.md"), op)
		if err != nil || !mv.Same || mv.Path != "hosts/a.md" || mv.Title != "A" {
			t.Fatalf("%+v: %+v %v", op, mv, err)
		}
	}
	if readFile(t, root, "hosts/a.md") != hostA || len(rec.changes()) != 0 {
		t.Fatal("a move onto itself wrote")
	}
	for _, op := range []note.MoveOp{{NewPath: "servers/b.md"}, {NewPath: "SERVERS/B.md"}, {NewTitle: "b", NewPath: "servers/b.md"}} {
		_, err := s.Move(resolvePath("hosts/a.md"), op)
		var ex *note.ExistsError
		if !errors.As(err, &ex) || ex.Path != "servers/b.md" {
			t.Fatalf("%+v: err %v, want exists at servers/b.md", op, err)
		}
	}
	if readFile(t, root, "hosts/a.md") != hostA {
		t.Fatal("refused move changed the note")
	}
}

func TestMove_SPEC0007_R20_R28_R9_refused(t *testing.T) {
	root := tree(t, map[string]string{
		"a.md":       hostA,
		"plain.md":   "no frontmatter\n",
		"bad.md":     "---\ntitle: [x\n---\nb\n",
		".gitignore": "drafts/\n",
	})
	s := openWith(t, root, fixedClock)
	cases := []struct {
		ref string
		op  note.MoveOp
		arg string // ArgError argument, or "" for a PathError
	}{
		{"a.md", note.MoveOp{}, "new_path"},
		{"a.md", note.MoveOp{NewTitle: "a\nb"}, "new_title"},
		{"a.md", note.MoveOp{NewPath: "drafts/a.md"}, ""},
		{"a.md", note.MoveOp{NewPath: ".hidden/a.md"}, ""},
		{"a.md", note.MoveOp{NewPath: "a.txt"}, ""},
		{"plain.md", note.MoveOp{NewTitle: "P"}, "new_title"},
		{"plain.md", note.MoveOp{NewPath: "p2.md"}, "new_path"},
		{"bad.md", note.MoveOp{NewTitle: "B"}, "new_title"},
	}
	for _, c := range cases {
		_, err := s.Move(resolvePath(c.ref), c.op)
		var ae *note.ArgError
		var pe *note.PathError
		switch {
		case c.arg == "" && !errors.As(err, &pe):
			t.Errorf("%s %+v: err %v, want PathError", c.ref, c.op, err)
		case c.arg != "" && (!errors.As(err, &ae) || ae.Arg != c.arg):
			t.Errorf("%s %+v: err %v, want ArgError for %s", c.ref, c.op, err, c.arg)
		}
	}
	assertIndexFresh(t, s)
}
