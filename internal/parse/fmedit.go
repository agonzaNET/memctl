package parse

import (
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
