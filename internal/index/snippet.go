package index

import (
	"strings"
	"unicode/utf8"

	"github.com/agonzaNET/memctl/internal/parse"
)

// snippetRunes is the longest snippet, without its ellipses (SPEC-0002 R14).
const snippetRunes = 200

// countTerms returns how many distinct query terms text contains.
func countTerms(text string, terms map[string]bool) int {
	seen := map[string]bool{}
	for _, t := range Tokenize(text) {
		if terms[t] {
			seen[t] = true
		}
	}
	return len(seen)
}

// snippet picks the observation, else the body line, with the most distinct
// query terms, else the first non-empty body line (SPEC-0002 R14).
func snippet(n parse.Note, terms map[string]bool) (string, int) {
	lines := strings.Split(n.Body, "\n")
	lineText := func(lineNo int) string {
		i := lineNo - n.BodyLine
		if i < 0 || i >= len(lines) {
			return ""
		}
		return strings.TrimSuffix(lines[i], "\r")
	}
	best, bestLine := 0, 0
	for _, o := range n.Observations {
		if c := countTerms(o.Text, terms); c > best {
			best, bestLine = c, o.Line
		}
	}
	if best == 0 {
		bodyLines(n, func(line int, text string) {
			if c := countTerms(text, terms); c > best {
				best, bestLine = c, line
			}
		})
	}
	if best == 0 {
		bodyLines(n, func(line int, text string) {
			if bestLine == 0 && strings.TrimSpace(text) != "" {
				bestLine = line
			}
		})
	}
	if bestLine == 0 {
		return "", 0
	}
	return cut(strings.TrimSpace(lineText(bestLine))), bestLine
}

func cut(s string) string {
	if utf8.RuneCountInString(s) <= snippetRunes {
		return s
	}
	r := []rune(s)
	return string(r[:snippetRunes]) + "…"
}
