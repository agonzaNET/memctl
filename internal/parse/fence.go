package parse

import "strings"

// fenceState tracks fenced code blocks (SPEC-0001 R9).
type fenceState struct {
	open bool
	char byte
	n    int
}

// leading returns the number of leading spaces, up to 4, and the rest.
func leading(text string) (int, string) {
	i := 0
	for i < len(text) && i < 4 && text[i] == ' ' {
		i++
	}
	return i, text[i:]
}

// fenceRun returns the fence character and run length at the start of rest,
// or 0 when rest does not start with at least three backticks or tildes.
func fenceRun(rest string) (byte, int) {
	if rest == "" || (rest[0] != '`' && rest[0] != '~') {
		return 0, 0
	}
	c := rest[0]
	n := 0
	for n < len(rest) && rest[n] == c {
		n++
	}
	if n < 3 {
		return 0, 0
	}
	return c, n
}

// step updates the state for one line and reports whether the line is part
// of a fence (an opening, inside, or closing line) and so carries no
// structure (SPEC-0001 R10).
func (f *fenceState) step(text string) bool {
	sp, rest := leading(text)
	if sp > 3 {
		return f.open
	}
	c, n := fenceRun(rest)
	if f.open {
		if c == f.char && n >= f.n && strings.TrimRight(rest[n:], " ") == "" {
			f.open = false
		}
		return true
	}
	if c == 0 {
		return false
	}
	if c == '`' && strings.Contains(rest[n:], "`") {
		return false
	}
	f.open, f.char, f.n = true, c, n
	return true
}
