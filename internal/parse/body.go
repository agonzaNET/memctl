package parse

import "strings"

// lineKind is what the previous body line was, for the setext and indented
// code rules (SPEC-0001 R20).
type lineKind int

const (
	kindNone lineKind = iota
	kindBlank
	kindText
	kindHeading
)

// scanBody walks the body lines once: fences, headings, sections,
// observations, relations, links and the line problems (SPEC-0001 R8–R20).
// first is the 0-based index of the first body line.
func scanBody(n *Note, ls []line, first int) {
	var fence fenceState
	cur := -1 // index into n.Sections of the open section, -1 for none
	closeSection := func(lastLine int) {
		if cur >= 0 {
			n.Sections[cur].End = lastLine
			cur = -1
		}
	}
	prev := kindNone
	htmlOpen, codeOpen, inList := false, false, false

	for i := first; i < len(ls); i++ {
		lineNo := i + 1
		text := ls[i].text
		if fence.step(text) {
			prev, htmlOpen, codeOpen = kindNone, false, false
			continue
		}
		if trimSpace(text) == "" {
			prev, htmlOpen = kindBlank, false
			continue
		}

		indented := strings.HasPrefix(text, "\t") || strings.HasPrefix(text, "    ")
		if indented {
			if prev == kindBlank && !inList && !codeOpen {
				n.addProblem(CodeIndentedCode, lineNo, "Text indented by four spaces or a tab after a blank line is code, which memctl does not read as structure; use a fenced code block (```) instead.")
				codeOpen = true
			}
			if codeOpen {
				prev = kindText
				continue
			}
		} else {
			codeOpen = false
		}

		if !htmlOpen && isHTMLStart(text) {
			n.addProblem(CodeHTMLBlock, lineNo, "An HTML block starts here; memctl reads no structure in it, so write the content as Markdown.")
			htmlOpen = true
		}
		if htmlOpen {
			prev = kindText
			continue
		}

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
			prev, inList = kindHeading, false
			continue
		}

		if isSetextUnderline(text) && prev == kindText {
			n.addProblem(CodeSetextHeading, lineNo, "This underline makes the line above a heading that memctl does not read; write the heading with # marks instead.")
			prev = kindText
			continue
		}

		if isListItem(text) {
			inList = true
		} else if !indented {
			inList = false
		}
		prev = kindText

		section := SectionKind("")
		if cur >= 0 {
			section = n.Sections[cur].Kind
		}
		if cat, rest, ok := observationForm(text); ok {
			if section == SectionObservations {
				n.Observations = append(n.Observations, Observation{Category: cat, Text: rest, Line: lineNo})
				n.addLinks(scanLinks(rest), lineNo)
				continue
			}
			n.addProblem(CodeObservationOutsideSection, lineNo, "This line has the form of an observation but is not under ## Observations, so it is not read as one; move it there or change its form.")
		}
		if typ, l, ok := relationForm(text); ok {
			if section == SectionRelations {
				n.Relations = append(n.Relations, Relation{Type: typ, Target: l.target, Label: l.label, Line: lineNo})
				continue
			}
			n.addProblem(CodeRelationOutsideSection, lineNo, "This line has the form of a relation but is not under ## Relations, so it is read as a plain link; move it there or change its form.")
		}
		n.addLinks(scanLinks(text), lineNo)
	}
	closeSection(len(ls))
}

// isHTMLStart reports a line that starts an HTML block: "<!--", or "<" and a
// letter, after at most three spaces (SPEC-0001 R20).
func isHTMLStart(text string) bool {
	sp, rest := leading(text)
	if sp > 3 {
		return false
	}
	if strings.HasPrefix(rest, "<!--") {
		return true
	}
	return len(rest) >= 2 && rest[0] == '<' && ((rest[1] >= 'a' && rest[1] <= 'z') || (rest[1] >= 'A' && rest[1] <= 'Z'))
}

// isSetextUnderline reports a line of only "=" or only "-", at least two,
// after at most three spaces (SPEC-0001 R20).
func isSetextUnderline(text string) bool {
	sp, rest := leading(text)
	if sp > 3 {
		return false
	}
	rest = strings.TrimRight(rest, " \t")
	if len(rest) < 2 {
		return false
	}
	return strings.Trim(rest, "=") == "" || strings.Trim(rest, "-") == ""
}

// isListItem reports a bullet or ordered list item after at most three
// spaces.
func isListItem(text string) bool {
	sp, rest := leading(text)
	if sp > 3 || rest == "" {
		return false
	}
	if (rest[0] == '-' || rest[0] == '*' || rest[0] == '+') && (len(rest) == 1 || rest[1] == ' ' || rest[1] == '\t') {
		return true
	}
	j := 0
	for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
		j++
	}
	return j > 0 && j < len(rest) && (rest[j] == '.' || rest[j] == ')') && (j+1 == len(rest) || rest[j+1] == ' ')
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
