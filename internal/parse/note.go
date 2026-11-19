package parse

import "time"

// Note is the parsed form of one Markdown note (SPEC-0001, "Outputs").
type Note struct {
	// Title is the frontmatter title, or the file stem of the note's path.
	Title string
	// TitleFromFrontmatter reports whether Title came from the frontmatter.
	TitleFromFrontmatter bool
	Type                 string
	Tags                 []string
	Aliases              []string
	// Created and Updated are zero when absent or invalid.
	Created time.Time
	Updated time.Time

	// HasFrontmatter reports whether the file has a closed frontmatter block,
	// valid or not. FrontmatterEnd is the 1-based line of its closing
	// delimiter, 0 without one.
	HasFrontmatter bool
	FrontmatterEnd int
	// Frontmatter holds every top-level key in file order with its source
	// lines, so a later write can keep unknown keys and their order.
	Frontmatter []Field

	// Body is everything after the frontmatter block; BodyLine is the
	// 1-based file line where it starts.
	Body     string
	BodyLine int

	Headings     []Heading
	Observations []Observation
	Relations    []Relation
	Links        []Link
	Problems     []Problem
}

// Field is one top-level frontmatter key and the raw YAML source lines of its
// entry (the key line up to the line before the next key), without the final
// newline.
type Field struct {
	Key  string
	Raw  string
	Line int
}

// Heading is an ATX heading outside fenced code.
type Heading struct {
	Level int
	Text  string
	Line  int
}

// Observation is a "- [category] text" line of the Observations section.
type Observation struct {
	Category string
	Text     string
	Line     int
}

// Relation is a "- type [[Target]]" line of the Relations section.
type Relation struct {
	Type   string
	Target string
	Label  string
	Line   int
}

// Link is a [[target]] or [[target|label]] outside fenced code, code spans and
// relation lines (a links_to edge).
type Link struct {
	Target string
	Label  string
	Line   int
}

// Problem is something wrong with a note, for memctl check.
type Problem struct {
	Code    string
	Line    int
	Message string
}

// Problem codes (SPEC-0001, "Error handling").
const (
	CodeBadFrontmatter            = "bad_frontmatter"
	CodeBadField                  = "bad_field"
	CodeBadLink                   = "bad_link"
	CodeObservationOutsideSection = "observation_outside_section"
	CodeRelationOutsideSection    = "relation_outside_section"
	CodeSetextHeading             = "setext_heading"
	CodeHTMLBlock                 = "html_block"
	CodeIndentedCode              = "indented_code"
	CodeInvalidUTF8               = "invalid_utf8"
)
