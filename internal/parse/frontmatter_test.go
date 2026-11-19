package parse_test

import (
	"testing"
	"time"

	"github.com/agonzaNET/memctl/internal/parse"
)

func TestFrontmatter_SPEC0001_R1_delimiters(t *testing.T) {
	cases := []struct {
		name     string
		data     []byte
		has      bool
		title    string
		bodyLine int
	}{
		{"dashes", lines("---", "title: A", "---", "body"), true, "A", 4},
		{"dots close", lines("---", "title: A", "...", "body"), true, "A", 4},
		{"trailing spaces", lines("---  ", "title: A", "--- \t", "body"), true, "A", 4},
		{"not first line", lines("", "---", "title: A", "---"), false, "n", 1},
		{"four dashes", lines("----", "title: A", "---"), false, "n", 1},
		{"indented", lines(" ---", "title: A", "---"), false, "n", 1},
		{"empty block", lines("---", "---", "body"), true, "n", 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := parse.Parse("n.md", c.data)
			if n.HasFrontmatter != c.has || n.Title != c.title || n.BodyLine != c.bodyLine {
				t.Fatalf("got has=%v title=%q bodyLine=%d, want %v %q %d",
					n.HasFrontmatter, n.Title, n.BodyLine, c.has, c.title, c.bodyLine)
			}
		})
	}
}

func TestFrontmatter_SPEC0001_R2_knownAndUnknownKeys(t *testing.T) {
	n := parse.Parse("ops/x.md", lines(
		"---",
		"title: PAN-OS upgrade procedure",
		"type: runbook",
		"# a comment",
		"source: manual",
		"tags: [palo-alto, firewall]",
		"aliases:",
		"  - PAN-OS upgrade",
		"created: 2026-11-19T10:12:00+01:00",
		"updated: 2026-11-19T10:12:00+01:00",
		"---",
		"body",
	))
	mustNoProblems(t, n)
	if n.Title != "PAN-OS upgrade procedure" || !n.TitleFromFrontmatter || n.Type != "runbook" {
		t.Fatalf("title/type: %q %v %q", n.Title, n.TitleFromFrontmatter, n.Type)
	}
	if !equalStrings(n.Tags, []string{"palo-alto", "firewall"}) || !equalStrings(n.Aliases, []string{"PAN-OS upgrade"}) {
		t.Fatalf("tags %v aliases %v", n.Tags, n.Aliases)
	}
	var keys []string
	for _, f := range n.Frontmatter {
		keys = append(keys, f.Key)
	}
	if !equalStrings(keys, []string{"title", "type", "source", "tags", "aliases", "created", "updated"}) {
		t.Fatalf("keys %v", keys)
	}
	want := map[string]parse.Field{
		"source":  {Key: "source", Raw: "source: manual", Line: 5},
		"aliases": {Key: "aliases", Raw: "aliases:\n  - PAN-OS upgrade", Line: 7},
		"type":    {Key: "type", Raw: "type: runbook\n# a comment", Line: 3},
	}
	for _, f := range n.Frontmatter {
		if w, ok := want[f.Key]; ok && f != w {
			t.Errorf("field %s = %+v, want %+v", f.Key, f, w)
		}
	}
}

func TestFrontmatter_SPEC0001_R3_tagsAndAliasesShapes(t *testing.T) {
	cases := []struct {
		name string
		line string
		tags []string
		bad  bool
	}{
		{"flow list", "tags: [a, b]", []string{"a", "b"}, false},
		{"single string", "tags: firewall", []string{"firewall"}, false},
		{"trim drop empty dedupe", "tags: [' a ', '', a, b]", []string{"a", "b"}, false},
		{"null", "tags:", nil, false},
		{"mapping", "tags: {a: 1}", nil, true},
		{"nested list", "tags: [[a]]", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := parse.Parse("n.md", lines("---", c.line, "---"))
			if !equalStrings(n.Tags, c.tags) {
				t.Fatalf("tags %q, want %q", n.Tags, c.tags)
			}
			if hasCode(n, parse.CodeBadField, 2) != c.bad {
				t.Fatalf("problems %v, want bad_field=%v", codes(n), c.bad)
			}
		})
	}
	n := parse.Parse("n.md", lines("---", "aliases: old name", "---"))
	if !equalStrings(n.Aliases, []string{"old name"}) {
		t.Fatalf("aliases %q", n.Aliases)
	}
}

