package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/agonzaNET/memctl/internal/parse"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type noteReadIn struct {
	Ref string `json:"ref"`
}

func (s *Server) registerRead() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "note_read",
		Description: descNoteRead,
		InputSchema: schemaNoteRead,
		Annotations: readOnly(),
	}, handle(s, "note_read", s.noteRead))
}

// noteRead implements note_read (SPEC-0004 R14–R18).
func (s *Server) noteRead(in noteReadIn, log *callLog) (*mcp.CallToolResult, error) {
	var res *mcp.CallToolResult
	err := s.opts.Notes.Call(func(idx *index.Index, _ note.Stats) error {
		p, err := resolveRef(idx, in.Ref)
		if err != nil {
			return err
		}
		n, m, _ := idx.Get(p)
		data, err := os.ReadFile(filepath.Join(s.opts.Notes.Root(), filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		content := string(data)

		var meta strings.Builder
		item := func(k, v string) {
			if v != "" {
				fmt.Fprintf(&meta, "- %s: %s\n", k, v)
			}
		}
		item("path", p)
		item("title", n.Title)
		item("type", n.Type)
		item("tags", strings.Join(n.Tags, ", "))
		item("aliases", strings.Join(n.Aliases, ", "))
		item("created", formatTime(n.Created))
		item("updated", formatTime(n.Updated))
		item("sha256", m.SHA256)

		links := linkLines(idx, n)
		var body strings.Builder
		body.WriteString(meta.String())
		body.WriteString("\n")
		body.WriteString(fenced(content))
		body.WriteString("\n\n")
		if len(links) == 0 {
			body.WriteString("Links: none.")
		} else {
			body.WriteString("Links:\n" + strings.Join(links, "\n"))
		}

		log.set("bytes", len(data))
		log.set("links", len(links))
		res = success(fmt.Sprintf("Read %q at %s.", n.Title, p), body.String())
		return nil
	})
	return res, err
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// linkLines renders every relation and link of a note, in file order, with
// its resolution (SPEC-0004 R15).
func linkLines(idx *index.Index, n parse.Note) []string {
	var items []lineItem
	for _, r := range n.Relations {
		items = append(items, lineItem{r.Line, fmt.Sprintf("- %s → %s", r.Type, describe(r.Target, idx.Resolve(r.Target)))})
	}
	for _, l := range n.Links {
		items = append(items, lineItem{l.Line, fmt.Sprintf("- links_to → %s", describe(l.Target, idx.Resolve(l.Target)))})
	}
	return sortItems(items)
}

type lineItem struct {
	line int
	text string
}

func sortItems(items []lineItem) []string {
	sort.SliceStable(items, func(i, j int) bool { return items[i].line < items[j].line })
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.text
	}
	return out
}
