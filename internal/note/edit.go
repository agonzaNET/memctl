package note

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

// MaxFind bounds find_replace's find text (SPEC-0007, "Inputs").
const MaxFind = 64 << 10

// Resolver turns a tool ref into a note path under the service mutex, as a
// read does (SPEC-0004 R14). Its errors are returned as they are.
type Resolver func(idx *index.Index) (string, error)

// EditOp is the input of Edit (note_edit).
type EditOp struct {
	ExpectedSHA string
	Op          string // append, replace_section or find_replace
	Text        string // append, replace_section
	Section     string // replace_section
	Find        string // find_replace
	Replace     string // find_replace
}

// Edited is the result of a successful Edit.
type Edited struct {
	Path, Title, SHA256 string
	Bytes               int
	Unchanged           bool // the edit left the file as it was; nothing was written
}

// ConflictError is an expected_sha that is not the note's current hash; it
// carries the current hash and content so the agent can retry without
// another read (SPEC-0007 R15). It wraps ErrConflict.
type ConflictError struct{ SHA256, Content string }

func (e *ConflictError) Error() string {
	return "note: the note changed since it was read; its sha256 is now " + e.SHA256
}
func (e *ConflictError) Unwrap() error { return ErrConflict }

// SectionError is a replace_section heading the note does not have; it
// lists the note's headings (SPEC-0007 R17).
type SectionError struct {
	Section  string
	Headings []string
}

func (e *SectionError) Error() string { return "note: no heading " + strconv.Quote(e.Section) }

// NotUniqueError is a find text that occurs Count times (SPEC-0007 R18).
type NotUniqueError struct{ Count int }

func (e *NotUniqueError) Error() string {
	return "note: the find text occurs " + strconv.Itoa(e.Count) + " times"
}

// ErrNoMatch is a find text that does not occur (SPEC-0007 R18).
var ErrNoMatch = errors.New("note: the find text does not occur")

// Edit changes a note with one operation (SPEC-0007 R15–R19), as one tool
// write.
func (s *Service) Edit(resolve Resolver, op EditOp) (Edited, error) {
	if err := checkEditOp(op); err != nil {
		return Edited{}, err
	}
	var out Edited
	err := s.mutate(func(idx *index.Index) (Change, error) {
		rel, err := resolve(idx)
		if err != nil {
			return Change{}, err
		}
		abs, err := s.SafePath(rel)
		if err != nil {
			return Change{}, err
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return Change{}, err
		}
		cur := hashOf(data)
		if cur != op.ExpectedSHA {
			return Change{}, &ConflictError{SHA256: cur, Content: string(data)}
		}
		n, _, _ := idx.Get(rel)
		edited, err := applyEdit(n, string(data), op)
		if err != nil {
			return Change{}, err
		}
		if edited == string(data) {
			out = Edited{Path: rel, Title: n.Title, SHA256: cur, Bytes: len(data), Unchanged: true}
			return Change{}, nil
		}
		if editableFrontmatter(n) {
			if b, err := parse.EditFrontmatter([]byte(edited), parse.FrontmatterSet{Key: "updated", Value: s.stamp()}); err == nil {
				edited = string(b)
			}
		}
		if len(edited) > MaxNoteSize {
			return Change{}, &ArgError{Arg: "text", Reason: "would make the note larger than 4 MiB"}
		}
		m, err := s.putFile(rel, []byte(edited), cur)
		if err != nil {
			return Change{}, err
		}
		after, _, _ := idx.Get(rel)
		out = Edited{Path: rel, Title: after.Title, SHA256: m.SHA256, Bytes: len(edited)}
		return Change{Tool: "note_edit", Title: after.Title, Paths: []string{rel}}, nil
	})
	if err != nil {
		return Edited{}, err
	}
	return out, nil
}

// checkEditOp applies the argument rules of note_edit.
func checkEditOp(op EditOp) error {
	if len(op.ExpectedSHA) != 64 || strings.Trim(op.ExpectedSHA, "0123456789abcdef") != "" {
		return &ArgError{Arg: "expected_sha", Reason: "must be the 64 lower-case hex characters of a sha256 from note_read"}
	}
	switch op.Op {
	case "append":
	case "replace_section":
		if strings.TrimSpace(op.Section) == "" {
			return &ArgError{Arg: "section", Reason: "is required for replace_section"}
		}
	case "find_replace":
		if op.Find == "" || len(op.Find) > MaxFind {
			return &ArgError{Arg: "find", Reason: "is required for find_replace and must be at most 64 KiB"}
		}
		if len(op.Replace) > MaxBodySize {
			return &ArgError{Arg: "replace", Reason: "must be at most 1 MiB"}
		}
	default:
		return &ArgError{Arg: "op", Reason: "must be append, replace_section or find_replace"}
	}
	if len(op.Text) > MaxBodySize {
		return &ArgError{Arg: "text", Reason: "must be at most 1 MiB"}
	}
	return nil
}

// applyEdit returns the file content after op, before updated is set.
func applyEdit(n parse.Note, content string, op EditOp) (string, error) {
	switch op.Op {
	case "append":
		// After a newline if the file does not end in one; the file ends in
		// exactly one newline (SPEC-0007 R16).
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return strings.TrimRight(content+op.Text, "\r\n") + "\n", nil
	case "replace_section":
		return replaceSection(n, content, op.Section, op.Text)
	default:
		switch c := strings.Count(content, op.Find); c {
		case 0:
			return "", ErrNoMatch
		case 1:
			return strings.Replace(content, op.Find, op.Replace, 1), nil
		default:
			return "", &NotUniqueError{Count: c}
		}
	}
}

// replaceSection keeps the heading line of the section and replaces the
// lines after it, up to the next heading of the same or a higher level, with
// text; the text ends in one newline and is followed by a blank line when a
// heading follows (SPEC-0007 R17).
func replaceSection(n parse.Note, content, name, text string) (string, error) {
	h, end, ok := parse.FindSection(n, name)
	if !ok {
		e := &SectionError{Section: name}
		for _, h := range n.Headings {
			e.Headings = append(e.Headings, h.Text)
		}
		return "", e
	}
	lines := strings.SplitAfter(content, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	head := strings.Join(lines[:min(h.Line, len(lines))], "")
	if !strings.HasSuffix(head, "\n") {
		head += "\n"
	}
	tail := ""
	if end > 0 && end-1 < len(lines) {
		tail = strings.Join(lines[end-1:], "")
	}
	body := strings.TrimRight(text, "\r\n")
	if body != "" {
		body += "\n"
	}
	if tail != "" {
		body += "\n"
	}
	return head + body + tail, nil
}

// editableFrontmatter reports whether note_edit may set updated: the note
// has a closed frontmatter block without bad_frontmatter (SPEC-0007 R9).
func editableFrontmatter(n parse.Note) bool {
	if !n.HasFrontmatter {
		return false
	}
	for _, p := range n.Problems {
		if p.Code == parse.CodeBadFrontmatter {
			return false
		}
	}
	return true
}

func hashOf(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
