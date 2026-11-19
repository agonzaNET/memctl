package parse_test

import (
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

func TestFormat_SPEC0007_R8_quoteOnlyWhenNeeded(t *testing.T) {
	scalars := map[string]string{
		"plain":          "plain",
		"PAN-OS upgrade": "PAN-OS upgrade",
		"a: b":           `"a: b"`,
		"true":           `"true"`,
		"123":            `"123"`,
		"null":           `"null"`,
		"":               `""`,
		"#x":             `"#x"`,
		"x #y":           `"x #y"`,
		"- a":            `"- a"`,
		"[a]":            `"[a]"`,
		"'q'":            `"'q'"`,
		` lead`:          `" lead"`,
		`say "hi"`:       `say "hi"`,
		"2026-11-19":     `"2026-11-19"`,
		"a,b":            "a,b",
		"Größe":          "Größe",
		"back\\slash":    "back\\slash",
		"@at":            `"@at"`,
	}
	for in, want := range scalars {
		if got := parse.FormatScalar(in); got != want {
			t.Errorf("FormatScalar(%q) = %s, want %s", in, got, want)
		}
	}
	lists := map[string][]string{
		"[]":                       nil,
		"[a, b]":                   {"a", "b"},
		`[plain tag, "a,b", "x]"]`: {"plain tag", "a,b", "x]"},
		`["true", "{y}"]`:          {"true", "{y}"},
	}
	for want, in := range lists {
		if got := parse.FormatList(in); got != want {
			t.Errorf("FormatList(%q) = %s, want %s", in, got, want)
		}
	}
}
