package mcpserver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// unchangedError is a failed file operation of a write tool: internal_error,
// with a message that says the note is unchanged (SPEC-0007, "Error
// handling").
type unchangedError struct{ err error }

func (e unchangedError) Error() string { return e.err.Error() }
func (e unchangedError) Unwrap() error { return e.err }

// unchangedMessage is the sentence of internal_error for a write tool.
const unchangedMessage = "memctl could not write the note, and the note is unchanged; try again, and if it fails again, tell the person."

// writeAnnotations are the hints of a write tool (SPEC-0007 R25).
func writeAnnotations(destructive, idempotent bool) *mcp.ToolAnnotations {
	no := false
	return &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &destructive, IdempotentHint: idempotent, OpenWorldHint: &no}
}

// writeError maps a note service error of a write to a tool error.
func writeError(err error) error {
	var (
		te       *toolError
		arg      *note.ArgError
		path     *note.PathError
		exists   *note.ExistsError
		conflict *note.ConflictError
		section  *note.SectionError
		many     *note.NotUniqueError
	)
	switch {
	case errors.As(err, &te):
		return err // from ref resolution
	case errors.As(err, &conflict):
		return errorf(CodeConflict, "The note changed since you read it, so nothing was changed. Its current sha256 and content follow; decide again on that content and retry with that sha256 as expected_sha.\n\nsha256: %s\n\n%s", conflict.SHA256, fenced(conflict.Content))
	case errors.As(err, &section):
		return noHeading(section.Section, section.Headings, "add the section with append, starting text with the heading line", "pass one of them as section, or add the section with append, starting text with the heading line")
	case errors.Is(err, note.ErrNoMatch):
		return errorf(CodeNoMatch, "The find text does not occur in the note; read the note again and pass text that it contains exactly.")
	case errors.As(err, &many):
		return errorf(CodeNotUnique, "The find text occurs %d times in the note; include more of the surrounding text so that it occurs once, and call again.", many.Count)
	case errors.As(err, &arg):
		return errorf(CodeInvalidArgument, "The %s %s; fix it and call again.", arg.Arg, arg.Reason)
	case errors.As(err, &path):
		return errorf(CodeInvalidArgument, "The folder or path %s cannot hold a note: %s; choose another folder or title and call again.", path.Path, path.Reason)
	case errors.As(err, &exists):
		return errorf(CodeExists, "A note exists at %s; use note_edit to change it, or choose another title or folder.", exists.Path)
	case errors.Is(err, note.ErrFrontmatterInBody):
		return errorf(CodeInvalidArgument, "The body starts with a frontmatter block; pass title, type and tags as arguments and only the note text in body.")
	case errors.Is(err, note.ErrConflict):
		return errorf(CodeConflict, "The file changed while memctl was writing it, and memctl did not overwrite it; read the note with note_read and decide again.")
	}
	return unchangedError{err}
}

// unresolvedList renders a written note's dangling and ambiguous links in
// the format of note_read's Links list, or nothing (SPEC-0007 R29).
func unresolvedList(es []index.Edge) string {
	if len(es) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nUnresolved links:")
	for _, e := range es {
		fmt.Fprintf(&b, "\n- %s → %s", e.Type, describe(e.Target, e.Resolution))
	}
	return b.String()
}

type noteWriteIn struct {
	Title  string   `json:"title"`
	Folder string   `json:"folder,omitempty"`
	Body   string   `json:"body"`
	Type   string   `json:"type,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

func (s *Server) registerWrite() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "note_write",
		Description: descNoteWrite,
		InputSchema: schemaNoteWrite,
		Annotations: writeAnnotations(false, false),
	}, handle(s, "note_write", s.noteWrite))
}

// noteWrite implements note_write (SPEC-0007 R10–R14, R27).
func (s *Server) noteWrite(in noteWriteIn, log *callLog) (*mcp.CallToolResult, error) {
	log.set("bytes", 0)
	log.set("paths", 0)
	log.set("unresolved", 0)
	w, err := s.opts.Notes.Write(note.NewNote{Title: in.Title, Folder: in.Folder, Body: in.Body, Type: in.Type, Tags: in.Tags})
	if err != nil {
		return nil, writeError(err)
	}
	log.set("bytes", w.Bytes)
	log.set("paths", 1)
	body := "sha256: " + w.SHA256
	if len(w.SameTitle) > 0 {
		body += fmt.Sprintf("\n\nOther notes have the same title: %s. Link to this note by its path, [[%s]], so the link stays unique.",
			strings.Join(w.SameTitle, ", "), strings.TrimSuffix(w.Path, ".md"))
	}
	log.set("unresolved", len(w.Unresolved))
	body += unresolvedList(w.Unresolved)
	return success(fmt.Sprintf("Created %q at %s.", w.Title, w.Path), body), nil
}
