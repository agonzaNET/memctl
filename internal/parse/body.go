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
		if h, ok := atxHeading(ls[i].text); ok {
			h.Line = lineNo
			n.Headings = append(n.Headings, h)
			if h.Level <= 2 {
				closeSection(lineNo - 1)
				if kind, ok := sectionKind(h); ok {
					n.Sections = append(n.Sections, Section{Kind: kind, Start: lineNo})
					cur = len(n.Sections) - 1
				}
			}
			continue
		}
	}
	closeSection(len(ls))
}
