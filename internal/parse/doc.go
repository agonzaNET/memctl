// Package parse turns the bytes of one Markdown note into its structure:
// frontmatter fields, title, headings, observations, relations, links, and the
// problems that memctl check reports (SPEC-0001).
//
// Parse is a pure function: it reads no files, uses no clock and does not
// log. It never fails; everything wrong with a note is a Problem. This package
// is the only importer of gopkg.in/yaml.v3 (ADR-0010).
package parse
