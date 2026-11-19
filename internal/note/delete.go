package note

// Deleted is the result of Delete.
type Deleted struct {
	Path, Title string
	Dangling    int
}

// Delete removes a note.
func (s *Service) Delete(resolve Resolver) (Deleted, error) { return Deleted{}, ErrConflict }
