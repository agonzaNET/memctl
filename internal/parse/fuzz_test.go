package parse_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

func seeds(t testing.TB) [][]byte {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join("testdata", "*.md"))
	var out [][]byte
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, b)
	}
	return out
}

// checkNote asserts the properties every parse must have: deterministic, and
// every reported line inside the file (SPEC-0001 R22, R24).
func checkNote(t testing.TB, data []byte) {
	n := parse.Parse("f/x.md", data)
	if again := parse.Parse("f/x.md", data); !reflect.DeepEqual(n, again) {
		t.Fatalf("not deterministic for %q", data)
	}
	lines := bytes.Count(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")), []byte("\n")) + 1
	inFile := func(l int) bool { return l >= 1 && l <= lines }
	for _, h := range n.Headings {
		if !inFile(h.Line) {
			t.Fatalf("heading line %d outside 1..%d", h.Line, lines)
		}
	}
	for _, l := range n.Links {
		if !inFile(l.Line) {
			t.Fatalf("link line %d outside 1..%d", l.Line, lines)
		}
	}
	for _, p := range n.Problems {
		if !inFile(p.Line) {
			t.Fatalf("problem %s line %d outside 1..%d", p.Code, p.Line, lines)
		}
	}
	if n.Title == "" {
		t.Fatalf("empty title")
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range seeds(f) {
		f.Add(s)
	}
	f.Add([]byte("---\n"))
	f.Add([]byte("[[a|b]] `[[c]]` ``` ~~~"))
	f.Fuzz(func(t *testing.T, data []byte) { checkNote(t, data) })
}

func TestParse_SPEC0001_R24_testdataDeterministic(t *testing.T) {
	for _, s := range seeds(t) {
		checkNote(t, s)
	}
}

func TestParse_SPEC0001_R22_messyNote(t *testing.T) {
	b, _ := os.ReadFile(filepath.Join("testdata", "messy.md"))
	n := parse.Parse("messy.md", b)
	want := []string{"bad_link@7", "setext_heading@10", "html_block@12", "indented_code@14"}
	if !equalStrings(codes(n), want) {
		t.Fatalf("problems %v, want %v", codes(n), want)
	}
	if !reflect.DeepEqual(n.Links, []parse.Link{{"real", "label", 7}}) || n.Tags[0] != "firewall" || len(n.Frontmatter) != 3 {
		t.Fatalf("note %+v", n)
	}
}
