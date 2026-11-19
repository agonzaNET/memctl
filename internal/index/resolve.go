package index

import "strings"

// Status is the outcome of a resolution.
type Status string

const (
	Resolved  Status = "resolved"
	Ambiguous Status = "ambiguous"
	Dangling  Status = "dangling"
)

// Resolution is the result of resolving a link target or a tool ref
// (SPEC-0002 R3–R6, design D23).
type Resolution struct {
	Status Status
	// Path is the resolved note path when Status is Resolved.
	Path string
	// Candidates holds the matching paths, sorted, when Status is Ambiguous.
	Candidates []string
	// Stale is true when the note was found only through a path-like alias.
	Stale bool
}

// normName trims and case-folds a title, alias or stem (SPEC-0002 R3).
func normName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// normPath normalises a path-form target: trimmed, case-folded, one leading
// "./" or "/" and one trailing ".md" removed (SPEC-0002 R3).
func normPath(s string) string {
	s = normName(s)
	if strings.HasPrefix(s, "./") {
		s = s[2:]
	} else if strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	return strings.TrimSuffix(s, ".md")
}

// isPathLike reports whether an alias is an old path (SPEC-0002 R4).
func isPathLike(a string) bool {
	a = strings.TrimSpace(a)
	return strings.Contains(a, "/") || strings.HasSuffix(strings.ToLower(a), ".md")
}

// Resolve resolves a link target or ref by path, title, alias and file stem.
// The first step that matches any note decides (SPEC-0002 R4).
func (x *Index) Resolve(target string) Resolution {
	name, asPath := normName(target), normPath(target)
	if name == "" {
		return Resolution{Status: Dangling}
	}
	if r, ok := decide(x.paths[asPath], nil); ok {
		return r
	}
	if r, ok := decide(x.titles[name], nil); ok {
		return r
	}
	if r, ok := decide(x.aliases[name], x.pathAliases[asPath]); ok {
		return r
	}
	if r, ok := decide(x.stems[name], nil); ok {
		return r
	}
	return Resolution{Status: Dangling}
}

// decide turns the matches of one step into a resolution. stale holds the
// matches found only through path-like aliases; a note in both sets counts
// once and is not stale (SPEC-0002 R5, R6).
func decide(plain, stale map[string]struct{}) (Resolution, bool) {
	all := map[string]bool{}
	for p := range plain {
		all[p] = false
	}
	for p := range stale {
		if _, ok := all[p]; !ok {
			all[p] = true
		}
	}
	switch len(all) {
	case 0:
		return Resolution{}, false
	case 1:
		for p, s := range all {
			return Resolution{Status: Resolved, Path: p, Stale: s}, true
		}
	}
	set := map[string]struct{}{}
	for p := range all {
		set[p] = struct{}{}
	}
	return Resolution{Status: Ambiguous, Candidates: sortedSet(set)}, true
}
