package note

import (
	"os"

	"github.com/agonzaNET/memctl/internal/index"
)

// Deleted is the result of a successful Delete.
type Deleted struct {
	Path, Title string
	Dangling    int // other notes whose links to it now resolve to nothing
}

// Delete removes a note and its index entry (SPEC-0007 R24), as one tool
// write. git history keeps the file.
func (s *Service) Delete(resolve Resolver) (Deleted, error) {
	var out Deleted
	err := s.mutate(func(idx *index.Index) (Change, error) {
		rel, err := resolve(idx)
		if err != nil {
			return Change{}, err
		}
		abs, err := s.SafePath(rel)
		if err != nil {
			return Change{}, err
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return Change{}, err
		}
		n, _, _ := idx.Get(rel)
		linkers := idx.Linkers(rel)
		if err := s.removeFile(rel, hashOf(data)); err != nil {
			return Change{}, err
		}
		out = Deleted{Path: rel, Title: n.Title, Dangling: linkers}
		return Change{Tool: "note_delete", Title: n.Title, Paths: []string{rel}}, nil
	})
	if err != nil {
		return Deleted{}, err
	}
	return out, nil
}
