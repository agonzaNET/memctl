package mcpserver

import (
	"fmt"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type noteEditIn struct {
	Ref         string  `json:"ref"`
	ExpectedSHA string  `json:"expected_sha"`
	Op          string  `json:"op"`
	Text        *string `json:"text,omitempty"`
	Section     string  `json:"section,omitempty"`
	Find        string  `json:"find,omitempty"`
	Replace     *string `json:"replace,omitempty"`
}

func (s *Server) registerEdit() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "note_edit",
		Description: descNoteEdit,
		InputSchema: schemaNoteEdit,
		Annotations: writeAnnotations(false, false),
	}, handle(s, "note_edit", s.noteEdit))
}

// noteEdit implements note_edit (SPEC-0007 R15–R19, R27).
func (s *Server) noteEdit(in noteEditIn, log *callLog) (*mcp.CallToolResult, error) {
	log.set("bytes", 0)
	log.set("paths", 0)
	log.set("unresolved", 0)
	if in.Text == nil && (in.Op == "append" || in.Op == "replace_section") {
		return nil, errorf(CodeInvalidArgument, "The text argument is required for %s; pass the text and call again.", in.Op)
	}
	if in.Replace == nil && in.Op == "find_replace" {
		return nil, errorf(CodeInvalidArgument, "The replace argument is required for find_replace; pass it, empty to delete the find text, and call again.")
	}
	op := note.EditOp{ExpectedSHA: in.ExpectedSHA, Op: in.Op, Section: in.Section, Find: in.Find}
	if in.Text != nil {
		op.Text = *in.Text
	}
	if in.Replace != nil {
		op.Replace = *in.Replace
	}
	e, err := s.opts.Notes.Edit(func(idx *index.Index) (string, error) { return resolveRef(idx, in.Ref) }, op)
	if err != nil {
		return nil, writeError(err)
	}
	body := "sha256: " + e.SHA256
	if e.Unchanged {
		return success(fmt.Sprintf("%q at %s is unchanged.", e.Title, e.Path), body), nil
	}
	log.set("bytes", e.Bytes)
	log.set("paths", 1)
	log.set("unresolved", len(e.Unresolved))
	body += unresolvedList(e.Unresolved)
	return success(fmt.Sprintf("Edited %q at %s.", e.Title, e.Path), body), nil
}
