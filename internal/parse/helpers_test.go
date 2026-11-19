package parse_test

import (
	"strings"
	"testing"

	"github.com/agonzaNET/memctl/internal/parse"
)

// lines joins its arguments with "\n" and adds a final newline.
func lines(ls ...string) []byte {
	return []byte(strings.Join(ls, "\n") + "\n")
}

// codes returns "code@line" for every problem, in order.
func codes(n parse.Note) []string {
	var out []string
	for _, p := range n.Problems {
		out = append(out, p.Code+"@"+itoa(p.Line))
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasCode(n parse.Note, code string, line int) bool {
	for _, p := range n.Problems {
		if p.Code == code && p.Line == line {
			return true
		}
	}
	return false
}

func mustNoProblems(t *testing.T, n parse.Note) {
	t.Helper()
	if len(n.Problems) != 0 {
		t.Fatalf("unexpected problems: %v", codes(n))
	}
}
