package note

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/parse"
)

// Argument limits of the write tools (SPEC-0007, "Inputs"). The input
// schemas in internal/mcpserver state the same limits.
const (
	MaxTitle    = 200     // characters
	MaxTypeTag  = 64      // characters
	MaxTags     = 20      // tags per note
	MaxBodySize = 1 << 20 // bytes
)

// NewNote is the input of Write (note_write).
type NewNote struct {
	Title, Folder, Body, Type string
	Tags                      []string
}

// Written is the result of a successful Write.
type Written struct {
	Path, Title, SHA256 string
	Bytes               int          // size of the written file
	SameTitle           []string     // other notes with the same title (R14)
	Unresolved          []index.Edge // dangling and ambiguous links of the note (R29)
}

// ExistsError reports a note at the target path, or at a path equal to it
// apart from letter case (SPEC-0007 R11, R21).
type ExistsError struct{ Path string }

func (e *ExistsError) Error() string { return "note: a note exists at " + e.Path }

// ArgError is a tool argument that breaks a rule of SPEC-0007 "Inputs".
type ArgError struct{ Arg, Reason string }

func (e *ArgError) Error() string { return "note: " + e.Arg + " " + e.Reason }

// ErrFrontmatterInBody is returned for a note_write body that starts with a
// frontmatter delimiter line (SPEC-0007 R12).
var ErrFrontmatterInBody = errors.New("note: the body starts with frontmatter")

// Write creates a note (SPEC-0007 R10–R14) as one tool write.
func (s *Service) Write(n NewNote) (Written, error) {
	if err := checkNewNote(n); err != nil {
		return Written{}, err
	}
	name, err := FileName(n.Title)
	if err != nil {
		return Written{}, err
	}
	rel := name
	if folder := normFolder(n.Folder); folder != "" {
		rel = folder + "/" + name
	}

	var w Written
	err = s.mutate(func(idx *index.Index) (Change, error) {
		abs, err := s.targetPath(rel)
		if err != nil {
			return Change{}, err
		}
		if p := pathTaken(idx, rel, abs); p != "" {
			return Change{}, &ExistsError{Path: p}
		}
		data := newNoteFile(n, s.stamp())
		m, err := s.putFile(rel, data, "")
		if err != nil {
			return Change{}, err
		}
		w = Written{Path: rel, Title: n.Title, SHA256: m.SHA256, Bytes: len(data), Unresolved: idx.Unresolved(rel)}
		for _, p := range idx.Titled(n.Title) {
			if p != rel {
				w.SameTitle = append(w.SameTitle, p)
			}
		}
		return Change{Tool: "note_write", Title: n.Title, Paths: []string{rel}}, nil
	})
	if err != nil {
		return Written{}, err
	}
	return w, nil
}

// checkNewNote applies the argument rules of note_write.
func checkNewNote(n NewNote) error {
	if err := checkLine("title", n.Title, MaxTitle); err != nil {
		return err
	}
	if n.Type != "" {
		if err := checkLine("type", n.Type, MaxTypeTag); err != nil {
			return err
		}
	}
	if len(n.Tags) > MaxTags {
		return &ArgError{Arg: "tags", Reason: "may hold at most 20 tags"}
	}
	for _, t := range n.Tags {
		if err := checkLine("tags", t, MaxTypeTag); err != nil {
			return err
		}
	}
	if len(n.Body) > MaxBodySize {
		return &ArgError{Arg: "body", Reason: "must be at most 1 MiB"}
	}
	if startsWithFrontmatter(n.Body) {
		return ErrFrontmatterInBody
	}
	return nil
}

// checkLine checks a one-line argument of 1 to max characters.
func checkLine(arg, v string, max int) error {
	switch n := utf8.RuneCountInString(v); {
	case n == 0:
		return &ArgError{Arg: arg, Reason: "must not be empty"}
	case n > max:
		return &ArgError{Arg: arg, Reason: "must be at most " + strconv.Itoa(max) + " characters"}
	case strings.ContainsAny(v, "\r\n"):
		return &ArgError{Arg: arg, Reason: "must not contain a line break"}
	}
	return nil
}

// startsWithFrontmatter reports whether the first line of body, after an
// optional byte order mark, is a frontmatter delimiter.
func startsWithFrontmatter(body string) bool {
	body = strings.TrimPrefix(body, "\ufeff")
	first, _, _ := strings.Cut(body, "\n")
	return strings.TrimRight(first, " \t\r") == "---"
}

// normFolder trims a folder argument: spaces, a leading "./" and trailing
// slashes. Its safety is checked with the whole path (SPEC-0007 R28).
func normFolder(f string) string {
	f = strings.TrimSpace(f)
	f = strings.TrimPrefix(f, "./")
	return strings.TrimRight(f, "/")
}

// pathTaken returns the path of a note at rel, or at a path equal to it
// apart from letter case, or rel when any file is there (SPEC-0007 R11).
func pathTaken(idx *index.Index, rel, abs string) string {
	for _, p := range idx.Paths() {
		if strings.EqualFold(p, rel) {
			return p
		}
	}
	if _, err := os.Lstat(abs); err == nil {
		return rel
	}
	return ""
}

// newNoteFile is the content of a new note: frontmatter with title, type,
// tags, created and updated in that order, a blank line and the body, ending
// in exactly one newline (SPEC-0007 R13).
func newNoteFile(n NewNote, now string) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: " + parse.FormatScalar(n.Title) + "\n")
	if n.Type != "" {
		b.WriteString("type: " + parse.FormatScalar(n.Type) + "\n")
	}
	if len(n.Tags) > 0 {
		b.WriteString("tags: " + parse.FormatList(n.Tags) + "\n")
	}
	b.WriteString("created: " + now + "\n")
	b.WriteString("updated: " + now + "\n")
	b.WriteString("---\n")
	if body := strings.TrimRight(n.Body, "\r\n"); body != "" {
		b.WriteString("\n" + body + "\n")
	}
	return []byte(b.String())
}
