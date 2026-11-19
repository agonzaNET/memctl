package parse_test

import (
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

func TestProblems_SPEC0001_R19_outsideSection(t *testing.T) {
	n := parse.Parse("n.md", lines(
		"intro",                    // 1
		"- [risk] in the body",     // 2 observation outside
		"- implements [[ODR-1]]",   // 3 relation outside
		"## Observations",          // 4
		"- applies_to [[PA-850]]",  // 5 relation under Observations
		"- [note] fine",            // 6
		"## Relations",             // 7
		"- [risk] under relations", // 8 observation under Relations
		"- runs_on [[host]]",       // 9 fine
	))
	want := []string{
		"observation_outside_section@2",
		"relation_outside_section@3",
		"relation_outside_section@5",
		"observation_outside_section@8",
	}
	if !equalStrings(codes(n), want) {
		t.Fatalf("problems %v, want %v", codes(n), want)
	}
	if len(n.Observations) != 1 || len(n.Relations) != 1 {
		t.Fatalf("misplaced lines parsed: %+v %+v", n.Observations, n.Relations)
	}
	var targets []string
	for _, l := range n.Links {
		targets = append(targets, l.Target)
	}
	if !reflect.DeepEqual(targets, []string{"ODR-1", "PA-850"}) {
		t.Fatalf("links %v", targets)
	}
}

func TestProblems_SPEC0001_R20_unsupportedForms(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
		want []string
	}{
		{"setext h1", lines("Title", "====="), []string{"setext_heading@2"}},
		{"setext h2", lines("Title", "--"), []string{"setext_heading@2"}},
		{"thematic break after blank", lines("text", "", "---"), nil},
		{"after heading", lines("# H", "---"), nil},
		{"single dash", lines("text", "-"), nil},
		{"html comment", lines("<!-- note", "still", "-->", "", "<div>", "x", "</div>"), []string{"html_block@1", "html_block@5"}},
		{"not html", lines("a < b", "<3 love", "</closing>"), nil},
		{"indented code", lines("text", "", "    code", "    more", "", "    again"), []string{"indented_code@3"}},
		{"indented after text is not code", lines("text", "    continuation"), nil},
		{"tab", lines("", "\tcode"), []string{"indented_code@2"}},
		{"list continuation", lines("- item", "", "    continued", "1. num", "", "    also"), nil},
		{"after list ends", lines("- item", "text", "", "    code"), []string{"indented_code@4"}},
		{"inside fence", lines("```", "Title", "===", "<div>", "", "    code", "```"), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := codes(parse.Parse("n.md", c.src)); !equalStrings(got, c.want) {
				t.Fatalf("problems %v, want %v", got, c.want)
			}
		})
	}
}
