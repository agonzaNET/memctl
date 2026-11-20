package index

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/agonzaNET/memctl/internal/parse"
)

// snippetRunes is the longest snippet, without its ellipses (SPEC-0002 R14,
// R27).
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
// query terms, else the first non-empty body line, cut by window (SPEC-0002
// R14, R27).
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
	return window(strings.TrimSpace(lineText(bestLine)), terms), bestLine
}

// window cuts a line longer than snippetRunes to that many runes, centred on
// its first matched token or from its start when it has none, drops a
// partial word at a cut edge, trims spaces there and marks each cut side
// with "…" (SPEC-0002 R27).
func window(s string, terms map[string]bool) string {
	r := []rune(s)
	if len(r) <= snippetRunes {
		return s
	}
	from := 0
	start, n, ok := firstMatch(s, terms)
	if ok {
		from = start
		if n < snippetRunes {
			from = start - (snippetRunes-n)/2
		}
	}
	from = max(0, min(from, len(r)-snippetRunes))
	to := from + snippetRunes
	from, to = wordCuts(r, from, to, start, start+n, ok)
	out := string(r[from:to])
	if from > 0 {
		out = "…" + strings.TrimLeftFunc(out, unicode.IsSpace)
	}
	if to < len(r) {
		out = strings.TrimRightFunc(out, unicode.IsSpace) + "…"
	}
	return out
}

// wordCuts moves a cut edge that splits a word inward to the nearest
// whitespace, dropping the partial word, unless that removes part of the
// token [tokStart, tokEnd) (when hasTok) or leaves only whitespace
// (SPEC-0002 R27).
func wordCuts(r []rune, from, to, tokStart, tokEnd int, hasTok bool) (int, int) {
	keeps := func(f, t int) bool {
		if hasTok && (f > tokStart || t < tokEnd) {
			return false
		}
		return strings.TrimSpace(string(r[f:t])) != ""
	}
	if from > 0 && !unicode.IsSpace(r[from-1]) && !unicode.IsSpace(r[from]) {
		i := from
		for i < to && !unicode.IsSpace(r[i]) {
			i++
		}
		if i < to && keeps(i, to) {
			from = i
		}
	}
	if to < len(r) && !unicode.IsSpace(r[to-1]) && !unicode.IsSpace(r[to]) {
		j := to
		for j > from && !unicode.IsSpace(r[j-1]) {
			j--
		}
		if j > from && keeps(from, j) {
			to = j
		}
	}
	return from, to
}

// firstMatch returns the rune offset and rune length of the first token of
// text that is a query term: of the tokens that start at the earliest rune,
// the longest. ok is false when text holds no query term. Lower-casing maps
// rune to rune, so offsets in the lower-cased text are offsets in text.
func firstMatch(text string, terms map[string]bool) (start, length int, ok bool) {
	lower := strings.ToLower(text)
	consider := func(tok string, byteOff int) {
		s, n := utf8.RuneCountInString(lower[:byteOff]), utf8.RuneCountInString(tok)
		if !ok || s < start || (s == start && n > length) {
			start, length, ok = s, n, true
		}
	}
	for _, c := range chains(text) {
		for i, p := range c.parts {
			if terms[p] {
				consider(p, c.starts[i])
			}
		}
		if len(c.parts) > 1 && terms[c.text] {
			consider(c.text, c.first)
		}
	}
	return start, length, ok
}
