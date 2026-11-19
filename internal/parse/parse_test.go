package parse_test

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

func TestParse_SPEC0001_R8_bomAndCRLF(t *testing.T) {
	lf := "---\ntitle: A\n---\nbody line\n"
	crlf := "\xef\xbb\xbf---\r\ntitle: A\r\n---\r\nbody line\r\n"
	a := parse.Parse("n.md", []byte(lf))
	b := parse.Parse("n.md", []byte(crlf))
	if a.Title != "A" || b.Title != "A" || a.BodyLine != 4 || b.BodyLine != 4 {
		t.Fatalf("lf %q/%d crlf %q/%d", a.Title, a.BodyLine, b.Title, b.BodyLine)
	}
	if b.Body != "body line\r\n" {
		t.Fatalf("crlf body %q", b.Body)
	}
	mustNoProblems(t, b)
}

func TestParse_SPEC0001_R21_invalidUTF8(t *testing.T) {
	data := []byte("ok\nbad \xff here\nalso \xfe\n")
	n := parse.Parse("n.md", data)
	if !equalStrings(codes(n), []string{"invalid_utf8@2"}) {
		t.Fatalf("problems %v", codes(n))
	}
	if n.Title != "n" {
		t.Fatalf("title %q", n.Title)
	}
}

func TestParse_SPEC0001_R22_neverFails(t *testing.T) {
	inputs := [][]byte{
		nil,
		{},
		[]byte("---"),
		[]byte("---\n---"),
		[]byte("---\ntitle: x\n---"),
		[]byte("\xef\xbb\xbf"),
		[]byte("\r\n\r\n"),
		[]byte{0, 1, 2, 0xff, 0xfe, '\n', '-', '-', '-'},
	}
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		b := make([]byte, r.Intn(300))
		r.Read(b)
		inputs = append(inputs, b)
	}
	for _, in := range inputs {
		n := parse.Parse("x.md", in)
		if n.Title != "x" && !n.TitleFromFrontmatter {
			t.Fatalf("title %q for %q", n.Title, in)
		}
	}
}

func TestParse_SPEC0001_R24_deterministic(t *testing.T) {
	data := lines("---", "title: A", "tags: {x: 1}", "created: soon", "---", "x \xff", "body")
	a := parse.Parse("n.md", data)
	b := parse.Parse("n.md", data)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("not deterministic:\n%+v\n%+v", a, b)
	}
	want := []string{"bad_field@3", "bad_field@4", "invalid_utf8@6"}
	if !equalStrings(codes(a), want) {
		t.Fatalf("problems %v, want %v (sorted by line, then code)", codes(a), want)
	}
}
