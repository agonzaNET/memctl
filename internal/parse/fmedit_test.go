package parse_test

import (
	"errors"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

const stamp = "2026-11-20T09:00:00+01:00"

func TestEditFrontmatter_SPEC0007_R8_golden(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		changes []parse.FrontmatterSet
		want    string
	}{
		{
			name: "comments, unknown keys, block list, quoted values",
			in: `---
# kept comment
title: "Old: title"   # trailing comment
custom:
  nested: [1, 2]
aliases:
  - first
  - "second, quoted"
# comment before tags
tags: [a, b]
updated: 2026-01-01T00:00:00+01:00
unknown_key: 'single quoted'
---
Body line one.
  indented body with title: x
`,
			changes: []parse.FrontmatterSet{
				{Key: "updated", Value: stamp},
				{Key: "title", Value: parse.FormatScalar("New title")},
				{Key: "aliases", Value: parse.FormatList([]string{"first", "second, quoted", "old/path"})},
			},
			want: `---
# kept comment
title: New title
custom:
  nested: [1, 2]
aliases: [first, "second, quoted", old/path]
# comment before tags
tags: [a, b]
updated: ` + stamp + `
unknown_key: 'single quoted'
---
Body line one.
  indented body with title: x
`,
		},
		{
			name: "new keys before the closing delimiter, in order",
			in:   "---\ntitle: T\n---\n\nbody\n",
			changes: []parse.FrontmatterSet{
				{Key: "aliases", Value: parse.FormatList([]string{"x"})},
				{Key: "updated", Value: stamp},
			},
			want: "---\ntitle: T\naliases: [x]\nupdated: " + stamp + "\n---\n\nbody\n",
		},
		{
			name:    "flow list replaced",
			in:      "---\naliases: [a, b]   # note\ntitle: T\n---\n",
			changes: []parse.FrontmatterSet{{Key: "aliases", Value: parse.FormatList([]string{"a", "b", "c"})}},
			want:    "---\naliases: [a, b, c]\ntitle: T\n---\n",
		},
		{
			name:    "last key with trailing blank line kept",
			in:      "---\ntitle: T\nupdated: x\n\n---\nbody",
			changes: []parse.FrontmatterSet{{Key: "updated", Value: stamp}},
			want:    "---\ntitle: T\nupdated: " + stamp + "\n\n---\nbody",
		},
		{
			name:    "CRLF line endings",
			in:      "---\r\ntitle: T\r\nupdated: x\r\n---\r\nbody\r\n",
			changes: []parse.FrontmatterSet{{Key: "updated", Value: stamp}, {Key: "aliases", Value: "[a]"}},
			want:    "---\r\ntitle: T\r\nupdated: " + stamp + "\r\naliases: [a]\r\n---\r\nbody\r\n",
		},
		{
			name:    "byte order mark kept",
			in:      "\ufeff---\ntitle: T\n---\n",
			changes: []parse.FrontmatterSet{{Key: "title", Value: "U"}},
			want:    "\ufeff---\ntitle: U\n---\n",
		},
		{
			name:    "closing ... delimiter",
			in:      "---\ntitle: T\n...\nbody\n",
			changes: []parse.FrontmatterSet{{Key: "updated", Value: stamp}},
			want:    "---\ntitle: T\nupdated: " + stamp + "\n...\nbody\n",
		},
		{
			name:    "empty frontmatter",
			in:      "---\n---\nbody\n",
			changes: []parse.FrontmatterSet{{Key: "updated", Value: stamp}},
			want:    "---\nupdated: " + stamp + "\n---\nbody\n",
		},
		{
			name:    "multi-line block scalar replaced",
			in:      "---\ntitle: |\n  long\n  # not a comment\nother: 1\n---\n",
			changes: []parse.FrontmatterSet{{Key: "title", Value: "Short"}},
			want:    "---\ntitle: Short\nother: 1\n---\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parse.EditFrontmatter([]byte(c.in), c.changes...)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != c.want {
				t.Fatalf("got:\n%q\nwant:\n%q", got, c.want)
			}
			if n := parse.Parse("x.md", got); len(n.Problems) != 0 {
				t.Fatalf("result has problems: %+v", n.Problems)
			}
		})
	}
}

func TestEditFrontmatter_SPEC0007_R9_refused(t *testing.T) {
	for name, in := range map[string]string{
		"no frontmatter": "just a body\n",
		"empty file":     "",
		"never closed":   "---\ntitle: T\nbody\n",
		"invalid YAML":   "---\ntitle: [unclosed\n---\n",
		"not a mapping":  "---\n- a\n- b\n---\n",
		"flow mapping":   "---\n{title: T, type: x}\n---\n",
	} {
		if _, err := parse.EditFrontmatter([]byte(in), parse.FrontmatterSet{Key: "updated", Value: stamp}); !errors.Is(err, parse.ErrNoFrontmatter) {
			t.Errorf("%s: err %v, want ErrNoFrontmatter", name, err)
		}
	}
}
