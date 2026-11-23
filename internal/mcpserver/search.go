package mcpserver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agonzaNET/memctl/internal/index"
	"github.com/agonzaNET/memctl/internal/note"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchIn struct {
	Query  string   `json:"query"`
	Type   string   `json:"type,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Folder string   `json:"folder,omitempty"`
	Limit  int      `json:"limit,omitempty"`
}

func (s *Server) registerSearch() {
	mcp.AddTool(s.sdk, &mcp.Tool{
		Name:        "search",
		Description: withScope(descSearch, s.opts.Scope),
		InputSchema: schemaSearch,
		Annotations: readOnly(),
	}, handle(s, "search", s.search))
}

// normFolder normalises a folder argument (SPEC-0004 R20): a leading "./"
// and a trailing "/" are removed; "..", "\" and absolute paths are refused.
func normFolder(f string) (string, error) {
	f = strings.TrimSpace(f)
	f = strings.TrimPrefix(f, "./")
	f = strings.TrimRight(f, "/")
	bad := strings.HasPrefix(f, "/") || strings.Contains(f, `\`)
	for _, seg := range strings.Split(f, "/") {
		if seg == ".." {
			bad = true
		}
	}
	if bad {
		return "", errorf(CodeInvalidArgument, "The folder must be a relative folder inside the notes, without .. or backslashes; fix it and call again.")
	}
	return f, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// search implements the search tool (SPEC-0004 R19–R21, R32).
func (s *Server) search(in searchIn, log *callLog) (*mcp.CallToolResult, error) {
	folder, err := normFolder(in.Folder)
	if err != nil {
		return nil, err
	}
	terms := len(uniqWords(in.Query))
	log.set("terms", terms)
	var res *mcp.CallToolResult
	err = s.opts.Notes.Call(func(idx *index.Index, _ note.Stats) error {
		found, err := idx.Search(index.Query{Text: in.Query, Type: in.Type, Tags: in.Tags, Folder: folder, Limit: in.Limit})
		if errors.Is(err, index.ErrEmptyQuery) {
			return errorf(CodeInvalidArgument, "The query has no word to search for; pass at least one word and call again.")
		}
		if err != nil {
			return err
		}
		hits := found.Hits
		log.set("results", len(hits))
		if len(hits) == 0 {
			res = success(fmt.Sprintf("Found no notes for %q.", in.Query), nearHint(idx, in, folder))
			return nil
		}
		var b strings.Builder
		for i, h := range hits {
			typ := ""
			if h.Type != "" {
				typ = " (" + h.Type + ")"
			}
			tags := ""
			if len(h.Tags) > 0 {
				tags = " · tags " + strings.Join(h.Tags, ", ")
			}
			fmt.Fprintf(&b, "%d. **%s** — %s%s%s · score %.2f\n", i+1, h.Title, h.Path, typ, tags, h.Score)
			if h.Snippet != "" {
				fmt.Fprintf(&b, "   > L%d: %s\n", h.SnippetLine, h.Snippet)
			}
		}
		first := fmt.Sprintf("Found %d %s for %q.", len(hits), plural(len(hits), "note", "notes"), in.Query)
		if found.Total > len(hits) {
			first = fmt.Sprintf("Found %d notes for %q; showing the first %d.", found.Total, in.Query, len(hits))
			if len(hits) < index.MaxLimit {
				fmt.Fprintf(&b, "\nMore notes match than are shown; pass a higher limit (up to %d) to see more, or narrow the search with type, tags or folder.", index.MaxLimit)
			} else {
				b.WriteString("\nMore notes match than can be shown; narrow the search with type, tags or folder, or use more specific words.")
			}
		}
		res = success(first, b.String())
		return nil
	})
	return res, err
}

func uniqWords(q string) map[string]bool {
	m := map[string]bool{}
	for _, t := range index.QueryTerms(q) {
		m[t] = true
	}
	return m
}

// maxNearWords caps the words named by the hint of a search without hits.
const maxNearWords = 5

// nearHint returns, for a search without hits, the line that names the words
// in matching notes that are a query word followed by digits, or "" when
// there are none (SPEC-0004 R32).
func nearHint(idx *index.Index, in searchIn, folder string) string {
	found, err := idx.Search(index.Query{Text: in.Query, Type: in.Type, Tags: in.Tags, Folder: folder, Limit: in.Limit, NearNames: true})
	if err != nil || len(found.Near) == 0 {
		return ""
	}
	words := found.Near
	more := ""
	if len(words) > maxNearWords {
		more = fmt.Sprintf(", and %d more", len(words)-maxNearWords)
		words = words[:maxNearWords]
	}
	return "Some notes hold a query word followed by digits: " + strings.Join(words, ", ") + more + "; search for one of these words."
}
