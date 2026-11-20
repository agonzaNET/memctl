package note_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
)

const fixedStamp = "2026-11-19T10:12:00+01:00"

// resolvePath resolves a ref that is a path, for the service tests.
func resolvePath(rel string) note.Resolver {
	return func(idx *index.Index) (string, error) {
		if _, _, ok := idx.Get(rel); !ok {
			return "", errors.New("test: no note " + rel)
		}
		return rel, nil
	}
}

func shaOf(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return sum(b)
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const fm = "---\ntitle: T\ntype: runbook\nupdated: 2026-01-01T00:00:00+01:00\n---\n"
const fmNew = "---\ntitle: T\ntype: runbook\nupdated: " + fixedStamp + "\n---\n"

func TestEdit_SPEC0007_R15_staleSha(t *testing.T) {
	root := tree(t, map[string]string{"a.md": fm + "body\n"})
	rec := &recCommitter{}
	s := openWith(t, root, fixedClock, note.WithCommitter(rec))
	stale := sum([]byte("an older version"))
	_, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: stale, Op: "append", Text: "x"})
	var c *note.ConflictError
	if !errors.As(err, &c) || !errors.Is(err, note.ErrConflict) {
		t.Fatalf("err %v, want ConflictError", err)
	}
	if c.SHA256 != shaOf(t, root, "a.md") || c.Content != fm+"body\n" {
		t.Fatalf("conflict %+v", c)
	}
	if readFile(t, root, "a.md") != fm+"body\n" || len(rec.changes()) != 0 {
		t.Fatal("a stale edit wrote or handed off")
	}
}

func TestEdit_SPEC0007_R16_append(t *testing.T) {
	cases := []struct{ in, text, want string }{
		{fm + "body", "more\n\n\n", fmNew + "body\n\nmore\n"},
		{fm + "body\n", "more", fmNew + "body\n\nmore\n"},
		{fm + "body\n\n\n\n", "more", fmNew + "body\n\nmore\n"},
		{fm + "body\n   \n", "more", fmNew + "body\n\nmore\n"},
		{fm + "body\n", "\n\n  \n  more", fmNew + "body\n\n  more\n"},
		{fm + "line with a break  \n", "more", fmNew + "line with a break  \n\nmore\n"},
		{fm, "first line", fmNew + "\nfirst line\n"},
	}
	for _, c := range cases {
		root := tree(t, map[string]string{"a.md": c.in})
		rec := &recCommitter{}
		s := openWith(t, root, fixedClock, note.WithCommitter(rec))
		e, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: shaOf(t, root, "a.md"), Op: "append", Text: c.text})
		if err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, root, "a.md"); got != c.want {
			t.Fatalf("append to %q:\n%q\nwant\n%q", c.in, got, c.want)
		}
		if e.Path != "a.md" || e.Title != "T" || e.SHA256 != sum([]byte(c.want)) || e.Bytes != len(c.want) || e.Unchanged {
			t.Fatalf("result %+v", e)
		}
		if got := rec.changes(); !reflect.DeepEqual(got, []note.Change{{Tool: "note_edit", Title: "T", Paths: []string{"a.md"}}}) {
			t.Fatalf("committer got %+v", got)
		}
		assertIndexFresh(t, s)
	}
}

func TestEdit_SPEC0007_R16_appendEmptyFile(t *testing.T) {
	root := tree(t, map[string]string{"a.md": ""})
	s := openWith(t, root, fixedClock)
	if _, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: shaOf(t, root, "a.md"), Op: "append", Text: "\nfirst\n"}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, root, "a.md"); got != "first\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEdit_SPEC0007_R17_replaceSection(t *testing.T) {
	body := "# A\nx\n## B\ny\n\n### B1\nz\n# C\nw\n"
	spaced := "# A\n\nx\n# C\nw\n"
	cases := []struct{ body, section, text, want string }{
		{body, "B", "new\n\n", "# A\nx\n## B\n\nnew\n\n# C\nw\n"},
		{body, "  b ", "new", "# A\nx\n## B\n\nnew\n\n# C\nw\n"},
		{body, "B", "\n\n  new\n\n", "# A\nx\n## B\n\n  new\n\n# C\nw\n"},
		{body, "A", "only", "# A\n\nonly\n\n# C\nw\n"},
		{body, "C", "last", "# A\nx\n## B\ny\n\n### B1\nz\n# C\n\nlast\n"},
		{body, "B1", "", "# A\nx\n## B\ny\n\n### B1\n\n# C\nw\n"},
		{body, "C", "", "# A\nx\n## B\ny\n\n### B1\nz\n# C\n\n"},
		{spaced, "A", "y", "# A\n\ny\n\n# C\nw\n"},
	}
	for _, c := range cases {
		body := c.body
		root := tree(t, map[string]string{"a.md": fm + body})
		s := openWith(t, root, fixedClock)
		if _, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: shaOf(t, root, "a.md"), Op: "replace_section", Section: c.section, Text: c.text}); err != nil {
			t.Fatalf("%s: %v", c.section, err)
		}
		if got := readFile(t, root, "a.md"); got != fmNew+c.want {
			t.Fatalf("section %q:\n%q\nwant\n%q", c.section, got, fmNew+c.want)
		}
		assertIndexFresh(t, s)
	}

	root := tree(t, map[string]string{"a.md": fm + body})
	s := openWith(t, root, fixedClock)
	_, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: shaOf(t, root, "a.md"), Op: "replace_section", Section: "Missing", Text: "x"})
	var se *note.SectionError
	if !errors.As(err, &se) || !reflect.DeepEqual(se.Headings, []string{"A", "B", "B1", "C"}) {
		t.Fatalf("missing section: %v", err)
	}
}

