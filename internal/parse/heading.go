package parse

import "strings"

// atxHeading parses an ATX heading (SPEC-0001 R11).
func atxHeading(text string) (Heading, bool) {
	sp, rest := leading(text)
	if sp > 3 {
		return Heading{}, false
	}
	level := 0
	for level < len(rest) && rest[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return Heading{}, false
	}
	after := rest[level:]
	if after != "" && after[0] != ' ' && after[0] != '\t' {
		return Heading{}, false
	}
	t := strings.TrimSpace(after)
	// An optional closing sequence of #, preceded by a space, is removed.
	if strings.Trim(t, "#") == "" {
		t = ""
	} else if trimmed := strings.TrimRight(t, "#"); trimmed != t {
		if strings.HasSuffix(trimmed, " ") || strings.HasSuffix(trimmed, "\t") {
			t = strings.TrimSpace(trimmed)
		}
	}
	return Heading{Level: level, Text: t}, true
}

// sectionKind returns the section a level-2 heading opens, if any
// (SPEC-0001 R12).
func sectionKind(h Heading) (SectionKind, bool) {
	if h.Level != 2 {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(h.Text)) {
	case "observations":
		return SectionObservations, true
	case "relations":
		return SectionRelations, true
	}
	return "", false
}

// FindSection returns the first heading whose text equals name, trimmed and
// without regard to letter case, and the 1-based line of the next heading of
// the same or a higher level (0 when the section runs to the end of the
// file). This is the section of note_read and of note_edit's
// replace_section (SPEC-0004 R16, SPEC-0007 R17).
func FindSection(n Note, name string) (h Heading, end int, ok bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for i, h := range n.Headings {
		if strings.ToLower(strings.TrimSpace(h.Text)) != want {
			continue
		}
		for _, next := range n.Headings[i+1:] {
			if next.Level <= h.Level {
				return h, next.Line, true
			}
		}
		return h, 0, true
	}
	return Heading{}, 0, false
}
