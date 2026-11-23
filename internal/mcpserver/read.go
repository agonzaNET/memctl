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
		Description: withScope(descNoteRead, s.opts.Scope),
		InputSchema: schemaNoteRead,
		Annotations: readOnly(),
	}, handle(s, "note_read", s.noteRead))
}

// noteRead implements note_read (SPEC-0004 R14–R18).
func (s *Server) noteRead(in noteReadIn, log *callLog) (*mcp.CallToolResult, error) {
	var res *mcp.CallToolResult
	err := s.opts.Notes.Call(func(idx *index.Index, _ note.Stats) error {
		ref, err := resolveRefFull(idx, in.Ref)
		if err != nil {
			return err
		}
		p := ref.Path
		n, m, _ := idx.Get(p)
		data, err := os.ReadFile(filepath.Join(s.opts.Notes.Root(), filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		content := string(data)
		first := fmt.Sprintf("Read %q at %s.", n.Title, p)
		inRange := func(int) bool { return true }
		linksHead := "Links"
		if in.Section != "" {
			h, part, from, end, err := section(n, content, in.Section)
			if err != nil {
				return err
			}
			content = part
			first = fmt.Sprintf("Read section %q of %q at %s.", h, n.Title, p)
			inRange = func(line int) bool { return line >= from && (end == 0 || line < end) }
			linksHead = "Links in this section"
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
		step, others := resolvedBy(in.Ref, ref)
		item("resolved by", step)
		item("other notes with this name", others)

		links := linkLines(idx, n, inRange)
		var body strings.Builder
		body.WriteString(meta.String())
		body.WriteString("\n")
		body.WriteString(fenced(content))
		body.WriteString("\n\n")
		if len(links) == 0 {
			body.WriteString(linksHead + ": none.")
		} else {
			body.WriteString(linksHead + ":\n" + strings.Join(links, "\n"))
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

// linkLines renders the relations and links of a note on the lines that
// keep says to keep, in file order, with their resolution (SPEC-0004 R15,
// R16).
func linkLines(idx *index.Index, n parse.Note, keep func(line int) bool) []string {
	var items []lineItem
	for _, r := range n.Relations {
		if !keep(r.Line) {
			continue
		}
		items = append(items, lineItem{r.Line, fmt.Sprintf("- %s → %s", r.Type, describe(r.Target, idx.Resolve(r.Target)))})
	}
	for _, l := range n.Links {
		if !keep(l.Line) {
			continue
		}
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

// section returns the text of the first heading that matches name, the
// lines from it to the next heading of the same or a higher level, and the
// section's first line and the line after it (0 at the end of the file)
// (SPEC-0004 R16).
func section(n parse.Note, content, name string) (string, string, int, int, error) {
	h, end, ok := parse.FindSection(n, name)
	if !ok {
		var names []string
		for _, h := range n.Headings {
			names = append(names, h.Text)
		}
		return "", "", 0, 0, noHeading(name, names, "read it without section", "pass one of them as section, or read the note without section")
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
	return h.Text, strings.Join(lines[from:to], ""), h.Line, end, nil
}

// noHeading is the section_not_found error for a missing section: it lists
// up to 20 of the note's headings with the advice in with, or gives the
// advice in without when the note has none.
func noHeading(name string, headings []string, without, with string) error {
	var names []string
	for i, h := range headings {
		if i == 20 {
			names = append(names, fmt.Sprintf("and %d more", len(headings)-20))
			break
		}
		names = append(names, h)
	}
	if len(names) == 0 {
		return errorf(CodeSectionNotFound, "The note has no heading %q, and no headings at all; %s.", name, without)
	}
	return errorf(CodeSectionNotFound, "The note has no heading %q; its headings are: %s; %s.", name, strings.Join(names, ", "), with)
}