func TestEdit_SPEC0007_R18_findReplace(t *testing.T) {
	root := tree(t, map[string]string{"a.md": fm + "one two two\n"})
	s := openWith(t, root, fixedClock)
	sha := shaOf(t, root, "a.md")
	if _, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: "three", Replace: "3"}); !errors.Is(err, note.ErrNoMatch) {
		t.Fatalf("zero matches: %v", err)
	}
	_, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: "two", Replace: "2"})
	var nu *note.NotUniqueError
	if !errors.As(err, &nu) || nu.Count != 2 {
		t.Fatalf("two matches: %v", err)
	}
	if _, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: "one", Replace: ""}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, root, "a.md"); got != fmNew+" two two\n" {
		t.Fatalf("got %q", got)
	}
	// Inside the frontmatter.
	sha = shaOf(t, root, "a.md")
	if _, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: "type: runbook", Replace: "type: host"}); err != nil {
		t.Fatal(err)
	}
	n, _, _ := s.Index().Get("a.md")
	if n.Type != "host" {
		t.Fatalf("type %q", n.Type)
	}
	assertIndexFresh(t, s)
}

func TestEdit_SPEC0007_R19_unchanged(t *testing.T) {
	root := tree(t, map[string]string{"a.md": fm + "body\n"})
	rec := &recCommitter{}
	s := openWith(t, root, fixedClock, note.WithCommitter(rec))
	sha := shaOf(t, root, "a.md")
	e, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: "body", Replace: "body"})
	if err != nil || !e.Unchanged || e.SHA256 != sha || e.Title != "T" || e.Path != "a.md" {
		t.Fatalf("%+v %v", e, err)
	}
	if readFile(t, root, "a.md") != fm+"body\n" || len(rec.changes()) != 0 {
		t.Fatal("an unchanged edit wrote or handed off")
	}
}

func TestEdit_SPEC0007_R9_noUsableFrontmatter(t *testing.T) {
	for name, in := range map[string]string{
		"none": "plain body\n",
		"bad":  "---\ntitle: [x\nupdated: old\n---\nbody\n",
	} {
		root := tree(t, map[string]string{"a.md": in})
		s := openWith(t, root, fixedClock)
		if _, err := s.Edit(resolvePath("a.md"), note.EditOp{ExpectedSHA: shaOf(t, root, "a.md"), Op: "append", Text: "added"}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := readFile(t, root, "a.md"); got != in+"added\n" {
			t.Fatalf("%s: %q", name, got)
		}
	}
}

func TestEdit_SPEC0007_inputRules(t *testing.T) {
	root := tree(t, map[string]string{"a.md": fm})
	s := openWith(t, root)
	sha := shaOf(t, root, "a.md")
	big := string(make([]byte, 1<<20+1))
	for _, c := range []struct {
		op  note.EditOp
		arg string
	}{
		{note.EditOp{ExpectedSHA: "abc", Op: "append", Text: "x"}, "expected_sha"},
		{note.EditOp{ExpectedSHA: sha, Op: "rewrite", Text: "x"}, "op"},
		{note.EditOp{ExpectedSHA: sha, Op: "replace_section", Text: "x"}, "section"},
		{note.EditOp{ExpectedSHA: sha, Op: "find_replace", Replace: "x"}, "find"},
		{note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: string(make([]byte, 64<<10+1))}, "find"},
		{note.EditOp{ExpectedSHA: sha, Op: "append", Text: big}, "text"},
		{note.EditOp{ExpectedSHA: sha, Op: "find_replace", Find: "x", Replace: big}, "replace"},
	} {
		_, err := s.Edit(resolvePath("a.md"), c.op)
		var ae *note.ArgError
		if !errors.As(err, &ae) || ae.Arg != c.arg {
			t.Errorf("%+v: err %v, want ArgError for %s", c.op.Op, err, c.arg)
		}
	}
}
