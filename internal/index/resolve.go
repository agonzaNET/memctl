package index

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

// Resolve resolves a link target or ref by path, title, alias and file stem.
func (x *Index) Resolve(target string) Resolution { return Resolution{} }
