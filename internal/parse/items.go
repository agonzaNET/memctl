package parse

// isLower reports a byte in a–z.
func isLower(c byte) bool { return c >= 'a' && c <= 'z' }

// observationForm parses "- [category] text" (SPEC-0001 R13): the bullet at
// the start of the line, one space, a category matching [a-z][a-z0-9_-]+,
// one or more spaces, non-empty text.
func observationForm(text string) (category, rest string, ok bool) {
	if len(text) < 4 || text[0] != '-' || text[1] != ' ' || text[2] != '[' {
		return "", "", false
	}
	i := 3
	if i >= len(text) || !isLower(text[i]) {
		return "", "", false
	}
	j := i + 1
	for j < len(text) && (isLower(text[j]) || (text[j] >= '0' && text[j] <= '9') || text[j] == '_' || text[j] == '-') {
		j++
	}
	if j-i < 2 || j >= len(text) || text[j] != ']' {
		return "", "", false
	}
	category = text[i:j]
	k := j + 1
	if k >= len(text) || text[k] != ' ' {
		return "", "", false
	}
	for k < len(text) && text[k] == ' ' {
		k++
	}
	rest = trimSpace(text[k:])
	if rest == "" {
		return "", "", false
	}
	return category, rest, true
}

// relationForm parses "- type [[Target|label]]" (SPEC-0001 R16): the bullet
// at the start of the line, one space, a type matching [a-z][a-z0-9_]*, one
// or more spaces, and a link; text after the link is ignored.
func relationForm(text string) (typ string, link foundLink, ok bool) {
	if len(text) < 3 || text[0] != '-' || text[1] != ' ' || !isLower(text[2]) {
		return "", foundLink{}, false
	}
	j := 3
	for j < len(text) && (isLower(text[j]) || (text[j] >= '0' && text[j] <= '9') || text[j] == '_') {
		j++
	}
	typ = text[2:j]
	k := j
	for k < len(text) && text[k] == ' ' {
		k++
	}
	if k == j {
		return "", foundLink{}, false
	}
	links := scanLinks(text)
	if len(links) == 0 || links[0].start != k || links[0].bad {
		return "", foundLink{}, false
	}
	return typ, links[0], true
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
