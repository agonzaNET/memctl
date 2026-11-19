package parse

import "strings"

// foundLink is a [[link]] found on a line: its target, label and the byte
// offset where its "[[" starts. bad marks an empty target (SPEC-0001 R17).
type foundLink struct {
	target, label string
	start         int
	bad           bool
}

// codeMask marks the bytes of text that are inside inline code spans: a run
// of backticks opens a span that closes at the next run of exactly the same
// length on the line; a run with no such partner is literal (SPEC-0001 R18).
func codeMask(text string) []bool {
	var mask []bool
	i := 0
	for i < len(text) {
		if text[i] != '`' {
			i++
			continue
		}
		n := 0
		for i+n < len(text) && text[i+n] == '`' {
			n++
		}
		close := -1
		for j := i + n; j < len(text); {
			if text[j] != '`' {
				j++
				continue
			}
			m := 0
			for j+m < len(text) && text[j+m] == '`' {
				m++
			}
			if m == n {
				close = j
				break
			}
			j += m
		}
		if close < 0 {
			i += n
			continue
		}
		if mask == nil {
			mask = make([]bool, len(text))
		}
		for k := i; k < close+n; k++ {
			mask[k] = true
		}
		i = close + n
	}
	return mask
}

func inCode(mask []bool, from, to int) bool {
	if mask == nil {
		return false
	}
	for k := from; k < to && k < len(mask); k++ {
		if mask[k] {
			return true
		}
	}
	return false
}

// scanLinks returns the links of one line outside code spans, in order
// (SPEC-0001 R17, R18).
func scanLinks(text string) []foundLink {
	if !strings.Contains(text, "[[") {
		return nil
	}
	mask := codeMask(text)
	var out []foundLink
	i := 0
	for {
		k := strings.Index(text[i:], "[[")
		if k < 0 {
			return out
		}
		open := i + k
		end := strings.Index(text[open+2:], "]]")
		if end < 0 {
			return out
		}
		closeAt := open + 2 + end
		inner := text[open+2 : closeAt]
		if strings.ContainsAny(inner, "[]") || inCode(mask, open, closeAt+2) {
			i = open + 1
			continue
		}
		target, label, _ := strings.Cut(inner, "|")
		target, label = strings.TrimSpace(target), strings.TrimSpace(label)
		out = append(out, foundLink{target: target, label: label, start: open, bad: target == ""})
		i = closeAt + 2
	}
}
