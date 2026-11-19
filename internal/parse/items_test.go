package parse_test

import (
	"reflect"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

func TestLinks_SPEC0001_R17_scan(t *testing.T) {
	cases := []struct {
		name string
		line string
		want []parse.Link
		bad  bool
	}{
		{"one", "see [[Target]] here", []parse.Link{{"Target", "", 1}}, false},
		{"label", "see [[ network/intpxy | the proxy ]]", []parse.Link{{"network/intpxy", "the proxy", 1}}, false},
		{"several", "[[a]] and [[b|B]] and [[ops/c.md]]", []parse.Link{{"a", "", 1}, {"b", "B", 1}, {"ops/c.md", "", 1}}, false},
		{"empty target", "x [[]] y", nil, true},
		{"blank target with label", "x [[  |label]] y", nil, true},
		{"bracket inside", "x [[a[b]] y", nil, false},
		{"unclosed", "x [[a y", nil, false},
		{"single brackets", "x [a] y", nil, false},
		{"triple brackets: the inner pair is the link", "[[[a]]]", []parse.Link{{"a", "", 1}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := parse.Parse("n.md", lines(c.line))
			if !reflect.DeepEqual(n.Links, c.want) {
				t.Fatalf("links %+v, want %+v", n.Links, c.want)
			}
			if hasCode(n, parse.CodeBadLink, 1) != c.bad {
				t.Fatalf("problems %v, want bad_link=%v", codes(n), c.bad)
			}
		})
	}
}

func TestLinks_SPEC0001_R18_frontmatterAndCodeSpans(t *testing.T) {
	n := parse.Parse("n.md", lines(
		"---",
		"title: '[[not a link]]'",
		"---",
		"code `[[x]]` and ``[[y]] with ` inside`` then [[real]]", // 4
		"unclosed ` then [[after]]",                              // 5
		"```",
		"[[fenced]]",
		"```",
		"# Heading [[in heading]]", // 9
	))
	want := []parse.Link{{"real", "", 4}, {"after", "", 5}, {"in heading", "", 9}}
	if !reflect.DeepEqual(n.Links, want) {
		t.Fatalf("links %+v, want %+v", n.Links, want)
	}
	mustNoProblems(t, n)
}

func TestObservations_SPEC0001_R13_R14_R15(t *testing.T) {
	src := lines(
		"## Observations",                     // 1
		"- [prereq] Download content updates", // 2 valid
		"- [risk]   HA pair first  ",          // 3 valid, trimmed
		"- [ ] task",                          // 4 task item
		"- [x] done",                          // 5 task item
		"- [X] done",                          // 6 task item
		"- [a] one-letter category",           // 7
		"- [Risk] upper case",                 // 8
		"  - [risk] indented",                 // 9
		"* [risk] star bullet",                // 10
		"+ [risk] plus bullet",                // 11
		"- [risk]",                            // 12 no text
		"- [risk]    ",                        // 13 blank text
		"- [port_no-2] keeps #tag (context) and [[Link]]", // 14
		"plain prose with [[Other]]",                      // 15
		"- [risk]no space",                                // 16
	)
	n := parse.Parse("n.md", src)
	want := []parse.Observation{
		{"prereq", "Download content updates", 2},
		{"risk", "HA pair first", 3},
		{"port_no-2", "keeps #tag (context) and [[Link]]", 14},
	}
	if !reflect.DeepEqual(n.Observations, want) {
		t.Fatalf("observations %+v", n.Observations)
	}
	wantLinks := []parse.Link{{"Link", "", 14}, {"Other", "", 15}}
	if !reflect.DeepEqual(n.Links, wantLinks) {
		t.Fatalf("links %+v", n.Links)
	}
	mustNoProblems(t, n)
}

func TestRelations_SPEC0001_R16(t *testing.T) {
	src := lines(
		"## Relations",                           // 1
		"- implements [[ODR-017]]",               // 2
		"- applies_to [[PA-850|the firewall]]",   // 3
		"- depends_on [[a]] trailing text [[b]]", // 4 trailing ignored
		"- Bad [[x]]",                            // 5 bad type
		"- see_also",                             // 6 no link
		"- part-of [[y]]",                        // 7 hyphen not allowed in type
		"prose [[z]]",                            // 8 link
		"- runs_on `[[code]]`",                   // 9 code span
	)
	n := parse.Parse("n.md", src)
	want := []parse.Relation{
		{"implements", "ODR-017", "", 2},
		{"applies_to", "PA-850", "the firewall", 3},
		{"depends_on", "a", "", 4},
	}
	if !reflect.DeepEqual(n.Relations, want) {
		t.Fatalf("relations %+v", n.Relations)
	}
	wantLinks := []parse.Link{{"x", "", 5}, {"y", "", 7}, {"z", "", 8}}
	if !reflect.DeepEqual(n.Links, wantLinks) {
		t.Fatalf("links %+v", n.Links)
	}
}

func TestParse_SPEC0001_R1_R2_designExample(t *testing.T) {
	n := parse.Parse("runbooks/PAN-OS upgrade procedure.md", lines(
		"---",
		"title: PAN-OS upgrade procedure",
		"type: runbook",
		"tags: [palo-alto, firewall]",
		"aliases: [PAN-OS upgrade]",
		"created: 2026-11-19T10:12:00+01:00",
		"updated: 2026-11-19T10:12:00+01:00",
		"---",
		"",
		"# PAN-OS upgrade procedure",
		"",
		"Free Markdown body. A mention of [[PA-850]] here is indexed as links_to.",
		"",
		"## Observations",
		"- [prereq] Download content updates before the base image",
		"- [risk] HA pair must be upgraded passive first",
		"",
		"## Relations",
		"- implements [[ODR-017]]",
		"- applies_to [[PA-850]]",
	))
	mustNoProblems(t, n)
	if n.Title != "PAN-OS upgrade procedure" || n.Type != "runbook" ||
		!equalStrings(n.Tags, []string{"palo-alto", "firewall"}) || !equalStrings(n.Aliases, []string{"PAN-OS upgrade"}) {
		t.Fatalf("fields %+v", n)
	}
	wantH := []parse.Heading{{1, "PAN-OS upgrade procedure", 10}, {2, "Observations", 14}, {2, "Relations", 18}}
	wantO := []parse.Observation{{"prereq", "Download content updates before the base image", 15}, {"risk", "HA pair must be upgraded passive first", 16}}
	wantR := []parse.Relation{{"implements", "ODR-017", "", 19}, {"applies_to", "PA-850", "", 20}}
	wantL := []parse.Link{{"PA-850", "", 12}}
	if !reflect.DeepEqual(n.Headings, wantH) || !reflect.DeepEqual(n.Observations, wantO) ||
		!reflect.DeepEqual(n.Relations, wantR) || !reflect.DeepEqual(n.Links, wantL) {
		t.Fatalf("structure:\n%+v\n%+v\n%+v\n%+v", n.Headings, n.Observations, n.Relations, n.Links)
	}
}
