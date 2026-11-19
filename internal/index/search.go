package index

import (
	"errors"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/agonzaNET/memctl/internal/parse"
)

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

// Search limits (SPEC-0002 R12) and BM25F parameters (R9).
const (
	DefaultLimit = 10
	MaxLimit     = 50
	k1           = 1.2
	bParam       = 0.75
	snippetRunes = 200
)

// The five indexed fields and their weights (SPEC-0002 R8, R9).
const (
	fTitle = iota
	fAliases
	fTags
	fObservations
	fBody
	numFields
)

var fieldWeight = [numFields]float64{3, 2, 2, 1.5, 1}

// docTerms is the per-note part of the inverted index.
type docTerms struct {
	lens [numFields]int
	tf   map[string]*[numFields]int
}

// Tokenize lower-cases text and splits it into runs of letters and digits
// (SPEC-0002 R7).
func Tokenize(text string) []string {
	f := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(f) == 0 {
		return nil
	}
	return f
}

// observationLines returns the set of file lines that are observations.
func observationLines(n parse.Note) map[int]bool {
	m := map[int]bool{}
	for _, o := range n.Observations {
		m[o.Line] = true
	}
	return m
}

// bodyLines calls fn for every body line that is not an observation, with
// its file line number.
func bodyLines(n parse.Note, fn func(line int, text string)) {
	obs := observationLines(n)
	for i, l := range strings.Split(n.Body, "\n") {
		lineNo := n.BodyLine + i
		if obs[lineNo] {
			continue
		}
		fn(lineNo, strings.TrimSuffix(l, "\r"))
	}
}

// fieldTexts returns the text of each indexed field of a note (R8).
func fieldTexts(n parse.Note) [numFields]string {
	var out [numFields]string
	out[fTitle] = n.Title
	out[fAliases] = strings.Join(n.Aliases, " ")
	out[fTags] = strings.Join(n.Tags, " ")
	obs := make([]string, len(n.Observations))
	for i, o := range n.Observations {
		obs[i] = o.Text
	}
	out[fObservations] = strings.Join(obs, " ")
	var body strings.Builder
	bodyLines(n, func(_ int, text string) {
		body.WriteString(text)
		body.WriteByte('\n')
	})
	out[fBody] = body.String()
	return out
}

func newDocTerms(n parse.Note) *docTerms {
	d := &docTerms{tf: map[string]*[numFields]int{}}
	for f, text := range fieldTexts(n) {
		toks := Tokenize(text)
		d.lens[f] = len(toks)
		for _, tok := range toks {
			c := d.tf[tok]
			if c == nil {
				c = &[numFields]int{}
				d.tf[tok] = c
			}
			c[f]++
		}
	}
	return d
}

// addTerms and removeTerms keep the postings and field totals in step with
// Put and Remove.
func (x *Index) addTerms(p string, d *docTerms) {
	for tok := range d.tf {
		set := x.postings[tok]
		if set == nil {
			set = map[string]struct{}{}
			x.postings[tok] = set
		}
		set[p] = struct{}{}
	}
	for f := range d.lens {
		x.totalLen[f] += d.lens[f]
	}
}

func (x *Index) removeTerms(p string, d *docTerms) {
	for tok := range d.tf {
		if set := x.postings[tok]; set != nil {
			delete(set, p)
			if len(set) == 0 {
				delete(x.postings, tok)
			}
		}
	}
	for f := range d.lens {
		x.totalLen[f] -= d.lens[f]
	}
}

// matches reports whether a note passes the filters of q (R11).
func matches(p string, n parse.Note, q Query) bool {
	if q.Type != "" && !strings.EqualFold(strings.TrimSpace(q.Type), n.Type) {
		return false
	}
	for _, want := range q.Tags {
		found := false
		for _, tag := range n.Tags {
			if strings.EqualFold(strings.TrimSpace(want), tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return inFolder(p, q.Folder)
}

// inFolder reports whether p is under folder, at a "/" boundary; an empty
// folder matches every path.
func inFolder(p, folder string) bool {
	folder = strings.Trim(folder, "/")
	return folder == "" || strings.HasPrefix(p, folder+"/")
}

// Search ranks the notes for q with BM25F (SPEC-0002 R7–R14).
func (x *Index) Search(q Query) ([]Hit, error) {
	terms := uniq(Tokenize(q.Text))
	if len(terms) == 0 {
		return nil, ErrEmptyQuery
	}
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	n := float64(len(x.notes))
	var avg [numFields]float64
	for f := range avg {
		if n > 0 {
			avg[f] = float64(x.totalLen[f]) / n
		}
	}
	scores := map[string]float64{}
	for _, term := range terms {
		docs := x.postings[term]
		df := float64(len(docs))
		if df == 0 {
			continue
		}
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		for p := range docs {
			d := x.terms[p]
			c := d.tf[term]
			wtf := 0.0
			for f := 0; f < numFields; f++ {
				if c[f] == 0 {
					continue
				}
				norm := 1 - bParam + bParam*float64(d.lens[f])/avg[f]
				wtf += fieldWeight[f] * float64(c[f]) / norm
			}
			scores[p] += idf * wtf / (k1 + wtf)
		}
	}
	var hits []Hit
	for p, s := range scores {
		e := x.notes[p]
		if !matches(p, e.note, q) {
			continue
		}
		hits = append(hits, Hit{Path: p, Title: e.note.Title, Type: e.note.Type, Score: s})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Path < hits[j].Path
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	termSet := map[string]bool{}
	for _, t := range terms {
		termSet[t] = true
	}
	for i := range hits {
		hits[i].Snippet, hits[i].SnippetLine = snippet(x.notes[hits[i].Path].note, termSet)
	}
	return hits, nil
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// countTerms returns how many distinct query terms text contains.
func countTerms(text string, terms map[string]bool) int {
	seen := map[string]bool{}
	for _, t := range Tokenize(text) {
		if terms[t] {
			seen[t] = true
		}
	}
	return len(seen)
}

// snippet picks the observation, else the body line, with the most distinct
// query terms, else the first non-empty body line (SPEC-0002 R14).
func snippet(n parse.Note, terms map[string]bool) (string, int) {
	lines := strings.Split(n.Body, "\n")
	lineText := func(lineNo int) string {
		i := lineNo - n.BodyLine
		if i < 0 || i >= len(lines) {
			return ""
		}
		return strings.TrimSuffix(lines[i], "\r")
	}
	best, bestLine := 0, 0
	for _, o := range n.Observations {
		if c := countTerms(o.Text, terms); c > best {
			best, bestLine = c, o.Line
		}
	}
	if best == 0 {
		bodyLines(n, func(line int, text string) {
			if c := countTerms(text, terms); c > best {
				best, bestLine = c, line
			}
		})
	}
	if best == 0 {
		bodyLines(n, func(line int, text string) {
			if bestLine == 0 && strings.TrimSpace(text) != "" {
				bestLine = line
			}
		})
	}
	if bestLine == 0 {
		return "", 0
	}
	return cut(strings.TrimSpace(lineText(bestLine))), bestLine
}

func cut(s string) string {
	if utf8.RuneCountInString(s) <= snippetRunes {
		return s
	}
	r := []rune(s)
	return string(r[:snippetRunes]) + "…"
}
