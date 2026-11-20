package index

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// nearNames returns the indexed tokens that are a query term of two or more
// letters followed by one or more digits, such as cisco1 for cisco, sorted
// (SPEC-0002 R29). It scans every indexed token, which is fine for the
// not_found suggestions, the only caller.
func (x *Index) nearNames(terms []string) []string {
	var words []string
	for _, t := range terms {
		if utf8.RuneCountInString(t) >= 2 && strings.IndexFunc(t, func(r rune) bool { return !unicode.IsLetter(r) }) < 0 {
			words = append(words, t)
		}
	}
	if len(words) == 0 {
		return nil
	}
	var out []string
	for tok := range x.postings {
		for _, w := range words {
			rest, ok := strings.CutPrefix(tok, w)
			if ok && rest != "" && strings.IndexFunc(rest, func(r rune) bool { return !unicode.IsDigit(r) }) < 0 {
				out = append(out, tok)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
