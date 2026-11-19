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
	Ref     string `json:"ref"`
	Section string `json:"section,omitempty"`
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
		first := fmt.Sprintf("Read %q at %s.", n.Title, p)
		if in.Section != "" {
			h, part, err := section(n, content, in.Section)
			if err != nil {
				return err
			}
			content = part
			first = fmt.Sprintf("Read section %q of %q at %s.", h, n.Title, p)
		}

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

		log.set("bytes", len(content))
		log.set("links", len(links))
		res = success(first, body.String())
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

// section returns the text of the first heading that matches name and the
// lines from it to the next heading of the same or a higher level
// (SPEC-0004 R16).
func section(n parse.Note, content, name string) (string, string, error) {
	want := strings.ToLower(strings.TrimSpace(name))
	for i, h := range n.Headings {
		if strings.ToLower(strings.TrimSpace(h.Text)) != want {
			continue
		}
		end := -1
		for _, next := range n.Headings[i+1:] {
			if next.Level <= h.Level {
				end = next.Line
				break
			}
		}
		lines := strings.SplitAfter(content, "\n")
		from := h.Line - 1
		to := len(lines)
		if end > 0 {
			to = end - 1
		}
		if from >= len(lines) {
			from = len(lines)
		}
		if to < from {
			to = from
		}
		return h.Text, strings.Join(lines[from:to], ""), nil
	}
	var names []string
	for i, h := range n.Headings {
		if i == 20 {
			names = append(names, fmt.Sprintf("and %d more", len(n.Headings)-20))
			break
		}
		names = append(names, h.Text)
	}
	if len(names) == 0 {
		return "", "", errorf(CodeNotFound, "The note has no heading %q, and no headings at all; read it without section.", name)
	}
	return "", "", errorf(CodeNotFound, "The note has no heading %q; its headings are: %s.", name, strings.Join(names, ", "))
}
