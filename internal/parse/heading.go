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
