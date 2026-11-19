package parse

import (
	"bytes"
	"errors"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FormatScalar returns s as a YAML value for a "key: value" line: plain when
// YAML reads the plain text back as the same string, double-quoted otherwise
// (SPEC-0007 R8, R13).
func FormatScalar(s string) string {
	if plainScalarOK(s) {
		return s
	}
	return strconv.Quote(s)
}

// FormatList returns items as a YAML flow list, "[a, b]", with each item
// quoted only when YAML needs it (SPEC-0007 R8, R13).
func FormatList(items []string) string {
	out := make([]string, len(items))
	for i, it := range items {
		if plainItemOK(it) {
			out[i] = it
		} else {
			out[i] = strconv.Quote(it)
		}
	}
	return "[" + strings.Join(out, ", ") + "]"
}

// plainScalarOK reports whether "k: s" decodes to the string s.
func plainScalarOK(s string) bool {
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return false
	}
	v := decodeValue("k: " + s)
	return v != nil && isStr(v, s)
}

// plainItemOK reports whether "k: [s]" decodes to a list holding only s.
func plainItemOK(s string) bool {
	if s == "" || strings.ContainsAny(s, "\r\n") {
		return false
	}
	v := decodeValue("k: [" + s + "]")
	return v != nil && v.Kind == yaml.SequenceNode && len(v.Content) == 1 && isStr(v.Content[0], s)
}

func isStr(n *yaml.Node, s string) bool {
	return n.Kind == yaml.ScalarNode && n.Tag == "!!str" && n.Style == 0 && n.Value == s
}

// decodeValue returns the value node of the one key in text, or nil.
func decodeValue(text string) *yaml.Node {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return nil
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
		return nil
	}
	m := doc.Content[0]
	if m.Kind != yaml.MappingNode || len(m.Content) != 2 {
		return nil
	}
	return m.Content[1]
}

// FrontmatterSet sets one top-level frontmatter key to Value, which is YAML
// text: a FormatScalar or FormatList result, or a timestamp.
type FrontmatterSet struct{ Key, Value string }

// ErrNoFrontmatter is returned by EditFrontmatter for a note without a
// frontmatter block, or with one that is not a valid block mapping
// (SPEC-0007 R9).
var ErrNoFrontmatter = errors.New("parse: the note has no frontmatter that can be edited")

// EditFrontmatter changes frontmatter keys by editing the text (SPEC-0007
// R8): the line, or the block, of an existing key is replaced by one
// "key: value" line; a new key is inserted before the closing delimiter, in
// the order given. Every other byte stays as it was: comments, unknown keys,
// blank lines, the body, a byte order mark and the line endings.
func EditFrontmatter(data []byte, changes ...FrontmatterSet) ([]byte, error) {
	var prefix []byte
	if bytes.HasPrefix(data, bom) {
		prefix, data = bom, data[len(bom):]
	}
	n := Parse("", data)
	if !n.HasFrontmatter {
		return nil, ErrNoFrontmatter
	}
	for _, p := range n.Problems {
		if p.Code == CodeBadFrontmatter {
			return nil, ErrNoFrontmatter
		}
	}
	ls := splitLines(data)
	closing := n.FrontmatterEnd - 1 // 0-based line of the closing delimiter
	lineStart := func(i int) int {
		if i == 0 {
			return 0
		}
		return ls[i-1].end
	}
	eol := "\n"
	if e := ls[0].end; e >= 2 && data[e-2] == '\r' {
		eol = "\r\n"
	}

	// The span of each top-level key: from its line to the next key or the
	// closing delimiter, without trailing blank lines and column-0 comments,
	// which belong to what follows.
	type span struct{ from, to int }
	spans := map[string]span{}
	seenLine := map[int]bool{}
	for j, f := range n.Frontmatter {
		from := f.Line - 1
		if seenLine[from] || from <= 0 || from >= closing {
			return nil, ErrNoFrontmatter // a flow mapping holds several keys on one line
		}
		seenLine[from] = true
		if t := ls[from].text; t == "" || t[0] == ' ' || t[0] == '\t' || t[0] == '{' {
			return nil, ErrNoFrontmatter
		}
		to := closing
		if j+1 < len(n.Frontmatter) {
			to = n.Frontmatter[j+1].Line - 1
		}
		for to-1 > from {
			t := ls[to-1].text
			if strings.TrimSpace(t) != "" && !strings.HasPrefix(t, "#") {
				break
			}
			to--
		}
		if _, dup := spans[f.Key]; !dup {
			spans[f.Key] = span{from, to}
		}
	}

	replace := map[int]span{} // from → span, for keys that exist
	text := map[int]string{}  // from → new line
	var insert []string
	for _, c := range changes {
		line := c.Key + ": " + c.Value + eol
		if sp, ok := spans[c.Key]; ok {
			replace[sp.from] = sp
			text[sp.from] = line
			continue
		}
		insert = append(insert, line)
	}

	out := make([]byte, 0, len(prefix)+len(data)+64)
	out = append(out, prefix...)
	for i := 0; i < len(ls); {
		if sp, ok := replace[i]; ok {
			out = append(out, text[i]...)
			i = sp.to
			continue
		}
		if i == closing {
			for _, l := range insert {
				out = append(out, l...)
			}
		}
		out = append(out, data[lineStart(i):ls[i].end]...)
		i++
	}
	return out, nil
}