func TestFrontmatter_SPEC0001_R4_timestamps(t *testing.T) {
	cases := []struct {
		name string
		line string
		want time.Time
		bad  bool
	}{
		{"rfc3339", "created: 2026-11-19T10:12:00+01:00", time.Date(2026, 11, 19, 10, 12, 0, 0, time.FixedZone("", 3600)), false},
		{"quoted rfc3339", "created: \"2026-11-19T09:12:00Z\"", time.Date(2026, 11, 19, 9, 12, 0, 0, time.UTC), false},
		{"date", "created: 2026-11-19", time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC), false},
		{"word", "created: soon", time.Time{}, true},
		{"list", "created: [2026-11-19]", time.Time{}, true},
		{"null", "created:", time.Time{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := parse.Parse("n.md", lines("---", c.line, "---"))
			if !n.Created.Equal(c.want) {
				t.Fatalf("created %v, want %v", n.Created, c.want)
			}
			if hasCode(n, parse.CodeBadField, 2) != c.bad {
				t.Fatalf("problems %v, want bad_field=%v", codes(n), c.bad)
			}
		})
	}
	n := parse.Parse("n.md", lines("---", "updated: 2026-11-20", "---"))
	if !n.Updated.Equal(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("updated %v", n.Updated)
	}
}

func TestFrontmatter_SPEC0001_R5_invalidYAML(t *testing.T) {
	cases := map[string][]byte{
		"broken":   lines("---", "title: [unclosed", "---", "# Body", "text"),
		"scalar":   lines("---", "just text", "---", "# Body", "text"),
		"sequence": lines("---", "- a", "- b", "---", "# Body", "text"),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			n := parse.Parse("dir/stem.md", data)
			if !hasCode(n, parse.CodeBadFrontmatter, 1) {
				t.Fatalf("problems %v, want bad_frontmatter@1", codes(n))
			}
			if n.Title != "stem" || n.TitleFromFrontmatter || len(n.Frontmatter) != 0 {
				t.Fatalf("known fields not absent: %q %v %v", n.Title, n.TitleFromFrontmatter, n.Frontmatter)
			}
			if !n.HasFrontmatter || n.Body != "# Body\ntext\n" {
				t.Fatalf("body %q has=%v", n.Body, n.HasFrontmatter)
			}
		})
	}
}

func TestFrontmatter_SPEC0001_R6_unclosed(t *testing.T) {
	data := lines("---", "title: A", "body")
	n := parse.Parse("u.md", data)
	if !hasCode(n, parse.CodeBadFrontmatter, 1) {
		t.Fatalf("problems %v", codes(n))
	}
	if n.HasFrontmatter || n.Title != "u" || n.Body != string(data) || n.BodyLine != 1 {
		t.Fatalf("got has=%v title=%q body=%q line=%d", n.HasFrontmatter, n.Title, n.Body, n.BodyLine)
	}
}

func TestFrontmatter_SPEC0001_R7_titleFallback(t *testing.T) {
	cases := []struct {
		name, rel string
		data      []byte
		title     string
		fromFM    bool
	}{
		{"no frontmatter", "hosts/PA-850.md", lines("# Main firewall"), "PA-850", false},
		{"no title key", "a/b/Note name.md", lines("---", "type: x", "---", "# Other"), "Note name", false},
		{"empty title", "x.md", lines("---", "title: '  '", "---"), "x", false},
		{"title trimmed", "x.md", lines("---", "title: '  T  '", "---"), "T", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := parse.Parse(c.rel, c.data)
			if n.Title != c.title || n.TitleFromFrontmatter != c.fromFM {
				t.Fatalf("title %q fromFM=%v, want %q %v", n.Title, n.TitleFromFrontmatter, c.title, c.fromFM)
			}
		})
	}
}
