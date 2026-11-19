package parse

import (
	"fmt"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// isDelimiter reports whether a line is exactly one of the delimiters, with
// trailing spaces or tabs allowed (SPEC-0001 R1).
func isDelimiter(text string, delims ...string) bool {
	t := strings.TrimRight(text, " \t")
	for _, d := range delims {
		if t == d {
			return true
		}
	}
	return false
}

// parseFrontmatter reads the frontmatter block, if any, into n and sets the
// body (SPEC-0001 R1–R7).
func parseFrontmatter(n *Note, data []byte, ls []line) {
	if len(ls) == 0 || !isDelimiter(ls[0].text, "---") {
		return
	}
	end := -1
	for i := 1; i < len(ls); i++ {
		if isDelimiter(ls[i].text, "---", "...") {
			end = i
			break
		}
	}
	if end < 0 {
		n.addProblem(CodeBadFrontmatter, 1, "The frontmatter that starts on line 1 is never closed; add a line with --- after it.")
		return
	}
	n.HasFrontmatter = true
	n.FrontmatterEnd = end + 1
	n.Body = string(data[ls[end].end:])
	n.BodyLine = end + 2

	yamlLines := make([]string, 0, end-1)
	for _, l := range ls[1:end] {
		yamlLines = append(yamlLines, l.text)
	}
	root, ok := decodeMapping(strings.Join(yamlLines, "\n"))
	if !ok {
		n.addProblem(CodeBadFrontmatter, 1, "The frontmatter is not a valid YAML mapping of keys to values; fix its syntax.")
		return
	}
	if root == nil {
		return
	}

	seen := map[string]bool{}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, val := root.Content[i], root.Content[i+1]
		keyLine := key.Line // 1-based within the YAML text
		nextLine := len(yamlLines) + 1
		if i+2 < len(root.Content) {
			nextLine = root.Content[i+2].Line
		}
		n.Frontmatter = append(n.Frontmatter, Field{
			Key:  key.Value,
			Raw:  rawLines(yamlLines, keyLine, nextLine),
			Line: keyLine + 1,
		})
		if seen[key.Value] {
			continue
		}
		seen[key.Value] = true
		readKnown(n, key.Value, deref(val), keyLine+1)
	}
}

// decodeMapping decodes YAML text into its top-level mapping node. It returns
// (nil, true) for an empty document and (nil, false) for invalid YAML or a
// top level that is not a mapping. A panic in the decoder counts as invalid.
func decodeMapping(text string) (root *yaml.Node, ok bool) {
	defer func() {
		if recover() != nil {
			root, ok = nil, false
		}
	}()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return nil, false
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return nil, true
	}
	top := deref(doc.Content[0])
	if top.Kind == yaml.ScalarNode && top.Tag == "!!null" {
		return nil, true
	}
	if top.Kind != yaml.MappingNode {
		return nil, false
	}
	return top, true
}

func deref(v *yaml.Node) *yaml.Node {
	for v != nil && v.Kind == yaml.AliasNode && v.Alias != nil {
		v = v.Alias
	}
	return v
}

// rawLines returns lines [from, to) of the YAML text (1-based), without
// trailing blank lines; at least the line from.
func rawLines(ls []string, from, to int) string {
	if to <= from {
		to = from + 1
	}
	if from < 1 {
		from = 1
	}
	if to-1 > len(ls) {
		to = len(ls) + 1
	}
	part := ls[from-1 : to-1]
	for len(part) > 1 && strings.TrimSpace(part[len(part)-1]) == "" {
		part = part[:len(part)-1]
	}
	return strings.Join(part, "\n")
}

func isNull(v *yaml.Node) bool {
	return v == nil || (v.Kind == yaml.ScalarNode && v.Tag == "!!null")
}

// readKnown sets a known field from its value node, or reports bad_field.
func readKnown(n *Note, key string, v *yaml.Node, line int) {
	bad := func(want string) {
		n.addProblem(CodeBadField, line, fmt.Sprintf("The %s field must be %s; fix its value.", key, want))
	}
	switch key {
	case "title", "type":
		if isNull(v) {
			return
		}
		if v.Kind != yaml.ScalarNode {
			bad("a single text value")
			return
		}
		s := strings.TrimSpace(v.Value)
		if key == "title" {
			n.Title = s
			n.TitleFromFrontmatter = s != ""
		} else {
			n.Type = s
		}
	case "tags", "aliases":
		list, ok := stringList(v)
		if !ok {
			bad("a list of text values or one text value")
			return
		}
		if key == "tags" {
			n.Tags = list
		} else {
			n.Aliases = list
		}
	case "created", "updated":
		if isNull(v) {
			return
		}
		t, ok := timestamp(v)
		if !ok {
			bad("a date (2026-11-19) or an RFC 3339 timestamp")
			return
		}
		if key == "created" {
			n.Created = t
		} else {
			n.Updated = t
		}
	}
}

// stringList reads a sequence of scalars or one scalar (SPEC-0001 R3): items
// trimmed, empty ones dropped, duplicates removed, order kept.
func stringList(v *yaml.Node) ([]string, bool) {
	if isNull(v) {
		return nil, true
	}
	var raw []string
	switch v.Kind {
	case yaml.ScalarNode:
		raw = []string{v.Value}
	case yaml.SequenceNode:
		for _, item := range v.Content {
			item = deref(item)
			if isNull(item) {
				continue
			}
			if item.Kind != yaml.ScalarNode {
				return nil, false
			}
			raw = append(raw, item.Value)
		}
	default:
		return nil, false
	}
	var out []string
	seen := map[string]bool{}
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, true
}

// timestamp reads an RFC 3339 timestamp or a plain date, as midnight UTC
// (SPEC-0001 R4).
func timestamp(v *yaml.Node) (time.Time, bool) {
	if v.Kind != yaml.ScalarNode {
		return time.Time{}, false
	}
	s := strings.TrimSpace(v.Value)
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}
