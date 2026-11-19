package note

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxFileNameBytes bounds a file name without .md (SPEC-0007 R10).
const maxFileNameBytes = 200

// FileName returns the file name of a note with this title: the title as
// written, with characters unsafe in file names and leading dots replaced by
// "-", runs of spaces collapsed, trimmed, cut to 200 bytes on a character
// boundary, plus .md (SPEC-0007 R10). A title with nothing left is an
// *ArgError.
func FileName(title string) (string, error) {
	var b strings.Builder
	space := false
	for _, r := range title {
		switch {
		case strings.ContainsRune(`/\:*?"<>|`, r), unicode.IsControl(r), r == utf8.RuneError:
			r = '-'
		case r == ' ':
			if space {
				continue
			}
		}
		space = r == ' '
		b.WriteRune(r)
	}
	name := strings.TrimSpace(b.String())
	if trimmed := strings.TrimLeft(name, "."); len(trimmed) < len(name) {
		name = strings.Repeat("-", len(name)-len(trimmed)) + trimmed
	}
	if len(name) > maxFileNameBytes {
		cut := maxFileNameBytes
		for cut > 0 && !utf8.RuneStart(name[cut]) {
			cut--
		}
		name = strings.TrimSpace(name[:cut])
	}
	if name == "" {
		return "", &ArgError{Arg: "title", Reason: "has no characters that can form a file name"}
	}
	return name + ".md", nil
}
