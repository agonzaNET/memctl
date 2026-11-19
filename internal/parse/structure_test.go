package parse_test

import (
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

func headings(n parse.Note) []parse.Heading { return n.Headings }

func TestHeadings_SPEC0001_R11_atx(t *testing.T) {
	cases := []struct {
		line string
		want []parse.Heading
	}{
		{"# Title", []parse.Heading{{Level: 1, Text: "Title", Line: 1}}},
		{"## Title ##", []parse.Heading{{Level: 2, Text: "Title", Line: 1}}},
		{"###### six", []parse.Heading{{Level: 6, Text: "six", Line: 1}}},
		{"   ### indented three", []parse.Heading{{Level: 3, Text: "indented three", Line: 1}}},
		{"#", []parse.Heading{{Level: 1, Text: "", Line: 1}}},
		{"#\tTab", []parse.Heading{{Level: 1, Text: "Tab", Line: 1}}},
		{"## C# notes", []parse.Heading{{Level: 2, Text: "C# notes", Line: 1}}},
		{"## trailing #hash", []parse.Heading{{Level: 2, Text: "trailing #hash", Line: 1}}},
		{"####### seven", nil},
		{"#tag", nil},
		{"#123", nil},
		{"    # four spaces", nil},
		{"text # not heading", nil},
	}
	for _, c := range cases {
		t.Run(c.line, func(t *testing.T) {
			got := headings(parse.Parse("n.md", lines(c.line)))
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestHeadings_SPEC0001_R8_lineNumbersCountFrontmatter(t *testing.T) {
	n := parse.Parse("n.md", lines("---", "title: x", "---", "", "# One", "text", "## Two"))
	want := []parse.Heading{{1, "One", 5}, {2, "Two", 7}}
	if !reflect.DeepEqual(n.Headings, want) {
		t.Fatalf("got %+v", n.Headings)
	}
}

func TestFences_SPEC0001_R9_R10(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
		want []string // heading texts outside fences
	}{
		{"backticks", lines("# A", "```", "# in", "```", "# B"), []string{"A", "B"}},
		{"tildes", lines("~~~", "# in", "~~~", "# B"), []string{"B"}},
		{"info string", lines("```go", "# in", "```", "# B"), []string{"B"}},
		{"longer closing", lines("```", "# in", "`````", "# B"), []string{"B"}},
		{"shorter closing does not close", lines("````", "```", "# in", "````", "# B"), []string{"B"}},
		{"other char does not close", lines("```", "~~~", "# in", "```", "# B"), []string{"B"}},
		{"closing with text does not close", lines("```", "``` x", "# in", "```", "# B"), []string{"B"}},
		{"backtick in info is not a fence", lines("``` a`b", "# out"), []string{"out"}},
		{"indented three", lines("   ```", "# in", "   ```", "# B"), []string{"B"}},
		{"indented four is not a fence", lines("    ```", "# out"), []string{"out"}},
		{"two backticks", lines("``", "# out"), []string{"out"}},
		{"unclosed runs to end", lines("# A", "```", "# in", "# in2"), []string{"A"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got []string
			for _, h := range parse.Parse("n.md", c.src).Headings {
				got = append(got, h.Text)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

// sections returns, for each observation-like probe line, which section the
// parser put it in, using the exported section state of the parsed note.
func TestSections_SPEC0001_R12(t *testing.T) {
	n := parse.Parse("n.md", lines(
		"# Top",           // 1
		"## observations", // 2 opens Observations (case-insensitive)
		"### Detail",      // 3 does not end it
		"## Relations",    // 4 ends Observations, opens Relations
		"#### deep",       // 5
		"## Next",         // 6 ends Relations
		"## Observations", // 7
		"# Again",         // 8 ends it
	))
	want := []parse.Section{
		{Kind: parse.SectionObservations, Start: 2, End: 3},
		{Kind: parse.SectionRelations, Start: 4, End: 5},
		{Kind: parse.SectionObservations, Start: 7, End: 7},
	}
	if !reflect.DeepEqual(n.Sections, want) {
		t.Fatalf("got %+v, want %+v", n.Sections, want)
	}
	n = parse.Parse("n.md", lines("### Observations", "x"))
	if len(n.Sections) != 0 {
		t.Fatalf("level-3 heading opened a section: %+v", n.Sections)
	}
	n = parse.Parse("n.md", lines("```", "## Observations", "```"))
	if len(n.Sections) != 0 {
		t.Fatalf("fenced heading opened a section: %+v", n.Sections)
	}
}
