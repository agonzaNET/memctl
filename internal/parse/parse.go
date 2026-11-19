package parse

import (
	"bytes"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

var bom = []byte("\xef\xbb\xbf")

// line is one line of the input without its line ending. end is the byte
// offset just after the line ending (or the end of the data).
type line struct {
	text string
	end  int
}

// splitLines splits data on "\n" and drops a "\r" before it (SPEC-0001 R8).
// A final line without a newline counts; an empty input has no lines.
func splitLines(data []byte) []line {
	var out []line
	start := 0
	for start < len(data) {
		i := bytes.IndexByte(data[start:], '\n')
		var text []byte
		end := len(data)
		if i < 0 {
			text = data[start:]
		} else {
			text = data[start : start+i]
			end = start + i + 1
		}
		text = bytes.TrimSuffix(text, []byte("\r"))
		out = append(out, line{text: string(text), end: end})
		start = end
	}
	return out
}

// Parse parses the note at rel (a slash-separated path relative to the notes
// root) from its raw bytes. It never fails and never panics (SPEC-0001 R22).
func Parse(rel string, data []byte) (n Note) {
	data = bytes.TrimPrefix(data, bom)
	ls := splitLines(data)
	n.Body = string(data)
	n.BodyLine = 1

	parseFrontmatter(&n, data, ls)

	for i, l := range ls {
		if !utf8.ValidString(l.text) {
			n.addProblem(CodeInvalidUTF8, i+1, "The line is not valid UTF-8; save the note as UTF-8.")
			break
		}
	}

	if n.Title == "" {
		n.Title = strings.TrimSuffix(path.Base(rel), ".md")
		n.TitleFromFrontmatter = false
	}

	sort.SliceStable(n.Problems, func(i, j int) bool {
		a, b := n.Problems[i], n.Problems[j]
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	return n
}

func (n *Note) addProblem(code string, line int, msg string) {
	n.Problems = append(n.Problems, Problem{Code: code, Line: line, Message: msg})
}
