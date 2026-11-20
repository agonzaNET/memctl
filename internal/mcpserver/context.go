package mcpserver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type contextIn struct {
	Ref      string   `json:"ref"`
	Depth    int      `json:"depth,omitempty"`
	RelTypes []string `json:"rel_types,omitempty"`
	Edges    string   `json:"edges,omitempty"`
}

type listIn struct {
	Folder string `json:"folder,omitempty"`
	Type   string `json:"type,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

// MaxListEntries caps a list result (SPEC-0004 R24).
const MaxListEntries = 500

func (s *Server) registerGraph() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "context",
		Description: descContext,
		InputSchema: schemaContext,
		Annotations: readOnly(),
	}, handle(s, "context", s.context))
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "list",
		Description: descList,
		InputSchema: schemaList,
		Annotations: readOnly(),
	}, handle(s, "list", s.list))
}

// context implements the context tool (SPEC-0004 R22, R23, R31).
func (s *Server) context(in contextIn, log *callLog) (*mcp.CallToolResult, error) {
	depth := in.Depth
	if depth == 0 {
		depth = 1
	}
	log.set("depth", depth)
	var res *mcp.CallToolResult
	err := s.opts.Notes.Call(func(idx *index.Index, _ note.Stats) error {
		ref, err := resolveRefFull(idx, in.Ref)
		if err != nil {
			return err
		}
		p := ref.Path
		scope, heading := index.EdgesAll, "Edges:"
		if in.Edges == "start" {
			scope, heading = index.EdgesStart, "Edges of the start note:"
		}
		g, err := idx.Walk(p, depth, in.RelTypes, scope)
		if errors.Is(err, index.ErrBadDepth) {
			return errorf(CodeInvalidArgument, "The depth must be 1, 2 or 3; fix it and call again.")
		}
		if err != nil {
			return err
		}
		log.set("notes", len(g.Nodes))
		log.set("edges", len(g.Edges))

		var b strings.Builder
		step, others := resolvedBy(in.Ref, ref)
		if step != "" {
			fmt.Fprintf(&b, "Resolved by %s.\n", step)
		}
		if others != "" {
			fmt.Fprintf(&b, "Other notes with this name: %s.\n", others)
		}
		if step != "" || others != "" {
			b.WriteString("\n")
		}
		b.WriteString("Notes:\n")
		for _, n := range g.Nodes {
			fmt.Fprintf(&b, "- %d · %s — %s", n.Distance, n.Path, n.Title)
			if scope == index.EdgesStart && n.Via != "" {
				fmt.Fprintf(&b, " (via %s)", n.Via)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n" + heading)
		if len(g.Edges) == 0 {
			b.WriteString(" none.")
		}
		for _, e := range g.Edges {
			fmt.Fprintf(&b, "\n- %s —%s→ %s", e.Source, e.Type, describe(e.Target, e.Resolution))
		}
		if g.Truncated {
			fmt.Fprintf(&b, "\n\nThe walk stopped at %d notes; pass a smaller depth or rel_types to narrow it.", index.MaxWalkNotes)
		}
		start, _, _ := idx.Get(p)
		found := len(g.Nodes) - 1
		count := "no notes"
		if found > 0 {
			count = fmt.Sprintf("%d %s", found, plural(found, "note", "notes"))
		}
		res = success(fmt.Sprintf("Found %s within %d %s of %q, plus the note itself.", count, depth, plural(depth, "hop", "hops"), start.Title), b.String())
		return nil
	})
	return res, err
}

// list implements the list tool (SPEC-0004 R24, R25, R30).
func (s *Server) list(in listIn, log *callLog) (*mcp.CallToolResult, error) {
	folder, err := normFolder(in.Folder)
	if err != nil {
		return nil, err
	}
	var res *mcp.CallToolResult
	err = s.opts.Notes.Call(func(idx *index.Index, _ note.Stats) error {
		entries := idx.List(index.Filter{Folder: folder, Type: in.Type, Tag: in.Tag})
		log.set("results", len(entries))
		if len(entries) == 0 {
			res = success("Found no notes that match.", "")
			return nil
		}
		shown := entries
		if len(shown) > MaxListEntries {
			shown = shown[:MaxListEntries]
		}
		var b strings.Builder
		for _, e := range shown {
			fmt.Fprintf(&b, "- %s — %s", e.Path, e.Title)
			if e.Type != "" {
				fmt.Fprintf(&b, " (%s)", e.Type)
			}
			if len(e.Tags) > 0 {
				fmt.Fprintf(&b, " · tags %s", strings.Join(e.Tags, ", "))
			}
			switch {
			case !e.Updated.IsZero():
				fmt.Fprintf(&b, " · updated %s", e.Updated.Format("2006-01-02"))
			case !e.ModTime.IsZero():
				fmt.Fprintf(&b, " · file changed %s", e.ModTime.Local().Format("2006-01-02"))
			}
			b.WriteString("\n")
		}
		first := fmt.Sprintf("Listed %d %s.", len(entries), plural(len(entries), "note", "notes"))
		if len(entries) > MaxListEntries {
			first = fmt.Sprintf("Listed %d of %d notes.", MaxListEntries, len(entries))
			fmt.Fprintf(&b, "\nThe list shows the first %d of %d notes; pass folder, type or tag to narrow it.", MaxListEntries, len(entries))
		}
		res = success(first, b.String())
		return nil
	})
	return res, err
}
