package parse

// scanBody walks the body lines once: fences, headings and sections
// (SPEC-0001 R8–R12). first is the 0-based index of the first body line.
func scanBody(n *Note, ls []line, first int) {
	var fence fenceState
	cur := -1 // index into n.Sections of the open section, -1 for none
	closeSection := func(lastLine int) {
		if cur >= 0 {
			n.Sections[cur].End = lastLine
			cur = -1
		}
	}
	for i := first; i < len(ls); i++ {
		lineNo := i + 1
		if fence.step(ls[i].text) {
			continue
		}
		text := ls[i].text
		if h, ok := atxHeading(text); ok {
			h.Line = lineNo
			n.Headings = append(n.Headings, h)
			if h.Level <= 2 {
				closeSection(lineNo - 1)
				if kind, ok := sectionKind(h); ok {
					n.Sections = append(n.Sections, Section{Kind: kind, Start: lineNo})
					cur = len(n.Sections) - 1
				}
			}
			n.addLinks(scanLinks(text), lineNo)
			continue
		}
		section := SectionKind("")
		if cur >= 0 {
			section = n.Sections[cur].Kind
		}
		switch section {
		case SectionObservations:
			if cat, rest, ok := observationForm(text); ok {
				n.Observations = append(n.Observations, Observation{Category: cat, Text: rest, Line: lineNo})
				n.addLinks(scanLinks(rest), lineNo)
				continue
			}
		case SectionRelations:
			if typ, l, ok := relationForm(text); ok {
				n.Relations = append(n.Relations, Relation{Type: typ, Target: l.target, Label: l.label, Line: lineNo})
				continue
			}
		}
		n.addLinks(scanLinks(text), lineNo)
	}
	closeSection(len(ls))
}

// addLinks records the links of one line; an empty target is a bad_link
// problem and is not recorded (SPEC-0001 R17).
func (n *Note) addLinks(links []foundLink, lineNo int) {
	for _, l := range links {
		if l.bad {
			n.addProblem(CodeBadLink, lineNo, "A link on this line has an empty target; write [[target]] or remove the brackets.")
			continue
		}
		n.Links = append(n.Links, Link{Target: l.target, Label: l.label, Line: lineNo})
	}
}
