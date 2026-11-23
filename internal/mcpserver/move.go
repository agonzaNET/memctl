package mcpserver

import (
	"fmt"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type noteMoveIn struct {
	Ref      string `json:"ref"`
	NewPath  string `json:"new_path,omitempty"`
	NewTitle string `json:"new_title,omitempty"`
}

func (s *Server) registerMove() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "note_move",
		Description: withScope(descNoteMove, s.opts.Scope),
		InputSchema: schemaNoteMove,
		Annotations: writeAnnotations(false, false),
	}, handle(s, "note_move", s.noteMove))
}

// noteMove implements note_move (SPEC-0007 R20–R23, R27).
func (s *Server) noteMove(in noteMoveIn, log *callLog) (*mcp.CallToolResult, error) {
	log.set("bytes", 0)
	log.set("paths", 0)
	log.set("linkers", 0)
	mv, err := s.opts.Notes.Move(func(idx *index.Index) (string, error) { return resolveRef(idx, in.Ref) },
		note.MoveOp{NewPath: in.NewPath, NewTitle: in.NewTitle})
	if err != nil {
		return nil, writeError(err)
	}
	if mv.Same {
		return success(fmt.Sprintf("%q is already at %s.", mv.Title, mv.Path), ""), nil
	}
	paths := 2
	if mv.OldPath == mv.Path {
		paths = 1
	}
	log.set("bytes", mv.Bytes)
	log.set("paths", paths)
	log.set("linkers", len(mv.Linkers))
	body := "sha256: " + mv.SHA256 + "\n\n" + linkersNotice(mv.Linkers,
		"No other note links to it.",
		"%s Those links keep working, because the note's old names are kept in its aliases; memctl check lists them as stale, so they can be updated to the new path.")
	return success(fmt.Sprintf("Moved %q from %s to %s.", mv.Title, mv.OldPath, mv.Path), body), nil
}

// MaxNamedLinkers caps the linkers a move or delete names (SPEC-0007 R23,
// R24).
const MaxNamedLinkers = 10

// linkersNotice says which notes link to a moved or deleted note: their
// number and up to MaxNamedLinkers of their paths.
func linkersNotice(paths []string, none, some string) string {
	n := len(paths)
	if n == 0 {
		return none
	}
	named := paths
	more := ""
	if n > MaxNamedLinkers {
		named = paths[:MaxNamedLinkers]
		more = fmt.Sprintf(" and %d more", n-MaxNamedLinkers)
	}
	who := fmt.Sprintf("%s to it: %s%s.", plural(n, "1 note links", fmt.Sprintf("%d notes link", n)), strings.Join(named, ", "), more)
	return fmt.Sprintf(some, who)
}
