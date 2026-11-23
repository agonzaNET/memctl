package mcpserver

import (
	"fmt"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type noteDeleteIn struct {
	Ref string `json:"ref"`
}

func (s *Server) registerDelete() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "note_delete",
		Description: withScope(descNoteDelete, s.opts.Scope),
		InputSchema: schemaNoteDelete,
		Annotations: writeAnnotations(true, true),
	}, handle(s, "note_delete", s.noteDelete))
}

// noteDelete implements note_delete (SPEC-0007 R24, R27).
func (s *Server) noteDelete(in noteDeleteIn, log *callLog) (*mcp.CallToolResult, error) {
	log.set("bytes", 0)
	log.set("paths", 0)
	log.set("linkers", 0)
	d, err := s.opts.Notes.Delete(func(idx *index.Index) (string, error) { return resolveRef(idx, in.Ref) })
	if err != nil {
		return nil, writeError(err)
	}
	log.set("paths", 1)
	log.set("linkers", len(d.Dangling))
	body := linkersNotice(d.Dangling,
		"No other note linked to it.",
		"%s Those links now point to no note, and memctl check lists them as dangling.") +
		" git history in the notes folder keeps the file, so a person can restore it."
	return success(fmt.Sprintf("Deleted %q at %s.", d.Title, d.Path), body), nil
}
