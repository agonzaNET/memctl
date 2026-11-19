package index

import "errors"

// ErrEmptyQuery is returned for a query with no word (SPEC-0002 R13).
var ErrEmptyQuery = errors.New("index: the query has no word")

// Query is a search request (SPEC-0002 R10–R12).
type Query struct {
	Text   string
	Type   string
	Tags   []string
	Folder string
	Limit  int
}

// Hit is one search result.
type Hit struct {
	Path        string
	Title       string
	Type        string
	Score       float64
	Snippet     string
	SnippetLine int
}

// Tokenize lower-cases text and splits it into runs of letters and digits
// (SPEC-0002 R7).
func Tokenize(text string) []string { return nil }

// Search ranks the notes for q with BM25F (SPEC-0002 R7–R14).
func (x *Index) Search(q Query) ([]Hit, error) { return nil, nil }
